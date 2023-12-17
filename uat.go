package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// UnrealAutomationToolResult is the result of running the Unreal Automation Tool
type UnrealAutomationToolResult struct {
	ExitCode int
	Warnings []string
	Errors   []string
}

// Process Unreal Automation Tool result
func (r *UnrealAutomationToolResult) Process() {
	if r.ExitCode != 0 {
		for k, e := range r.Errors {
			pattern := "^Log(\\w+)\\:\\s?(Error)\\:\\s?(.+)$"

			reg, err := regexp.Compile(pattern)
			if err != nil {
				log.Printf("failed to compile regex: %s", err)
			}

			matches := reg.FindStringSubmatch(e)

			if len(matches) == 4 {
				r.Errors[k] = fmt.Sprintf("Category: %s\nSeverity: %s\nMessage: %s", matches[1], matches[2], matches[3])
			}
		}

	}
}

// runUnrealAutomationTool runs the Unreal Automation Tool, logs its output to stdout and waits for it to exit completing the automation command then returns
func runUnrealAutomationTool(args []string) (result UnrealAutomationToolResult, err error) {
	result = UnrealAutomationToolResult{
		ExitCode: 0,
		Warnings: nil,
		Errors:   nil,
	}

	if _, err = os.Stat(uatPath); errors.Is(err, os.ErrNotExist) {
		return result, fmt.Errorf("no AutomationTool binary found at %s", uatPath)
	}

	uatDir := filepath.Dir(uatPath)
	cmd := exec.Command(uatPath, args...)
	cmd.Dir = uatDir
	rd, err := cmd.StdoutPipe()
	if err != nil {
		return result, fmt.Errorf("failed to attach to the AutomationTool stdout pipe: %v", err)
	}

	var exitCode = -1

	var wg sync.WaitGroup

	go func() {
		wg.Add(1)

		logFile, err := os.Create("uat.log")
		if err != nil {
			Logger.Errorf("failed to create uat.log: %s", err)
		}

		defer func(f *os.File) {
			err := f.Close()
			if err != nil {
				Logger.Errorf("failed to close uat.log: %s", err)
			}
		}(logFile)

		defer func(rd io.ReadCloser) {
			err := rd.Close()
			if err != nil {
				Logger.Errorf("failed to close rd: %s", err)
			}
		}(rd)

		reader := bufio.NewReader(rd)
		for {
			line, err := reader.ReadString('\n')
			s := strings.ReplaceAll(strings.ReplaceAll(line, "\\", "/"), "\r\n", "\n")
			Logger.Infof(line)
			_, err1 := logFile.WriteString(line)
			if err1 != nil {
				Logger.Errorf("failed to write to uat.log: %s", err)
			}
			if strings.HasPrefix(s, "AutomationTool exiting with ExitCode") {
				i0 := strings.Index(s, "ExitCode=")
				i1 := strings.Index(s, " (")
				if i0 > 0 && i1 > 0 {
					code := s[i0+9 : i1]
					if code != "0" {
						if exitCode, err = strconv.Atoi(code); err != nil {
							Logger.Errorf("failed to convert code %s to int: %v", code, err)
						}
					} else {
						exitCode = 0
					}
					err = io.EOF
				}
			}

			// Check for errors and exit conditions.
			if err != nil {
				if err == io.EOF {
					Logger.Infof("AutomationTool process has exited")
				} else {
					Logger.Errorf("failed to read the UAT process pipe: %v", err)
				}

				// Flush the log file.
				if err := logFile.Sync(); err != nil {
					Logger.Errorf("failed to sync uat.log: %s", err)
				}

				// Close the log file.
				if err := logFile.Close(); err != nil {
					Logger.Errorf("failed to close uat.log: %s", err)
				}

				// Open the log file again.
				logFile, err := os.Open("uat.log")
				if err != nil {
					Logger.Errorf("failed to open uat.log: %s", err)
				}

				// Scan the log file for errors.
				scanner := bufio.NewScanner(logFile)
				for scanner.Scan() {
					line := scanner.Text()
					if strings.Contains(line, "Warning:") {
						result.Warnings = append(result.Warnings, line)
					}
					if strings.Contains(line, "Error:") {
						result.Errors = append(result.Errors, line)
					}
				}

				// Check for errors.
				if err := scanner.Err(); err != nil {
					Logger.Errorf("failed to read uat.log: %s", err)
				}

				// Close the log file.
				err = logFile.Close()
				if err != nil {
					Logger.Errorf("failed to close uat.log: %s", err)
				}

				result.Process()

				wg.Done()

				return // Exit goroutine.
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return result, fmt.Errorf("cmd.Start() error: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return result, fmt.Errorf("cmd.Wait() error: %v", err)
	}

	if exitCode > 0 {
		return result, fmt.Errorf("UAT exit code: %d", exitCode)
	}

	wg.Wait()

	return result, nil
}
