package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// pushCodeRelease Tries to push the new jobs for the new code release
func pushCodeRelease() error {
	Logger.Infof("projectDir: %s", projectDir)

	r, err := gitRepo(projectDir)
	if err != nil {
		return fmt.Errorf("failed to open the repo at %s: %v", projectDir, err)
	}

	contentVersion, err := projectLatestContentVersion()
	if err != nil {
		return fmt.Errorf("failed to get project latest version: %v", err)
	}

	codeVersion, err := gitLatestTag(r)
	if err != nil {
		return fmt.Errorf("failed to get latest tag from repo: %v", err)
	}
	Logger.Warningf("latest tag: %s", codeVersion)

	// Prepare an HTTP request
	reqUrl := fmt.Sprintf("%s/jobs/release", api2Url)
	bodyJson := fmt.Sprintf(`{"codeVersion":"%s","contentVersion":"%s"}`, codeVersion, contentVersion)
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer([]byte(bodyJson)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Send HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %s", err)
	}

	// Process response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %s", err)
	}

	// Validate response
	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to push a code release to %s, status code: %d, content: %s", reqUrl, resp.StatusCode, string(body))
	}

	return nil
}
