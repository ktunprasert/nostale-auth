package gfClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	_challengeUrl = "https://challenge.gameforge.com/challenge/%s" // uuid
	_imageDropUrl = "https://image-drop-challenge.gameforge.com/challenge/%s/en-B"
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

func SolveCaptcha(uuid string) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}

	res, err := client.Get(fmt.Sprintf(_challengeUrl, uuid))
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
	imgDropRes, err := client.Get(fmt.Sprintf(_imageDropUrl, uuid))
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

	for i := 1; i <= 4; i++ {
		solvePayload.Answer = i

		// Marshal the payload to JSON bytes
		bytePayload, err := json.Marshal(solvePayload)
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON payload: %w", err)
		}

		// Create an io.Reader from the JSON bytes
		payload := io.NopCloser(io.Reader(bytes.NewBuffer(bytePayload)))

		// Send the POST request
		resp, err := client.Post(fmt.Sprintf(_imageDropUrl, uuid), "application/json", payload)
		if err != nil {
			return "", fmt.Errorf("failed to submit captcha answer %d: %w", i, err)
		}
		defer resp.Body.Close()

		// Read and check the response

		var respData ImageDropResponse
		if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
			return "", fmt.Errorf("failed to decode response for answer %d: %w", i, err)
		}

		if respData.Status == "solved" {
			fmt.Printf("Captcha solved with answer %d\n", i)
			return uuid, nil
		}
	}

	return "", fmt.Errorf("failed to solve captcha after trying all answers")
}
