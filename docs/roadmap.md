# Roadmap

## v0.1.0, released on May 17th, 2021

- [x] Create Function, Builder and Serving CRDs and corresponding controllers
- [x] Support using existing function framework & buildpacks such as Google Cloud Function to build functions
- [x] Support using Tekton and Cloud Native Buildpacks as Builder backend to build functions
- [x] Support Knative as Serving backend
- [x] Optimize and localize existing function framework & buildpacks

## v0.2.0, plan to be released in June, 2021

- [ ] Support Dapr + KEDA + Deployment/Job as Serving backend
- [ ] Add FUNC_CONTEXT for Input/Output data bindings
- [ ] Functions Frameworks Spec
- [ ] Develop customized go function framework & buildpacks for both Knative and Dapr + KEDA backend
- [ ] Support asynchronous Non-HTTP function types
- [ ] Support batch jobs function type

## v0.3.0, 2021 H2

- [ ] Support python functions frameworks
- [ ] Support nodejs functions frameworks
- [ ] Support scheduling functions to Edge nodes (KubeEdge)
- [ ] Test and support existing application buildpacks to run application as Serverless workload directly
- [ ] Support [ShipWright](https://github.com/shipwright-io/build) as Builder backend to build functions or apps with Docker file (without buildpacks)
- [ ] Support AI Inference functions, for example, AI models loaded by Tensorflow Serving or KFServing