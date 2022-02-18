---
title: "Alienvault OTX"
date: 2022-02-05T14:05:25+11:00
draft: false
---

### Summary
Indicator context from the [Alienvault OTX](https://otx.alienvault.com/) threat intelligence community.

No API key is required for lookups.

### Supports
`ipv4`, `domain`, `url`

### Example Result
```
Alienvault OTX has 1 matches for '127.0.0.1', in the following pulses:
IPQS Abusive IP List

More information at: https://otx.alienvault.com/browse/global/pulses?q=127.0.0.1
```

### Setup
No setup required.

### Environment Variables
`ONLY_LOG_MATCHES` : Set to `true` (in template.yaml) to only decorate an alert if the indicator was found in Alienvault OTX. Default=`false`.