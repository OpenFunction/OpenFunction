# Setup OpenFunction Serving with Knative

You can refer to the installation steps on the [Knative docs](https://knative.dev/docs/install/any-kubernetes-cluster/) for setup OpenFunction Serving. Or follow these steps:
> :grey_exclamation: Refer to [this section](#Installation with poor network connections to GitHub/Googleapis) when you are in a **poor network connections to GitHub/Googleapis**.

## Installation

### Install knative CLI

Choose a suitable installation of knative CLI for your cluster by refer to thie [docs](https://knative.dev/docs/client/install-kn/).

### Install knative serving

#### Install the required custom resources

```bash
kubectl apply -f https://github.com/knative/serving/releases/download/v0.22.0/serving-crds.yaml
```

#### Install the core components of Serving

```bash
kubectl apply -f https://github.com/knative/serving/releases/download/v0.22.0/serving-core.yaml
```

#### Install a networking layer

- **Kouier**

1. Install the Knative Kourier controller
   
    ```bash
    kubectl apply -f https://github.com/knative/net-kourier/releases/download/v0.22.0/kourier.yaml
    ```
2. To configure Knative Serving to use Kourier by default
   
    ```bash
    kubectl patch configmap/config-network \
      --namespace knative-serving \
      --type merge \
      --patch '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'
    ```

3. Fetch the External IP or CNAME

    >We recommend you to configure the ```EXTERNAL-IP``` according to your cluster environment.
 
    ```bash
    kubectl --namespace kourier-system get service kourier
    
    NAME      TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)                      AGE
    kourier   LoadBalancer   10.233.40.58   <pending>     80:30410/TCP,443:31324/TCP   3m38s
    ```

4. Confirm the service readiness
    
    >The following ```<external-ip>``` indicates an external url that can be accessed normally.

    ```bash
    kubectl --namespace kourier-system get service kourier
    
    NAME      TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)                      AGE
    kourier   LoadBalancer   10.233.40.58    <external-ip>    80:30807/TCP,443:32762/TCP   18m
    ```

#### Verify the installation

```bash
kubectl get pods --namespace knative-serving

NAME                                     READY   STATUS    RESTARTS   AGE
3scale-kourier-control-6745584b9-cgv6w   1/1     Running   0          8m47s
activator-6875896748-tjgb9               1/1     Running   0          11m
autoscaler-6bbc885cfd-f6lz5              1/1     Running   0          11m
controller-64dd4bd56-rl6k6               1/1     Running   0          11m
default-domain-pq82l                     1/1     Running   0          2m2s
webhook-75f5d4845d-lg8j5                 1/1     Running   0          11m
```

#### Configure DNS

- **Magic DNS (xip.io)**

    ```bash
    kubectl apply -f https://github.com/knative/serving/releases/download/v0.22.0/serving-default-domain.yaml
    ```

### Install knative eventing

#### Install the required custom resource definitions (CRDs)

```bash
kubectl apply -f https://github.com/knative/eventing/releases/download/v0.22.0/eventing-crds.yaml
```

#### Install the core components of Eventing

```bash
kubectl apply -f https://github.com/knative/eventing/releases/download/v0.22.0/eventing-core.yaml
```

#### Verify the installation

```bash
kubectl get pods --namespace knative-eventing

NAME                                   READY   STATUS    RESTARTS   AGE
eventing-controller-d666b4657-jwsrk    1/1     Running   0          23m
eventing-webhook-778b6b8cf4-mgjtr      1/1     Running   0          23m
```

## Installation with poor network connections to GitHub/Googleapis

### Install knative CLI

Choose a suitable installation of knative CLI for your cluster by refer to thie [docs](https://knative.dev/docs/client/install-kn/).

### Install knative serving

#### Install the required custom resources

```bash
kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.22.0/serving-crds.yaml
```

#### Install the core components of Serving

```bash
kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.22.0/serving-core.yaml
```

#### Install a networking layer

- **Kouier**

1. Install the Knative Kourier controller

    ```bash
    kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/net-kourier/v0.22.0/kourier.yaml
    ```
2. To configure Knative Serving to use Kourier by default

    ```bash
    kubectl patch configmap/config-network \
      --namespace knative-serving \
      --type merge \
      --patch '{"data":{"ingress.class":"kourier.ingress.networking.knative.dev"}}'
    ```

3. Fetch the External IP or CNAME

   >We recommend you to configure the ```EXTERNAL-IP``` according to your cluster environment.

    ```bash
    kubectl --namespace kourier-system get service kourier
    
    NAME      TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)                      AGE
    kourier   LoadBalancer   10.233.40.58   <pending>     80:30410/TCP,443:31324/TCP   3m38s
    ```

4. Confirm the service readiness

   >The following ```<external-ip>``` indicates an external url that can be accessed normally.

    ```bash
    kubectl --namespace kourier-system get service kourier
    
    NAME      TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)                      AGE
    kourier   LoadBalancer   10.233.40.58    <external-ip>    80:30807/TCP,443:32762/TCP   18m
    ```

#### Verify the installation

```bash
kubectl get pods --namespace knative-serving

NAME                                     READY   STATUS    RESTARTS   AGE
3scale-kourier-control-6745584b9-cgv6w   1/1     Running   0          8m47s
activator-6875896748-tjgb9               1/1     Running   0          11m
autoscaler-6bbc885cfd-f6lz5              1/1     Running   0          11m
controller-64dd4bd56-rl6k6               1/1     Running   0          11m
default-domain-pq82l                     1/1     Running   0          2m2s
webhook-75f5d4845d-lg8j5                 1/1     Running   0          11m
```

#### Configure DNS

- **Magic DNS (xip.io)**

    ```bash
    kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.22.0/serving-default-domain.yaml
    ```

### Install knative eventing

#### Install the required custom resource definitions (CRDs)

```bash
kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/eventing/v0.22.0/eventing-crds.yaml
```

#### Install the core components of Eventing

```bash
kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/eventing/v0.22.0/eventing-core.yaml
```

#### Verify the installation

```bash
kubectl get pods --namespace knative-eventing

NAME                                   READY   STATUS    RESTARTS   AGE
eventing-controller-d666b4657-jwsrk    1/1     Running   0          23m
eventing-webhook-778b6b8cf4-mgjtr      1/1     Running   0          23m
```