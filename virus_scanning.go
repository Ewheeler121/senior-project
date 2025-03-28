package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
)

type FoundVirus struct {
	FileName  string `json:"FileName"`
	VirusName string `json:"VirusName"`
}

type ScanResponse struct {
	CleanResult  bool         `json:"CleanResult"`
	FoundViruses []FoundVirus `json:"FoundViruses"`
	Message      string       `json:"Message"`
}

func scanFile(fileBytes []byte, fileName string) (bool, error) {
	apiKey := os.Getenv("API_KEY")

	url := "https://api.cloudmersive.com/virus/scan/file"
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return false, err
	}
	_, err = part.Write(fileBytes)
	if err != nil {
		return false, err
	}
	writer.Close()

	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Apikey", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Cloudmersive API returned status: %d", resp.StatusCode)
	}

	var scanResponse ScanResponse
	err = json.NewDecoder(resp.Body).Decode(&scanResponse)
	if err != nil {
		return false, err
	}

	return scanResponse.CleanResult, nil
}

func validateAndScanFiles(files []File) ([]File, error) {
	var cleanFiles []File

	for _, file := range files {
		isClean, err := scanFile(file.file, "uploaded_file")
		if err != nil {
			return nil, err
		}
		if !isClean {
			return nil, fmt.Errorf("malicious file detected")
		}
		cleanFiles = append(cleanFiles, file)
	}
	return cleanFiles, nil
}
