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
	"net"
	"regexp"
	"strings"
	"time"
)

var privateBlocks []*net.IPNet

const (
	StepFunctionTimeout = 15
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
		RawMessage: alert.Message,
		Id:         alert.CorrelationId,
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
		RawMessage: alert.Alert.Message,
		Id:         alert.Alert.AlertId,
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

func setupIPBlocks() {
	privateBlockStrs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
	}

	privateBlocks = make([]*net.IPNet, len(privateBlockStrs))

	for i, blockStr := range privateBlockStrs {
		_, block, _ := net.ParseCIDR(blockStr)
		privateBlocks[i] = block
	}
}

func isPrivateIP(ip_str string) bool {
	ip := net.ParseIP(ip_str)

	for _, priv := range privateBlocks {
		if priv.Contains(ip) {
			return true
		}
	}
	return false
}

func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func extractIPs(details string) []hellarad.Subject {
	var subjectList []hellarad.Subject
	re := regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)
	submatchall := re.FindAllString(details, -1)

	if len(privateBlocks) < 1 {
		setupIPBlocks()
	}

	submatchall = removeDuplicateStr(submatchall)

	for _, address := range submatchall {
		var subject = hellarad.Subject{
			IP: address,
		}

		// Ignore private IP addresses
		if isPrivateIP(address) == false {
			subjectList = append(subjectList, subject)
		}
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

func waitForSfn(svc *sfn.SFN, execArn *string) (bool, error){
	iter := 1
	for iter < StepFunctionTimeout {
		result, _ := svc.DescribeExecution(&sfn.DescribeExecutionInput{
			ExecutionArn: execArn,
		})
		if aws.StringValue(result.Output) != "" {
			break
		}
		time.Sleep(time.Second)
		iter += iter
	}
	if iter < StepFunctionTimeout {
		return true, nil	
	} else {
		return false, errors.New("Timed out waiting for Sfn execution to complete!")
	}
}

func sendAlertToSfn(alert hellarad.Alert, sfnName string) (string, error) {
		// Convert alert to a Json string ready to pass to our AWS Step Function
		alertJson, _ := json.Marshal(alert)

		// Find the Arn of the required step function
		sesh := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		cfnsvc := cloudformation.New(sesh)
		sfnArn, err := getStackResourceArn(cfnsvc, "hellarad", sfnName)
		if err != nil {
			return string(err.Error()), err
		}

		sfnsvc := sfn.New(sesh)
		result, err := sfnsvc.StartExecution(&sfn.StartExecutionInput{
			StateMachineArn: &sfnArn,
			Input:           aws.String(string(alertJson)),
		})
		if err != nil {
			return string(err.Error()), err
		}
		log.Printf("Started IP Lookup with execution %s\n", aws.StringValue(result.ExecutionArn))
		success, err := waitForSfn(sfnsvc, result.ExecutionArn)
		if success {
			return aws.StringValue(result.ExecutionArn), nil
		} else {
			return string(err.Error()), err
		}
}

func HandleRequest(ctx context.Context, snsEvent events.SNSEvent) (string, error) {
	for _, record := range snsEvent.Records {
		snsRecord := record.SNS
		var alert hellarad.Alert
		var subjects []hellarad.Subject

		log.Printf("Processing message %s\n", snsRecord.MessageID)

		if strings.Contains(snsRecord.Message, "search_name") {
			log.Println("Auto detected Splunk alert")
			alert = convertSplunkAlert(snsRecord.Message)
		} else {
			log.Println("Auto detected OpsGenie alert")
			alert = convertOpsGenieAlert(snsRecord.Message)
		}
		alert.Subjects = extractIPs(alert.Details)

		if len(alert.Subjects ) == 0 {
			return "", errors.New("No public IP addresses found to process!")
		}
		// Have finished adding the extracted subjects to our alert

		execArn, err := sendAlertToSfn(alert, "IPLookupStateMachine")
		if err != nil {
			return string(err.Error()), err
		}
		log.Printf("Successfully processed %d entries for alert %s!\n\n", len(alert.Subjects), alert.Id)
	}

	return fmt.Sprintf("Processed %d SNS messages.", len(snsEvent.Records)), nil
}

func main() {
	lambda.Start(HandleRequest)
}
