package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// runWailsCli runs the Wails CLI, logs its output to stdout and waits for it to exit completing the automation command then returns
func runWailsCli(args []string, workDir string) error {
	if _, err := os.Stat(wailsPath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no wails binary found at %s", wailsPath)
	}

	cmd := exec.Command(wailsPath, args...)
	cmd.Dir = workDir
	rd, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to attach to the wails stdout pipe: %v", err)
	}

	var exitCode = -1

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
				if strings.HasPrefix(s, "Built") {
					exitCode = 0
				} else if strings.Contains(s, "exit status") {
					i0 := strings.Index(s, "status ") + len("status ")
					i1 := i0 + 1
					if i0 > 0 && i1 > 0 {
						code := s[i0:i1]
						if exitCode, err = strconv.Atoi(code); err != nil {
							Logger.Errorf("failed to convert code %s to int: %v", code, err)
							exitCode = -1
						}
					}
				}
			}
			if err != nil {
				if err == io.EOF {
					Logger.Infof("wails process has exited")
				} else {
					Logger.Errorf("failed to read the wails process pipe: %v", err)
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

	if exitCode > 0 {
		return fmt.Errorf("wails exit code: %d", exitCode)
	}

	return nil
}
