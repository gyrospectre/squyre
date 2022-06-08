---
title: "A Note On Error Handling"
date: 2022-02-05T14:16:07+11:00
draft: false
---

Because of the way Squyre is used, we're looking for short and sharp lookups for which failures are not the end of the world. 

For example; if an alert fires for suspicious activity and we're not able to get additional information from Greynoise, this doesn't stop the analyst from getting on with things, it's just that some manual work might be needed to do what Squyre would otherwise had done!

*Ultimately, the main priority is the alert, and enrichments are a bonus.*

For this reason, the enrichment functions will swallow errors experienced when calling different services, reporting the error in a `Result` object that is passed back into the alert/ticket. This `Result` has the `Success` attribute set to `False` to indicate this.

It's important to note that this means that enrichment lambdas will rarely fail (so neither will step function executions), but errors will be reported like all other enrichments - in alert tickets. This is intended to ensure that we get maximum benefit from each Squyre run, errors cause the least amount of impact on the real job of alert triage, but that errors are still made visible to the analyst so they know what manual rework they might need to do.
