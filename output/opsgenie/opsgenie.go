package opsgenie

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gyrospectre/hellarad"
	"log"
	"net/http"
	"strings"
)

const (
	SecretLocation = "OpsGenieAPI"
	BaseURL        = "https://api.opsgenie.com/v2"
)

type apiKeySecret struct {
	Key string `json:"apikey"`
}

type opsgenieNote struct {
	User   string `json:"user"`
	Source string `json:"source"`
	Note   string `json:"note"`
}

func Send(result hellarad.Result, alertId string) {

	var secret apiKeySecret

	message, _ := json.Marshal(result.Message)
	fmt.Printf("Debug: Message is - %s", message)
	smresponse, err := hellarad.GetSecret(SecretLocation)
	if err != nil {
		log.Fatalf("Failed to fetch OpsGenie secret: %s", err)
	}

	json.Unmarshal([]byte(*smresponse.SecretString), &secret)

	ogurl := fmt.Sprintf("%s/alerts/%s/notes", strings.TrimSuffix(BaseURL, "/"), alertId)
	auth := fmt.Sprintf("GenieKey %s", secret.Key)

	fmt.Printf("%s", message)
	note := &opsgenieNote{
		User:   "Hella Rad!",
		Source: result.Source,
		Note:   string(message),
	}
	note.Note = string(message)
	jsonData, err := json.Marshal(note)
	if err != nil {
		log.Fatalf("Could not marshal JSON into Note: %s", err)
	}

	request, _ := http.NewRequest("POST", ogurl, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	request.Header.Set("Authorization", auth)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Fatalf("Error posting data to OpsGenie: %s", err)
	}
	defer response.Body.Close()

	return
}
