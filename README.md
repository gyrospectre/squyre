# Squyre

<img src="https://media.giphy.com/media/l0MYylLtnC1ADCGys/giphy.gif" alt="ooh so rad" width="300" align="right" />

Easy alert enrichment for overworked security teams!

[![Build CI](https://github.com/gyrospectre/squyre/actions/workflows/build.yml/badge.svg)](https://github.com/gyrospectre/squyre/actions/workflows/build.yml)
[![Build Docs](https://github.com/gyrospectre/squyre/actions/workflows/gh-pages.yml/badge.svg)](https://github.com/gyrospectre/squyre/actions/workflows/gh-pages.yml)

Squyre will help you deal with threats more effectively, decorating your security alerts by adding helpful information to provide context and help decide if this alert is cause for concern.

Check out the docs at https://gyrospectre.github.io/squyre/ for more information on the problem it solves and how it can work for you.

## Enrichment Functions
It's easy to add enrichment functions, and more will be added over time. Feel free to PR and contribute!

Currently supported:
- Alienvault OTX (https://otx.alienvault.com/) : Open Threat Exchange is the neighborhood watch of the global intelligence community. (Indicator types: ipv4, domain, url)
- Greynoise (https://www.greynoise.io/) : Tells security analysts what not to worry about. (Indicator types: ipv4)
- IP API (https://ipapi.com/) : IP address geolocation information. (Indicator types: ipv4)
- CrowdStrike Falcon (https://www.crowdstrike.com/endpoint-security-products/falcon-platform/) : Primarily utilising Falcon X for threat intelligence. (Indicator types: ipv4, domain, sha256, hostname)
