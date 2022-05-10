# Roadmap

## v0.7.0+, 2022 Q2 ~ Q4

- [x] Nodejs async functions frameworks.
- [ ] Python functions frameworks & builder.
- [ ] Java functions frameworks & builder.
- [ ] .NET frameworks & builder.
- [ ] Compile Java code to native binary and run Java functions in Quarkus
- [ ] Add KEDA http-add-on as another sync function runtime.
- [ ] Add helm chart for OpenFunction and its dependencies.
- [ ] Support more EventSources like Github.
- [ ] OpenFunction API & Console.
- [ ] Add the ability to control concurrent access to functions.
- [ ] Use helm in ofn cli to replace the original yaml installation method.
- [ ] Add function invoking ability to ofn cli.
- [ ] Use Contour as Knative network layer and K8s Ingress & Gateway.
- [ ] [Serverless workflow](https://serverlessworkflow.io/) support, refer to [Serverless Workflow Project Deep Dive](https://www.youtube.com/watch?v=dsuo1VQQZ2E&list=PLj6h78yzYM2MqBm19mRz9SYLsw4kfQBrC&index=166) for more info. Reference implementations include [
SYNAPSE](https://github.com/serverlessworkflow/synapse), [FunctionGraph](https://www.huaweicloud.com/en-us/product/functiongraph.html), [Kogito](https://kogito.kie.org/), [Automatiko](https://automatiko.io/).
- [ ] Use [ShipWright](https://github.com/shipwright-io/build) to build functions with Dockerfile.
- [ ] Function tracing: support using OpenTelemetry for tracing.
- [ ] Better support for Functions running in Edge
- [ ] Support Rust functions & WebAssembly runtime.

## v0.6.0, Mar 2022

- [x] Upgrade core API from v1alpha2 to v1beta1
- [x] Functions framework refactoring.
- [x] Add plug-in mechanism to functions framework.
- [x] Refactoring OpenFunctionAsync runtime definition.
- [x] Add binding to knative sync functions (Integrate Dapr with Knative).
- [x] Add the ability to control min/max replicas.
- [x] Nodejs sync functions frameworks & builder.
- [x] Function tracing: support using Skywalking for function tracing.
- [x] Add more e2e tests

## v0.5.0, Dec 2021

- [x] Deprecate the old install/uninstall scripts in favor of the cli tool [ofn](https://github.com/OpenFunction/cli/releases)
- [x] Now OpenFunction is compatible with K8s 1.17 ~ 1.20+.
- [x] Deprecate the core v1alpha1 API which will be removed in the next release.
- [x] Add build and serving timeout.
- [x] Add MQTT EventSource to OpenFunction Events.
- [x] Add the unified function entry point(ingress) support for a sync function.
- [x] Add buildah, kaniko, ko support, user can build apps with Dockerfile now.
- [x] Add OpenFunction ClientSet.
- [x] Support to keep build history.

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

## v0.3.0, July ~ August 2021

- [x] Add OpenFunction Events: OpenFunction's own event management framework
- [x] Support using [ShipWright](https://github.com/shipwright-io/build) as Builder backend to build functions or apps
- [x] Build and serving can be launched separately
- [x] Support running an application (container image) as a serverless workload directly

## v0.2.0, June 2021

- [x] Support OpenFunctionAsync serving runtime（backed by Dapr + KEDA + Deployment/Job)
- [x] Functions Frameworks async function support
- [x] Customized go function framework & builders for both Knative and OpenFunctionAsync serving runtime

## v0.1.0, May 2021

- [x] Create Function, Builder and Serving CRDs and corresponding controllers
- [x] Support using existing function framework & buildpacks such as Google Cloud Function to build functions
- [x] Support using Tekton and Cloud Native Buildpacks as Builder backend to build functions
- [x] Support Knative as Serving backend
- [x] Optimize and localize existing function framework & buildpacks


For more information, please refer to [OpenFunction Roadmap](https://github.com/orgs/OpenFunction/projects/3)