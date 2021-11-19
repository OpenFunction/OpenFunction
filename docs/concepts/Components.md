[Events CRDs](#events-crds)

[Core CRDs](#core-crds)

* [Function](#function)
* [Builder](#builder)
  + [Shipwright and Cloud Native Buildpacks](#shipwright-and-cloud-native-buildpacks)
* [Serving](#serving)
  + [Knative](#knative)
  + [OpenFuncAsync](#openfuncasync)
* [Domain](#domain)

### Events CRDs

OpenFunction provides an event handling framework called OpenFunction events that complements the event-driven capabilities of OpenFunction as a FaaS framework.

You can refer to [OpenFunction Events Framework Concepts](https://github.com/OpenFunction/OpenFunction/blob/main/docs/concepts/OpenFunction-events-framework.md) for more information.

### Core CRDs

The core capability of OpenFunction is to enable users to develop, run and manage applications as executable function code. OpenFunction implements the following [custom resource definitions (CRDs)](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/):

- **Function**, defines a function.
- **Builder**, defines a function builder.
- **Serving**, defines a function workload.
- **Domain**, defines a unified entry point for sync functions.

#### Function

The goal of `Function` is to control the lifecycle management from user code to the final application that can respond to events through a single profile.

`Function` manages and coordinates `Builder` and `Serving` resources to handle the details of the process.

#### Builder

The goal of `Builder` is to compile the user's function source code into an application image.

It will fetch the code from the code repository, build the application image locally and publish it to the container image repository.

Currently, OpenFunction Builder uses [Shipwright and Cloud Native Buildpacks](https://github.com/OpenFunction/OpenFunction#shipwright-and-cloud-native-buildpacks) to build container images.

##### Shipwright and Cloud Native Buildpacks

[Shipwright](https://github.com/shipwright-io/build) is an extensible framework for building container images on Kubernetes.

Cloud Native Buildpacks is an OCI standard image building framework that transforms your application source code into container images without a Dockerfile.

OpenFunction Builder controls the build process of application images by [Shipwright](https://github.com/shipwright-io/build), including fetching code, building and publishing images via Cloud Native Buildpacks.

#### Serving

The goal of `Serving` is to run functions in a highly elastic manner (dynamically scale 0 <-> N).

Currently, OpenFunction supports two serving runtimes, [Knative](https://github.com/OpenFunction/OpenFunction#knative) and [OpenFuncAsync](https://github.com/OpenFunction/OpenFunction#openfuncasync). At least one of these runtimes is required.

##### Knative

Knative Serving builds on Kubernetes to support deploying and serving serverless applications and functions. Knative Serving is easy to get started with and scales to support advanced scenarios.

##### OpenFuncAsync

OpenFuncAsync is an event-driven Serving runtime. It is implemented based on KEDA + Dapr.

You can refer to [Prerequisites](https://github.com/OpenFunction/OpenFunction#prerequisites) and use `--with-openFuncAsync` to install OpenFuncAsync runtime.

The OpenFuncAsync functions can be triggered by various event types, such as MQ, cronjob, DB events, etc. In a Kubernetes cluster, OpenFuncAsync functions run in the form of deployments or jobs.

#### Domain

`Domain` defines a unified entry point for sync functions using ingress, user can use

```
http://<domain-name>.<domain-namespace>/<function-namespace>/<function-name>
```

to access a function.

Only one `Domain` can be defined in a cluster. A `Domain` requires a `Ingress Controller`. By default, we use `nginx-ingress`. You can refer to [Prerequisites](https://github.com/OpenFunction/OpenFunction#prerequisites) and use `--with-ingress` to install it, or install it manually. If the `nginx-ingress` does not use the default namespace and name, please modify the `config/domain/default-domain.yaml`, and run

```
make manifests
```

to update the `config/bundle.yaml`, and use this file to deploy `openFunction`.