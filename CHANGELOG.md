- [OpenFunction/OpenFunction](#openfunction)
  * [0.1.0 / 2021-05-17](#010--2021-05-17)
  * [0.2.0 / 2021-06-30](#020--2021-06-30)
  * [0.3.0 / 2021-08-19](#030--2021-08-19)
  * [0.3.1 / 2021-08-27](#031--2021-08-27)
  * [0.4.0 / 2021-10-19](#040--2021-10-19)
  * [0.5.0 / 2021-12-31](#050--2021-12-31)
- [OpenFunction/samples](#openfunctionsamples)
- [OpenFunction/website](#openfunctionwebsite)
- [OpenFunction/builder](#openfunctionbuilder)
- [OpenFunction/cli](#openfunctioncli)
- [OpenFunction/functions-framework](#openfunctionfunctions-framework)
- [OpenFunction/functions-framework-go](#openfunctionfunctions-framework-go)
- [OpenFunction/functions-framework-nodejs](#openfunctionfunctions-framework-nodejs)
- [OpenFunction/events-handlers](#openfunctionevents-handlers)

# OpenFunction

## 0.1.0 / 2021-05-17

### Features

- Add Function, Builder, and Serving CRDs and corresponding controllers
- Support using existing function framework & buildpacks such as Google Cloud Functions to build functions
- Support Tekton and Cloud Native Buildpacks as Builder backend 
- Support Knative as Serving backend
- Optimize and localize existing function framework & buildpacks.

## 0.2.0 / 2021-06-30

### Features

- Support OpenFunctionAsync serving runtime（backed by Dapr + KEDA + Deployment/Job)
- Functions frameworks async function support
- Customized go function framework & builders for both Knative and OpenFunctionAsync serving runtime

## 0.3.0 / 2021-08-19

### Features

-  Add OpenFunction Events: OpenFunction's own event management framework. [#78](https://github.com/OpenFunction/OpenFunction/pull/78) [#83](https://github.com/OpenFunction/OpenFunction/pull/83) [#89](https://github.com/OpenFunction/OpenFunction/pull/89) [#90](https://github.com/OpenFunction/OpenFunction/pull/90) [#99](https://github.com/OpenFunction/OpenFunction/pull/99) [#100](https://github.com/OpenFunction/OpenFunction/pull/100)
-  Support using ShipWright as Builder backend to build functions or apps. [#82](https://github.com/OpenFunction/OpenFunction/pull/82) [#95](https://github.com/OpenFunction/OpenFunction/pull/95)
-  Build and serving can be launched separately. [#82](https://github.com/OpenFunction/OpenFunction/pull/82) [#95](https://github.com/OpenFunction/OpenFunction/pull/95)
-  Support running an application (container image) as a serverless workload directly. [#82](https://github.com/OpenFunction/OpenFunction/pull/82) [#95](https://github.com/OpenFunction/OpenFunction/pull/95)


## 0.3.1 / 2021-08-27

### Enhancement

- Delete old serving after new serving running. #107


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

- Depreciate the old install/uninstall scripts in favor of the cli tool [ofn](https://github.com/OpenFunction/cli/releases), now users can use [ofn](https://github.com/OpenFunction/cli/releases) to install/uninstall/demo OpenFunction
- Now OpenFunction is compatible with K8s 1.17 ~ 1.20+, thanks to [ofn](https://github.com/OpenFunction/cli/releases)
- Optimized Function status to reflect serving workload status more accurately [151](https://github.com/OpenFunction/OpenFunction/pull/151)
- Adjust samples repo to move samples to each version's release branch

### Docs


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

## 0.4.0 / 2021-10-19

### Features

- Support OpenFunction type function. [#7](https://github.com/OpenFunction/functions-framework-nodejs/pull/7)

# OpenFunction/events-handlers

## 0.4.0 / 2021-10-19

### Features

- Add handler functions. [#7](https://github.com/OpenFunction/events-handlers/pull/7)
