# Roadmap

## v0.1.0, released on May 17th, 2021

- [x] Create Function, Builder and Serving CRDs and corresponding controllers
- [x] Support using existing function framework & buildpacks such as Google Cloud Function to build functions
- [x] Support using Tekton and Cloud Native Buildpacks as Builder backend to build functions
- [x] Support Knative as Serving backend
- [x] Optimize and localize existing function framework & buildpacks

## v0.2.0, plan to be released in June, 2021

- [ ] Support OpenFunctionAsync serving runtime（backed by Dapr + KEDA + Deployment/Job)
- [ ] Functions Frameworks OpenFunction Async function support
- [ ] Customized go function framework & builders for both Knative and OpenFunctionAsync serving runtime

## v0.3.0, 2021 H2

- [ ] Async function dead letter sink support
- [ ] OpenFunction Events: OpenFunction's own event management including EventSource and Trigger supports
- [ ] Support python functions frameworks
- [ ] Support nodejs functions frameworks
- [ ] Support scheduling functions to Edge nodes (KubeEdge)
- [ ] Test and support existing application builders to run an application as Serverless workload directly
- [ ] Support [ShipWright](https://github.com/shipwright-io/build) as Builder backend to build functions or apps with Docker file
- [ ] Support AI Inference functions, for example, AI models loaded by Tensorflow Serving or KFServing