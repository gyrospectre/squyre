package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/andygrunwald/go-jira"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/gyrospectre/squyre"
	"sort"
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
	CreateTicketForAlert = mockCreateTicketForAlert
	AddComment = mockAddComment

	// By default we create tickets for each alert
	CreateTicket = true

	// Reset fake ticket number count
	MockTicket = 1
}

func mockInitClient() (*jira.Client, error) {
	return &jira.Client{}, nil
}

func mockGetSecret(location string) (secretsmanager.GetSecretValueOutput, error) {
	secret := `{"user":"test","apikey": "test123"}`
	return secretsmanager.GetSecretValueOutput{
		SecretString: &secret,
	}, nil
}

func mockCreateTicketForAlert(client *jira.Client, alert squyre.Alert) (string, error) {
	ticketnumber := fmt.Sprintf("CREATED-%d", MockTicket)
	MockTicket = MockTicket + 1
	return ticketnumber, nil
}

func mockAddComment(client *jira.Client, ticket string, rawComment string) error {
	return nil
}

func makeTestAlerts(number int, groups int, prefix string, includeResults bool, sameID bool) ([][]string, []string) {
	alert := squyre.Alert{
		RawMessage: "Testing",
	}
	if includeResults {
		alert.Results = []squyre.Result{
			{
				Source:         "Gyro",
				AttributeValue: "127.0.0.1",
				Message:        "Test",
				Success:        true,
			},
		}
	}

	var grouplist [][]string
	var alerts []string
	var alertlist []string
	for j := 0; j < groups; j++ {
		for i := 1; i <= number; i++ {
			if sameID {
				alert.ID = fmt.Sprintf("%s1", prefix)
			} else {
				alert.ID = fmt.Sprintf("%s%d", prefix, i+(j*number))
			}

			alertlist = append(alertlist, alert.ID)
			alertJSON, _ := json.Marshal(alert)
			alerts = append(alerts, string(alertJSON))
		}
		grouplist = append(grouplist, alerts)
	}
	return grouplist, alertlist
}

// tests Handler when Create is set
func TestHandlerCreateSuccess(t *testing.T) {
	setup()
	numgroups := 2
	numalerts := 5

	alerts, _ := makeTestAlerts(numalerts, numgroups, "EXISTING-", true, false)

	output, err := handleRequest(Ctx, alerts)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	var alertList []string
	for i := 1; i <= numalerts*numgroups; i++ {
		alertList = append(alertList, fmt.Sprintf("%s-%d", "CREATED", i))
	}
	sort.Strings(alertList)

	have := string(output)
	want := fmt.Sprintf("Success: %d alerts processed (%d groups). Created alerts: %s", numalerts*numgroups, numgroups, alertList)

	if have != want {
		t.Fatalf("Unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}

// tests Handler when Create is not set
func TestHandlerNoCreateSuccess(t *testing.T) {
	setup()
	CreateTicket = false
	numgroups := 1
	numalerts := 3

	alerts, alertlist := makeTestAlerts(numalerts, numgroups, "EXISTING-", true, false)

	output, err := handleRequest(Ctx, alerts)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	have := string(output)
	want := fmt.Sprintf("Success: %d alerts processed (%d groups). Updated alerts: %s", numalerts*numgroups, numgroups, alertlist)

	if have != want {
		t.Fatalf("unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}

func TestHandlerNoResults(t *testing.T) {
	setup()

	numgroups := 1
	numalerts := 3

	alerts, _ := makeTestAlerts(numalerts, numgroups, "EXISTING-", false, false)

	output, err := handleRequest(Ctx, alerts)
	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}

	have := string(output)
	want := "No results found to process"

	if have != want {
		t.Fatalf("unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}

func TestHandlerCreateSameIds(t *testing.T) {
	setup()

	numgroups := 1
	numalerts := 5

	alerts, _ := makeTestAlerts(numalerts, numgroups, "EXISTING-", true, true)

	output, err := handleRequest(Ctx, alerts)

	var alertList []string
	alertList = append(alertList, "CREATED-1")

	if err != nil {
		t.Fatalf("unexpected error %s", err)
	}
	have := string(output)

	want := fmt.Sprintf("Success: 1 alerts processed (%d groups). Created alerts: %s", numgroups, alertList)

	if have != want {
		t.Fatalf("Unexpected output. \nHave: %s\nWant: %s", have, want)
	}
}
