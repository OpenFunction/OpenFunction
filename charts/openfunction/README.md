# openfunction

![Version: 0.6.0](https://img.shields.io/badge/Version-0.6.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.6.0](https://img.shields.io/badge/AppVersion-0.6.0-informational?style=flat-square)

A Helm chart for OpenFunction on Kubernetes

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| wangyifei | <wangyifei@kubesphere.io> |  |

## Source Code

* <https://github.com/OpenFunction/OpenFunction>

## Prerequisites

* Kubernetes cluster with RBAC (Role-Based Access Control) enabled is required
* Helm 3.4.0 or newer
* Knative Serving 1.0.1
* Kourier 1.0.1
* Serving Default Domain 1.0.1
* Dapr 1.5.1
* Keda 2.4.0
* Shipwright build 0.6.0
* Tekton Pipelines 0.30.0
* Ingress Nginx 1.1.0
* CertManager 1.5.4

## Install the Chart

Ensure Helm is initialized in your Kubernetes cluster.

For more details on initializing Helm, [read the Helm docs](https://helm.sh/docs/)

1. Add openfunction.github.io as an helm repo
    ```
    helm repo add openfunction https://openfunction.github.io/helm-charts/
    helm repo update
    ```

2. Install the openfunction chart on your cluster in the openfunction namespace:
    ```
    helm install openfunction openfunction/openfunction -n openfunction --create-namespace
    ```

## Verify installation

```
kubectl get pods -namespace openfunction
```

## Uninstall the Chart

To uninstall/delete the `openfunction` release:
```
helm uninstall openfunction -n openfunction
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| config.Example | string | `"openfunction.namespace: \"openfunction\"\nopenfunction.config-features.name: \"config-features\"\n# Configuration of the order of the plugins\nplugins: |\n  pre:\n  - plugin1\n  - plugin2\n  post:\n  - plugin2\n  - plugin1\nplugins.tracing: |\n  # Switch for tracing, default to false\n  enabled: true\n  # Provider name can be set to \"skywalking\", \"opentelemetry\"\n  # A valid provider must be set if tracing is enabled.\n  provider:\n    name: \"skywalking\"\n    oapServer: \"localhost:xxx\"\n  # Custom tags to add to tracing\n  tags:\n    func: function-with-tracing\n    layer: faas\n    tag1: value1\n    tag2: value2\n  baggage:\n  # baggage key is `sw8-correlation` for skywalking and `baggage` for opentelemetry\n  # Correlation context for skywalking: https://skywalking.apache.org/docs/main/latest/en/protocols/skywalking-cross-process-correlation-headers-protocol-v1/\n  # baggage for opentelemetry: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/baggage/api.md\n  # W3C Baggage Specification/: https://w3c.github.io/baggage/\n    key: sw8-correlation # key should be baggage for opentelemetry\n    value: \"base64(string key):base64(string value),base64(string key2):base64(string value2)\"\n"` |  |
| config.knativeServingConfigFeaturesName | string | `"config-features"` |  |
| config.knativeServingNamespace | string | `"openfunction"` |  |
| config.pluginsTracing | string | `"enabled: false\n# Provider name can be set to \"skywalking\", \"opentelemetry\"\n# A valid provider must be set if tracing is enabled.\nprovider:\n  name: \"skywalking\"\n  oapServer: \"localhost:xxx\"\n# Custom tags to add to tracing\ntags:\n  func: function-with-tracing\n  layer: faas\n  tag1: value1\n  tag2: value2\nbaggage:\n# baggage key is `sw8-correlation` for skywalking and `baggage` for opentelemetry\n# Correlation context for skywalking: https://skywalking.apache.org/docs/main/latest/en/protocols/skywalking-cross-process-correlation-headers-protocol-v1/\n# baggage for opentelemetry: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/baggage/api.md\n# W3C Baggage Specification/: https://w3c.github.io/baggage/\n  key: sw8-correlation # key should be baggage for opentelemetry\n  value: \"base64(string key):base64(string value),base64(string key2):base64(string value2)\"\n"` |  |
| controllerManager.kubeRbacProxy.image.repository | string | `"openfunction/kube-rbac-proxy"` |  |
| controllerManager.kubeRbacProxy.image.tag | string | `"v0.8.0"` |  |
| controllerManager.openfunction.image.repository | string | `"openfunction/openfunction"` |  |
| controllerManager.openfunction.image.tag | string | `"v0.6.0"` |  |
| controllerManager.openfunction.resources.limits.cpu | string | `"500m"` |  |
| controllerManager.openfunction.resources.limits.memory | string | `"500Mi"` |  |
| controllerManager.openfunction.resources.requests.cpu | string | `"100m"` |  |
| controllerManager.openfunction.resources.requests.memory | string | `"20Mi"` |  |
| controllerManager.replicas | int | `1` |  |
| kubernetesClusterDomain | string | `"cluster.local"` |  |
| managerConfig.controllerManagerConfigYaml.health.healthProbeBindAddress | string | `":8081"` |  |
| managerConfig.controllerManagerConfigYaml.leaderElection.leaderElect | bool | `true` |  |
| managerConfig.controllerManagerConfigYaml.leaderElection.resourceName | string | `"79f0111e.openfunction.io"` |  |
| managerConfig.controllerManagerConfigYaml.metrics.bindAddress | string | `"127.0.0.1:8080"` |  |
| managerConfig.controllerManagerConfigYaml.webhook.port | int | `9443` |  |
| metricsService.ports[0].name | string | `"https"` |  |
| metricsService.ports[0].port | int | `8443` |  |
| metricsService.ports[0].targetPort | string | `"https"` |  |
| metricsService.type | string | `"ClusterIP"` |  |
| webhookService.ports[0].port | int | `443` |  |
| webhookService.ports[0].targetPort | int | `9443` |  |
| webhookService.type | string | `"ClusterIP"` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.10.0](https://github.com/norwoodj/helm-docs/releases/v1.10.0)
