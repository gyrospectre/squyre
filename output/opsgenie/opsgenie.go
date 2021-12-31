package opsgenie

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gyrospectre/hellarad"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	SecretLocation = "OpsGenieAPI"
	BaseURL  = "https://api.opsgenie.com/v2"
)

func Send(results hellarad.Result, alertId string) (bool, error) {
	apiKey, _ := hellarad.getSecret(SecretLocation)

    url := fmt.Sprintf("%s/alerts/%s/notes", strings.TrimSuffix(BaseURL, "/"), alertId)
	auth := fmt.Sprintf("Authorization: GenieKey %s", apikey)

	var jsonData = []byte(`{
		"name": "morpheus",
		"job": "leader"
	}`)
	request, error := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	request.Header.Set("Authorization", fmt.Sprintf("GenieKey %s"),apiKey)

	client := &http.Client{}
	response, error := client.Do(request)
	if error != nil {
		panic(error)
	}
	defer response.Body.Close()

	fmt.Println("response Status:", response.Status)
	fmt.Println("response Headers:", response.Header)
	body, _ := ioutil.ReadAll(response.Body)
	fmt.Println("response Body:", string(body))

    return true
}

