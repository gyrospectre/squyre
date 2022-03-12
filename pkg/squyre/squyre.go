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
	Scope      string // The types of Subjects in this alert, used by the step function
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

// SumoLogicAlert defines the standard format alerts come to us from Sumo
// See https://help.sumologic.com/Manage/Connections-and-Integrations/Webhook-Connections/Set_Up_Webhook_Connections#Webhook_payload_variables
type SumoLogicAlert struct {
	AlertID     string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	EventType   string `json:"event_type"`
	Client      string `json:"client"`
	ClientURL   string `json:"client_url"`
	TimeRange   string `json:"time_range"`
	TimeTrigger string `json:"time_trigger"`
	NumResults  string `json:"num_results"`
	Results     string `json:"results"`
}

// Normaliser comverts a Sumo Logic alert to our standard form
func (alert SumoLogicAlert) Normaliser() Alert {
	return Alert{
		RawMessage: alert.Results,
		ID:         alert.AlertID,
		Name:       alert.Name,
		Timestamp:  alert.TimeTrigger,
		URL:        alert.ClientURL,
	}
}

// CombineResultsbyAlertID merges a slice of alerts for the same Id into one
func CombineResultsbyAlertID(raw [][]string) map[string]Alert {

	/*
		The output function(s) are called with a slice of slice of srings.
		Each slice is an output from a group of enrichments e.g. IPv4, Domain etc. which contains
		another slice of the outputs from each function (as Alerts). This is the way the Step
		Function groups the results of parallel executions.

		This function collapses all of that structure down, grouping by alert ID with all the separate
		results within.
	*/

	resultsmap := make(map[string][]Result)
	alerts := make(map[string]Alert)

	// First, collapse the enrichment groups down into one big slice
	mergedGroups := []string{}
	for _, group := range raw {
		mergedGroups = append(mergedGroups, group...)
	}

	// Now, go through and organise as unique alerts with results contained within
	for _, alertStr := range mergedGroups {
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
