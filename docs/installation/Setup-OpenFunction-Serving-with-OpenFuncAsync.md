# Setup OpenFunction Serving with OpenFuncAsync

## Installation

### Install Dapr

#### Install Dapr CLI

Installs the latest Dapr CLI to /usr/local/bin:

```bash
wget -q https://raw.githubusercontent.com/dapr/cli/master/install/install.sh -O - | /bin/bash
```

#### Init Dapr in kubernetes

```bash
dapr init -k
```

#### Verify the installation

```bash
kubectl get po -n dapr-system 
NAME                                    READY   STATUS    RESTARTS   AGE
dapr-dashboard-564485bbb7-7drhp         1/1     Running   0          6m45s
dapr-operator-7f6d6cd94d-qvgtz          1/1     Running   0          6m45s
dapr-placement-server-0                 1/1     Running   0          6m45s
dapr-sentry-86dc6d67f-dndt4             1/1     Running   0          6m45s
dapr-sidecar-injector-584c69d8f-mx5nx   1/1     Running   0          6m45s
```

### Install Keda

Installs the latest release version

```bash
kubectl apply -f https://github.com/kedacore/keda/releases/download/v2.2.0/keda-2.2.0.yaml
```
```