# Roadmap

- [x] Create Function, Builder and Serving CRDs and corresponding controllers
- [x] Support using existing function framework & buildpacks such as Google Cloud Function to build functions
- [x] Support using Tekton and Cloud Native Buildpacks as Builder backend to build functions
- [x] Support Knative as Serving backend
- [ ] Optimize and localize existing function framework & buildpacks
- [ ] Support Dapr + KEDA + Deployment/Job as Serving backend
- [ ] Add Input CRD field to route events to a specific function service
- [ ] Add Output CRD field to send function result to a specific sink
- [ ] Develop customized function framework & buildpacks for both Knative and Dapr + KEDA backend
- [ ] Support scheduling functions to Edge nodes (KubeEdge)
- [ ] Test and support existing application buildpacks to run application as Serverless workload directly
- [ ] Support [ShipWright](https://github.com/shipwright-io/build) as Builder backend to build functions or apps with Docker file (without buildpacks)
- [ ] Support AI Inference functions, for example, AI models loaded by Tensorflow Serving or KFServing