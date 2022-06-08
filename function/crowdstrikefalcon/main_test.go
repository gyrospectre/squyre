package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/crowdstrike/gofalcon/falcon/client"
	"github.com/crowdstrike/gofalcon/falcon/models"
	"github.com/gyrospectre/squyre/pkg/squyre"
	"testing"
	"time"
)

var (
	// MockTicket is a fake ticket for tests
	MockTicket int
	Ctx        context.Context
)

func setup() {
	// Mock out calls to real things
	InitClient = mockInitClient

	// Reset fake ticket number count
	MockTicket = 1

	getIndicator = mockGetFalconIndicator
	OnlyLogMatches = false
}

func mockInitClient() (*client.CrowdStrikeAPISpecification, error) {
	return &client.CrowdStrikeAPISpecification{}, nil
}

func mockGetFalconIndicator(client *client.CrowdStrikeAPISpecification, name string) (*models.DomainPublicIndicatorV3, error) {
	conf := "high"
	now := time.Now()
	epoch := now.Unix()

	if name == "8.8.8.8" {
		indicator := &models.DomainPublicIndicatorV3{
			Indicator:           &name,
			MaliciousConfidence: &conf,
			PublishedDate:       &epoch,
			LastUpdated:         &epoch,
		}
		return indicator, nil
	} else if name == "9.9.9.9" {
		return nil, errors.New("Error!")
	}

	return nil, nil
}

func mockGetSecret(location string) (secretsmanager.GetSecretValueOutput, error) {
	secret := `{"user":"test","apikey": "test123"}`
	return secretsmanager.GetSecretValueOutput{
		SecretString: &secret,
	}, nil
}

func makeTestAlert() (squyre.Alert, string) {
	alert := squyre.Alert{
		RawMessage: "Testing",
	}
	finalJSON, _ := json.Marshal(alert)

	return alert, string(finalJSON)
}

// tests main Handler
func TestAlertWithNoSubjects(t *testing.T) {
	setup()

	alert, alertStr := makeTestAlert()
	output, err := handleRequest(Ctx, alert)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	have := string(output)
	want := alertStr

	if have != want {
		t.Fatalf("Unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}

func TestAlertWithSubjectsNoIgnore(t *testing.T) {
	setup()

	alert, _ := makeTestAlert()
	alert.Subjects = []squyre.Subject{
		{
			Type:  "ipv4",
			Value: "8.8.8.8",
		},
		{
			Type:  "ipv4",
			Value: "4.4.4.4",
		},
	}

	output, err := handleRequest(Ctx, alert)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	var response squyre.Alert
	json.Unmarshal([]byte(output), &response)

	client, _ := mockInitClient()
	ind, _ := mockGetFalconIndicator(client, "8.8.8.8")

	have := string(response.Results[0].Message)
	want := messageFromIndicator(ind)
	if have != want {
		t.Fatalf("Unexpected output. \nHave: %s\nWant: %s", have, want)
	}

	havenum := len(response.Results)
	wantnum := 0

	if have != want {
		t.Errorf("Expected %x results, got %x", wantnum, havenum)
	}

}

func TestAlertWithSubjectsIgnore(t *testing.T) {
	setup()
	OnlyLogMatches = true

	alert, _ := makeTestAlert()
	alert.Subjects = []squyre.Subject{
		{
			Type:  "ipv4",
			Value: "8.8.8.8",
		},
		{
			Type:  "ipv4",
			Value: "4.4.4.4",
		},
	}

	output, err := handleRequest(Ctx, alert)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	var response squyre.Alert
	json.Unmarshal([]byte(output), &response)

	have := len(response.Results)
	want := 1

	if have != want {
		t.Errorf("Expected %x results, got %x", want, have)
	}
}

func TestAlertFailedLookup(t *testing.T) {
	setup()

	alert, _ := makeTestAlert()
	alert.Subjects = []squyre.Subject{
		{
			Type:  "ipv4",
			Value: "9.9.9.9",
		},
	}

	output, err := handleRequest(Ctx, alert)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	var response squyre.Alert
	json.Unmarshal([]byte(output), &response)

	have := string(response.Results[0].Message)
	want := "Error!"
	if have != want {
		t.Fatalf("Unexpected output. \nHave: %s\nWant: %s", have, want)
	}

	have2 := response.Results[0].Success
	want2 := false

	if have2 != want2 {
		t.Fatalf("Unexpected output. \nHave: %t\nWant: %t", have2, want2)
	}
}
