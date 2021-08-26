# Setup OpenFunction Serving with Knative

You can refer to the steps in [Knative docs](https://knative.dev/docs/install/any-kubernetes-cluster/) to install Knative or follow the steps below:
> :grey_exclamation: Refer to [this section](#installation-when-having-poor-network-connectivity-to-githubgoogleapis) when you have **poor network connections to GitHub/Googleapis**.

## Installation

### Install Knative CLI

Choose a suitable installation of Knative CLI for your cluster by refer to this [docs](https://knative.dev/docs/client/install-kn/).

### Install Knative Serving

#### Install the required custom resources

```bash
kubectl apply -f https://github.com/knative/serving/releases/download/v0.23.0/serving-crds.yaml
```

#### Install the core components of Serving

```bash
kubectl apply -f https://github.com/knative/serving/releases/download/v0.23.0/serving-core.yaml
```

#### Install a networking layer

- **Kouier**

1. Install the Knative Kourier controller
   
    ```bash
    kubectl apply -f https://github.com/knative/net-kourier/releases/download/v0.23.0/kourier.yaml
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

- **Magic DNS (sslip.io)**

    ```bash
    kubectl apply -f https://github.com/knative/serving/releases/download/v0.23.0/serving-default-domain.yaml
    ```

## Installation when having poor network connectivity to GitHub/Googleapis

### Install Knative CLI

Choose a suitable installation of Knative CLI for your cluster by refer to this [docs](https://knative.dev/docs/client/install-kn/).

### Install Knative Serving

#### Install the required custom resources

```bash
kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.23.0/serving-crds.yaml
```

#### Install the core components of Serving

```bash
kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.23.0/serving-core.yaml
```

#### Install a networking layer

- **Kouier**

1. Install the Knative Kourier controller

    ```bash
    kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/net-kourier/v0.23.0/kourier.yaml
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

- **Magic DNS (sslip.io)**

    ```bash
    kubectl apply -f https://openfunction.sh1a.qingstor.com/knative/serving/v0.23.0/serving-default-domain.yaml
    ```
  