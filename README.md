# Squyre

Easy alert enrichment for overworked security teams!

![ooh so rad](https://media.giphy.com/media/l0MYylLtnC1ADCGys/giphy.gif)

Squyre will help you deal with threats more effectively, decorating your security alerts by adding helpful information to provide context and help decide if this alert is cause for concern.

Check out the docs at https://gyrospectre.github.io/squyre/ for more information on the problem it solves and how it can work for you.

[![Build Docs](https://github.com/gyrospectre/squyre/actions/workflows/gh-pages.yml/badge.svg)](https://github.com/gyrospectre/squyre/actions/workflows/gh-pages.yml)

## How can I use this?

As an example, let's say that your security team uses Splunk for alerting and investigation, and Atlassian Jira for ticketing. By using the SNS alert action in the free Splunk Add-on for AWS, you can set your alerts to send to Squyre, which will take the results you define as interesting, extract any public IP addresses from them, and then run them through a bunch of services to get information about them. Squyre will then create a Jira ticket for your alert, and add this information as comments.

Woot. Enjoy all that sweet, sweet extra time back in your day.

## Enrichment Functions
It's easy to add enrichment functions, and more will be added over time. Feel free to PR and contribute!

Currently supported:
- Greynoise (https://www.greynoise.io/) : Tells security analysts what not to worry about. (Indicator types: ipv4)
- IP API (https://ip-api.com/) : IP address geolocation information. (Indicator types: ipv4)
- CrowdStrike Falcon (https://www.crowdstrike.com/endpoint-security-products/falcon-platform/) : Primarily utilising Falcon X for threat intelligence. (Indicator types: ipv4, domain, sha256, hostname)

## Developing

### Data Structures
`squyre.Alert`   - The main data structure used by Squyre. It encapsulates everything about an alert, it's details and the enrichment results. `Alerts` are the standard way data is passed around between components.

`squyre.Subject` - Any collection of data points which can be used for enrichment. At the time of writing, either an IP address or a domain name. `Subjects` are stored within `Alerts`.

`squyre.Result`  - Stores enrichment results, the subject used, and the source of the data. `Results` are also stored within `Alerts`.

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
If all is working, then add the new function to the `statemachine/enrich.asl.json` file, so that it executes as part of the main workflow. Then you can `sam deploy` and try it out!

## Testing

Run Go unit tests
```
make test
```

Integration tests (requires AWS credentials in session, live calls)
```
make build
# Test enrichment functions
sam local invoke IPAPIFunction --event event/ip-alert.json
sam local invoke GreynoiseFunction --event event/ip-alert.json

# Test Conductor from both potential sources of the SNS
sam local invoke ConductorFunction --event event/sns_from_splunk.json
sam local invoke ConductorFunction --event event/sns_from_opsgenie.json

# Test whichever output function you're using (either Jira or Opsgenie)
sam local invoke OutputFunction --event event/output.json
```
