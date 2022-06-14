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
	// 4.4.4.4 is a Tor node, all other IPs are not
	if ipv4 == "4.4.4.4" {
		mockResponse = "blah blah blah Result is positive <html woot yeh"
	} else {
		mockResponse = "blah blah blah Result is negative <html woot yeh"
	}

	return &http.Response{
		Body:       ioutil.NopCloser(bytes.NewReader([]byte(mockResponse))),
		StatusCode: 200,
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
	want := messageFromResponse("8.8.8.8", false)

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
	want := messageFromResponse("4.4.4.4", true)

	if have != want {
		t.Errorf("Expected '%s', got '%s'", want, have)
	}
}

func TestMultiSubject(t *testing.T) {
	setup()
	OnlyLogMatches = true

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

	haveNum := len(respAlert.Results)
	wantNum := 1

	if haveNum != wantNum {
		t.Errorf("Expected %x results, got %x", wantNum, haveNum)
	}

	have := respAlert.Results[0].Message
	want := messageFromResponse("4.4.4.4", true)

	if have != want {
		t.Fatalf("unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}
