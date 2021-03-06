package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
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
	"github.com/gyrospectre/squyre/pkg/squyre"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/publicsuffix"
	"mvdan.cc/xurls/v2"
)

var (
	privateBlocks []*net.IPNet
	// Stack defines the main stack in use
	Stack CloudformationStack
	// SendAlert abstracts the sendAlertToSfn function to allow for testing
	SendAlert = sendAlertToSfn
	// BuildDestination abstracts the BuildStateMachine function to allow for testing
	BuildDestination = BuildStateMachine
	// HostRegex defines the pattern for hostnames in your organisation, comes from an env var
	HostRegex = os.Getenv("HOST_REGEX")
	// IgnoreDomain optionally specifies a domain to ignore when extracting domains, comes from an env var
	IgnoreDomain = os.Getenv("IGNORE_DOMAIN")
)

const (
	stepFunctionTimeout = 15
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
			log.Infof("Step function exec succeeded after %d seconds.", iter)
			return nil
		}
		if execStatus == "FAILED" {
			log.Errorf("Step function exec failed. Full details: %s", result.GoString())
			return errors.New("Step function execution failed")
		}
		if execStatus == "TIMED_OUT" || execStatus == "ABORTED" {
			break
		}

		time.Sleep(time.Second)
		iter++
	}

	log.Errorf("Step function exec timed out after %d seconds.", iter)
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
		StackName: os.Getenv("STACK_NAME"),
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

func removeDuplicateTrimmedStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		trimmed := strings.Trim(item, " {}=[],")
		if _, value := allKeys[trimmed]; !value {
			allKeys[trimmed] = true
			list = append(list, trimmed)
		}
	}
	return list
}

func extractHosts(details string) []squyre.Subject {
	if HostRegex == "" {
		log.Warn("Env var IGNORE_DOMAIN is not set!")
		return nil
	}
	var subjectList []squyre.Subject

	regex := `(^|[ =\{\}\[])` + HostRegex + `($|[ ,\{\}\]])`

	re := regexp.MustCompile(regex)

	submatchall := re.FindAllString(details, -1)
	submatchall = removeDuplicateTrimmedStr(submatchall)

	for _, hostname := range submatchall {
		var subject = squyre.Subject{
			Type:  "hostname",
			Value: hostname,
		}
		subjectList = append(subjectList, subject)
	}
	return subjectList
}

func extractIPs(details string) []squyre.Subject {
	var subjectList []squyre.Subject

	// Only match IP addresses bounded by space, start/end of line, '=' or braces.
	// Prevents a lot of false positive matches!
	re := regexp.MustCompile(`(^|[ =\{\}\[])(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}($|[ ,\{\}\]])`)

	submatchall := re.FindAllString(details, -1)

	if len(privateBlocks) < 1 {
		setupIPBlocks()
	}

	submatchall = removeDuplicateTrimmedStr(submatchall)

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

func extractDomains(details string) []squyre.Subject {
	var subjectList []squyre.Subject
	re := regexp.MustCompile(`(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z]`)

	submatchall := re.FindAllString(details, -1)

	submatchall = removeDuplicateTrimmedStr(submatchall)

	if IgnoreDomain == "" {
		log.Warn("Env var IGNORE_DOMAIN is not set!")
	}

	for _, domain := range submatchall {
		if (IgnoreDomain != "" || !strings.Contains(domain, IgnoreDomain)) || IgnoreDomain == "" {
			var subject = squyre.Subject{
				Type:  "domain",
				Value: domain,
			}
			// Ignore TLDs that are not official
			_, icann := publicsuffix.PublicSuffix(domain)
			if icann {
				log.Infof("Adding domain %s.", domain)
				subjectList = append(subjectList, subject)
			} else {
				log.Infof("Ignoring internal domain %s.", domain)
			}
		} else if IgnoreDomain != "" {
			log.Infof("Ignoring domain %s per env var.", domain)
		}
	}
	return subjectList
}

func extractUrls(details string) []squyre.Subject {
	var subjectList []squyre.Subject

	rxStrict := xurls.Strict()
	submatchall := rxStrict.FindAllString(details, -1)
	submatchall = removeDuplicateTrimmedStr(submatchall)

	for _, url := range submatchall {
		if strings.Contains(url, "safelinks.protection.outlook.com") {
			url = normaliseAtpSafeLink(url)
		}

		var subject = squyre.Subject{
			Type:  "url",
			Value: url,
		}

		subjectList = append(subjectList, subject)
	}
	return subjectList
}

// normaliseAtpSafeLink extracts the target Url from a M365 ATP safe link. It will return the raw safe link if parsing fails for any reason.
func normaliseAtpSafeLink(safeurl string) string {
	splitUrl := strings.Split(safeurl, "?url=")
	if len(splitUrl) < 2 {
		log.Error("Could not parse ATP Safe Link! URL missing.")
		return safeurl
	}
	encodedUrl := splitUrl[1]
	decodedUrl, err := url.PathUnescape(encodedUrl)
	if err != nil {
		log.Errorf("Could not parse ATP Safe Link!: %s", err)
		return safeurl
	}

	splitUrl = strings.Split(decodedUrl, "&")

	if len(splitUrl) == 1 {
		log.Error("Could not parse ATP Safe Link! Data missing.")
		return safeurl
	}

	return splitUrl[0]
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

func convertSumoAlert(alertBody string) squyre.Alert {
	var messageObject squyre.SumoLogicAlert
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
	log.Infof("Started %s with execution %s\n", sfnName, aws.StringValue(result.ExecutionArn))
	err = stepFunction.WaitForExecCompletion(result.ExecutionArn)

	return err
}

func handleRequest(ctx context.Context, event map[string]interface{}) (string, error) {
	eventStr, _ := json.Marshal(event)

	var snsEvent events.SNSEvent
	var apiEvent events.APIGatewayProxyRequest
	var messages []string
	if strings.Contains(string(eventStr), "\"EventSource\":\"aws:sns\"") {
		log.Info("Detected SNS source.")
		json.Unmarshal(eventStr, &snsEvent)
		if len(snsEvent.Records) == 0 {
			return "Aborted", errors.New("No records in SNS event to process")
		}
		for _, record := range snsEvent.Records {
			snsRecord := record.SNS
			log.Infof("Processing message %s\n", snsRecord.MessageID)
			messages = append(messages, snsRecord.Message)
		}
	} else if strings.Contains(string(eventStr), "apiId") {
		log.Info("Detected API GW source.")
		json.Unmarshal(eventStr, &apiEvent)
		messages = append(messages, apiEvent.Body)
	} else {
		return "Aborted", errors.New("Invocation service not supported. Can only use SNS or API GW!")
	}

	var scope []string
	for _, message := range messages {
		var alert squyre.Alert

		if strings.Contains(message, "search_name") {
			log.Info("Auto detected Splunk alert")
			alert = convertSplunkAlert(message)
		} else if strings.Contains(message, "integrationName") {
			log.Info("Auto detected OpsGenie alert")
			alert = convertOpsGenieAlert(message)
		} else if strings.Contains(message, "Sumo Logic") {
			log.Info("Auto detected Sumo Logic alert")
			alert = convertSumoAlert(message)
		} else {
			return "", errors.New("Could not determine alert type")
		}

		// IPV4
		ipSubjects := extractIPs(alert.RawMessage)
		if len(ipSubjects) == 0 {
			log.WithFields(log.Fields{
				"alert": alert.ID,
			}).Info("No public IP addresses found to process")
		} else {
			for _, sub := range ipSubjects {
				alert.Subjects = append(alert.Subjects, sub)
			}
			log.WithFields(log.Fields{
				"alert": alert.ID,
			}).Infof("Extracted %d public IP addresses from the alert message", len(ipSubjects))
			scope = append(scope, "ipv4")
		}

		// Domains
		domainSubjects := extractDomains(alert.RawMessage)
		if len(domainSubjects) == 0 {
			log.WithFields(log.Fields{
				"alert": alert.ID,
			}).Info("No domains found to process")
		} else {
			for _, sub := range domainSubjects {
				alert.Subjects = append(alert.Subjects, sub)
			}
			log.WithFields(log.Fields{
				"alert": alert.ID,
			}).Infof("Extracted %d domains from the alert message", len(domainSubjects))
			scope = append(scope, "domain")
		}

		// Hosts
		hostSubjects := extractHosts(alert.RawMessage)
		if len(hostSubjects) == 0 {
			log.WithFields(log.Fields{
				"alert": alert.ID,
			}).Info("No hosts found to process")
		} else {
			for _, sub := range hostSubjects {
				alert.Subjects = append(alert.Subjects, sub)
			}
			log.WithFields(log.Fields{
				"alert": alert.ID,
			}).Infof("Extracted %d hosts from the alert message", len(hostSubjects))
			scope = append(scope, "hostname")
		}

		// Urls
		urlSubjects := extractUrls(alert.RawMessage)
		if len(urlSubjects) == 0 {
			log.WithFields(log.Fields{
				"alert": alert.ID,
			}).Info("No urls found to process")
		} else {
			for _, sub := range urlSubjects {
				alert.Subjects = append(alert.Subjects, sub)
			}
			log.WithFields(log.Fields{
				"alert": alert.ID,
			}).Infof("Extracted %d urls from the alert message", len(urlSubjects))
			scope = append(scope, "url")
		}

		// Have finished adding the extracted subjects to our alert
		if len(scope) == 0 {
			log.WithFields(log.Fields{
				"alert": alert.ID,
			}).Info("No subjects founds to process")
			return "", errors.New("No subjects found to process")
		}
		alert.Scope = strings.Join(scope, ",")

		err := SendAlert(alert, "EnrichStateMachine")
		if err != nil {
			log.WithFields(log.Fields{
				"alert": alert.ID,
			}).Error("Enrichment function failed")
			return string(err.Error()), err
		}
		log.WithFields(log.Fields{
			"alert": alert.ID,
		}).Infof("Successfully processed %d entries for alert %s!\n\n", len(alert.Subjects), alert.ID)
	}

	return fmt.Sprintf("Processed %d SNS messages.", len(snsEvent.Records)), nil
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
	lambda.Start(handleRequest)
}
