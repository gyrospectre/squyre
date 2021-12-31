package opsgenie

import (
	"fmt"
	"github.com/gyrospectre/hellarad"
	"io/ioutil"
	"net/http"
	"strings"
	"bytes"
	"encoding/json"
)

const (
	SecretLocation = "OpsGenieAPI"
	BaseURL  = "https://api.opsgenie.com/v2"
)

type apiKeySecret struct {
	Key	  string	`json:"apikey"`
}

func Send(results hellarad.Result, alertId string) (bool, error) {
	smresponse, _ := hellarad.GetSecret(SecretLocation)
	//fmt.Printf("%s", *apiKey.SecretString)

	var secret apiKeySecret
	json.Unmarshal([]byte(*smresponse.SecretString), &secret)
	fmt.Println(secret.Key)
    url := fmt.Sprintf("%s/alerts/%s/notes", strings.TrimSuffix(BaseURL, "/"), alertId)
	auth := fmt.Sprintf("GenieKey %s", secret.Key)

	var jsonData = []byte(`{
		"user": "Hella Rad!",
		"source": "hellarad",
		"Note": "Oooh yeah"
	}`)
	request, error := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	request.Header.Set("Authorization", auth)

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

    return true, nil
}

