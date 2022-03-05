---
title: "Customising for your environment"
date: 2022-02-05T14:16:07+11:00
draft: false
---

There are a couple of features which can be customised for your environment.

## Enrichment Functions

You can modify `statemachine/enrich.asl.json` to change which enrichment functions run. The easiest way to do this is to use the included helper script, which provides a nice wizard to select functions.

```
make setup
```
Then run the deploy again to save your changes to AWS.

Don't have CrowdStrike? No problem, just remove that function! If you want, you can also remove unnecessary functions from `template.yaml` to cut down what gets deployed to AWS. If you save the new definition to a filename other than `statemachine/enrich.asl.json`, then don't forget to update `template.yaml` accordingly.

If you're `\m/` hardcore `\m/`, you can also edit the state machine definition from the [AWS Step Functions Workflow Studio](https://aws.amazon.com/blogs/aws/new-aws-step-functions-workflow-studio-a-low-code-visual-tool-for-building-state-machines/) in the AWS Console, then export as JSON back into `statemachine/enrich.asl.json`.

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
