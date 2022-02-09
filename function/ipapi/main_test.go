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
)

func mockInitClient() (*apiClient, error) {
	return &apiClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		apiKey: "secret!",
	}, nil
}

func mockIPInfo(c *apiClient, ipv4 string) (*http.Response, error) {
	return &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte(mockResponse))),
	}, nil
}

func setup() {
	GetIPInfo = mockIPInfo
	InitClient = mockInitClient
}

// tests GetSecret return an expected value
func TestHandlerSuccess(t *testing.T) {
	setup()

	alert := squyre.Alert{
		RawMessage: "Testing",
		ID:         "1234-1234",
		Name:       "Test Search",
		URL:        "https://127.0.0.1/test.html",
		Timestamp:  "2022-12-12 18:00:00",
	}
	alert.Subjects = []squyre.Subject{
		{
			Type:  "ipv4",
			Value: "8.8.8.8",
		},
	}
	var ctx context.Context
	mockResponse = `{"ip":"8.8.8.8", "city":"Okayville", "country_name":"Atlantis"}`

	output, _ := handleRequest(ctx, alert)

	var response squyre.Alert
	json.Unmarshal([]byte(output), &response)

	have := response.Results[0].Message
	want := "IP API result for 8.8.8.8:\n\nCountry: Atlantis\nCity: Okayville, \n"

	if have != want {
		t.Errorf("Expected '%s', got '%s'", have, want)
	}
}
