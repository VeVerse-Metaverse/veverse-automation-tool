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
	goRuntime "runtime"
	"strings"
)

func getReleaseIgnoredFiles() ([]string, error) {
	const ignoreFile = ".veverse-automation-ignore"

	file, err := os.OpenFile(ignoreFile, os.O_RDONLY, 0644)
	if err != nil {
		if err != os.ErrNotExist {
			return nil, fmt.Errorf("failed to open %s file: %v", ignoreFile, err)
		} else {
			return nil, nil
		}
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read from %s file: %v", ignoreFile, err)
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

// processServerRelease runs the server release processing
func processServerRelease(job JobMetadata) (err error) {
	if err1 := updateJobStatus(job, JobStatusProcessing, ""); err1 != nil {
		Logger.Errorf("failed to update job status: %v", err1)
	}

	// Can not build server release for a mobile platform
	if job.Platform == "Android" || job.Platform == "IOS" {
		err = fmt.Errorf("invalid platform %s for server release", job.Platform)
		return
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

		if //goland:noinspection GoBoolExpressions
		goRuntime.GOOS == "darwin" {
			targetBranch = targetBranch + "-mac"
		}

		// Check if we need to switch branches
		if currentBranch != targetBranch {
			Logger.Infof("current branch %s, checking out branch %s", currentBranch, targetBranch)
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

	// Switch project to code version
	if err = switchProjectEngineVersion(ueVersionCode); err != nil {
		return fmt.Errorf("failed to switch engine version: %v", err)
	}

	projectStagingDir := filepath.Join(projectDir, "Saved", "StagedBuilds")
	commandLine := fmt.Sprintf(
		"BuildCookRun -project=%s -noP4 -unrealexe=%s -noclient -server -serverconfig=%s -serverplatform=%s -ini:Game:[/Script/UnrealEd.ProjectPackagingSettings]:BlueprintNativizationMethod=Disabled -build -cook -unversionedcookedcontent -SkipCookingEditorContent -map=%s -pak -compressed -package -createreleaseversion=%s -stage -stagingdirectory=%s -VeryVerbose -NoCodeSign -BuildMachine -AllowCommandletRendering -utf8output",
		projectName,
		editorPath,
		job.Configuration,
		job.Platform,
		job.Release.Map,
		job.Release.ContentVersion,
		projectStagingDir,
	)

	if job.Configuration == "Shipping" {
		commandLine += " -CrashReporter -nodebug -nodebuginfo -distribution -prereqs"
	} else if job.Configuration == "Development" || job.Configuration == "DebugGame" || job.Configuration == "Debug" || job.Configuration == "Test" {
		commandLine += " -debug"
	}

	if _, err1 := runUnrealAutomationTool(strings.Split(commandLine, " ")); err1 != nil {
		err = fmt.Errorf("failed to run AutomationTool: %v", err1)
		return
	}

	ignoredFiles, err2 := getReleaseIgnoredFiles()
	if err2 != nil {
		err = fmt.Errorf("failed to get ignored files: %v", err2)
		return
	}

	platformStagingDir := filepath.Join(projectStagingDir, getPlatformName(job))
	files, err := listFilesRecursive(platformStagingDir, ignoredFiles)

	for _, file := range files {
		path := filepath.Join(platformStagingDir, file)
		originalPath := filepath.ToSlash(file)
		err = uploadReleaseFile(job, path, originalPath, nil)
		if err != nil {
			return fmt.Errorf("failed to upload release file: %v", err)
		}
	}

	return nil
}

// processClientRelease runs the client release processing
func processClientRelease(job JobMetadata) (err error) {
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

	// Checkout the matching tag
	tag := job.Release.CodeVersion

	err = gitPull(r)
	if err != nil {
		return fmt.Errorf("failed to pull repository: %v", err)
	}

	s, err := gitLatestTag(r)
	fmt.Printf(s)

	// Checkout the branch corresponding to the job configuration
	//if targetBranch, ok := configurationBranchMapping[job.Configuration]; !ok {
	//	return fmt.Errorf("failed to map the job configuration %s to a branch", job.Configuration)
	//} else {
	// Get the current branch
	hash, err := gitTag(r, tag)
	if err != nil || hash == nil {
		return fmt.Errorf("failed to get the tag ref: %v", err)
	}

	// Check if we need to switch branches
	//if currentBranch != targetBranch {
	if err1 := vcsGit.TagSync(projectDir, tag); err1 != nil {
		return fmt.Errorf("failed to checkout tag: %v", err1)
	}
	//if err = gitCheckoutCommit(r, *hash); err != nil {
	//	return fmt.Errorf("failed to checkout the target branch: %v", err)
	//}
	//}
	//}

	// Update the code base to the branch head
	//err = gitPull(r)
	//if err != nil {
	//	return fmt.Errorf("failed to update the repo: %v", err)
	//}

	// Switch project to code version
	if err = switchProjectEngineVersion(ueVersionCode); err != nil {
		return fmt.Errorf("failed to switch engine version: %v", err)
	}

	projectStagingDir := filepath.Join(projectDir, "Saved", "StagedBuilds")
	commandLine := fmt.Sprintf(
		"BuildCookRun -project=%s -noP4 -unrealexe=%s -clientconfig=%s -platform=%s -ini:Game:[/Script/UnrealEd.ProjectPackagingSettings]:BlueprintNativizationMethod=Disabled -build -cook -unversionedcookedcontent -SkipCookingEditorContent -map=%s -pak -compressed -package -createreleaseversion=%s -stage -stagingdirectory=%s -VeryVerbose -NoCodeSign -BuildMachine -AllowCommandletRendering -utf8output",
		projectName,
		editorPath,
		job.Configuration,
		job.Platform,
		job.Release.Map,
		job.Release.ContentVersion,
		projectStagingDir,
	)

	if job.Configuration == "Shipping" {
		commandLine += " -CrashReporter -nodebug -nodebuginfo -distribution -prereqs"
	} else if job.Configuration == "Development" || job.Configuration == "DebugGame" || job.Configuration == "Debug" || job.Configuration == "Test" {
		commandLine += " -debug"
	}

	if _, err1 := runUnrealAutomationTool(strings.Split(commandLine, " ")); err1 != nil {
		err = fmt.Errorf("failed to run AutomationTool: %v", err1)
		return
	}

	ignore, err2 := getReleaseIgnoredFiles()
	if err2 != nil {
		return err2
	}

	platformStagingDir := filepath.Join(projectStagingDir, getPlatformName(job))
	files, err := listFilesRecursive(platformStagingDir, ignore)

	var releaseArchiveFileMap = map[string]string{}
	for _, file := range files {
		path := filepath.Join(platformStagingDir, file)
		originalPath := filepath.ToSlash(file)
		releaseArchiveFileMap[path] = originalPath
		err = uploadReleaseFile(job, path, originalPath, nil)
		if err != nil {
			return fmt.Errorf("failed to upload release file: %v", err)
		}
	}

	zipName := fmt.Sprintf("%s-%s-%s-%s-%s.zip", job.Release.AppName, job.Release.Version, job.Deployment, job.Configuration, job.Platform)
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

	err = uploadReleaseArchiveFile(job, zipName, zipName, nil)
	if err != nil {
		return fmt.Errorf("failed to upload release archive file: %v", err)
	}

	return nil
}

// uploadReleaseFile uploads the release job results to the API for storage
func uploadReleaseFile(job JobMetadata, path string, originalPath string, params map[string]string) error {
	Logger.Infof("uploading release file %s", originalPath)

	if job.Release.Id == nil || job.Release.Id.IsNil() {
		return fmt.Errorf("invalid job package id")
	}

	var (
		fileType = "release"
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

// uploadReleaseFile uploads the release archive file to the API for storage
func uploadReleaseArchiveFile(job JobMetadata, path string, originalPath string, params map[string]string) error {
	Logger.Infof("uploading release archive file %s", path)

	if job.Release.Id == nil || job.Release.Id.IsNil() {
		return fmt.Errorf("invalid job package id")
	}

	var (
		fileType = "release-archive"
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
