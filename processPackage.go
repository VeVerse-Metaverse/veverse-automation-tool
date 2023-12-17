package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// uploadPackageFile uploads the package job results to the API for storage
func uploadPackageFile(job JobMetadata, path string, params map[string]string) error {
	Logger.Infof("uploading package file %s", path)

	if job.Package.Id == nil || job.Package.Id.IsNil() {
		return fmt.Errorf("invalid job package id")
	}

	// Build the request URL
	fileType := "pak"
	fileMime := url.QueryEscape("application/octet-stream")

	return uploadJobEntityFile(job, job.Package.Id, fileType, fileMime, path, "", params)
}

// processServerPackage runs the server package processing
func processServerPackage(job JobMetadata) (err error) {
	if err1 := updateJobStatus(job, JobStatusProcessing, ""); err1 != nil {
		Logger.Errorf("failed to update job status: %v", err1)
	}

	// Can not build server package for a mobile platform
	if job.Platform == "Android" || job.Platform == "IOS" {
		err = fmt.Errorf("invalid platform %s for server package", job.Platform)
		return
	}

	if job.Package.Id.IsNil() {
		err = fmt.Errorf("invalid job package id")
		return
	}

	if job.Package.Name == "" {
		err = fmt.Errorf("invalid job package name")
		return
	}

	if len(job.Files) == 0 {
		err = fmt.Errorf("job package has no files")
		return
	}

	// Create directories as required
	var pluginDir = filepath.Join(projectDir, "Plugins", job.Package.Name)
	var pluginContentDir = filepath.Join(pluginDir, "Content")
	if err1 := os.MkdirAll(pluginContentDir, 0755); err1 != nil {
		err = fmt.Errorf("failed to create a directory %s: %s", pluginContentDir, err1)
		return
	}

	var pluginDescriptorExists = false
	var pluginContentZipExists = false
	for _, f := range job.Files {
		if f.Type == "uplugin" {
			var err1 error = nil

			// Download a plugin descriptor
			var pluginDescriptorPath = filepath.Join(pluginDir, job.Package.Name+".uplugin")
			if f.Size != nil {
				err1 = downloadFile(pluginDescriptorPath, f.Url, *f.Size, true)
			} else {
				err1 = downloadFile(pluginDescriptorPath, f.Url, 0, true)
			}

			if err1 != nil {
				err = fmt.Errorf("failed to download a plugin descriptor: %s", err1)
			}

			pluginDescriptorExists = true
		} else if f.Type == "uplugin_content" {
			var err1 error = nil

			// Download a plugin content
			var pluginContentZipPath = filepath.Join(pluginDir, job.Package.Id.String()+".zip")
			if f.Size != nil {
				err1 = downloadFile(pluginContentZipPath, f.Url, *f.Size, true)
			} else {
				err1 = downloadFile(pluginContentZipPath, f.Url, 0, true)
			}

			if err1 != nil {
				err = fmt.Errorf("failed to download a plugin content: %v", err1)
				return
			}

			Logger.Infof("unzipping file %s to %s", pluginContentZipPath, pluginContentDir)

			if err1 = unzip(pluginContentZipPath, pluginContentDir); err != nil {
				err = fmt.Errorf("failed to unzip a plugin content: %s", err1)
				return
			}

			pluginContentZipExists = true
		}
	}

	if !pluginContentZipExists || !pluginDescriptorExists {
		err = fmt.Errorf("job is missing content zip %d or descriptor %d", pluginContentZipExists, pluginDescriptorExists)
		return
	}

	// Switch project to code version
	if err = switchProjectEngineVersion(ueVersionCode); err != nil {
		return fmt.Errorf("failed to switch engine version: %v", err)
	}

	commandLine := fmt.Sprintf("BuildCookRun -project=%s -noP4 -serverconfig=%s -unrealexe=%s -utf8output -cook -map=%s -unversionedcookedcontent -pak -dlcname=%s -DLCIncludeEngineContent -basedonreleaseversion=%s -distribution -compressed -package -noclient -server -serverplatform=%s -skipstage -VeryVerbose -BuildMachine", projectName, job.Configuration, editorPath, job.Package.Map, job.Package.Name, job.Package.Release, job.Platform)

	if result, err1 := runUnrealAutomationTool(strings.Split(commandLine, " ")); err != nil {
		err = fmt.Errorf("failed to run AutomationTool: %v", err1)
		err1 := reportJobLog(job, result.Warnings, result.Errors)
		if err1 != nil {
			Logger.Errorf("failed to report job log: %v", err1)
		}
		return
	}

	//region Upload job results

	// Mark the job with uploading status explicitly
	if err1 := updateJobStatus(job, JobStatusUploading, ""); err1 != nil {
		Logger.Errorf("failed to update job status: %v", err1)
	}

	//region Determine built package path

	packagePlatform := getPlatformName(job)

	// Package file name
	var packageName = fmt.Sprintf("%s%s-%s.pak", job.Package.Name, projectName, packagePlatform)

	// Path to the package file
	var packagePath string
	if job.Platform != "IOS" {
		// Most of the platforms have similar path to the resulting package file
		packagePath = fmt.Sprintf("%s/Plugins/%s/Saved/StagedBuilds/%s/%s/Plugins/%s/Content/Paks/%s/%s", projectDir, job.Package.Name, packagePlatform, projectName, job.Package.Name, packagePlatform, packageName)
	} else {
		// IOS has different resulting package file path comparing to other platforms
		packagePath = fmt.Sprintf("%s/Plugins/%s/Saved/StagedBuilds/%s/cookeddata/%s/plugins/%s/content/paks/%s/%s", projectDir, job.Package.Name, packagePlatform, strings.ToLower(projectName), strings.ToLower(job.Package.Name), strings.ToLower(packagePlatform), packageName)
	}

	//endregion

	err1 := uploadPackageFile(job, packagePath, nil)
	if err1 != nil {
		err = fmt.Errorf("failed to upload package: %v", err1)
		return
	}

	//endregion

	return
}

// processClientPackage runs the client package processing
func processClientPackage(job JobMetadata) (err error) {
	if err1 := updateJobStatus(job, JobStatusProcessing, ""); err1 != nil {
		Logger.Errorf("failed to update job status: %v", err1)
	}

	if job.Package.Id.IsNil() {
		err = fmt.Errorf("invalid job package id")
		return
	}

	if job.Package.Name == "" {
		err = fmt.Errorf("invalid job package name")
		return
	}

	if len(job.Files) == 0 {
		err = fmt.Errorf("job package has no files")
		return
	}

	// Create directories as required
	var pluginDir = filepath.Join(projectDir, "Plugins", job.Package.Name)
	var pluginContentDir = filepath.Join(pluginDir, "Content")
	if err1 := os.MkdirAll(pluginContentDir, 0755); err1 != nil {
		err = fmt.Errorf("failed to create a directory %s: %s", pluginContentDir, err1)
		return
	}

	var pluginDescriptorExists = false
	var pluginContentZipExists = false
	for _, f := range job.Files {
		if f.Type == "uplugin" {
			var err1 error = nil

			// Download a plugin descriptor
			var pluginDescriptorPath = filepath.Join(pluginDir, job.Package.Name+".uplugin")
			if f.Size != nil {
				err1 = downloadFile(pluginDescriptorPath, f.Url, *f.Size, true)
			} else {
				err1 = downloadFile(pluginDescriptorPath, f.Url, 0, true)
			}

			if err1 != nil {
				err = fmt.Errorf("failed to download a plugin descriptor: %s", err1)
			}

			pluginDescriptorExists = true
		} else if f.Type == "uplugin_content" {
			var err1 error = nil

			// Download a plugin content
			var pluginContentZipPath = filepath.Join(pluginDir, job.Package.Id.String()+".zip")
			if f.Size != nil {
				err1 = downloadFile(pluginContentZipPath, f.Url, *f.Size, true)
			} else {
				err1 = downloadFile(pluginContentZipPath, f.Url, 0, true)
			}

			if err1 != nil {
				err = fmt.Errorf("failed to download a plugin content: %v", err1)
				return
			}

			Logger.Infof("unzipping file %s to %s", pluginContentZipPath, pluginContentDir)

			if err1 = unzip(pluginContentZipPath, pluginContentDir); err != nil {
				err = fmt.Errorf("failed to unzip a plugin content: %s", err1)
				return
			}

			pluginContentZipExists = true
		}
	}

	if !pluginContentZipExists || !pluginDescriptorExists {
		err = fmt.Errorf("job is missing content zip %d or descriptor %d", pluginContentZipExists, pluginDescriptorExists)
		return
	}

	var commandLine string
	if job.Configuration == "Shipping" {
		// Add -distribution for the shipping builds
		commandLine = fmt.Sprintf(
			"BuildCookRun -project=%s -noP4 -clientconfig=%s -unrealexe=%s -utf8output -platform=%s -cook -map=%s -unversionedcookedcontent -pak -dlcname=%s -DLCIncludeEngineContent -basedonreleaseversion=%s -compressed -package -skipstage -VeryVerbose -BuildMachine -distribution",
			projectName,
			job.Configuration,
			editorPath,
			job.Platform,
			job.Package.Map,
			job.Package.Name,
			job.Package.Release,
		)
	} else {
		commandLine = fmt.Sprintf(
			"BuildCookRun -project=%s -noP4 -clientconfig=%s -unrealexe=%s -utf8output -platform=%s -cook -map=%s -unversionedcookedcontent -pak -dlcname=%s -DLCIncludeEngineContent -basedonreleaseversion=%s -compressed -package -skipstage -VeryVerbose -BuildMachine",
			projectName,
			job.Configuration,
			editorPath,
			job.Platform,
			job.Package.Map,
			job.Package.Name,
			job.Package.Release,
		)
	}

	if result, err1 := runUnrealAutomationTool(strings.Split(commandLine, " ")); err1 != nil {
		err = fmt.Errorf("failed to run AutomationTool: %v", err1)
		err1 := reportJobLog(job, result.Warnings, result.Errors)
		if err1 != nil {
			Logger.Errorf("failed to report job log: %v", err1)
		}
		return
	}

	//region Upload job results

	// Mark the job with uploading status explicitly
	if err1 := updateJobStatus(job, JobStatusUploading, ""); err1 != nil {
		Logger.Errorf("failed to update job status: %v", err1)
	}

	//region Determine built package path

	var packagePlatform string
	if job.Platform == "Win64" {
		// Windows packages have platform name part equal to "Win64" although resulting package names have the "Windows" platform name part
		if job.Deployment == "Server" {
			packagePlatform = "WindowsServer"
		} else {
			packagePlatform = "Windows"
		}
	} else if job.Platform == "IOS" || job.Platform == "Android" {
		// Mobile server builds have no sense
		packagePlatform = job.Platform
	} else {
		// Other supported platforms (Linux, Mac) can have server builds
		if job.Deployment == "Server" {
			packagePlatform = job.Platform + "Server"
		} else {
			packagePlatform = job.Platform
		}
	}

	// Package file name
	var packageName = fmt.Sprintf("%s%s-%s.pak", job.Package.Name, projectName, packagePlatform)

	// Path to the package file
	var packagePath string
	if job.Platform != "IOS" {
		// Most of the platforms have similar path to the resulting package file
		packagePath = fmt.Sprintf("%s/Plugins/%s/Saved/StagedBuilds/%s/%s/Plugins/%s/Content/Paks/%s/%s", projectDir, job.Package.Name, packagePlatform, projectName, job.Package.Name, packagePlatform, packageName)
	} else {
		// IOS has different resulting package file path comparing to other platforms
		packagePath = fmt.Sprintf("%s/Plugins/%s/Saved/StagedBuilds/%s/cookeddata/%s/plugins/%s/content/paks/%s/%s", projectDir, job.Package.Name, packagePlatform, strings.ToLower(projectName), strings.ToLower(job.Package.Name), strings.ToLower(packagePlatform), packageName)
	}

	//endregion

	err1 := uploadPackageFile(job, packagePath, nil)
	if err1 != nil {
		err = fmt.Errorf("failed to upload package: %v", err1)
		return
	}

	//endregion

	return
}
