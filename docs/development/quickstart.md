# QuickStart

---

This document walks you through how to get started with building OpenFunction in your local environment.

## Prerequisites

---

If you are interested in developing controller-manager, see [sample-controller](https://github.com/kubernetes/sample-controller) and [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder).

### Go

OpenFunction is based on [Kubernetes](https://github.com/kubernetes/kubernetes). Both of them are written in [Go](http://golang.org/). If you don't have a Go development environment, please [set it up](http://golang.org/doc/code.html) first.

| Kubernetes | requires Go |
| ---------- | ----------- |
| 1.18+      | go>=1.12    |

> Tips:
>
> - Ensure your GOPATH and PATH have been configured in accordance with the Go environment instructions.
> - It's recommended to install [macOS GNU tools](https://www.topbug.net/blog/2013/04/14/install-and-use-gnu-command-line-tools-in-mac-os-x) when using MacOS for development.

### Docker

OpenFunction components are often deployed as containers in Kubernetes. If you need to rebuild the OpenFunction components in the Kubernetes cluster, you'll need to [install Docker](https://docs.docker.com/install/) in advance.

### Dependency Management

OpenFunction uses [Go Modules](https://github.com/golang/go/wiki/Modules) to manage dependencies.

## Build Image & Apply

---

You can build ```openfunction``` image for your local environment by modifying ``cmd/Dockerfile``. 

After uploading the image to your personal image repository, change the image url corresponding to the ```openfunction``` container in the workload (Deployment) openfunction-controller-manager and switch it to your personal image repository.

```shell
kubectl edit deployments.apps -n openfunction openfunction-controller-manager
```

Kubernetes will then automatically apply your controllers.

