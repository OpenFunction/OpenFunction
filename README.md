<div align=center><img  width="334" height="117" src=docs/images/logo.png></div>

## 👀 Overview

[OpenFunction](https://openfunction.dev/) is a cloud-native open source FaaS (Function as a Service) platform aiming to let you focus on your business logic without having to maintain the underlying runtime environment and infrastructure. You only need to submit business-related source code in the form of functions.

<div align=center><img src=docs/images/function-lifecycle.svg></div>

OpenFunction features include:

- Converting business-related function source code to application source code.
- Generating ready-to-run container images from the converted application source code.
- Deploying generated container images to any underlying runtime environments such as Kubernetes, and automatically scaling up and down from 0 to N according to business traffic.
- Providing event management functions for trigger functions.
- Providing additional functions for function version management, ingress management, etc.

## ☸ Custom Resource Definitions

<div align=center><img src=docs/images/openfunction-overview.svg></div>

OpenFunction manages resources in the form of Custom Resource Definitions (CRD) throughout the lifecycle of a function. To learn more about it, visit [Components](docs/concepts/Components.md) or [Concepts](https://openfunction.dev/docs/concepts/).

## ✔️ Compatibility

### Kubernetes compatibility matrix

The following Kubernetes versions are supported as we tested against these versions in their respective branches. Besides, OpenFunction might also work well with other Kubernetes versions!

| OpenFunction                                                 | Kubernetes 1.17 | Kubernetes 1.18 | Kubernetes 1.19 | Kubernetes 1.20+ |
| ------------------------------------------------------------ | --------------- | --------------- | --------------- | ---------------- |
| [`release-0.3`](https://github.com/OpenFunction/OpenFunction/tree/v0.3.0) | &radic;         | &radic;         | &radic;         | &radic;          |
| [`release-0.4`](https://github.com/OpenFunction/OpenFunction/tree/v0.4.0) | &radic;         | &radic;         | &radic;         | &radic;          |
| [`HEAD`](https://github.com/OpenFunction/OpenFunction/tree/main) | &radic; *         | &radic; *         | &radic;         | &radic;          |

\****Note***: OpenFunction has added the [function ingress](docs/concepts/Components.md#domain) feature since *release-0.4*, which means that:

- You have to install OpenFunction in Kuberenetes ***v1.19*** or later if you enable this feature.
- You can still use OpenFunction in Kubernetes ***v1.17—v1.20+*** without this feature enabled.

## 🚀 QuickStart

### Install OpenFunction

Visit [ofn releases page](https://github.com/OpenFunction/cli/releases) to dowload the latest version of `ofn`, the CLI of OpenFunction, to install OpenFunction and its dependencies on your Kubernetes cluster.

Besides, you can perform the following steps to install the latest version of OpenFunction.

1. Run the following command to download `ofn`.

   ```
   wget -c  https://github.com/OpenFunction/cli/releases/latest/download/ofn_linux_amd64.tar.gz -O - | tar -xz
   ```

2. Run the following commands to make `ofn` executable and move it to `/usr/local/bin/`.

   ```
   chmod +x ofn && mv ofn /usr/local/bin/
   ```

3. Run the following command to install OpenFunction.

   ```
   ofn install --all
   ```

For more information about how to use the `ofn install` command, refer to [ofn install document](https://github.com/OpenFunction/cli/blob/main/docs/install.md).

### Run a function sample

After you install OpenFunction, refer to [OpenFunction samples](https://github.com/OpenFunction/samples) to learn more about function samples.

Here is an example of a synchronous function:

> This function writes "Hello, World!" to the HTTP response.

```go
package hello

import (
	"fmt"
	"net/http"
)

func HelloWorld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, World!\n")
}
```

[Function ingress](docs/concepts/Components.md#domain) defines a unified entry point for a synchronous function. You can use it as in below format to access a synchronous function without [configuring LB for Knative](https://github.com/OpenFunction/samples/tree/main/functions/Knative/hello-world-go).

```bash
curl http://<domain-name>.<domain-namespace>/<function-namespace>/<function-name>
```

Here is an example of asynchronous function:

> This function receives a greeting message and then send it to "another-target".

```go
package bindings

import (
	"encoding/json"
	"log"
  
	ofctx "github.com/OpenFunction/functions-framework-go/openfunction-context"
)

func BindingsOutput(ctx *ofctx.OpenFunctionContext, in []byte) ofctx.RetValue {
	log.Printf("receive greeting: %s", string(in))
	_ := ctx.Send("another-target", in)
	return 200
}
```

You can also run the following command to make a quick demo:

```shell
ofn demo
```

>By default, a demo environment will be deleted when a demo finishes.
>You can keep the demo kind cluster for further exploration by running `ofn demo --auto-prune=false`.
>The demo kind cluster can be deleted by running `kind delete cluster --name openfunction`.

For more information about how to use the `ofn demo` command, refer to [ofn demo document](https://github.com/OpenFunction/cli/blob/main/docs/demo.md).

### Uninstall OpenFunction

Run the following command to uninstall OpenFunction and its dependencies.

```shell
ofn uninstall --all
```

For more information about how to use the `ofn uninstall` command, refer to [ofn uninstall document](https://github.com/OpenFunction/cli/blob/main/docs/uninstall.md).

## 💻 Development

See the [Development Guide](docs/development/README.md) to get started with developing this project.

## 🛣️ Roadmap

Learn more about OpenFunction [roadmap](docs/roadmap.md).

## 🏘️ Community

### Community Call

Meeting room: [Zoom](https://us02web.zoom.us/j/89684762679?pwd=U1JNWVdzbElScVFMSEdQQnV0YnR4UT09)

Meeting time: Wednesday at 15:30—16:30 Beijing Time (biweekly, starting from June 23rd, 2021)

Check out the [meeting calendar](https://kubesphere.io/contribution/) and [meeting notes](https://docs.google.com/document/d/1bh5-kVPegjNlIjjq_e37mS3ZhyXWhmmUaysFgeI9_-o/edit?usp=sharing).

### Contact Us

OpenFunction is sponsored and open-sourced by the [KubeSphere](http://kubesphere.io/) Team and maintained by the OpenFunction community.

- Slack: [#sig-serverless](https://kubesphere.slack.com/archives/C021XAR3CG3)
- Wechat: join the OpenFunction user group by following the KubeSphere WeChat subscription

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
