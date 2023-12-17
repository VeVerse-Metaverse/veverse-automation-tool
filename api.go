package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

// login authenticates user with the API
func login() (string, error) {
	var (
		requestBody []byte
		err         error
	)

	requestBody, err = json.Marshal(map[string]string{
		"email":    apiEmail,
		"password": apiPassword,
	})

	if err != nil {
		log.Fatalln(err)
	}

	url := fmt.Sprintf("%s/auth/login", api2Url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Printf("error closing http response body: %s\n", err.Error())
		}
	}(resp.Body)

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("failed to login to %s, status code: %d, error: %s", url, resp.StatusCode, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var v map[string]string
	if err = json.Unmarshal(body, &v); err != nil {
		return "", err
	}

	if v["status"] == "error" {
		return "", errors.New(fmt.Sprintf("authentication error %d: %s\n", resp.StatusCode, v["message"]))
	} else if v["status"] == "ok" {
		return v["data"], nil
	}

	return "", errors.New(v["message"])
}
