package gfClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

const (
	_challengeUrl = "https://challenge.gameforge.com/challenge/%s" // uuid
	_imageDropUrl = "https://image-drop-challenge.gameforge.com/challenge/%s/en-GB"
)

type CaptchaReponse struct {
	Script string `json:"script"`
	Type   string `json:"type"`
}

type ImageDropResponse struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	// LastUpdated int    `json:"lastUpdated"`
}

type ChallengeBody struct {
	Answer int `json:"answer"`
}

var client *http.Client

func SolveCaptcha(uuid string, headers http.Header) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequest("GET", fmt.Sprintf(_challengeUrl, uuid), nil)
	if err != nil {
		return "", err
	}
	req.Header = headers

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("initialise captcha challenge: %s", res.Status)
	}
	defer res.Body.Close()

	// decode
	var captchaResponse CaptchaReponse
	if err := json.NewDecoder(res.Body).Decode(&captchaResponse); err != nil {
		return "", err
	}

	// check if type is image-drop-challenge
	if captchaResponse.Type != "gf-image-drop-captcha" {
		return "", fmt.Errorf("invalid captcha type: %s", captchaResponse.Type)
	}

	// begin the challenge
	imgDropReq, err := http.NewRequest("GET", fmt.Sprintf(_imageDropUrl, uuid), nil)
	if err != nil {
		return "", err
	}
	imgDropReq.Header = headers

	imgDropRes, err := client.Do(imgDropReq)
	if err != nil {
		return "", err
	}

	var dropResponse ImageDropResponse
	if err := json.NewDecoder(imgDropRes.Body).Decode(&dropResponse); err != nil {
		return "", err
	}
	defer imgDropRes.Body.Close()

	// check for status and ID
	if dropResponse.Status != "presented" || dropResponse.Id != uuid {
		return "", fmt.Errorf("invalid response: %+v", dropResponse)
	}

	// tries answer from 1 to 4
	solvePayload := &ChallengeBody{}

	for range 10 {
		time.Sleep(1 * time.Second) // Sleep for 1 second before each attempt
		solvePayload.Answer = rand.Intn(3) + int(1)

		// Marshal the payload to JSON bytes
		bytePayload, err := json.Marshal(solvePayload)
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON payload: %w", err)
		}

		// Create an io.Reader from the JSON bytes
		payload := bytes.NewBuffer(bytePayload)

		// Create POST request
		req, err := http.NewRequest("POST", fmt.Sprintf(_imageDropUrl, uuid), payload)
		if err != nil {
			return "", fmt.Errorf("failed to create request with answer %d: %w", solvePayload.Answer, err)
		}

		// Set headers from parameter
		req.Header = headers.Clone()
		req.Header.Set("Content-Type", "application/json")

		// Send the POST request
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to submit captcha answer %d: %w", solvePayload.Answer, err)
		}
		defer resp.Body.Close()

		// Read and check the response

		var respData ImageDropResponse
		if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
			return "", fmt.Errorf("failed to decode response for answer %d: %w", solvePayload.Answer, err)
		}

		if respData.Status == "solved" {
			fmt.Printf("Captcha solved with answer %d\n", solvePayload.Answer)
			return uuid, nil
		}
	}

	return "", fmt.Errorf("failed to solve captcha after trying all answers")
}
