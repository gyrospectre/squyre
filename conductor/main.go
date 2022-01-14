package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/aws/aws-sdk-go/service/sfn/sfniface"
	"github.com/gyrospectre/squyre"
)

var (
	privateBlocks []*net.IPNet
	// Stack defines the main stack in use
	Stack CloudformationStack
	// SendAlert abstracts the sendAlertToSfn function to allow for testing
	SendAlert = sendAlertToSfn
	// BuildDestination abstracts the BuildStateMachine function to allow for testing
	BuildDestination = BuildStateMachine
)

const (
	stepFunctionTimeout = 15
	stackName           = "squyre"
)

// CloudformationStack abstracts AWS Cloudformation stacks
type CloudformationStack struct {
	Client    cloudformationiface.CloudFormationAPI
	StackName string
}

func (s *CloudformationStack) getStackResourceArn(resourceName string) (string, error) {
	req := cloudformation.ListStackResourcesInput{
		StackName: aws.String(s.StackName),
	}

	for {
		resp, err := s.Client.ListStackResources(&req)
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
	return "", errors.New("No matching stack resources found")
}

// StateMachine abstracts AWS Step Functions
type StateMachine struct {
	Client      sfniface.SFNAPI
	FunctionArn string
}

// Execute starts a step function execution with the provided input data
func (s *StateMachine) Execute(input string) (*sfn.StartExecutionOutput, error) {
	result, err := s.Client.StartExecution(&sfn.StartExecutionInput{
		StateMachineArn: aws.String(s.FunctionArn),
		Input:           aws.String(input),
	})
	if err != nil {
		return nil, err
	}

	return result, err
}

// WaitForExecCompletion waits for a given step function execution to complete
func (s *StateMachine) WaitForExecCompletion(execArn *string) error {
	iter := 1
	var execStatus string

	for iter <= stepFunctionTimeout {
		result, err := s.Client.DescribeExecution(&sfn.DescribeExecutionInput{
			ExecutionArn: execArn,
		})
		if err != nil {
			return err
		}

		execStatus = aws.StringValue(result.Status)

		if execStatus == "SUCCEEDED" {
			log.Printf("Step function exec succeeded after %d seconds.", iter)
			return nil
		}
		if execStatus == "FAILED" {
			log.Printf("Step function exec failed. Full details: %s", result.GoString())
			return errors.New("Step function execution failed")
		}
		if execStatus == "TIMED_OUT" || execStatus == "ABORTED" {
			break
		}

		time.Sleep(time.Second)
		iter++
	}

	log.Printf("Step function exec timed out after %d seconds.", iter)
	return errors.New("Step function timed out")
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

func init() {
	sess := session.Must(session.NewSession())

	Stack = CloudformationStack{
		Client:    cloudformation.New(sess),
		StackName: stackName,
	}
}

func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)

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

func extractIPs(details string) []squyre.Subject {
	var subjectList []squyre.Subject
	re := regexp.MustCompile(`(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}`)
	submatchall := re.FindAllString(details, -1)

	if len(privateBlocks) < 1 {
		setupIPBlocks()
	}

	submatchall = removeDuplicateStr(submatchall)

	for _, address := range submatchall {
		var subject = squyre.Subject{
			Type:  "ipv4",
			Value: address,
		}

		// Ignore private IP addresses
		if isPrivateIP(address) == false {
			subjectList = append(subjectList, subject)
		}
	}
	return subjectList
}

func convertSplunkAlert(alertBody string) squyre.Alert {
	var messageObject squyre.SplunkAlert
	json.Unmarshal([]byte(alertBody), &messageObject)

	return messageObject.Normaliser()
}

func convertOpsGenieAlert(alertBody string) squyre.Alert {
	var messageObject squyre.OpsGenieAlert
	json.Unmarshal([]byte(alertBody), &messageObject)

	return messageObject.Normaliser()
}

// BuildStateMachine builds a connection to the Step Function at the provided arn
func BuildStateMachine(arn string) StateMachine {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	return StateMachine{
		Client:      sfn.New(sess),
		FunctionArn: arn,
	}
}
func sendAlertToSfn(alert squyre.Alert, sfnName string) error {
	// Convert alert to a Json string ready to pass to our AWS Step Function
	alertJSON, _ := json.Marshal(alert)

	// Find the Arn of the required step function
	sfnArn, err := Stack.getStackResourceArn(sfnName)
	if err != nil {
		return err
	}
	stepFunction := BuildDestination(sfnArn)
	result, err := stepFunction.Execute(string(alertJSON))

	if err != nil {
		return err
	}
	log.Printf("Started %s with execution %s\n", sfnName, aws.StringValue(result.ExecutionArn))
	err = stepFunction.WaitForExecCompletion(result.ExecutionArn)

	return err
}

func handleRequest(ctx context.Context, snsEvent events.SNSEvent) (string, error) {
	if len(snsEvent.Records) == 0 {
		return "Aborted", errors.New("No records in SNS event to process")
	}
	for _, record := range snsEvent.Records {
		snsRecord := record.SNS
		var alert squyre.Alert
		log.Printf("Processing message %s\n", snsRecord.MessageID)

		if strings.Contains(snsRecord.Message, "search_name") {
			log.Println("Auto detected Splunk alert")
			alert = convertSplunkAlert(snsRecord.Message)
		} else if strings.Contains(snsRecord.Message, "integrationName") {
			log.Println("Auto detected OpsGenie alert")
			alert = convertOpsGenieAlert(snsRecord.Message)
		} else {
			return "", errors.New("Could not determine alert type")
		}
		alert.Subjects = extractIPs(alert.RawMessage)
		if len(alert.Subjects) == 0 {
			return "", errors.New("No public IP addresses found to process")
		}
		log.Printf("Extracted %d public IP addresses from the alert message", len(alert.Subjects))

		// Have finished adding the extracted subjects to our alert

		err := SendAlert(alert, "EnrichIPStateMachine")
		if err != nil {
			return string(err.Error()), err
		}
		log.Printf("Successfully processed %d entries for alert %s!\n\n", len(alert.Subjects), alert.ID)
	}

	return fmt.Sprintf("Processed %d SNS messages.", len(snsEvent.Records)), nil
}

func main() {
	lambda.Start(handleRequest)
}
