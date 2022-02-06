---
title: "Architecture"
date: 2022-02-05T14:16:07+11:00
draft: false
---

Squyre is serverless, and uses AWS services SNS, Lambda and Step Functions to do it's thing.

Alerts are sent to the SNS topic, which triggers the first Lambda function, `conductor`. This function take the alert body, extracts IP addresses, domain names and hostnames, and then starts the step function with this information.

The step function (or state machine) then invokes enrichment functions depending on what sort of info was in the alert. There are currently two categories of functions:

1. Multipurpose. These functions can enrich based on various data types, so are run on every alert.
2. IPv4. These functions can only enrich IP addresses, so only run if the alert contained at least one IP.

Enrichment functions run in parallel, and then once everything is done the output is passed on to the final Lambda, `output`. This function is responsible for adding the results to the chosen destination (either Jira or Opsgenie) as comments.

All of this is deployed via Cloudformation, to make it easy to spin up and down.
