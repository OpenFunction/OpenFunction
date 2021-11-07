# Roadmap

## v0.1.0, May 2021

- [x] Create Function, Builder and Serving CRDs and corresponding controllers
- [x] Support using existing function framework & buildpacks such as Google Cloud Function to build functions
- [x] Support using Tekton and Cloud Native Buildpacks as Builder backend to build functions
- [x] Support Knative as Serving backend
- [x] Optimize and localize existing function framework & buildpacks

## v0.2.0, June 2021

- [x] Support OpenFunctionAsync serving runtime（backed by Dapr + KEDA + Deployment/Job)
- [x] Functions Frameworks async function support
- [x] Customized go function framework & builders for both Knative and OpenFunctionAsync serving runtime

## v0.3.0, July ~ August 2021

- [x] Add OpenFunction Events: OpenFunction's own event management framework
- [x] Support using [ShipWright](https://github.com/shipwright-io/build) as Builder backend to build functions or apps
- [x] Build and serving can be launched separately
- [x] Support running an application (container image) as a serverless workload directly

## v0.4.0, Oct 2021

- [x] Upgrade core.openfunction.io from **v1alpha1** to **v1alpha2**.
- [x] Make event handlers self-driven.
- [x] Update dependent components(Tekton, Knative, Shipwright, and Dapr) and go version.
- [x] Add [OpenFunction Website](https://openfunction.dev/).
- [x] Add [OpenFunction CLI](https://github.com/OpenFunction/cli).
- [x] Add Ruby builder. 
- [x] Supports multiple input sources
- [x] OpenFunction/functions-framework-nodejs now support OpenFunctionAsync function.
- [x] Add [event source & trigger functions](https://github.com/OpenFunction/events-handlers).

## v0.5.0+, 2021 Q4 ~ 2022 Q1

- [ ] Function ingress support
- [ ] Add ScaleOptionTemplate and ComponentTemplate
- [ ] Support more EventSources
- [ ] Add OpenFunction sync function
- [ ] Python functions frameworks & builder
- [ ] OpenFunction API & Console
- [ ] [Serverless workflow](https://serverlessworkflow.io/) support, refer to [Serverless Workflow Project Deep Dive](https://www.youtube.com/watch?v=dsuo1VQQZ2E&list=PLj6h78yzYM2MqBm19mRz9SYLsw4kfQBrC&index=166) for more info. Reference implementations include [
SYNAPSE](https://github.com/serverlessworkflow/synapse), [FunctionGraph](https://www.huaweicloud.com/en-us/product/functiongraph.html), [Kogito](https://kogito.kie.org/), [Automatiko](https://automatiko.io/).
- [ ] Use [ShipWright](https://github.com/shipwright-io/build) to build functions or apps with Dockerfile.
- [ ] Support Rust functions & WebAssembly runtime.
- [ ] Function Tracing: Integrate [go2sky](https://github.com/SkyAPM/go2sky) with go function framework and send tracing data to Skywalking. (Needs to use other [instrument SDKs](https://github.com/SkyAPM) to integrate with more function frameworks)
- [ ] Integration with KubeSphere
- [ ] Integration with Notification Manager to provide Serverless notification, maybe by a new project called `Elastic Notification`.
