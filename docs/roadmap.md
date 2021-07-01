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

- [ ] OpenFunction Events: OpenFunction's own event management
- [ ] Async function dead letter sink
- [ ] Run an existing container image as serverless workload without the build process
- [ ] Support existing application builders or customized builders to run an application as Serverless workload directly
- [ ] Nodejs functions frameworks
- [ ] Python functions frameworks

## v0.4.0+, 2021 Q4

- [ ] OpenFunction sync function
- [ ] OpenFunction CLI
- [ ] OpenFunction Console (WebUI)
- [ ] Support scheduling functions to Edge nodes (KubeEdge)
- [ ] Test and support existing application builders to run an application as Serverless workload directly
- [ ] Support [ShipWright](https://github.com/shipwright-io/build) as Builder backend to build functions or apps with Docker file
- [ ] Support AI Inference functions, for example, AI models loaded by Tensorflow Serving or KFServing