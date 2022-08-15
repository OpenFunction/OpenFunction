- [OpenFunction/OpenFunction](#openfunction)
  * [0.1.0 / 2021-05-17](#010--2021-05-17)
  * [0.2.0 / 2021-06-30](#020--2021-06-30)
  * [0.3.0 / 2021-08-19](#030--2021-08-19)
  * [0.3.1 / 2021-08-27](#031--2021-08-27)
  * [0.4.0 / 2021-10-19](#040--2021-10-19)
  * [0.5.0 / 2021-12-31](#050--2021-12-31)
  * [0.6.0-rc.0 / 2022-03-08](#060-rc0--2022-03-08)
  * [0.6.0 / 2022-03-21](#060--2022-03-21)
  * [0.7.0-rc.0 / 2022-08-11](#070-rc0--2022-08-11)
- [OpenFunction/samples](#openfunctionsamples)
- [OpenFunction/website](#openfunctionwebsite)
- [OpenFunction/builder](#openfunctionbuilder)
- [OpenFunction/cli](#openfunctioncli)
- [OpenFunction/functions-framework](#openfunctionfunctions-framework)
- [OpenFunction/functions-framework-go](#openfunctionfunctions-framework-go)
- [OpenFunction/functions-framework-nodejs](#openfunctionfunctions-framework-nodejs)
- [OpenFunction/events-handlers](#openfunctionevents-handlers)

# OpenFunction

## 0.7.0-rc.0 / 2022-08-11

> Note: This release contains a few breaking changes.
- The `ofn install` and `ofn uninstall` CLI was deprecated.
- The `domains.core.openfunction.io` CRD was deprecated and removed.
- The cert-manager was removed.
- The Nginx ingress controller was removed.
- Use contour as the network layer of knative-serving instead of kourier.

### OpenFunction

#### Features

- [Add the parameter validation capabilities for Function](https://github.com/OpenFunction/OpenFunction/issues/274). [OpenFunction#290](https://github.com/OpenFunction/OpenFunction/pull/290)
- [Add Gateway & Route for OpenFunction](https://github.com/OpenFunction/OpenFunction/blob/main/docs/proposals/202207-openfunction-gateway.md). [OpenFunction#321](https://github.com/OpenFunction/OpenFunction/pull/321)

#### Enhancement
- Remove cert-manager, use generate-cert.sh to generate caBundle and tls.* files. [OpenFunction#261](https://github.com/OpenFunction/OpenFunction/pull/261)
- Remove the crd description to avoid "metadata.annotations too long" error when using "kubectl apply -f". [OpenFunction#264](https://github.com/OpenFunction/OpenFunction/pull/264)
- Add e2e testing for local environments. [OpenFunction#266](https://github.com/OpenFunction/OpenFunction/pull/266)
- Change the function sample's sourceSubPath & upgrade kustomize version. [OpenFunction#304](https://github.com/OpenFunction/OpenFunction/pull/304)
- Use fixed strings instead of knativeAutoscalingPrefix. [OpenFunction#311](https://github.com/OpenFunction/OpenFunction/pull/311)
- Remove domain crd & optimize path-based mode routing. [OpenFunction#327](https://github.com/OpenFunction/OpenFunction/pull/327)
- Add samples to gateway & improve gateway controller compatibility. [OpenFunction#333](https://github.com/OpenFunction/OpenFunction/pull/333)
- Add the compatibility with v0.6.0 functions. [OpenFunction#344](https://github.com/OpenFunction/OpenFunction/pull/344)

### builder

#### Features
- Update go builder to support declarative function api. [builder#56](https://github.com/OpenFunction/builder/pull/56)
- Bump node functions framework to v0.5.0. [builder#57](https://github.com/OpenFunction/builder/pull/57)
- Add java builder. [builder#58](https://github.com/OpenFunction/builder/pull/58)
- Add go117 builder & bump function-framework-go to v0.4.0. [builder#60](https://github.com/OpenFunction/builder/pull/60)

### functions-framework-go

#### Features
- Support declarative multiple functions. [functions-framework-go#48](https://github.com/OpenFunction/functions-framework-go/pull/48)
- Support defining path-parameters and HTTP method. [functions-framework-go#52](https://github.com/OpenFunction/functions-framework-go/pull/52)
- Add GetEventInputName func for context interface. [functions-framework-go#55](https://github.com/OpenFunction/functions-framework-go/pull/55)

#### Enhancement
- Set the exit span before sending the payload to the target. [functions-framework-go#45](https://github.com/OpenFunction/functions-framework-go/pull/45)
- [Plugin-SkyWalking] Set instance layer to FAAS. [functions-framework-go#46](https://github.com/OpenFunction/functions-framework-go/pull/46)
- Use innerEvent to encapsulate user data only when the tracing is enabled. [functions-framework-go#49](https://github.com/OpenFunction/functions-framework-go/pull/49)
- [Plugin-SkyWalking] Report pod name and namespace. [functions-framework-go#50](https://github.com/OpenFunction/functions-framework-go/pull/50)
- Update cloud event input data to json format. [functions-framework-go#53](https://github.com/OpenFunction/functions-framework-go/pull/53)
- Upgrade dapr to v1.8.3 & dapr-go-sdk to v1.5.0. [functions-framework-go#56](https://github.com/OpenFunction/functions-framework-go/pull/56) [functions-framework-go#59](https://github.com/OpenFunction/functions-framework-go/pull/59)
- Combine declarative test cases into one test case. [functions-framework-go#60](https://github.com/OpenFunction/functions-framework-go/pull/60)

### functions-framework-nodejs
#### Features
- Initialize openfunction knative and async runtime. [functions-framework-nodejs#4](https://github.com/OpenFunction/functions-framework-nodejs/pull/4)
- Enable HTTP function trigger async functions. [functions-framework-nodejs#10](https://github.com/OpenFunction/functions-framework-nodejs/pull/10)

### functions-framework-java

OpenFunction now supports java!

#### Features
- Support OpenFunction framework. [functions-framework-java#1](https://github.com/OpenFunction/functions-framework-java/pull/1)

### openfunction.dev
Renaming OpenFunction' website repository to openfunction.dev.

Docs have been refactored and updated with all the new features and changes of this release, see [OpenFunction docs](https://openfunction.dev/docs/).

### charts
Now you can install OpenFunction and all its dependencies with helm charts.
#### **TL;DR**
```shell
helm repo add openfunction https://openfunction.github.io/charts/
helm repo update
helm install openfunction openfunction/openfunction -n openfunction --create-namespace
```

#### Component Upgrade
- Upgrade knative-serving from v1.0.1 to v1.3.2
- Upgrade shipwright-build from v0.6.1 to v0.10.0
- Upgrade tekton-pipelines from v0.30.0 to v0.37.2
- Upgrade keda from v2.4.0 to v2.7.1
- Upgrade dapr from v1.5.1 to v1.8.3

#### Features
- Add helm chart for openfunction and its dependencies. [charts#1](https://github.com/OpenFunction/charts/pull/1)

#### Enhancement
- Update helm chart for openfunction v0.7.0. [charts#14](https://github.com/OpenFunction/charts/pull/14)
- Adjust helm chart for release v0.7.0-rc.0. [charts#22](https://github.com/OpenFunction/charts/pull/22)

## 0.6.0 / 2022-03-21

The core v1alpha1 API was deprecated and removed.

### Features

- [Refactor OpenFuncAsync runtime definition](https://github.com/OpenFunction/OpenFunction/issues/184) and upgrade the core api to v1beta1. #222
- [Add HTTP trigger to async function](https://github.com/OpenFunction/OpenFunction/issues/185) by enabling Knative runtime to use Dapr. #222
- [Add an unified scaleOptions to control the scaling of both the Knative and Async runtime](https://github.com/OpenFunction/OpenFunction/issues/173). #222
- Add function plugin support as described in the [proposal](https://github.com/OpenFunction/OpenFunction/blob/main/docs/proposals/202112_functions_framework_refactoring.md). #222
- [Add Skywalking tracing support for both Sync and Async function](https://github.com/OpenFunction/OpenFunction/blob/main/docs/proposals/202112_support_function_tracing.md) as discussed in issue [#146](https://github.com/OpenFunction/OpenFunction/issues/146) and [#9](https://github.com/OpenFunction/functions-framework/issues/9). #222 [#36](https://github.com/OpenFunction/functions-framework-go/pull/36) [#30](https://github.com/OpenFunction/functions-framework-go/pull/30)

### Enhancement

- Move Keda's trigger to the top level of function serving as discussed in [proposal](https://hackmd.io/FW4WAM6CSmuZt6zrpsqWXA?view). #232
- Add URI to EventSource sink. #207 #216
- Add more e2e tests. #226
- Regenerate OpenFunction client. #243
- Update OpenFunction events docs. #244

### Fixes

- Fix [controller failed because dependent CRD was not found](https://github.com/OpenFunction/OpenFunction/issues/199). #222
- Fix [function build failure issue after renaming the secret used](https://github.com/OpenFunction/OpenFunction/issues/219). #220
- Change tracing plugin switch from `Enable` to `Enabled`. #246
- Fix [Updates to the event api do not trigger updates to the function](https://github.com/OpenFunction/OpenFunction/issues/164). #247

## 0.6.0-rc.0 / 2022-03-08

The core v1alpha1 API was deprecated and removed.

### Features

- [Refactor OpenFuncAsync runtime definition](https://github.com/OpenFunction/OpenFunction/issues/184) and upgrade the core api to v1beta1. [#222](https://github.com/OpenFunction/OpenFunction/pull/222)
- [Add HTTP trigger to async function](https://github.com/OpenFunction/OpenFunction/issues/185) by enabling Knative runtime to use Dapr. [#222](https://github.com/OpenFunction/OpenFunction/pull/222)
- [Add an unified scaleOptions to control the scaling of both the Knative and Async runtime](https://github.com/OpenFunction/OpenFunction/issues/173). [#222](https://github.com/OpenFunction/OpenFunction/pull/222)
- Add function plugin support as described in the [proposal](https://github.com/OpenFunction/OpenFunction/blob/main/docs/proposals/202112_functions_framework_refactoring.md). [#222](https://github.com/OpenFunction/OpenFunction/pull/222)
- [Add Skywalking tracing support for both Sync and Async function](https://github.com/OpenFunction/OpenFunction/blob/main/docs/proposals/202112_support_function_tracing.md) as discussed in issue [#146](https://github.com/OpenFunction/OpenFunction/issues/146) and [#9](https://github.com/OpenFunction/functions-framework/issues/9). [#222](https://github.com/OpenFunction/OpenFunction/pull/222) [#36](https://github.com/OpenFunction/functions-framework-go/pull/36) [#30](https://github.com/OpenFunction/functions-framework-go/pull/30)

### Enhancement

- Move Keda's trigger to the top level of function serving as discussed in [proposal](https://hackmd.io/FW4WAM6CSmuZt6zrpsqWXA?view). [#232](https://github.com/OpenFunction/OpenFunction/pull/232)
- Add URI to EventSource sink. [#207](https://github.com/OpenFunction/OpenFunction/pull/207) [#216](https://github.com/OpenFunction/OpenFunction/pull/216)
- Add license header check with skywalking-eyes. [#212](https://github.com/OpenFunction/OpenFunction/pull/212)
- Add more e2e tests. [#226](https://github.com/OpenFunction/OpenFunction/pull/226)

### Fixes

- Fix [controller failed because dependent CRD was not found](https://github.com/OpenFunction/OpenFunction/issues/199). [#222](https://github.com/OpenFunction/OpenFunction/pull/222)
- Fix [function build failure issue after renaming the secret used](https://github.com/OpenFunction/OpenFunction/issues/219). [#220](https://github.com/OpenFunction/OpenFunction/pull/220)

## 0.5.0 / 2021-12-31

The core v1alpha1 API is deprecated and will be removed in the next release.
### Features

- Add github action CI workflow [#140](https://github.com/OpenFunction/OpenFunction/pull/140) [179](https://github.com/OpenFunction/OpenFunction/pull/179)
- Add build and serving timeout [#147](https://github.com/OpenFunction/OpenFunction/pull/147)
- Add MQTT EventSource to OpenFunction Events [#149](https://github.com/OpenFunction/OpenFunction/pull/149)
- Add Domain CRD to define the entry point of a sync function [#158](https://github.com/OpenFunction/OpenFunction/pull/158)
- Add buildah, kaniko, ko support, user can build apps with Dockerfile using buildah or kaniko, and build go application with ko now [#170](https://github.com/OpenFunction/OpenFunction/pull/170) [#171](https://github.com/OpenFunction/OpenFunction/pull/171)
- Add OpenFunction ClientSet [#176](https://github.com/OpenFunction/OpenFunction/pull/176)
- Support to keep build history [#179](https://github.com/OpenFunction/OpenFunction/pull/179)
- Support to add labels and annotations to function workloads [#181](https://github.com/OpenFunction/OpenFunction/pull/181)

### Enhancement

- Deprecate the old install/uninstall scripts in favor of the cli tool [ofn](https://github.com/OpenFunction/cli/releases), now users can use [ofn](https://github.com/OpenFunction/cli/releases) to install/uninstall/demo OpenFunction
- Now OpenFunction is compatible with K8s 1.17 ~ 1.20+, thanks to [ofn](https://github.com/OpenFunction/cli/releases)
- Optimize Function status to reflect serving workload status more accurately [151](https://github.com/OpenFunction/OpenFunction/pull/151)
- Adjust samples repo to move samples to each version's release branch [33](https://github.com/OpenFunction/samples/pull/33)

## 0.4.0 / 2021-10-19

### Features

- Upgrade core.openfunction.io from **v1alpha1** to **v1alpha2**. [#115](https://github.com/OpenFunction/OpenFunction/pull/115)
- Make event handlers self driven. [#115](https://github.com/OpenFunction/OpenFunction/pull/115)

### Enhancement

- Update dependent Dapr version to v1.3.1. [#123](https://github.com/OpenFunction/OpenFunction/pull/123)
- Update dependent Tekton pipeline version to 0.28.1. [#131](https://github.com/OpenFunction/OpenFunction/pull/131)
- Update dependent Knative serving version to 0.26.0. [#131](https://github.com/OpenFunction/OpenFunction/pull/131)
- Update dependent Shipwright build version to 0.6.0. [#131](https://github.com/OpenFunction/OpenFunction/pull/131)
- Update go version to 1.16. [#131](https://github.com/OpenFunction/OpenFunction/pull/131)
- Now Function supports environment variables with commas. [#131](https://github.com/OpenFunction/OpenFunction/pull/131)

### Fixes

- Fix bug rerun serving failed. [#132](https://github.com/OpenFunction/OpenFunction/pull/132)

### Docs

- Use installation script to deploy OpenFunction and deprecate the installation guide. [#122](https://github.com/OpenFunction/OpenFunction/pull/122)

## 0.3.1 / 2021-08-27

### Enhancement

- Delete old serving after new serving running. #107

## 0.3.0 / 2021-08-19

### Features

-  Add OpenFunction Events: OpenFunction's own event management framework. [#78](https://github.com/OpenFunction/OpenFunction/pull/78) [#83](https://github.com/OpenFunction/OpenFunction/pull/83) [#89](https://github.com/OpenFunction/OpenFunction/pull/89) [#90](https://github.com/OpenFunction/OpenFunction/pull/90) [#99](https://github.com/OpenFunction/OpenFunction/pull/99) [#100](https://github.com/OpenFunction/OpenFunction/pull/100)
-  Support using ShipWright as Builder backend to build functions or apps. [#82](https://github.com/OpenFunction/OpenFunction/pull/82) [#95](https://github.com/OpenFunction/OpenFunction/pull/95)
-  Build and serving can be launched separately. [#82](https://github.com/OpenFunction/OpenFunction/pull/82) [#95](https://github.com/OpenFunction/OpenFunction/pull/95)
-  Support running an application (container image) as a serverless workload directly. [#82](https://github.com/OpenFunction/OpenFunction/pull/82) [#95](https://github.com/OpenFunction/OpenFunction/pull/95)

## 0.2.0 / 2021-06-30

### Features

- Support OpenFunctionAsync serving runtime（backed by Dapr + KEDA + Deployment/Job)
- Functions frameworks async function support
- Customized go function framework & builders for both Knative and OpenFunctionAsync serving runtime

## 0.1.0 / 2021-05-17

### Features

- Add Function, Builder, and Serving CRDs and corresponding controllers
- Support using existing function framework & buildpacks such as Google Cloud Functions to build functions
- Support Tekton and Cloud Native Buildpacks as Builder backend 
- Support Knative as Serving backend
- Optimize and localize existing function framework & buildpacks.


The following repos' release schedule is the same as OpenFunction/OpenFunction

# OpenFunction/samples

## 0.4.0 / 2021-10-19

### Docs

- Archive by version. [#20](https://github.com/OpenFunction/samples/pull/20)

# OpenFunction/website

## 0.4.0 / 2021-10-19

### Features

- Add OpenFunction Website. [#1](https://github.com/OpenFunction/website/pull/1)
- Support Algolia search. [#14](https://github.com/OpenFunction/website/pull/14)

# OpenFunction/builder

## 0.6.0 / 2022-03-08

### Enhancement

- Change run base image to busybox for go to reduce function image size [28](https://github.com/OpenFunction/builder/pull/28)
- Upgrade lifecycle, buildpack api and libcnb to latest version [32](https://github.com/OpenFunction/builder/pull/32)
- Use alpine as run image for nodejs [38](https://github.com/OpenFunction/builder/pull/38)
- Add functions framework version environment variable. [45](https://github.com/OpenFunction/builder/pull/45)
- Update functions-framework-go to v0.2.0 [46](https://github.com/OpenFunction/builder/pull/46)

## 0.4.0 / 2021-10-19

### Features

- Upgrade the functions-framework-go from **v0.0.0-20210628081257-4137e46a99a6** to **v0.0.0-20210922063920-81a7b2951b8a**. [#17](https://github.com/OpenFunction/builder/pull/17)
- Add Ruby builder. [#11](https://github.com/OpenFunction/builder/pull/11)

#### Fixes

- Enables the OpenFunction functions framework to use the **FUNC_GOPROXY** environment variable. [#16](https://github.com/OpenFunction/builder/pull/16)

# OpenFunction/cli

## 0.4.0 / 2021-10-19

### Features

- Add OpenFunction CLI. [#11](https://github.com/OpenFunction/cli/pull/1)

# OpenFunction/functions-framework

> Timeline based on primary repository releases

## 0.4.0 / 2021-10-19

### Docs

- Add **functions-framework-nodejs** case. [#6](https://github.com/OpenFunction/functions-framework/pull/6)
- Archive by version. [#7](https://github.com/OpenFunction/functions-framework/pull/7)

# OpenFunction/functions-framework-go

## 0.4.0 / 2021-10-19

### Features

- Supports multiple input sources && Replace int return with ctx.Return. [#13](https://github.com/OpenFunction/functions-framework-go/pull/13)

# OpenFunction/functions-framework-nodejs

## 0.3.6 / 2022-03-08

### Features

- Change to OpenFunction. [#1](https://github.com/OpenFunction/functions-framework-nodejs/pull/1)

# OpenFunction/events-handlers

## 0.4.0 / 2021-10-19

### Features

- Add handler functions. [#7](https://github.com/OpenFunction/events-handlers/pull/7)