package squyre

import (
	"encoding/json"
)

// Subject defines attributes about a thing that we want to know about
type Subject struct {
	Type  string // ipv4, domain, sha256 or hostname
	Value string
}

// Result holds enrichment results, and where they came from
type Result struct {
	Source         string
	AttributeValue string
	Message        string
	Success        bool
}

// Alert holds information about an incoming alert
type Alert struct {
	Timestamp  string
	Name       string
	RawMessage string
	URL        string
	ID         string
	Subjects   []Subject
	Results    []Result
	Scope      string
}

// Alerter defines common functions for all alert types
type Alerter interface {
	Normaliser() Alert
}

// SplunkAlert defines the standard format alerts come to us from Splunk
type SplunkAlert struct {
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id"`
	SearchName    string `json:"search_name"`
	Timestamp     string `json:"timestamp"`
	Entity        string `json:"entity"`
	Source        string `json:"source"`
	Event         string `json:"event"`
	ResultsLink   string `json:"results_link"`
	App           string `json:"app"`
	Owner         string `json:"owner"`
}

// Normaliser comverts a Splunk alert to our standard form
func (alert SplunkAlert) Normaliser() Alert {
	return Alert{
		RawMessage: alert.Message,
		ID:         alert.CorrelationID,
		Name:       alert.SearchName,
		URL:        alert.ResultsLink,
		Timestamp:  alert.Timestamp,
	}
}

// OpsGenieAlert defines the standard format alerts come to us from OpsGenie
// See https://support.atlassian.com/opsgenie/docs/opsgenie-edge-connector-alert-action-data/
type OpsGenieAlert struct {
	Action          string `json:"action"`
	IntegrationID   string `json:"integrationId"`
	IntegrationName string `json:"integrationName"`
	Source          struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"source"`
	Alert struct {
		AlertID   string `json:"alertId"`
		Message   string `json:"message"`
		CreatedAt string `json:"createdAt"`
		Details   struct {
			ResultsLink   string `json:"Results Link"`
			ResultsObject string `json:"Results Object"`
		} `json:"details"`
	} `json:"alert"`
}

// Normaliser comverts an OpsGenie alert to our standard form
func (alert OpsGenieAlert) Normaliser() Alert {
	return Alert{
		RawMessage: alert.Alert.Details.ResultsObject,
		ID:         alert.Alert.AlertID,
		Name:       alert.Alert.Message,
		Timestamp:  alert.Alert.CreatedAt,
		URL:        alert.Alert.Details.ResultsLink,
	}
}

// CombineResultsbyAlertID merges a slice of alerts for the same Id into one
func CombineResultsbyAlertID(raw []string) map[string]Alert {
	resultsmap := make(map[string][]Result)
	alerts := make(map[string]Alert)

	for _, alertStr := range raw {
		var alert Alert
		json.Unmarshal([]byte(alertStr), &alert)
		for _, result := range alert.Results {
			resultsmap[alert.ID] = append(resultsmap[alert.ID], result)
		}
		alert.Results = nil
		alerts[alert.ID] = alert
	}
	for id, results := range resultsmap {
		temp := alerts[id]
		temp.Results = results
		alerts[id] = temp
	}
	return alerts
}
