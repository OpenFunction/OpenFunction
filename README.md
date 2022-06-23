<p align="center">
<a href="https://openfunction.dev/"><img src="docs/images/openfunction-logo-gif.gif" alt="banner" width="500px"></a>
</p>

<p align="center">
<b>Cloud native FaaS platform for running Serverless workloads with ease</b>
</p>

<p align=center>
<a href="https://goreportcard.com/report/github.com/openfunction/openfunction"><img src="https://goreportcard.com/badge/github.com/openfunction/openfunction" alt="A+"></a>
<a href="https://hub.docker.com/r/openfunction/openfunction"><img src="https://img.shields.io/docker/pulls/openfunction/openfunction"></a>
<a href="https://github.com/OpenFunction/OpenFunction/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22"><img src="https://img.shields.io/github/issues/openfunction/openfunction?label=good%20first%20issues" alt="good first"></a>
<a href="https://twitter.com/intent/follow?screen_name=KubeSphere"><img src="https://img.shields.io/twitter/follow/KubeSphere?style=social" alt="follow on Twitter"></a>
<a href="https://join.slack.com/t/kubesphere/shared_invite/enQtNTE3MDIxNzUxNzQ0LTZkNTdkYWNiYTVkMTM5ZThhODY1MjAyZmVlYWEwZmQ3ODQ1NmM1MGVkNWEzZTRhNzk0MzM5MmY4NDc3ZWVhMjE"><img src="https://img.shields.io/badge/Slack-600%2B-blueviolet?logo=slack&amp;logoColor=white"></a>
<a href="https://www.youtube.com/channel/UCyTdUQUYjf7XLjxECx63Hpw"><img src="https://img.shields.io/youtube/channel/subscribers/UCyTdUQUYjf7XLjxECx63Hpw?style=social"></a>
</p>

## 👀 Overview

[OpenFunction](https://openfunction.dev/) is a cloud-native open source FaaS (Function as a Service) platform aiming to let you focus on your business logic without having to maintain the underlying runtime environment and infrastructure. You only need to submit business-related source code in the form of functions.

<div align=center><img src=docs/images/function-lifecycle.svg></div>

OpenFunction features include:

- Cloud agnostic and decoupled with cloud providers' BaaS
- Pluggable architecture that allows multiple function runtimes
- Support both sync and async functions
- Unique async functions support that can consume events directly from event sources
- Support generating OCI-Compliant container images directly from function source code.
- Flexible autoscaling between 0 and N
- Advanced async function autoscaling based on event sources' specific metrics
- Simplified BaaS integration for both sync and async functions by introducing [Dapr](https://dapr.io/) 
- Advanced function ingress & traffic management powered by [K8s Gateway API](https://gateway-api.sigs.k8s.io/) (In Progress)
- Flexible and easy-to-use events management framework

## ☸ Architecture

![OpenFunction Architecture](docs/images/openfunction-0.5-architecture.svg)

OpenFunction manages resources in the form of Custom Resource Definitions (CRD) throughout the lifecycle of a function. To learn more about it, visit [Components](docs/concepts/Components.md) or [Concepts](https://openfunction.dev/docs/concepts/).

<div align=center><img src=docs/images/OpenFunction-events-architecture.svg></div>

OpenFunction Events is OpenFunction's events framework, you can refer to [OpenFunction Events](https://github.com/OpenFunction/OpenFunction/blob/main/docs/concepts/OpenFunction-events-framework.md) for more information.
## ✔️ Compatibility

### Kubernetes compatibility matrix

The following Kubernetes versions are supported as we tested against these versions in their respective branches. Besides, OpenFunction might also work well with other Kubernetes versions!

| OpenFunction                                                 | Kubernetes 1.17 | Kubernetes 1.18 | Kubernetes 1.19 | Kubernetes 1.20+ |
| ------------------------------------------------------------ | --------------- | --------------- | --------------- | ---------------- |
| [`release-0.4`](https://github.com/OpenFunction/OpenFunction/tree/v0.4.0) | &radic;         | &radic;         | &radic;         | &radic;          |
| [`release-0.5`](https://github.com/OpenFunction/OpenFunction/tree/v0.5.0) | &radic; *         | &radic; *         | &radic;         | &radic;          |
| [`release-0.6`](https://github.com/OpenFunction/OpenFunction/tree/v0.6.0) | &radic; *         | &radic; *         | &radic;         | &radic;          |
| [`HEAD`](https://github.com/OpenFunction/OpenFunction/tree/main) | &radic; *         | &radic; *         | &radic;         | &radic;          |

\****Note***: OpenFunction has added the [function ingress](docs/concepts/Components.md#domain) feature in *release-0.5*, which means that:

- You have to install OpenFunction in Kuberenetes ***v1.19*** or later if you enable this feature.
- You can still use OpenFunction in Kubernetes ***v1.17—v1.20+*** without this feature enabled.

## 🚀 QuickStart

### Install OpenFunction
To install OpenFunction, please refer to [Installation Guide](https://openfunction.dev/docs/getting-started/installation/#install-openfunction).

### Run a function sample

After you install OpenFunction, refer to [OpenFunction samples](https://github.com/OpenFunction/samples) to learn more about function samples.

Here is an example of a synchronous function:

> This function writes "Hello, World!" to the HTTP response. Refer to [here](https://github.com/OpenFunction/samples/tree/main/functions/Knative) to find more samples of synchronous functions.

```go
package hello

import (
	"fmt"
	"net/http"
)

func HelloWorld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s!\n", r.URL.Path[1:])
}
```

[Function ingress](docs/concepts/Components.md#domain) defines a unified entry point for a synchronous function. You can use it as in below format to access a synchronous function without [configuring LB for Knative](https://github.com/OpenFunction/samples/tree/main/functions/Knative/hello-world-go).

```bash
curl http://<domain-name>.<domain-namespace>/<function-namespace>/<function-name>
```

Here is an example of an async function:

> This function receives a greeting message and then send it to "another-target". Refer to [here](https://github.com/OpenFunction/samples/tree/main/functions/Async) to find more samples of asynchronous functions.

```go
package bindings

import (
	"encoding/json"
	"log"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
)

func BindingsOutput(ctx ofctx.Context, in []byte) (ofctx.Out, error) {
	var greeting []byte
	if in != nil {
		greeting = in
	} else {
		greeting, _ = json.Marshal(map[string]string{"message": "Hello"})
	}

	_, err := ctx.Send("another-target", greeting)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return ctx.ReturnOnInternalError(), err
	}
	return ctx.ReturnOnSuccess(), nil
}
```

One more example with tracing capability: 

> SkyWalking provides solutions for observing and monitoring distributed systems, in many different scenarios.
>
> We have introduced SkyWalking (go2sky) for OpenFunction as a distributed tracing solution for Go language functions. 

You can find the method to enable SkyWalking tracing for Go functions in [tracing sample](https://github.com/OpenFunction/samples/blob/main/functions/tracing/README.md).

![](docs/images/tracing-topology.gif)

You can also run the following command to make a quick demo:

```shell
ofn demo
```

>By default, a demo environment will be deleted when a demo finishes.
>You can keep the demo kind cluster for further exploration by running `ofn demo --auto-prune=false`.
>The demo kind cluster can be deleted by running `kind delete cluster --name openfunction`.

For more information about how to use the `ofn demo` command, refer to [ofn demo document](https://github.com/OpenFunction/cli/blob/main/docs/demo.md).

### Uninstall OpenFunction

To uninstall OpenFunction, please refer to [Uninstallation Guide](https://openfunction.dev/docs/getting-started/installation/#uninstall-openfunction).

### FAQ

When you encounter any problems when using OpenFunction, you can refer to the [FAQ](https://openfunction.dev/docs/reference/faq/) for help.

## 💻 Development

See the [Development Guide](docs/development/README.md) to get started with developing this project.

## 🛣️ Roadmap

Learn more about OpenFunction [roadmap](https://github.com/orgs/OpenFunction/projects/3/views/1?layout=board).

## 🏘️ Community

### Community Call and Events

Meeting time：15:00-16:00(GMT+08:00), Thursday every two weeks starting from March 17th, 2022

Meeting room: [Tencent Meeting](https://meeting.tencent.com/dm/HMec1CjT8F2i)

Tencent Meeting Number: 443-6181-3052

Check out the [meeting calendar](https://kubesphere.io/contribution/) and [meeting notes](https://docs.google.com/document/d/1bh5-kVPegjNlIjjq_e37mS3ZhyXWhmmUaysFgeI9_-o/edit?usp=sharing).

OpenFunction team has presented OpenFunction project in some community events including [CNCF TAG-runtime meeting](https://youtu.be/qDH_LbagrVA?t=821), [Dapr community meeting](https://youtu.be/S9e3ol7JCDA?t=183), and [OpenFunction community call](https://space.bilibili.com/438908638/channel/seriesdetail?sid=495452). You can watch the recording videos and ask us anything.
### Contact Us

OpenFunction is sponsored and open-sourced by the [KubeSphere](http://kubesphere.io/) Team and maintained by the OpenFunction community.

- Discord: https://discord.gg/hG3yc6uAyC
- Slack: Create an account at [#CNCF Slack](https://slack.cncf.io/) and then search `#openfunction` to join or click [#openfunction](https://cloud-native.slack.com/archives/C03ETDMD3LZ) to join if you already have an [#CNCF Slack](https://slack.cncf.io/) account.
- Wechat: join the OpenFunction user group by following the `kubesphere` WeChat subscription and then reply `openfunction`

## Landscape

<p align="center">
<br/><br/>
<img src="https://landscape.cncf.io/images/left-logo.svg" width="150"/>&nbsp;&nbsp;<img src="https://landscape.cncf.io/images/right-logo.svg" width="200"/>&nbsp;&nbsp;
<br/><br/>
OpenFunction enriches the <a href="https://landscape.cncf.io/serverless?license=apache-license-2-0">CNCF Cloud Native Landscape.
</a>
</p>

## 📊 Status

![Alt](https://repobeats.axiom.co/api/embed/48814fec53572bf75ac4de9d4f447d2c978b26ee.svg "Repobeats analytics image")
