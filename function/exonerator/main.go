package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/squyre/pkg/squyre"
)

const (
	provider = "ExoneraTor"
	baseURL  = "https://metrics.torproject.org/exonerator.html"
	supports = "ipv4"
)

var (
	// GetIPInfo abstracts this function to allow for tests
	GetIPInfo         = getIPInfo
	InitClient        = initExoneraTorClient
	OnlyLogMatches, _ = strconv.ParseBool(os.Getenv("ONLY_LOG_MATCHES"))
)

var template = `
ExoneraTor believes %s was %srecently a Tor relay.

More information at: %s

`

type apiClient struct {
	httpClient *http.Client
	baseURL    string
}

func initExoneraTorClient() (*apiClient, error) {
	client := &apiClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}

	return client, nil
}

func dayBeforeYesterday() string {
	return time.Now().AddDate(0, 0, -2).Format("2006-01-02")
}

func getIPInfo(c *apiClient, ipv4 string) (*http.Response, error) {
	request, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s?ip=%s&timestamp=%s&lang=en", c.baseURL, ipv4, dayBeforeYesterday()),
		nil,
	)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Do(request)
}

func handleRequest(ctx context.Context, alert squyre.Alert) (string, error) {
	log.Infof("Starting %s run for alert %s", provider, alert.ID)
	log.Infof("OnlyLogMatches is set to %t", OnlyLogMatches)

	if len(alert.Subjects) == 0 {
		log.Info("Alert has no subjects to process.")
		finalJSON, _ := json.Marshal(alert)
		return string(finalJSON), nil
	}

	client, err := InitClient()
	if err != nil {
		return "Failed to initialise client", err
	}

	// Process each subject in the alert we were passed
	for _, subject := range alert.Subjects {
		if strings.Contains(supports, subject.Type) {

			// Build a result object to hold our goodies
			var result = squyre.Result{
				Source:         provider,
				AttributeValue: subject.Value,
				MatchFound:     false,
				Success:        false,
			}

			response, err := GetIPInfo(client, subject.Value)

			if err != nil || response.StatusCode != http.StatusOK {
				log.Errorf("Failed to fetch data from %s", provider)
				result.Success = false
				result.Message = err.Error() + fmt.Sprintf("(statuscode: %d)", response.StatusCode)
				alert.Results = append(alert.Results, result)
			} else {
				result.Success = true
				responseData, err := ioutil.ReadAll(response.Body)

				if err == nil {
					log.Infof("Received %s response for %s", provider, subject.Value)

					if strings.Contains(string(responseData), "Result is positive") {
						result.MatchFound = true
					} else if strings.Contains(string(responseData), "Result is negative") {
						result.MatchFound = false
					} else {
						log.Errorf("Unexpected response from %s", provider)
						result.Success = false
						result.Message = "Bad response, no result found in provider output!"
						alert.Results = append(alert.Results, result)
					}

					if !result.MatchFound && OnlyLogMatches {
						log.Infof("Skipping non match for %s", subject.Value)
					} else {
						// Match found. Add the enriched details back to the results
						result.Message = messageFromResponse(subject.Value, result.MatchFound)
						alert.Results = append(alert.Results, result)
						log.Infof("Added %s to result set", subject.Value)
					}
				} else {
					log.Errorf("Unexpected response from %s for %s", provider, subject.Value)
					return "Error decoding response from API!", err
				}
			}
		} else {
			log.Info("Subject not supported by this provider. Skipping.")
		}
	}

	log.Infof("Successfully ran %s. Yielded %d results for %d subjects.", provider, len(alert.Results), len(alert.Subjects))
	// Convert the alert object into Json for the step function
	finalJSON, _ := json.Marshal(alert)
	return string(finalJSON), nil
}

func messageFromResponse(ipv4 string, matchfound bool) string {
	negate := ""
	if !matchfound {
		negate = "NOT "
	}
	message := fmt.Sprintf(template,
		ipv4,
		negate,
		fmt.Sprintf("%s?ip=%s&timestamp=%s&lang=en", baseURL, ipv4, dayBeforeYesterday()),
	)

	return string(message)
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
	lambda.Start(handleRequest)
}
