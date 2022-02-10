package main

import (
	"bytes"
	"context"
	"encoding/json"
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
)

func mockInitClient() (*apiClient, error) {
	return &apiClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}, nil
}

func mockIPInfo(c *apiClient, ipv4 string) (*http.Response, error) {
	// 4.4.4.4 is bad, all other IPs good
	var gnResp greynoiseResponse
	if ipv4 == "4.4.4.4" {
		gnResp = greynoiseResponse{
			IP:             "4.4.4.4",
			Noise:          true,
			Riot:           false,
			Classification: "malicious",
			Link:           "http://localhost",
			Message:        "Success",
		}
	} else {
		gnResp = greynoiseResponse{
			IP:      "8.8.8.8",
			Noise:   false,
			Riot:    false,
			Message: "IP not observed scanning the internet or contained in RIOT data set.",
		}
	}
	gnRespJson, _ := json.Marshal(gnResp)
	mockResponse = string(gnRespJson)

	return &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte(mockResponse))),
	}, nil
}

func setup() {
	GetIPInfo = mockIPInfo
	InitClient = mockInitClient
	OnlyLogMatches = false
	TestAlert = squyre.Alert{
		RawMessage: "Testing",
		ID:         "1234-1234",
		Name:       "Test Search",
		URL:        "https://127.0.0.1/test.html",
		Timestamp:  "2022-12-12 18:00:00",
	}
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
	want := messageFromResponse(responseObject)

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
			Value: "8.8.8.8",
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

	have := respAlert.Results[1].Message
	want := "IP not observed scanning the internet or contained in RIOT data set."

	if have != want {
		t.Fatalf("unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}
