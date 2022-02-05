---
title: Squyre
---
<img src="https://media.giphy.com/media/l0MYylLtnC1ADCGys/giphy.gif" alt="ooh so rad" width="250" align="right" />

Easy alert enrichment for overworked security teams!

Squyre will help you deal with threats more effectively, decorating your security alerts by adding helpful information to provide context and help decide if this alert is cause for concern.

## The Problem

Once beyond the earliest stages of maturity, Security teams build processes to generate alerts to tell them when threats may be active in their environment. These alerts must be "triaged" by an analyst; as in, deciding which of the following applies.

- **"false positive"** the alert was triggered by an event that it was not designed to catch
- **"true positive benign"** the alert was triggered by the intended event, but the activity is acceptable and does not require further action.
- **"true positive malicious"** the alert was triggered by the intended event. It's bad and we need to call an incident!

The goal of threat detection is to be as accurate as possible with the last category, and minimise the other two. Unfortunately this is quite difficult! Enterprise environments are complex, and they have lots of humans doing complex stuff in them; you will always have some level of false positives/true positive benign alerts.

Unfortunately, alerts of these undesirable types can be quite hard on the analyst! Alerts almost never contain all the information needed to be able to triage. An analyst will perform research on the host, IP address, file hash etc in the alert, trying to get context on what all of this information means and whether it means something bad has happened. This is time consuming and requires switching to numerous tools, websites etc to gather various parts of the puzzle.

At scale, this leads to "alert fatigue": de-sensitising analysts with repetitive tasks, leading to missed or ignored alerts or delayed responses. It's also not much fun! Poor alert quality leads to frustrated security teams that are not very happy and likely to leave.

## Our Solution

[Ryan McGeehan's 2017 article "Lessons Learned in Detection Engineering"](https://medium.com/starting-up-security/lessons-learned-in-detection-engineering-304aec709856
) is one I keep coming back to - go read it if you haven't! In Ryan's words:

*"Great teams prepare the on-call analyst with as much information as possible."*
...
*"You should decorate alerts. This describes a standard of detail where an alert brings additional information to the analyst without requiring extra work. This helps avoids “tab hell” where an analyst needs to be logged into several tools to follow up on an incident, just to know what is going on."*
...
*"A rule should trigger automation that pulls in corresponding information, including log snippets, translation of IDs or employee names, hostnames, opinions from threat intelligence, etc."*

This is exactly what Squyre does for you. Alerts are sent in, automation runs to gather this information, and them added to your alert - "decorating" them.

It is cheap and relatively easy to run, improving your ability to catch threats, and keeping your team happy and focussed on what you pay them for! There are commercial products out there to do this, but they can get VERY expensive.

**That's the goal of the project - put good alert management into everyone's reach, regardless of their company's size or budget.**