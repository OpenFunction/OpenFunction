<p align="center">
<a href="https://openfunction.dev/"><img src="docs/images/openfunction-logo-gif.gif" alt="banner" width="500px"></a>
</p>

<p align="center">
<b>Cloud native FaaS platform for running Serverless workloads with ease</b>
</p>

<p align=center>
<a href="https://goreportcard.com/report/github.com/openfunction/openfunction"><img src="https://goreportcard.com/badge/github.com/openfunction/openfunction" alt="A+"></a>
<a href="https://hub.docker.com/r/openfunction/openfunction"><img src="https://img.shields.io/docker/pulls/openfunction/openfunction"></a>
<a href="https://github.com/OpenFunction/OpenFunction/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22"><img src="https://img.shields.io/github/issues/openfunction/openfunction?label=good%20first%20issues" alt="good first"></a>
<a href="https://twitter.com/intent/follow?screen_name=KubeSphere"><img src="https://img.shields.io/twitter/follow/KubeSphere?style=social" alt="follow on Twitter"></a>
<a href="https://cloud-native.slack.com/archives/C03ETDMD3LZ"><img src="https://img.shields.io/badge/Slack-600%2B-blueviolet?logo=slack&amp;logoColor=white"></a>
<a href="https://www.youtube.com/channel/UCyTdUQUYjf7XLjxECx63Hpw"><img src="https://img.shields.io/youtube/channel/subscribers/UCyTdUQUYjf7XLjxECx63Hpw?style=social"></a>
</p>

## 👀 Overview

[OpenFunction](https://openfunction.dev/) is a cloud-native open source FaaS (Function as a Service) platform aiming to let you focus on your business logic without having to maintain the underlying runtime environment and infrastructure. You only need to submit business-related source code in the form of functions.

<div align=center><img src=docs/images/function-lifecycle.svg></div>

OpenFunction features include:

- Cloud agnostic and decoupled with cloud providers' BaaS
- Pluggable architecture that allows multiple function runtimes
- Support both sync and async functions
- Unique async functions support that can consume events directly from event sources
- Support generating OCI-Compliant container images directly from function source code.
- Flexible autoscaling between 0 and N
- Advanced async function autoscaling based on event sources' specific metrics
- Simplified BaaS integration for both sync and async functions by introducing [Dapr](https://dapr.io/) 
- Advanced function ingress & traffic management powered by [K8s Gateway API](https://gateway-api.sigs.k8s.io/)
- Flexible and easy-to-use events management framework

## ☸ Architecture

<div align=center><img width="120%" height="120%" src=docs/images/openfunction-architecture.svg/></div>

OpenFunction manages its components in the form of Custom Resource Definitions (CRD) throughout the lifecycle of a function, you can find more details in the [Concepts](https://openfunction.dev/docs/concepts/) section.

<div align=center><img src=docs/images/OpenFunction-events-architecture.svg></div>

OpenFunction Events is OpenFunction's events framework, you can refer to [OpenFunction Events](https://github.com/OpenFunction/OpenFunction/blob/main/docs/concepts/OpenFunction-events-framework.md) for more information.

## 🚀 QuickStart

### Install OpenFunction

To install OpenFunction, please refer to [Installation Guide](https://openfunction.dev/docs/getting-started/installation/#install-openfunction).

### Create functions

You can find guides to create the sync and async functions in different languages [here](https://openfunction.dev/docs/getting-started/quickstarts/)

### Uninstall OpenFunction

To uninstall OpenFunction, please refer to [Uninstallation Guide](https://openfunction.dev/docs/getting-started/installation/#uninstall-openfunction).

### FAQ

When you encounter any problems when using OpenFunction, you can refer to the [FAQ](https://openfunction.dev/docs/reference/faq/) for help.

## 💻 Development

See the [Development Guide](docs/development/README.md) to get started with developing this project.

## 🛣️ Roadmap

Here you can find OpenFunction [roadmap](https://github.com/orgs/OpenFunction/projects/3/views/1?layout=board).

## 🏘️ Community

### [Contact Us](https://github.com/OpenFunction/community#contact-us)

### [Community Call](https://github.com/OpenFunction/community#community-call)

### [Events](https://github.com/OpenFunction/community#events)

## Landscape
 
<p align="center">
<br/><br/>
<img src="https://landscape.cncf.io/images/left-logo.svg" width="150"/>&nbsp;&nbsp;<img src="https://landscape.cncf.io/images/right-logo.svg" width="200"/>&nbsp;&nbsp;
<br/><br/>
OpenFunction is a CNCF Sandbox project now which also enriches the <a href="https://landscape.cncf.io/serverless?license=apache-license-2-0">CNCF Cloud Native Landscape.
</a>
</p>

## 📊 Status

![Alt](https://repobeats.axiom.co/api/embed/48814fec53572bf75ac4de9d4f447d2c978b26ee.svg "Repobeats analytics image")
