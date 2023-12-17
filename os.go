package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func getPlatformName(job JobMetadata) (platform string) {
	if job.Platform == "Win64" {
		// Windows packages have platform name part equal to "Win64" although resulting package names have the "Windows" platform name part
		if job.Deployment == "Server" {
			platform = "WindowsServer"
		} else {
			platform = "Windows"
		}
	} else if job.Platform == "IOS" || job.Platform == "Android" {
		// Mobile server builds have no sense
		platform = job.Platform
	} else {
		// Other supported platforms (Linux, Mac) can have server builds
		if job.Deployment == "Server" {
			platform = job.Platform + "Server"
		} else {
			platform = job.Platform
		}
	}

	return
}

func listFilesRecursive(dir string, fileList []string) ([]string, error) {
	files := []string{}

	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		skip := false

		// Loop : Ignore or include Files & Folders
		for _, i := range fileList {
			// Ignoring list records
			if strings.Contains(filepath.ToSlash(path), filepath.ToSlash(i)) {
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
