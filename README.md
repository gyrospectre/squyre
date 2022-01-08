# Squyre

Easy alert enrichment for overworked security teams! Squyre will help you deal with threats more effectively but, unlike the [historic role it was named after](https://en.wikipedia.org/wiki/Squire), it has a rad 'y' instead of an 'i' and is unlikely to scrub your armour.

![ooh so rad](https://media.giphy.com/media/l0MYylLtnC1ADCGys/giphy.gif)

Designed to be modular and extensible, it will consume your alerts, enrich them with information that helps with triage, and then feed the juicy results back into your alert pipeline (or ticketing system).

The only pre-requisite is that you must have an AWS account to host Squyre - it runs solely in AWS using serverless services (lambdas and step functions).

Currently, we support Splunk or OpsGenie as alert sources, and Jira or OpsGenie as output providers, but let me know if you need something else to make this work for you. Even better, build it yourself and contribute to the project!

## How can I use this?

As an example, let's say that your security team uses Splunk for alerting and investigation, and Atlassian Jira for ticketing. By using the SNS alert action in the free Splunk Add-on for AWS, you can set your alerts to send to Squyre, which will take the results you define as interesting, extract any public IP addresses from them, and then run them through a bunch of services to get information about then. Squyre will then create a Jira ticket for your alert, and add this information as comments.

Woot. Enjoy all that sweet, sweet extra time back in your day.

## Warnings
- Consider OpSec, some of your indicators may be sensitive, and you may not want them sent to the public service used by Squyre. Don't send things like this to it.
- This is not yet in Production use. It's been tested pretty extensively in a test environment, but probably still has plenty of bugs.
- This is my first foray into Go. I'm still learning, the code is not awesome. If you are a Go expert, I would love your feedback on how I can do better!

## Suggested Deployment Patterns
There are a couple of ways you can deploy, either directing between your alert source and ticketing system (pattern 1), or using an incident management platform like OpsGenie (pattern 2).

![diagram](https://github.com/gyrospectre/squyre/raw/main/diagram.png)

Pattern 1 is the out of the box configuration as it's the most generic. If you are using Splunk and Jira, but don't already have something in place to create tickets automatically when alerts fire, then this is for you.

Pattern 2 however, is a more scalable pattern. If you are already using Opsgenie in your alert pipeline, this is a better option. This allows you to add as many alert sources as you like, without having to change anything on the Squyre side.


## Getting Started - Pattern 1: Splunk to Jira Deployment
1. Clone this repo.
2. Install the AWS SAM CLI. See https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html
3. Update the consts at the top of `output/jira/main.go` with your destination Jira instance URL (`BaseURL`) and Project name (`Project`).
4. With appropriate AWS credentials in your terminal session, build and deploy the stack.
```
sam build
sam deploy --guided

Configuring SAM deploy
======================

	Looking for config file [samconfig.toml] :  Not found

	Setting default arguments for 'sam deploy'
	=========================================
	Stack Name [sam-app]: sqyre
	AWS Region [ap-southeast-2]:
	#Shows you resources changes to be deployed and require a 'Y' to initiate deploy
	Confirm changes before deploy [y/N]:
	#SAM needs permission to be able to create roles to connect to the resources in your template
	Allow SAM CLI IAM role creation [Y/n]:
	Save arguments to configuration file [Y/n]:
	SAM configuration file [samconfig.toml]:
	SAM configuration environment [default]:
```
5. Over on Splunk, install the Splunk Add-on for AWS (https://splunkbase.splunk.com/app/1876/), to give you an SNS alert action. 
6. Configure the app with some AWS creds. The IAM user or role must have SNS Publish/Get/List perms to SNS topic `squyre-Alert`. See https://docs.splunk.com/Documentation/AddOns/released/AWS/Setuptheadd-on
https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/

7. Create a Jira API key (https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/)
8. In AWS, create a new Secrets Manager secret called `JiraApi` in the same account/region as Squyre is deployed. Use the following content, obviously substituting your key and email.
```
{
  "apikey": <the API key you just created>,
  "user": <the email address of the Jira account the key is associated with>
}
```
9. Almost there! Update one of your Splunk saved searches, adding a `strcat` at the end to combine all the fields you think are of use to a new field called `interesting`.

`<awesome detection logic> | stats values(src_ip) as src_ip by dest_user | eval Detection="A test alert" | strcat src_ip "," dest_user interesting`

4. Add an 'AWS SNS Alert' action to your scheduled search (https://docs.splunk.com/Documentation/AddOns/released/AWS/ModularAlert), updating the 'Message' field of the action to `$result.interesting$`.
5. Also fill out the Account and Region fields per the doco for the AWS Tech Add-on. The topic should be set to `squyre-Alert`.

Next time this alert fires, the details will be sent to Squyre, which will create a Jira ticket for you, adding enrichment details for all extracted IP address to the same ticket as comments.

## Getting Started - Pattern 2: OpsGenie Deployment
1. Clone this repo.
2. Install the AWS SAM CLI. See https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html
3. Edit `template.yaml` to use OpsGenie instead of Jira. In the `OutputFunction` definition, change the `CodeUri` value to `output/opsgenie`.
4. With appropriate AWS credentials in your terminal session, build and deploy the stack.
```
sam build
sam deploy --guided

Configuring SAM deploy
======================

	Looking for config file [samconfig.toml] :  Not found

	Setting default arguments for 'sam deploy'
	=========================================
	Stack Name [sam-app]: sqyre
	AWS Region [ap-southeast-2]:
	#Shows you resources changes to be deployed and require a 'Y' to initiate deploy
	Confirm changes before deploy [y/N]:
	#SAM needs permission to be able to create roles to connect to the resources in your template
	Allow SAM CLI IAM role creation [Y/n]:
	Save arguments to configuration file [Y/n]:
	SAM configuration file [samconfig.toml]:
	SAM configuration environment [default]:
```
5. Create an OpsGenie integration API key. See https://support.atlassian.com/opsgenie/docs/create-a-default-api-integration/
6. In AWS, create a new Secrets Manager secret called `OpsGenieAPI` in the same account/region as Squyre is deployed. Use the following content, obviously substituting your key and email.
```
{
  "apikey": <the API key you just created>
}
```
7. Setup OpsGenie to send SNS messages to topic `squyre-Alert` on alert creation only. See https://support.atlassian.com/opsgenie/docs/integrate-opsgenie-with-outgoing-amazon-sns/

## Enrichment Functions
It's easy to add enrichment functions, and more will be added over time. Feel free to PR and contribute!

Currently supported:
- Greynoise (https://www.greynoise.io/) : Tells security analysts what not to worry about. Indicator types: IP
- IP API (https://ip-api.com/) : IP address geolocation information. Indicator types: IP

## Developing

### Data Structures
`squyre.Alert`   - The main data structure used by Squyre. It encapsulates everything about an alert, it's details and the enrichment results. `Alerts` are the standard way data is passed around between components.

`squyre.Subject` - Any collection of data points which can be used for enrichment. At the time of writing, either an IP address or a domain name. `Subjects` are stored within `Alerts`.

`squyre.Result`  - Stores Enrichment results, the subject used, and the source of the data. `Results` are also stored within `Alerts`.

### Enrichment Functions
An enrichment function is a Go lambda that takes a `squyre.Alert` as input (see `squyre.go`), performs some analysis, adds the results (as a slice of `squyre.Result` objects) to the Alert object, and returns a Json string representation of the updated Alert.

Have a look at any of the existing functions (in the `function`) folder, you should be able to copy paste a fair amount and get started pretty quick. If you need to work with API keys, please use AWS Secrets Manager to store your secrets; there is a built in function to fetch keys as required! For E.g. https://github.com/gyrospectre/squyre/blob/0ad801155f278d0e02894bd312eb4f0da2387341/output/jira/main.go#L49

Once you have something working, add the new function to the template.yaml (again copy one of the other stanzas) and then test:
```
make fmt
make lint
make test
make build
sam local invoke MyNewFunction --event event/alert.json
```
If all is working, then add the new function to the `statemachine/enrichIP.asl.json` file, so that it executes as part of the main workflow. Then you can `sam deploy` and try it out!

## Testing

Run Go unit tests
```
make test
```

Integration tests (requires AWS credentials in session, live calls)
```
make build
sam local invoke IPAPIFunction --event event/alert.json
sam local invoke GreynoiseFunction --event event/alert.json
sam local invoke ConductorFunction --event event/sns.json 
sam local invoke OutputFunction --event event/output.json 
```