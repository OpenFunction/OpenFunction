# Setup OpenFunction Builder with Tekton

You can refer to the steps in [Tekon docs](https://tekton.dev/docs/getting-started/) to install Tekton or follow the steps below:
> :grey_exclamation: Refer to [this section](#installation-with-poor-network-connections-to-githubgoogleapis) when you have **poor network connections to GitHub/Googleapis**.

## Installation

### Install Tekton pipeline

```bash
kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
```

#### Confirm that every component listed has the status Running

```bash
kubectl get pods --namespace tekton-pipelines

NAME                                          READY   STATUS    RESTARTS   AGE
tekton-pipelines-controller-6b94f5f96-hdjw5   1/1     Running   0          20m
tekton-pipelines-webhook-5bfbbd6475-6fl2r     1/1     Running   0          20m
```

### Install Tekton CLI

Choose a suitable installation of Tekton CLI for your cluster by refer to thie [docs](https://tekton.dev/docs/cli/).

### Install Tekton triggers

```bash
kubectl apply --filename https://storage.googleapis.com/tekton-releases/triggers/latest/release.yaml
```

### Install Tekton Dashboard

```bash
kubectl apply --filename https://github.com/tektoncd/dashboard/releases/latest/download/tekton-dashboard-release.yaml
```

If you want to use ```NodePort``` to expose the Tekton dashboard service, you need to modify the ```spec``` fields of ```svc tekton-dashboard``` like below：

1. Use this command:
    ```bash
    kubectl edit -n tekton-pipelines svc tekton-dashboard
    ```
2. And modify the information:
    ```yaml
    spec:
      clusterIP: 10.233.56.213
      externalTrafficPolicy: Cluster
      ports:
      - name: http
        nodePort: 31003 # Select a suitable port
        port: 9097
        protocol: TCP
        targetPort: 9097
      selector:
        app.kubernetes.io/component: dashboard
        app.kubernetes.io/instance: default
        app.kubernetes.io/name: dashboard
        app.kubernetes.io/part-of: tekton-dashboard
      sessionAffinity: None
      type: NodePort # change to NodePort
    ```

## Installation when having poor network connectivity to GitHub/Googleapis

> We keep track of the latest versions by default, and you can request a specific version by submitting an issue.

### Install Tekton pipeline

```bash
kubectl apply --filename https://openfunction.sh1a.qingstor.com/tekton/pipeline/v0.23.0/release.yaml
```

#### Confirm that every component listed has the status Running

```bash
kubectl get pods --namespace tekton-pipelines

NAME                                          READY   STATUS    RESTARTS   AGE
tekton-pipelines-controller-6b94f5f96-hdjw5   1/1     Running   0          20m
tekton-pipelines-webhook-5bfbbd6475-6fl2r     1/1     Running   0          20m
```

### Install Tekton CLI

Choose a suitable installation of Tekton CLI for your cluster by refer to thie [docs](https://tekton.dev/docs/cli/).

### Install Tekton triggers

```bash
kubectl apply --filename https://openfunction.sh1a.qingstor.com/tekton/trigger/v0.13.0/release.yaml
```

### Install Tekton Dashboard

```bash
kubectl apply --filename https://openfunction.sh1a.qingstor.com/tekton/dashboard/v0.16.0/release.yaml
```

If you want to use ```NodePort``` to expose the Tekton dashboard service, you need to modify the ```spec``` fields of ```svc tekton-dashboard``` like below：

1. Use this command:
    ```bash
    kubectl edit -n tekton-pipelines svc tekton-dashboard
    ```
2. And modify the information:
    ```yaml
    spec:
      clusterIP: 10.233.56.213
      externalTrafficPolicy: Cluster
      ports:
      - name: http
        nodePort: 31003 # Select a suitable port
        port: 9097
        protocol: TCP
        targetPort: 9097
      selector:
        app.kubernetes.io/component: dashboard
        app.kubernetes.io/instance: default
        app.kubernetes.io/name: dashboard
        app.kubernetes.io/part-of: tekton-dashboard
      sessionAffinity: None
      type: NodePort # change to NodePort
    ```