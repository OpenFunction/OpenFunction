- [OpenFunction/OpenFunction](#openfunction)
  * [0.1.0 / 2021-05-17](#010--2021-05-17)
  * [0.2.0 / 2021-06-30](#020--2021-06-30)
  * [0.3.0 / 2021-08-19](#030--2021-08-19)
  * [0.3.1 / 2021-08-27](#031--2021-08-27)
  * [0.4.0 / 2021-10-19](#040--2021-10-19)
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

# OpenFunction/samples

> Timeline based on primary repository releases

## 0.4.0 / 2021-10-19

### Docs

- Archive by version. [#20](https://github.com/OpenFunction/samples/pull/20)

# OpenFunction/website

> Timeline based on primary repository releases

## 0.4.0 / 2021-10-19

### Features

- Add OpenFunction Website. [#1](https://github.com/OpenFunction/website/pull/1)
- Support Algolia search. [#14](https://github.com/OpenFunction/website/pull/14)

# OpenFunction/builder

> Timeline based on primary repository releases

## 0.4.0 / 2021-10-19

### Features

- Upgrade the functions-framework-go from **v0.0.0-20210628081257-4137e46a99a6** to **v0.0.0-20210922063920-81a7b2951b8a**. [#17](https://github.com/OpenFunction/builder/pull/17)
- Add Ruby builder. [#11](https://github.com/OpenFunction/builder/pull/11)

#### Fixes

- Enables the OpenFunction functions framework to use the **FUNC_GOPROXY** environment variable. [#16](https://github.com/OpenFunction/builder/pull/16)

# OpenFunction/cli

> Timeline based on primary repository releases

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

> Timeline based on primary repository releases

## 0.4.0 / 2021-10-19

### Features

- Supports multiple input sources && Replace int return with ctx.Return. [#13](https://github.com/OpenFunction/functions-framework-go/pull/13)

# OpenFunction/functions-framework-nodejs

> Timeline based on primary repository releases

## 0.4.0 / 2021-10-19

### Features

- Support OpenFunction type function. [#7](https://github.com/OpenFunction/functions-framework-nodejs/pull/7)

# OpenFunction/events-handlers

> Timeline based on primary repository releases

## 0.4.0 / 2021-10-19

### Features

- Add handler functions. [#7](https://github.com/OpenFunction/events-handlers/pull/7)
