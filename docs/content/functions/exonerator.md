---
title: "ExoneraTor"
date: 2022-06-15T08:05:25+11:00
draft: false
---

### Summary
ExoneraTor is a handy service from the Tor Project, which tells you if an IP was a Tor relay on a given date. For more information, check out https://metrics.torproject.org/exonerator.html. If an alert sourced from a Tor exit node (relay), this can be an interesting piece of information when triaging.

No API key is required.

### Supports
`ipv4`

### Example Result
```
ExoneraTor believes 127.0.0.1 was recently a Tor relay.

More information at: https://metrics.torproject.org/exonerator.html?ip=127.0.0.1&timestamp=2022-06-12&lang=en

```

### Setup
No setup required.

### Environment Variables
`ONLY_LOG_MATCHES` : Set to `true` (in template.yaml) to only decorate an alert if the IP was found to be a recent Tor relay. Default=`false`.