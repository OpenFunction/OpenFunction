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

## v0.4.0+, 2021 Q4

- [ ] OpenFunction CLI
- [ ] Support more EventSources
- [ ] Use OpenFunction async functions to drive EventSource & EventTrigger workloads
- [ ] OpenFunction sync function
- [ ] Python functions frameworks & builder
- [ ] Nodejs functions frameworks & builder
- [ ] OpenFunction Console (WebUI)
- [ ] Support scheduling functions to Edge
- [ ] Use [ShipWright](https://github.com/shipwright-io/build) to build functions or apps with Dockerfile.
- [ ] Support Rust functions & WebAssembly runtime.
