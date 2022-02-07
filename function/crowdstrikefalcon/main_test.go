package main

import (
	"context"
	//"fmt"
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/crowdstrike/gofalcon/falcon/client"
	"github.com/gyrospectre/squyre/pkg/squyre"
	"testing"
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
}

func mockInitClient() (*client.CrowdStrikeAPISpecification, error) {
	return &client.CrowdStrikeAPISpecification{}, nil
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
	alert.Results = []squyre.Result{
		{
			Source:         "Gyro",
			AttributeValue: "127.0.0.1",
			Message:        "Test",
			Success:        true,
		},
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
