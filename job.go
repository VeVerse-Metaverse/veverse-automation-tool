package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gofrs/uuid"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

// updateJobStatus updates the job status using status code and error message
func updateJobStatus(job JobMetadata, statusCode int, message string) error {
	if job.Id == nil {
		return fmt.Errorf("invalid job id")
	}

	reqUrl := fmt.Sprintf("%s/jobs/%s/status", api2Url, job.Id.String())

	var (
		status string
		ok     bool
	)

	if status, ok = supportedJobStatuses[statusCode]; !ok {
		return fmt.Errorf("unknown job status")
	}

	var requestMetadata = JobStatusRequestMetadata{Status: status, Message: message}

	b, err := json.Marshal(requestMetadata)
	if err != nil {
		return fmt.Errorf("failed to serialize json: %s", err)
	}

	req, err := http.NewRequest("PATCH", reqUrl, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %s", err)
	}

	if resp.StatusCode >= 400 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %s", err)
		}
		return fmt.Errorf("failed to upload a file, status code: %d, content: %s", resp.StatusCode, string(body))
	}

	return nil
}

// reportJobLog reports the job log using warnings and errors
func reportJobLog(job JobMetadata, warnings []string, errors []string) error {
	if job.Id == nil {
		return fmt.Errorf("invalid job id")
	}

	reqUrl := fmt.Sprintf("%s/jobs/%s/log", api2Url, job.Id.String())

	var requestMetadata = JobLogRequestMetadata{Warnings: warnings, Errors: errors}

	b, err := json.Marshal(requestMetadata)
	if err != nil {
		return fmt.Errorf("failed to serialize json: %s", err)
	}

	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %s", err)
	}

	if resp.StatusCode >= 400 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %s", err)
		}
		return fmt.Errorf("failed to upload a file, status code: %d, content: %s", resp.StatusCode, string(body))
	}

	return nil
}

// uploadFile uploads the job results to the API for storage
func uploadJobEntityFile(job JobMetadata, entityId *uuid.UUID, fileType string, fileMime string, path string, originalPath string, params map[string]string) error {
	const chunkSize = 100 * 1024 * 1024 // 100MiB

	// Validate job
	if job.Id == nil || job.Id.IsNil() {
		return fmt.Errorf("invalid job id")
	}

	if entityId == nil || entityId.IsNil() {
		return fmt.Errorf("invalid job package id")
	}

	// Warning! For the package upload we don't set index and original-path to prevent duplicates, if these fields provided, we will get an error on DB index in future re-uploads of the package
	reqUrl := fmt.Sprintf("%s/entities/%s/files/upload?type=%s&mime=%s&deployment=%s&platform=%s&original-path=%s", api2Url, entityId.String(), fileType, fileMime, job.Deployment, job.Platform, originalPath)

	// Open file
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}

	// Get file info
	fi, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %v", err)
	}

	// Defer file close
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			Logger.Errorf("failed to close the uploading package file")
		}
	}(file)

	// Temporary buffer to get multipart form fields (header) and the boundary
	multipartFormBuffer := &bytes.Buffer{}

	// Add multipart form data parameters if any supplied
	multipartFormWriter := multipart.NewWriter(multipartFormBuffer)
	for key, value := range params {
		err = multipartFormWriter.WriteField(key, value)
	}

	// Add a file to the multipart form writer, the field name should be "file" as the API expects it
	_, err = multipartFormWriter.CreateFormFile("file", fi.Name())
	if err != nil {
		return fmt.Errorf("failed to create a multipart form file: %v", err)
	}

	// Get multipart form content type including boundary
	multipartFormDataContentType := multipartFormWriter.FormDataContentType()

	multipartFormOpeningHeaderSize := multipartFormBuffer.Len()

	// Save the opening multipart form header from the buffer
	multipartFormOpeningHeader := make([]byte, multipartFormOpeningHeaderSize)
	_, err = multipartFormBuffer.Read(multipartFormOpeningHeader)
	if err != nil {
		return fmt.Errorf("failed to read the multipart form buffer: %v", err)
	}

	// Write the multipart form closing boundary to the buffer
	err = multipartFormWriter.Close()
	if err != nil {
		return fmt.Errorf("failed to close the multipart form message")
	}

	multipartFormClosingBoundarySize := multipartFormBuffer.Len()

	// Save the closing multipart form boundary to the buffer
	multipartFormClosingBoundary := make([]byte, multipartFormClosingBoundarySize)
	_, err = multipartFormBuffer.Read(multipartFormClosingBoundary)
	if err != nil {
		return fmt.Errorf("failed to read boundary from the multipart form buffer")
	}

	// Calculate the total content size including opening header size, uploaded file size and closing boundary length
	multipartDataTotalSize := int64(multipartFormOpeningHeaderSize) + fi.Size() + int64(multipartFormClosingBoundarySize)

	// Use a pipe to write request data
	pipeReader, pipeWriter := io.Pipe()
	defer func(rd *io.PipeReader) {
		err := rd.Close()
		if err != nil {
			Logger.Errorf("failed to close a pipe reader")
		}
	}(pipeReader)

	go func() {
		defer func(pipeWriter *io.PipeWriter) {
			err := pipeWriter.Close()
			if err != nil {
				Logger.Errorf("failed to close a pipe writer: %v", err)
			}
		}(pipeWriter)

		// Write the multipart form opening header
		_, err = pipeWriter.Write(multipartFormOpeningHeader)
		if err != nil {
			Logger.Errorf("failed to write the opening header to the multipart form: %v", err)
		}

		// Write the file bytes to the temporary buffer
		buffer := make([]byte, chunkSize)
		for {
			n, err := file.Read(buffer)
			if err != nil {
				if err != io.EOF {
					Logger.Errorf("failed to read from the file pipe reader: %v", err)
				}
				break
			}
			_, err = pipeWriter.Write(buffer[:n])
			if err != nil {
				Logger.Errorf("failed to write file bytes to the multipart form: %v", err)
			}
		}

		// Write the closing boundary to the multipart form
		_, err = pipeWriter.Write(multipartFormClosingBoundary)
		if err != nil {
			Logger.Errorf("failed to write the closing boundary to the multipart form: %v", err)
		}
	}()

	// Create an HTTP request with the pipe reader
	req, err := http.NewRequest("PUT", reqUrl, pipeReader)
	req.Header.Set("Content-Type", multipartFormDataContentType)
	req.ContentLength = multipartDataTotalSize
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Process the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			Logger.Errorf("failed to close resp body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode >= 400 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read the response body: %v", err)
		}
		return fmt.Errorf("failed to upload a file, status code: %d, content: %s", resp.StatusCode, string(body))
	}

	return nil
}

// fetchUnclaimedJob Tries to fetch the unclaimed job supported by the runner, validates and returns it
func fetchUnclaimedJob() (*JobMetadata, error) {
	// Prepare an HTTP request
	reqUrl := fmt.Sprintf("%s/jobs/unclaimed?platform=%s&type=%s&deployment=%s", api2Url, platforms, jobTypes, deployments)
	req, err := http.NewRequest("GET", reqUrl, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Send HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %s", err)
	}

	// Process response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err)
	}

	// Validate response
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to fetch an unclaimed job, status code: %d, content: %s", resp.StatusCode, string(body))
	}

	// Parse the HTTP request json content
	var container JobMetadataContainer
	err = json.Unmarshal(body, &container)
	if err != nil {
		return nil, fmt.Errorf("failed to parse job json: %s", err.Error())
	}

	// Special case when there are no unclaimed jobs to process
	if container.Status == "no jobs" {
		return nil, nil
	}

	return &container.JobMetadata, nil
}
