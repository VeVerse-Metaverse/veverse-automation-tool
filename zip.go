package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func unzip(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open a zip file: %s", err.Error())
	}
	defer func() {
		if err := r.Close(); err != nil {
			Logger.Fatalf("failed to close a zip file: %s", err)
		}
	}()

	err = os.MkdirAll(dst, 0755)
	if err != nil {
		return fmt.Errorf("failed to make a directory tree: %s", err.Error())
	}

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open a zipped file: %s", err.Error())
		}
		defer func() {
			if err := rc.Close(); err != nil {
				Logger.Fatalf("failed to close a zipped file: %s", err.Error())
			}
		}()

		path := filepath.Join(dst, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			if err = os.MkdirAll(path, f.Mode()); err != nil {
				Logger.Errorf("failed to create a directory %s: %s", path, err.Error())
			}
		} else {
			if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				Logger.Errorf("failed to create a directory %s: %s", filepath.Dir(path), err.Error())
			}

			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					Logger.Fatalf("failed to close a file %s: %s", path, err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return fmt.Errorf("failed to copy file %s: %s", path, err.Error())
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return fmt.Errorf("failed to extract a file: %s", err.Error())
		}
	}

	return nil
}
