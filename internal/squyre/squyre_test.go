package squyre

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNormaliseSplunkAlert(t *testing.T) {
	splunk := SplunkAlert{
		Message:       "Testing",
		CorrelationID: "1234-1234",
		SearchName:    "Test Search",
		ResultsLink:   "https://127.0.0.1/test.html",
		Timestamp:     "2022-12-12 18:00:00",
	}

	expected, _ := json.Marshal(Alert{
		RawMessage: "Testing",
		ID:         "1234-1234",
		Name:       "Test Search",
		URL:        "https://127.0.0.1/test.html",
		Timestamp:  "2022-12-12 18:00:00",
	})
	output, _ := json.Marshal(splunk.Normaliser())
	if !cmp.Equal(expected, output) {
		t.Fatalf("expected value %s, got %s", expected, output)
	}
}

func TestNormaliseOGAlert(t *testing.T) {
	opsgenie := OpsGenieAlert{}

	opsgenie.Alert.AlertID = "1234-1234"
	opsgenie.Alert.Message = "Testing"
	opsgenie.Alert.CreatedAt = "2022-12-12 18:00:00"
	opsgenie.Alert.Details.ResultsObject = "{ 8.8.8.8 }"

	expected, _ := json.Marshal(Alert{
		RawMessage: "{ 8.8.8.8 }",
		ID:         "1234-1234",
		Name:       "Testing",
		Timestamp:  "2022-12-12 18:00:00",
	})
	output, _ := json.Marshal(opsgenie.Normaliser())
	if !cmp.Equal(expected, output) {
		t.Fatalf("expected value %s, got %s", expected, output)
	}
}
