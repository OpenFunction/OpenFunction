# Development Guide

---

本文将向您介绍如何在本地构建 OpenFunction 项目的开发环境。

## QuickStart

---

您可以通过访问 [sample-controller](https://github.com/kubernetes/sample-controller) 和 [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) 来获取开发 controllers 模块的帮助。

### Go

OpenFunction 基于 [Kubernetes](https://github.com/kubernetes/kubernetes) 。 皆使用 [Go](http://golang.org/) 语言开发。 您可以先遵循 [这里](http://golang.org/doc/code.html) 搭建Go开发环境。

| Kubernetes | requires Go |
| ---------- | ----------- |
| 1.18+      | 1.15<=go    |

> Tips:
>
> - 请确认您的Go环境中设置了GOPATH和PATH。
> - 当时用MacOs进行开发时，请务必安装 [macOS GNU tools](https://www.topbug.net/blog/2013/04/14/install-and-use-gnu-command-line-tools-in-mac-os-x) 。

### Docker

OpenFunction的组件通常以容器的方式运行于Kubernetes环境中，您可以使用容器的方式来重建您的OpenFunction组件。请参考 [install Docker](https://docs.docker.com/install/) 来搭建容器环境。

### Dependency Management

OpenFunction使用 [Go Modules](https://github.com/golang/go/wiki/Modules) 来管理`vendor/`目录下的依赖。

## Build Image&Apply

---

您可以通过修改```cmd/Dockerfile```来制作本地环境的controllers容器镜像。将镜像上传至个人镜像仓库后，可以通过以下命令修改工作负载（Deployment）openfunction-controller-manager中openfunction容器对应的镜像地址将其切换至您的个人镜像仓库：

```shell
kubectl edit deployments.apps -n openfunction openfunction-controller-manager
```

随后Kubernetes会自动应用您更新后的controllers。

