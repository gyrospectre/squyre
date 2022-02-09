package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gyrospectre/squyre/pkg/squyre"
)

const (
	provider       = "IP API"
	baseURL        = "http://api.ipapi.com/"
	supports       = "ipv4"
	secretLocation = "IPAPI"
)

var (
	// GetIPInfo abstracts this function to allow for tests
	GetIPInfo      = getIPInfo
	responseObject ipapiResponse
	InitClient     = initIPAPIClient
)

var template = `IP API result for %s:

Country: %s
City: %s, %s
`

type apiKeySecret struct {
	ApiKey string `json:"apikey"`
}

type apiClient struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

type ipapiResponse struct {
	IP                      string `json:"ip"`
	Type                    string `json:"type"`
	ContinentCode           string `json:"continent_code"`
	ContinentName           string `json:"continent_name"`
	CountryCode             string `json:"country_code"`
	CountryName             string `json:"country_name"`
	RegionCode              string `json:"region_code"`
	RegionName              string `json:"region_name"`
	City                    string `json:"city"`
	Zip                     string `json:"zip"`
	Latitude                string `json:"latitude"`
	Longitude               string `json:"longitude"`
	CountryFlag             string `json:"country_flag"`
	CountryFlagEmoji        string `json:"country_flag_emoji"`
	CountryFlagEmojiUnicode string `json:"country_flag_emoji_unicode"`
	calling_code            string `json:"calling_code"`
	IsEu                    bool   `json:"is_eu"`
}

func initIPAPIClient() (*apiClient, error) {
	// Fetch API key from Secrets Manager
	smresponse, err := squyre.GetSecret(secretLocation)
	if err != nil {
		log.Errorf("Failed to fetch %s secret: %s", provider, err)
	}

	var secret apiKeySecret
	json.Unmarshal([]byte(*smresponse.SecretString), &secret)

	client := &apiClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		apiKey: secret.ApiKey,
	}

	return client, nil
}

func getIPInfo(c *apiClient, ipv4 string) (*http.Response, error) {
	request, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/%s?access_key=%s", c.baseURL, ipv4, c.apiKey),
		nil,
	)
	if err != nil {
		return nil, err
	}
	return c.httpClient.Do(request)
}

func handleRequest(ctx context.Context, alert squyre.Alert) (string, error) {
	log.Infof("Starting %s run for alert %s", provider, alert.ID)

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
				Success:        false,
			}

			response, err := GetIPInfo(client, subject.Value)

			if err != nil {
				log.Errorf("Failed to fetch data from %s", provider)
				return "Error fetching data from API!", err
			}
			responseData, err := ioutil.ReadAll(response.Body)

			if err == nil {
				log.Infof("Received %s response for %s", provider, subject.Value)

				json.Unmarshal(responseData, &responseObject)

				result.Success = true
				result.Message = messageFromResponse(responseObject)
			} else {
				log.Errorf("Unexpected response from %s for %s", provider, subject.Value)
				return "Error decoding response from API!", err
			}
			// Add the enriched details back to the results
			alert.Results = append(alert.Results, result)
			log.Infof("Added %s to result set", subject.Value)
		} else {
			log.Info("Subject not supported by this provider. Skipping.")
		}
	}
	log.Infof("Successfully ran %s. Yielded %d results for %d subjects.", provider, len(alert.Results), len(alert.Subjects))

	// Convert the alert object into Json for the step function
	finalJSON, _ := json.Marshal(alert)
	return string(finalJSON), nil
}

func messageFromResponse(response ipapiResponse) string {
	message := fmt.Sprintf(template,
		response.IP,
		response.CountryName,
		response.City,
		response.RegionName,
	)

	return string(message)
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
	lambda.Start(handleRequest)
}
