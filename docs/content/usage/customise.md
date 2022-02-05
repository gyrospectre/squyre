---
title: "Customising for your environment"
date: 2022-02-05T14:16:07+11:00
draft: false
---

There are a couple of features which can be customised for your environment.

## Hostname Enrichment

Squyre will attempt to extract any internal hostnames from your alerts. Most organisations have a convention for endpoints and servers, but they vary considerably. As a result, you need to tell Squyre what your org's convention is.

Do this via an environment variable in `template.yaml` under the `ConductorFunction` section, specifying a Go compatible regular expression.
```
HOST_REGEX: A-[A-Z0-9]{6}
```
The above example will match hostnames such as `A-AB12CD`.

## Filtering out internal domains

In most cases, you don't want to enrich your internal domain names or email addresses, you're only concerned with domains unrelated to your organisation. Again, via an environment variable in `template.yaml` in the `ConductorFunction` section, you can tell Squyre to ignore your domain.
```
IGNORE_DOMAIN: your-internal-domain.int
```
