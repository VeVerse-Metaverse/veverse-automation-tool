package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

// downloadFile downloads file to the filepath from url
func downloadFile(filepath string, url string, size int64, force bool) (err error) {
	// Check if file exists
	stat, err := os.Stat(filepath)
	if err == nil {
		if size > 0 && stat.Size() == size {
			if !force {
				Logger.Infof("skipping download, file exists: %s, size matches: %d", filepath, size)
				return nil
			} else {
				// delete file
				err = os.Remove(filepath)
				if err != nil {
					return fmt.Errorf("failed to delete file: %s, error: %s", filepath, err.Error())
				}
			}
		}
	}

	Logger.Infof("downloading file %s of size %d to %s", url, size, filepath)

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to send a HTTP GET request: %s\n", err.Error())
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Printf("error closing http response body: %s\n", err.Error())
		}
	}(resp.Body)

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file %s to %s: bad status: %s\n", url, filepath, resp.Status)
	}

	// Create the dir
	dir := path.Dir(filepath)
	err = os.MkdirAll(dir, 0750)
	if err != nil {
		return fmt.Errorf("failed to create a directory %s: %s\n", dir, err.Error())
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create a file downloaded %s to %s: %s\n", url, filepath, err.Error())
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			log.Printf("error closing file: %s\n", err.Error())
		}
	}(out)

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write a file downloaded %s to %s: %s\n", url, filepath, err.Error())
	}

	// Change a file mode for known binaries to make them executable
	for s, b := range binarySuffixes {
		if b && strings.HasSuffix(filepath, s) {
			err = os.Chmod(filepath, 0755)
			if err != nil {
				log.Printf("failed to change file mode for %s: %s\n", filepath, err.Error())
			}
		}
	}

	return nil
}
