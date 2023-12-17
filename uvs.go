package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runUnrealVersionSelector runs the Unreal Version Selector, logs its output to stdout and waits for it to exit completing the automation command then returns
func runUnrealVersionSelector(args []string) error {
	if _, err := os.Stat(uvsPath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no VersionSelector binary found at %s", uvsPath)
	}

	uvsDir := filepath.Dir(uvsPath)
	cmd := exec.Command(uvsPath, args...)
	cmd.Dir = uvsDir
	rd, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to attach to the VersionSelector stdout pipe: %v", err)
	}

	go func() {
		defer func(rd io.ReadCloser) {
			err := rd.Close()
			if err != nil {
				Logger.Errorf("failed to close rd: %s", err)
			}
		}(rd)

		b := make([]byte, 2048)
		for {
			nn, err := rd.Read(b)
			if nn > 0 {
				s := string(b[:nn])
				Logger.Infof("%s", strings.ReplaceAll(strings.ReplaceAll(s, "\\", "/"), "\r\n", ""))
			}
			if err != nil {
				if err == io.EOF {
					Logger.Infof("VersionSelector process has exited")
				} else {
					Logger.Errorf("failed to read the UVS process pipe: %v", err)
				}

				return // Exit goroutine
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cmd.Start() error: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("cmd.Wait() error: %v", exitError)
		} else {
			return fmt.Errorf("cmd.Wait() error: %v", exitError)
		}
	}

	return nil
}
