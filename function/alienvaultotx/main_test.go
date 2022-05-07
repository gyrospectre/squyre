package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/gyrospectre/squyre/pkg/squyre"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

var (
	mockResponse string
	TestAlert    squyre.Alert
	ctx          context.Context
	attempt      int
)

func mockInitClient() (*apiClient, error) {
	return &apiClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}, nil
}

func mockInfo(c *apiClient, indicator string, indicatorType string) (*http.Response, error) {
	var otxResp otxResponse
	otxResp = otxResponse{
		Indicator:  indicator,
		Reputation: 0,
	}
	if indicator == "4.4.4.4" {
		otxResp.PulseInfo.Count = 1
		otxResp.PulseInfo.Pulses = []otxPulse{
			{
				Id:   "1234",
				Name: "test",
			},
		}
	} else if indicator == "1.1.1.1" {
		if attempt < 3 {
			attempt += 1
			return nil, errors.New("(Client.Timeout exceeded while awaiting headers)")
		}
		otxResp.PulseInfo.Count = 1
		otxResp.PulseInfo.Pulses = []otxPulse{
			{
				Id:   "7890",
				Name: "test2",
			},
		}
	} else {
		otxResp.PulseInfo.Count = 0
	}

	otxRespJson, _ := json.Marshal(otxResp)
	mockResponse = string(otxRespJson)

	attempt += 1
	return &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte(mockResponse))),
	}, nil
}

func setup() {
	GetIndictatorInfo = mockInfo
	InitClient = mockInitClient
	OnlyLogMatches = false
	TestAlert = squyre.Alert{
		RawMessage: "Testing",
		ID:         "1234-1234",
		Name:       "Test Search",
		URL:        "https://127.0.0.1/test.html",
		Timestamp:  "2022-12-12 18:00:00",
	}
	attempt = 1
}

func TestHandlerNonMatchNonIgnore(t *testing.T) {
	setup()

	TestAlert.Subjects = []squyre.Subject{
		{
			Type:  "ipv4",
			Value: "8.8.8.8",
		},
	}
	output, _ := handleRequest(ctx, TestAlert)

	var response squyre.Alert
	json.Unmarshal([]byte(output), &response)

	have := response.Results[0].Message
	json.Unmarshal([]byte(mockResponse), &responseObject)
	want := "Indicator not found in Alienvault OTX."

	if have != want {
		t.Errorf("Expected '%s', got '%s'", want, have)
	}
}

func TestHandlerNonMatchIgnore(t *testing.T) {
	setup()
	OnlyLogMatches = true

	TestAlert.Subjects = []squyre.Subject{
		{
			Type:  "ipv4",
			Value: "8.8.8.8",
		},
	}
	output, _ := handleRequest(ctx, TestAlert)

	var response squyre.Alert
	json.Unmarshal([]byte(output), &response)

	have := len(response.Results)
	want := 0

	if have != want {
		t.Errorf("Expected %x results, got %x", want, have)
	}
}

func TestHandlerMatch(t *testing.T) {
	setup()

	TestAlert.Subjects = []squyre.Subject{
		{
			Type:  "ipv4",
			Value: "4.4.4.4",
		},
	}
	output, _ := handleRequest(ctx, TestAlert)

	var response squyre.Alert
	json.Unmarshal([]byte(output), &response)

	have := response.Results[0].Message
	json.Unmarshal([]byte(mockResponse), &responseObject)
	want := messageFromResponse(responseObject)

	if have != want {
		t.Errorf("Expected '%s', got '%s'", want, have)
	}
}

func TestMultiSubject(t *testing.T) {
	setup()

	TestAlert.Subjects = []squyre.Subject{
		{
			Type:  "ipv4",
			Value: "4.4.4.4",
		},
		{
			Type:  "ipv4",
			Value: "8.8.8.8",
		},
	}
	output, _ := handleRequest(ctx, TestAlert)

	var respAlert squyre.Alert
	json.Unmarshal([]byte(output), &respAlert)

	have := respAlert.Results[0].Message
	want := "\nAlienvault OTX has 1 matches for '4.4.4.4', in the following pulses:\ntest\n\nMore information at: https://otx.alienvault.com/browse/global/pulses?q=4.4.4.4\n\n"

	if have != want {
		t.Fatalf("unexpected output. \nHave: %s\nWant: %s", have, want)
	}

	have = respAlert.Results[1].Message
	want = "Indicator not found in Alienvault OTX."

	if have != want {
		t.Fatalf("unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}

// Fail lookups the first two attempts, then work on the 3rd
func TestTempTimeout(t *testing.T) {
	setup()

	TestAlert.Subjects = []squyre.Subject{
		{
			Type:  "ipv4",
			Value: "1.1.1.1",
		},
	}
	output, _ := handleRequest(ctx, TestAlert)

	var response squyre.Alert
	json.Unmarshal([]byte(output), &response)

	have := response.Results[0].Message
	json.Unmarshal([]byte(mockResponse), &responseObject)
	want := messageFromResponse(responseObject)

	if have != want {
		t.Errorf("Expected '%s', got '%s'", want, have)
	}

	if attempt-1 != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempt-1)
	}
}
