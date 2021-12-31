package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/gyrospectre/hellarad"
	"log"
	"regexp"
	"strings"
	"time"
)

type Alert interface {
	Normaliser() hellarad.Alert
}

type SplunkAlert struct {
	Message       string `json:"message"`
	CorrelationId string `json:"correlation_id"`
	SearchName    string `json:"search_name"`
}

func (alert SplunkAlert) Normaliser() hellarad.Alert {
	return hellarad.Alert{
		Details: alert.Message,
		Id:      alert.CorrelationId,
	}
}

type OpsGenieAlert struct {
	Action string `json:"action"`
	Alert  struct {
		AlertId string `json:"alertId"`
		Message string `json:"message"`
	} `json:"alert"`
}

func (alert OpsGenieAlert) Normaliser() hellarad.Alert {
	return hellarad.Alert{
		Details: alert.Alert.Message,
		Id:      alert.Alert.AlertId,
	}
}

func getStackResourceArn(svc *cloudformation.CloudFormation, stackName string, resourceName string) (string, error) {
	req := cloudformation.ListStackResourcesInput{
		StackName: aws.String(stackName),
	}
	for {
		resp, err := svc.ListStackResources(&req)
		if err != nil {
			return "", err
		}
		for _, s := range resp.StackResourceSummaries {
			if *s.LogicalResourceId == resourceName {
				return *s.PhysicalResourceId, nil
			}
		}
		req.NextToken = resp.NextToken
		if aws.StringValue(req.NextToken) == "" {
			break
		}
	}
	return "", errors.New("No matching stack resources found!")
}

func extractAndAddIPs(details string, subjectList []hellarad.Subject) []hellarad.Subject {
	re := regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)
	submatchall := re.FindAllString(details, -1)

	for _, address := range submatchall {
		var subject = hellarad.Subject{
			IP: address,
		}
		subjectList = append(subjectList, subject)
	}
	return subjectList
}

func convertSplunkAlert(alertBody string) hellarad.Alert {
	var messageObject SplunkAlert
	json.Unmarshal([]byte(alertBody), &messageObject)

	return messageObject.Normaliser()
}

func convertOpsGenieAlert(alertBody string) hellarad.Alert {
	var messageObject OpsGenieAlert
	json.Unmarshal([]byte(alertBody), &messageObject)

	return messageObject.Normaliser()
}

func HandleRequest(ctx context.Context, snsEvent events.SNSEvent) (string, error) {
	var inputList []hellarad.Subject
	var results string
	for _, record := range snsEvent.Records {
		snsRecord := record.SNS
		var alert hellarad.Alert

		log.Printf("Processing message %s\n", snsRecord.MessageID)
        log.Printf("Raw message: %s\n", snsRecord.Message)

		if strings.Contains(snsRecord.Message, "search_name") {
			log.Println("Auto detected Splunk alert")
			alert = convertSplunkAlert(snsRecord.Message)
		} else {
			log.Println("Auto detected OpsGenie alert")
			alert = convertOpsGenieAlert(snsRecord.Message)
		}
		inputList = extractAndAddIPs(alert.Details, inputList)

		if len(inputList) == 0 {
			return "", errors.New("No IP addresses found to process!")
		}

		inputJson, _ := json.Marshal(inputList)

		sesh := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		cfnsvc := cloudformation.New(sesh)
		sfnArn, err := getStackResourceArn(cfnsvc, "hellarad", "IPLookupStateMachine")

		if err != nil {
			return sfnArn, err
		}

		sfnsvc := sfn.New(sesh)
		result, err := sfnsvc.StartExecution(&sfn.StartExecutionInput{
			StateMachineArn: &sfnArn,
			Input:           aws.String(string(inputJson)),
		})
		if err != nil {
			fmt.Print(string(err.Error()))
			return string(err.Error()), err
		}

		log.Printf("Started IP Lookup with execution %s\n", aws.StringValue(result.ExecutionArn))
		iter := 1
		for iter < 10 {
			result, _ := sfnsvc.DescribeExecution(&sfn.DescribeExecutionInput{
				ExecutionArn: result.ExecutionArn,
			})
			if aws.StringValue(result.Output) != "" {
				results = aws.StringValue(result.Output)
				break
			}
			time.Sleep(time.Second)
			iter += iter
		}
		log.Printf("Successfully processed %d entries for alert %s!\n\n", len(inputList), alert.Id)
		log.Printf("Results: %s\n", results)
	}

	return fmt.Sprintf("Processed %d SNS messages.", len(snsEvent.Records)), nil
}

func main() {
	lambda.Start(HandleRequest)
}
