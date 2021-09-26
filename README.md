<div align=center><img  width="334" height="117" src=docs/images/logo.png></div>

## Overview

**OpenFunction** is a cloud-native open source FaaS (Function as a Service) platform aiming to enable users to focus on their business logic without worrying about the underlying runtime environment and infrastructure. Users only need to submit business-related source code in the form of functions.

<div align=center><img src=docs/images/function-lifecycle.svg></div>

OpenFunction features but not limited to the following:

- Convert business-related function source code to runnable application source code.
- Generate a deployable container image from the converted application source code.
- Deploy the generated container image to the underlying runtime environment such as K8s, and automatically scale up and down according to business traffic, and scale to 0 when there is no traffic.
- Provide event management functions for trigger functions.
- Provide additional functions to manage function versions, ingress management etc.

## CustomResourceDefinitions

<div align=center><img src=docs/images/openfunction-overview.svg></div>

### Events CRDs

OpenFunction provides an event handling framework called OpenFunction events that complements the event-driven capabilities of OpenFunction as a FaaS framework.

You can refer to [OpenFunction Events Framework Concepts](docs/concepts/OpenFunction-events-framework.md) for more information.

### Core CRDs

The core capability of OpenFunction is to enable users to develop, run and manage applications as executable function code. OpenFunction implements the following [custom resource definitions (CRDs)](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/):

- **Function**, defines a function.
- **Builder**, defines a function builder.
- **Serving**, defines a function workload.

#### Function

The goal of `Function` is to control the lifecycle management from user code to the final application that can respond to events through a single profile.

`Function` manages and coordinates `Builder` and `Serving` resources to handle the details of the process.

#### Builder

The goal of `Builder` is to compile the user's function source code into an application image.

It will fetch the code from the code repository, build the application image locally and publish it to the container image repository.

Currently, OpenFunction Builder uses [Shipwright and Cloud Native Buildpacks](#shipwright-and-cloud-native-buildpacks) to build container images.

##### Shipwright and Cloud Native Buildpacks

[Shipwright](https://github.com/shipwright-io/build) is an extensible framework for building container images on Kubernetes.

Cloud Native Buildpacks is an OCI standard image building framework that transforms your application source code into container images without a Dockerfile.

OpenFunction Builder controls the build process of application images by [Shipwright](https://github.com/shipwright-io/build), including fetching code, building and publishing images via Cloud Native Buildpacks.

#### Serving

The goal of `Serving` is to serving user functions in a highly elastic manner (dynamically scale 0 <-> N).

Currently, OpenFunction supports two serving runtimes, [Knative](#knative) and [OpenFuncAsync](#openfuncasync). At least one of these runtimes is required.

##### Knative

Knative Serving builds on Kubernetes to support deploying and serving serverless applications and functions. Knative Serving is easy to get started with and scales to support advanced scenarios.

##### OpenFuncAsync

OpenFuncAsync is an event-driven Serving runtime. It is implemented based on KEDA + Dapr.

You can refer to [Prerequisites](#prerequisites) and use `--with-openFuncAsync` to install OpenFuncAsync runtime.

The OpenFuncAsync functions can be triggered by various event types, such as MQ, cronjob, DB events, etc. In a Kubernetes cluster, OpenFuncAsync functions run in the form of deployments or jobs.

## QuickStart

### Prerequisites

The current version of OpenFunction requires that you have a Kubernetes cluster with version ``>=1.18.6``.

In addition, you need to deploy several dependencies for the OpenFunction ```Builder``` and ```Serving```.

You can use the following command to setup OpenFunction ```Builder``` and ```Serving```:

```shell
sh hack/deploy.sh --all
```

This command will install dependencies of all supported ```Builder``` and ```Serving``` to your cluster.

You can also customize the installation with the following parameters:

|Parameter|Comment|
|---|---|
| --all                              | Install all supported ```Builder``` and ```Serving``` |
| --with-shipwright                  | Install Shipwright builder |
| --with-knative                     | Install Knative serving runtime |
| --with-openFuncAsync               | Install OpenFuncAsync serving runtime |
| --poor-network                     | Use this if you have poor network connectivity to GitHub/Googleapis |
| --tekton-dashboard-nodeport <port> | Expose the Tekton dashboard service with NodePort |

### Install OpenFunction

You can install the OpenFunction platform by the following command:

- Install the latest stable version

```shell
kubectl apply -f https://github.com/OpenFunction/OpenFunction/releases/download/v0.3.1/bundle.yaml
```

- Install the development version

```shell
kubectl apply -f https://raw.githubusercontent.com/OpenFunction/OpenFunction/main/config/bundle.yaml
```

> Note: When using non-default namespaces, make sure that the ClusterRoleBinding in the namespace is adapted.

### Verify the installation

```bash
kubectl get pods --namespace openfunction

NAME                                               READY   STATUS    RESTARTS   AGE
openfunction-controller-manager-6955498c9b-hjql7   2/2     Running   0         2m2s
```
### Sample: Run a function.

If you have already installed the OpenFunction platform, refer to [OpenFunction samples](https://github.com/OpenFunction/samples) to find more samples.

Here is an example of a sync function.

> This function will write "Hello, World!" to the HTTP response.

```go
package hello

import (
	"net/http"
	"fmt"
)

func HelloWorld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, World!\n")
}
```

And an async function example.

> This function will receive a greeting message and then send it to "another-target".

```go
package bindings

import (
	"encoding/json"
	"log"
  
	ofctx "github.com/OpenFunction/functions-framework-go/openfunction-context"
)

func BindingsOutput(ctx *ofctx.OpenFunctionContext, in []byte) int {
	log.Printf("receive greeting: %s", string(in))
	_ := ctx.SendTo(in, "another-target")
	return 200
}
```

### Uninstall OpenFunction

- Uninstall the latest stable version

```shell
kubectl delete -f https://raw.githubusercontent.com/OpenFunction/OpenFunction/release-0.3/config/bundle.yaml
```

- Uninstall the development version

```shell
kubectl delete -f https://raw.githubusercontent.com/OpenFunction/OpenFunction/main/config/bundle.yaml
```

## Development

You can get help on developing this project by visiting [Development Guide](docs/development/README.md).

## Roadmap

[Here](docs/roadmap.md) you can find OpenFunction's roadmap.

## Community

### Community Call

Meeting Info: [Zoom](https://us02web.zoom.us/j/89684762679?pwd=U1JNWVdzbElScVFMSEdQQnV0YnR4UT09) 

Meeting Time: Wednesday at 15:30~16:30 Beijing Time (biweekly, starting from June 23rd, 2021) [Meeting Calendar](https://kubesphere.io/contribution/)

[Meeting notes](https://docs.google.com/document/d/1bh5-kVPegjNlIjjq_e37mS3ZhyXWhmmUaysFgeI9_-o/edit?usp=sharing)

### Contact Us

- Slack [#sig-serverless](https://kubesphere.slack.com/archives/C021XAR3CG3)
