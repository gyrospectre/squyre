---
title: "Developing"
date: 2022-02-05T14:03:08+11:00
draft: false
---

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
