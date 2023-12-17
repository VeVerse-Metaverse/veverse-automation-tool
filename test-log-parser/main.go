package main

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"regexp"
	"strings"
)

var Logger *logrus.Logger

func init() {
	Logger = &logrus.Logger{
		Out: os.Stdout,
		Formatter: &logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		},
		Hooks: make(logrus.LevelHooks),
		Level: logrus.DebugLevel,
	}
}

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

			if len(matches) > 0 {
				if len(matches) == 4 {
					r.Errors[k] = fmt.Sprintf("Category: %s\nSeverity: %s\nMessage: %s", matches[1], matches[2], matches[3])
				}
			}
		}
	}
}

func main() {

	// Open the log file again.
	logFile, err := os.Open("uat.log")
	if err != nil {
		Logger.Errorf("failed to open uat.log: %s", err)
	}

	result := UnrealAutomationToolResult{}

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

	result.ExitCode = 25
	result.Process()

	// Close the log file.
	err = logFile.Close()
	if err != nil {
		Logger.Errorf("failed to close uat.log: %s", err)
	}

}
