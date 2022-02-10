---
title: "CrowdStrike Falcon"
date: 2022-02-05T14:05:25+11:00
draft: false
weight: 1
---

### Summary
Indicator context from the [CrowdStrike Falcon ](https://www.crowdstrike.com/endpoint-security-products/falcon-x-threat-intelligence/) threat intelligence database. Also provides information on corporate hosts running the Falcon agent.

Requires a paid Falcon Insight and Falcon X license.

### Supports
`ipv4`, `domain`, `sha256`, `hostname`

### Example Result

```
Found Falcon X indicator for 127.0.0.1:

Malicious confidence: 'High'.
Added: 2022-01-01 00:00:00 +0000 UTC
Updated: 2022-01-01 00:00:10 +0000 UTC

Labels: Killchain/C2,Malware/CobaltStrike
Kill Chains: C2
Malware Families: CobaltStrike
Vulnerabilities:
Threat Types: Commodity,Criminal,RAT
Targets:

More information at: https://falcon.crowdstrike.com/search/?term=_all:~'127.0.0.1'
```

### Setup
1. [Create a Falcon API key](https://help.falcon.io/hc/en-us/articles/360027409272-Getting-Access-to-Falcon-APIs)
2. In AWS, [create a new Secrets Manager secret](https://docs.aws.amazon.com/secretsmanager/latest/userguide/manage_create-basic-secret.html) called `CrowdstrikeAPI` in the same account/region as Squyre is deployed. Use the following content, obviously substituting your key and email. The secret should be of type `Other type of secret`.
```
{
  "ClientID": <the Client ID of the API key you just created>,
  "ClientSecret": <the Client Secret of the key>,
  "FalconCloud": <the Falcon Cloud region your account uses e.g. us-1, us-2, eu-1, us-gov-1>
}
```

### Environment Variables
`ONLY_LOG_MATCHES` : Set to true (in template.yaml) to only decorate an alert if the indicator was found in Greynoise. Default = `false`.