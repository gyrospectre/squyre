package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/go-cmp/cmp"
	"github.com/gyrospectre/squyre"
	"io/ioutil"
	"net/http"
	"testing"
)

// MockClient is the mock client
type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

var (
	// GetDoFunc fetches the mock client's `Do` func
	GetDoFunc func(req *http.Request) (*http.Response, error)
)

// Do is the mock client's `Do` func
func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return GetDoFunc(req)
}

// tests GetSecret return an expected value
func TestHandlerSuccess(t *testing.T) {
	Client = &MockClient{}

	// build response JSON
	gnResp := greynoiseResponse{
		IP:      "8.8.8.8",
		Noise:   false,
		Riot:    false,
		Message: "IP not observed scanning the internet or contained in RIOT data set.",
	}
	respJSON, _ := json.Marshal(gnResp)

	// create a new reader with that JSON
	r := ioutil.NopCloser(bytes.NewReader([]byte(respJSON)))
	GetDoFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       r,
		}, nil
	}

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
	output, err := handleRequest(ctx, alert)

	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	prettyresponse := messageFromResponse(gnResp)

	expected, _ := json.Marshal(squyre.Alert{
		RawMessage: "Testing",
		ID:         "1234-1234",
		Name:       "Test Search",
		URL:        "https://127.0.0.1/test.html",
		Timestamp:  "2022-12-12 18:00:00",
		Subjects: []squyre.Subject{
			{
				Type:  "ipv4",
				Value: "8.8.8.8",
			},
		},
		Results: []squyre.Result{
			{
				Source:         "GreyNoise",
				AttributeValue: "8.8.8.8",
				Message:        string(prettyresponse),
				Success:        true,
			},
		},
	})

	if !cmp.Equal(string(expected), output) {
		t.Fatalf("expected value %s, got %s", expected, output)
	}
}

func TestHandlerError(t *testing.T) {
	Client = &MockClient{}
	GetDoFunc = func(*http.Request) (*http.Response, error) {
		return nil, errors.New("Greynoise failed")
	}

	alert := squyre.Alert{}
	alert.Subjects = []squyre.Subject{
		{
			Type:  "ipv4",
			Value: "8.8.8.8",
		},
	}

	var ctx context.Context
	_, err := handleRequest(ctx, alert)

	if err == nil {
		t.Fatalf("unexpected non error")
	}
}
