package main

import (
	"context"
	"fmt"
	"github.com/gabriel-vasile/mimetype"
	"github.com/mholt/archiver/v4"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func getSdkIncludeFiles(err error) ([]string, error) {
	const includeFile = ".veverse-automation-sdk-include"

	file, err := os.OpenFile(includeFile, os.O_RDONLY, 0644)
	if err != nil {
		if err != os.ErrNotExist {
			return nil, fmt.Errorf("failed to open %s file: %v", includeFile, err)
		} else {
			return nil, nil
		}
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read from %s file: %v", includeFile, err)
	}

	// Split lines
	list := strings.Split(string(data), "\n")

	// Filter empty lines and comments
	var result []string
	for _, str := range list {
		if str != "" || !strings.HasPrefix(str, "#") {
			result = append(result, strings.TrimSpace(str))
		}
	}

	return result, nil
}

func listSdkFilesRecursive(dir string, fileList []string, projectName string) ([]string, error) {
	files := []string{}

	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		skip := false

		// Loop : Ignore or include Files & Folders
		for _, i := range fileList {
			left := filepath.ToSlash(path)
			right := filepath.ToSlash(filepath.Join("UnrealEngine", projectName, i))

			// Including list records
			if strings.Contains(left, right) {
				skip = false
				break
			} else {
				skip = true
			}
		}

		if !skip {
			f, err = os.Stat(path)
			if err != nil {
				return fmt.Errorf("failed to get file info: %v", err)
			}

			fileMode := f.Mode()
			if fileMode.IsRegular() {
				relativePath, err := filepath.Rel(dir, path)
				if err != nil {
					return err
				}

				files = append(files, relativePath)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func switchProjectEngineVersion(versionOrFolderPath string) error {
	Logger.Infof("switching engine version to %s", versionOrFolderPath)

	// Get path to the project descriptor file
	projectDescriptorPath := filepath.Join(projectDir, projectName+".uproject")

	// Switch engine version
	err := runUnrealVersionSelector([]string{"-switchversionsilent", projectDescriptorPath, versionOrFolderPath})
	if err != nil {
		return fmt.Errorf("failed to switch engine version: %v", err)
	}

	return nil
}

// processSdkRelease runs the SDK release processing
func processSdkRelease(job JobMetadata) (err error) {
	if err1 := updateJobStatus(job, JobStatusProcessing, ""); err1 != nil {
		Logger.Errorf("failed to update job status: %v", err1)
	}

	if job.Release.Id.IsNil() {
		err = fmt.Errorf("invalid job release id")
		return
	}

	r, err := gitRepo(projectDir)
	if err != nil {
		return fmt.Errorf("failed to open the repo at %s: %v", projectDir, err)
	}

	// Checkout the branch corresponding to the job configuration
	if targetBranch, ok := configurationBranchMapping[job.Configuration]; !ok {
		return fmt.Errorf("failed to map the job configuration %s to a branch", job.Configuration)
	} else {
		// Get the current branch
		currentBranch, err := gitBranch(r)
		if err != nil {
			return fmt.Errorf("failed to get the current branch name: %v", err)
		}

		// Check if we need to switch branches
		if currentBranch != targetBranch {
			if err = gitCheckout(r, targetBranch); err != nil {
				return fmt.Errorf("failed to checkout the target branch: %v", err)
			}
		}
	}

	// Update the code base to the branch head
	err = gitPull(r)
	if err != nil {
		return fmt.Errorf("failed to update the repo: %v", err)
	}

	// Switch project to marketplace version
	if err = switchProjectEngineVersion(ueVersionMarketplace); err != nil {
		return fmt.Errorf("failed to switch engine version: %v", err)
	}

	// Build editor
	commandLine := fmt.Sprintf("BuildEditor -project=%s", projectName)
	if _, err1 := runUnrealAutomationTool(strings.Split(commandLine, " ")); err1 != nil {
		err = fmt.Errorf("failed to run AutomationTool: %v", err1)
		return
	}

	// Build development release
	projectStagingDir := filepath.Join(projectDir, "Saved", "StagedBuilds")
	commandLine = fmt.Sprintf(
		"BuildCookRun -project=%s -noP4 -unrealexe=%s -clientconfig=Development -serveconfig=Development -platform=%s -ini:Game:[/Script/UnrealEd.ProjectPackagingSettings]:BlueprintNativizationMethod=Disabled -build -cook -unversionedcookedcontent -SkipCookingEditorContent -map=%s -pak -compressed -package -createreleaseversion=%s -stage -stagingdirectory=%s -VeryVerbose -NoCodeSign -BuildMachine -AllowCommandletRendering -utf8output -nodebug -nodebuginfo",
		projectName,
		editorPath,
		job.Platform,
		job.Release.Map,
		job.Release.ContentVersion,
		projectStagingDir,
	)

	if _, err1 := runUnrealAutomationTool(strings.Split(commandLine, " ")); err1 != nil {
		err = fmt.Errorf("failed to run AutomationTool: %v", err1)
		return
	}

	includeFiles, err2 := getSdkIncludeFiles(err)
	if err2 != nil {
		return err2
	}

	platformStagingDir := projectDir
	files, err := listSdkFilesRecursive(platformStagingDir, includeFiles, projectName)

	var releaseArchiveFileMap = map[string]string{}
	for _, file := range files {
		path := filepath.Join(platformStagingDir, file)
		originalPath := filepath.ToSlash(file)
		releaseArchiveFileMap[path] = originalPath
		//err = uploadReleaseFile(job, path, originalPath, nil)
		//if err != nil {
		//	return fmt.Errorf("failed to upload SDK release file: %v", err)
		//}
	}

	zipName := fmt.Sprintf("%s-%s-%s-%s-%s-SDK.zip", job.Release.AppName, job.Release.Version, job.Deployment, job.Configuration, job.Platform)
	zip, err := os.Create(zipName)
	if err != nil {
		return fmt.Errorf("failed to create a zip file: %v", err)
	}
	defer func(zip *os.File) {
		err := zip.Close()
		if err != nil {
			Logger.Errorf("failed to close a zip file: %v", err)
		}
	}(zip)

	format := archiver.CompressedArchive{
		Archival: archiver.Zip{},
	}

	releaseArchiveFiles, err := archiver.FilesFromDisk(nil, releaseArchiveFileMap)
	if err != nil {
		return fmt.Errorf("failed to enumerate release archive files to zip: %v", err)
	}

	err = format.Archive(context.Background(), zip, releaseArchiveFiles)
	if err != nil {
		return fmt.Errorf("failed to zip release archive files: %v", err)
	}

	err = uploadSdkArchiveFile(job, zipName, zipName, nil)
	if err != nil {
		return fmt.Errorf("failed to upload release archive file: %v", err)
	}

	return nil
}

// uploadSdkArchiveFile uploads the release archive file to the API for storage
func uploadSdkArchiveFile(job JobMetadata, path string, originalPath string, params map[string]string) error {
	Logger.Infof("uploading release archive file %s", path)

	if job.Release.Id == nil || job.Release.Id.IsNil() {
		return fmt.Errorf("invalid job package id")
	}

	var (
		fileType = "release-archive-sdk"
		fileMime = "application/octet-stream"
	)

	// Try to detect MIME
	pMIME, err := mimetype.DetectFile(path)
	if err != nil {
		Logger.Warningf("failed to detect MIME type of %s", path)
	} else {
		fileMime = pMIME.String()
	}

	fileMime = url.QueryEscape(fileMime)

	return uploadJobEntityFile(job, job.Release.Id, fileType, fileMime, path, originalPath, params)
}
