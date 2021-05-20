# ![OpenFunction](docs/images/logo.png)

## Overview

```OpenFunction``` is a cloud-native open source FaaS (Function as a Service) platform aiming to enable users to focus on their business logic without worrying about the underlying runtime environment and infrastructure. Users only need to submit business-related source code in the form of functions.

```OpenFunction``` features but not limited to the following:

- Convert business-related function source code to runnable application source code.
- Generate a deployable container image from the converted application source code.
- Deploy the generated container image to the underlying runtime environment such as K8s, and automatically scale up and down according to business traffic, and scale to 0 when there is no traffic.
- Provide event management functions for trigger functions.
- Provide additional functions to manage function versions, ingress management etc.

![](docs/images/OpenFunction-architecture.png)

---

## Prerequisites

The current version of OpenFunction requires that you have a Kubernetes cluster with version ``>=1.18.6``.

In addition, you need to deploy several dependencies for the OpenFunction ```Builder``` and ```Serving```.

You can refer to the [Installation Guide](docs/installation/README.md) to setup OpenFunction ```Builder``` and ```Serving```.

### Builder

You need to install at least one of the following options for builders:

- Currently, OpenFunction Builder uses Tekton and Cloud Native Buildpacks to build container images, you need to [install Tekton](https://tekton.dev/docs/getting-started/#installation).
    
### Serving

You need to install at least one of the following options for the serving component:

- Currently, OpenFunction Serving relies on Knative, so you need to [install Knative Serving](https://knative.dev/docs/install/).
- Another Serving runtime Dapr + KEDA will be supported soon.

### Tekton and Knative

You can deploy Tekton and Knative follow this command.

```bash
sh hack/deploy-tekon-and-knative.sh
```
You deploy Tekton and Knative follow this command if you having poor network connectivity to GitHub/Googleapis.

```bash
sh hack/deploy-tekon-and-knative.sh --poor-network
```

If you want to use NodePort to expose the Tekton dashboard service, follow this command.

```bash
sh hack/deploy-tekon-and-knative.sh --tekton-dashboard-nodeport <port>
```

## CustomResourceDefinitions

The core function of OpenFunction is to enable users to develop, run, and manage business applications as execution units of code functions. OpenFunction implements the following [custom resource definitions (CRDs)](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/):  

- **```Function```**, defines a function.
- **```Builder```**, defines a function builder.
- **```Serving```**, defines a function workload.

## QuickStart

### Install

You can install the OpenFunction platform by the following command:

- Install the latest stable version

```shell
kubectl apply -f https://github.com/OpenFunction/OpenFunction/releases/download/v0.1.0/release.yaml
```

- Install the development version

```shell
kubectl apply -f https://raw.githubusercontent.com/OpenFunction/OpenFunction/main/config/bundle.yaml
```

> Note: When using non-default namespaces, make sure that the ClusterRoleBinding in the namespace is adapted.

### Sample: Run a function.

If you have already installed the OpenFunction platform, follow the steps below to run a sample function.

1. Creating a secret

    In order to access your container registry, you need to create a secret. You can create this secret by editing the ``username`` and ``password`` fields in ``config/samples/registry-account.yaml``, and then apply it.

    ```shell
    kubectl apply -f config/samples/registry-account.yaml
    ```

2. Creating functions

    For sample function below, modify the ``spec.image`` field in ``config/samples/function-sample.yaml`` to your own container registry address: 

    ```yaml
    apiVersion: core.openfunction.io/v1alpha1
    kind: Function
    metadata:
      name: function-sample
    spec:
      image: "<your registry name>/sample-go-func:latest"
    ```

    Use the following command to create this Function:

    ```shell
    kubectl apply -f config/samples/function-sample.yaml
    ```

3. Result observation

    You can observe the process of a function with the following command:

    ```shell
    kubectl get functions.core.openfunction.io

    NAME              AGE
    function-sample   5s
    ```

    You can also observe the process of a builder in the [Tekton Dashboard](https://tekton.dev/docs/dashboard/).

    Finally, you can observe the final state of the function workload in the Serving:

    ```shell
    kubectl get servings.core.openfunction.io
     
    NAME                      AGE
    function-sample-serving   15s
    ```

    You can now find out the service entry of the function with the following command:

    ```shell
    kubectl get ksvc
     
    NAME                           URL                                                                  LATESTCREATED                        LATESTREADY                          READY   REASON
    function-sample-serving-ksvc   http://function-sample-serving-ksvc.default.<external-ip>.sslip.io   function-sample-serving-ksvc-00001   function-sample-serving-ksvc-00001   True
    ```

    Or get the service address directly with the following command:

    > where` <external-ip> `indicates the external address of your gateway service. 
    >
    > You can do a simple configuration to use the node ip as the `<external-ip>` as follows  (Assuming you are using Kourier as network layer of Knative). Where `1.2.3.4` can be replaced by your node ip.
    >
    > ```shell
    > kubectl patch svc -n kourier-system kourier \
    >   -p '{"spec": {"type": "LoadBalancer", "externalIPs": ["1.2.3.4"]}}'
    > 
    > kubectl patch configmap/config-domain -n knative-serving \
    >   --type merge --patch '{"data":{"1.2.3.4.sslip.io":""}}'
    > ```

    ```shell
    kubectl get ksvc function-sample-serving-ksvc -o jsonpath={.status.url}
     
    http://function-sample-serving-ksvc.default.<external-ip>.sslip.io
    ```

    Access the above service address via commands such as ``curl``:

    ```shell
    curl http://function-sample-serving-ksvc.default.<external-ip>.sslip.io
     
    Hello, World!
    ```

### Removal

You can uninstall the components of OpenFunction by executing the following command:

```shell
kubectl delete -f config/bundle.yaml
```

## Development

You can get help on developing this project by visiting [Development Guide](docs/development/README.md).

## Roadmap

[Here](docs/roadmap.md) you can find OpenFunction's roadmap.