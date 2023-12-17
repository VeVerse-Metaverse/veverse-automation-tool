package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	goRuntime "runtime"
)

// processClientLauncher runs the client launcher processing
func processClientLauncher(job JobMetadata) (err error) {
	if err1 := updateJobStatus(job, JobStatusProcessing, ""); err1 != nil {
		Logger.Errorf("failed to update job status: %v", err1)
	}

	if job.App.Id.IsNil() {
		err = fmt.Errorf("invalid job app id")
		return
	}

	if job.App.Name == "" {
		err = fmt.Errorf("invalid job app name")
		return
	}

	if len(job.Files) == 0 {
		err = fmt.Errorf("job app has no files, must have app-icons (image/png and image/x-icon)")
		return
	}

	// Create directories as required
	var buildDir = filepath.Join(launcherDir, "build")
	var buildBinaryDir = filepath.Join(launcherDir, "build", "bin")
	var buildWindowsDir = filepath.Join(buildDir, "windows")
	if err1 := os.MkdirAll(buildWindowsDir, 0755); err1 != nil {
		err = fmt.Errorf("failed to create a directory %s: %s", buildWindowsDir, err1)
		return
	}

	var appIconPngExists = false
	var appIconIcoExists = false
	for _, f := range job.Files {
		if f.Type == "image-app-icon" && *f.Mime == "image/png" {
			var err1 error = nil

			// Download an app icon PNG file
			var appIconPath = filepath.Join(buildDir, "appicon.png")
			if f.Size != nil {
				err1 = downloadFile(appIconPath, f.Url, *f.Size, true)
			} else {
				err1 = downloadFile(appIconPath, f.Url, 0, true)
			}

			if err1 != nil {
				err = fmt.Errorf("failed to download a plugin descriptor: %s", err1)
			}

			appIconPngExists = true
		} else if f.Type == "image-app-icon" && *f.Mime == "image/x-icon" {
			var err1 error = nil

			// Download an app icon in ICO format
			var appIconPath = filepath.Join(buildDir, "windows", "icon.ico")
			if f.Size != nil {
				err1 = downloadFile(appIconPath, f.Url, *f.Size, true)
			} else {
				err1 = downloadFile(appIconPath, f.Url, 0, true)
			}

			if err1 != nil {
				err = fmt.Errorf("failed to download a plugin content: %v", err1)
				return
			}

			appIconIcoExists = true
		}
	}

	if !appIconPngExists || !appIconIcoExists {
		err = fmt.Errorf("app icon is missing: png=%v, ico=%v", appIconPngExists, appIconIcoExists)
		return
	}

	var commandLine []string
	var appOriginalPath string
	var outPath string
	if job.Platform == "Win64" {
		appOriginalPath = job.App.Name + "Launcher.exe"
		outPath = filepath.Join(buildBinaryDir, appOriginalPath)
		commandLine = []string{
			`build`,
			`-nocolour`,
			//`-platform`, `"windows/amd64"`,
			`-clean`,
			`-ldflags`, fmt.Sprintf(`-s -w -X games.launch.launcher/config.AppId=%s`, job.App.Id),
			`-tags`, fmt.Sprintf(`%s`, job.Configuration),
			//`-upx`,
			//`-obfuscated`,
			//`-nsis`,
			`-o`, fmt.Sprintf(`%s`, filepath.Base(outPath)),
			`-v`, `2`}
	} else if job.Platform == "Mac" {
		appOriginalPath = job.App.Name + "Launcher"
		outPath = filepath.Join(buildBinaryDir, appOriginalPath)
		commandLine = []string{
			`build`,
			//`-platform "darwin/arm64"`,
			`-nocolour`,
			`-clean`,
			`-ldflags`, fmt.Sprintf(`"-s -w -X games.launch.launcher/config.AppId=%s"`, job.App.Id),
			`-tags`, fmt.Sprintf(`%s`, job.Configuration),
			//`-upx`,
			//`-obfuscated`,
			`-o`, fmt.Sprintf(`%s`, filepath.Base(outPath)),
			`-v`, `2`}
	} else {
		err = fmt.Errorf("unsupported platform: %s", job.Platform)
		return
	}

	if err1 := runWailsCli(commandLine, launcherDir); err1 != nil {
		err = fmt.Errorf("failed to run wails cli: %v", err1)
		return
	}

	if //goland:noinspection GoBoolExpressions
	goRuntime.GOOS == "windows" && certFile != "" && certPassword != "" {
		commandLine = []string{
			`sign`,
			`/tr`,
			`http://timestamp.digicert.com`,
			`/td`,
			`sha1`,
			`/fd`,
			`sha1`,
			`/f`,
			certFile,
			`/p`,
			certPassword,
			outPath,
		}
		if err1 := runSignTool(commandLine, buildBinaryDir); err1 != nil {
			err = fmt.Errorf("failed to run sign tool: %v", err1)
			return
		}

		commandLine = []string{
			`sign`,
			`/tr`,
			`http://timestamp.digicert.com`,
			`/as`,
			`/td`,
			`sha256`,
			`/fd`,
			`sha256`,
			`/f`,
			certFile,
			`/p`,
			certPassword,
			outPath,
		}
		if err1 := runSignTool(commandLine, buildBinaryDir); err1 != nil {
			err = fmt.Errorf("failed to run sign tool: %v", err1)
			return
		}
	}

	//region Upload job results

	// Mark the job with uploading status explicitly
	if err1 := updateJobStatus(job, JobStatusUploading, ""); err1 != nil {
		Logger.Errorf("failed to update job status: %v", err1)
	}

	if _, err := os.Stat(outPath); errors.Is(err, os.ErrNotExist) {
		if err1 := updateJobStatus(job, JobStatusUploading, ""); err1 != nil {
			Logger.Errorf("failed to update job status: %v", err1)
		}
		return fmt.Errorf("no result launcher binary found at %s: %v", outPath, err)
	}

	// Path to the package file
	err1 := uploadLauncherFile(job, outPath, appOriginalPath, nil)
	if err1 != nil {
		err = fmt.Errorf("failed to upload package: %v", err1)
		return
	}

	//endregion

	return
}

// uploadLauncherFile uploads the launcher job results to the API for storage
func uploadLauncherFile(job JobMetadata, path string, originalPath string, params map[string]string) error {
	Logger.Infof("uploading launcher file %s", path)

	if job.App.Id == nil || job.App.Id.IsNil() {
		return fmt.Errorf("invalid job app id")
	}

	// Build the request URL
	fileType := "app-launcher"
	var fileMime string
	if //goland:noinspection GoBoolExpressions
	goRuntime.GOOS == "windows" {
		fileMime = url.QueryEscape("application/vnd.microsoft.portable-executable")
	} else {
		fileMime = url.QueryEscape("application/octet-stream")
	}

	return uploadJobEntityFile(job, job.App.Id, fileType, fileMime, path, originalPath, params)
}
