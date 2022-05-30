# tekton-pipelines

![Version: 0.30.0](https://img.shields.io/badge/Version-0.30.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.30.0](https://img.shields.io/badge/AppVersion-0.30.0-informational?style=flat-square)

A Helm chart for Tekton Pipelines on Kubernetes

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| wangyifei | <wangyifei@kubesphere.io> |  |

## Source Code

* <https://github.com/tektoncd/pipeline>

## Prerequisites

* Kubernetes cluster with RBAC (Role-Based Access Control) enabled is required
* Helm 3.4.0 or newer

## Install the Chart

Ensure Helm is initialized in your Kubernetes cluster.

For more details on initializing Helm, [read the Helm docs](https://helm.sh/docs/)

1. Add openfunction.github.io as an helm repo
    ```
    helm repo add openfunction https://openfunction.github.io/helm-charts/
    helm repo update
    ```

2. Install the tekton-pipelines chart on your cluster in the tekton-pipelines namespace:
    ```
    helm install tekton-pipelines openfunction/tekton-pipelines -n tekton-pipelines --create-namespace
    ```

## Verify installation

```
kubectl get pods -namespace tekton-pipelines
```

## Uninstall the Chart

To uninstall/delete the `tekton-pipelines` release:
```
helm uninstall tekton-pipelines -n tekton-pipelines
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| configDefaults.Example | string | `"################################\n#                              #\n#    EXAMPLE CONFIGURATION     #\n#                              #\n################################\n\n# This block is not actually functional configuration,\n# but serves to illustrate the available configuration\n# options and document them in a way that is accessible\n# to users that `kubectl edit` this config map.\n#\n# These sample configuration options may be copied out of\n# this example block and unindented to be in the data block\n# to actually change the configuration.\n\n# default-timeout-minutes contains the default number of\n# minutes to use for TaskRun and PipelineRun, if none is specified.\ndefault-timeout-minutes: \"60\"  # 60 minutes\n\n# default-service-account contains the default service account name\n# to use for TaskRun and PipelineRun, if none is specified.\ndefault-service-account: \"default\"\n\n# default-managed-by-label-value contains the default value given to the\n# \"app.kubernetes.io/managed-by\" label applied to all Pods created for\n# TaskRuns. If a user's requested TaskRun specifies another value for this\n# label, the user's request supercedes.\ndefault-managed-by-label-value: \"tekton-pipelines\"\n\n# default-pod-template contains the default pod template to use\n# TaskRun and PipelineRun, if none is specified. If a pod template\n# is specified, the default pod template is ignored.\n# default-pod-template:\n\n# default-cloud-events-sink contains the default CloudEvents sink to be\n# used for TaskRun and PipelineRun, when no sink is specified.\n# Note that right now it is still not possible to set a PipelineRun or\n# TaskRun specific sink, so the default is the only option available.\n# If no sink is specified, no CloudEvent is generated\n# default-cloud-events-sink:\n\n# default-task-run-workspace-binding contains the default workspace\n# configuration provided for any Workspaces that a Task declares\n# but that a TaskRun does not explicitly provide.\n# default-task-run-workspace-binding: |\n#   emptyDir: {}\n"` |  |
| configLeaderElection.leaseDuration | string | `"15s"` |  |
| configLeaderElection.renewDeadline | string | `"10s"` |  |
| configLeaderElection.retryPeriod | string | `"2s"` |  |
| configLogging.loglevelController | string | `"info"` |  |
| configLogging.loglevelWebhook | string | `"info"` |  |
| configLogging.zapLoggerConfig | string | `"{\n  \"level\": \"info\",\n  \"development\": false,\n  \"sampling\": {\n    \"initial\": 100,\n    \"thereafter\": 100\n  },\n  \"outputPaths\": [\"stdout\"],\n  \"errorOutputPaths\": [\"stderr\"],\n  \"encoding\": \"json\",\n  \"encoderConfig\": {\n    \"timeKey\": \"ts\",\n    \"levelKey\": \"level\",\n    \"nameKey\": \"logger\",\n    \"callerKey\": \"caller\",\n    \"messageKey\": \"msg\",\n    \"stacktraceKey\": \"stacktrace\",\n    \"lineEnding\": \"\",\n    \"levelEncoder\": \"\",\n    \"timeEncoder\": \"iso8601\",\n    \"durationEncoder\": \"\",\n    \"callerEncoder\": \"\"\n  }\n}\n"` |  |
| configObservability.Example | string | `"################################\n#                              #\n#    EXAMPLE CONFIGURATION     #\n#                              #\n################################\n\n# This block is not actually functional configuration,\n# but serves to illustrate the available configuration\n# options and document them in a way that is accessible\n# to users that `kubectl edit` this config map.\n#\n# These sample configuration options may be copied out of\n# this example block and unindented to be in the data block\n# to actually change the configuration.\n\n# metrics.backend-destination field specifies the system metrics destination.\n# It supports either prometheus (the default) or stackdriver.\n# Note: Using Stackdriver will incur additional charges.\nmetrics.backend-destination: prometheus\n\n# metrics.stackdriver-project-id field specifies the Stackdriver project ID. This\n# field is optional. When running on GCE, application default credentials will be\n# used and metrics will be sent to the cluster's project if this field is\n# not provided.\nmetrics.stackdriver-project-id: \"<your stackdriver project id>\"\n\n# metrics.allow-stackdriver-custom-metrics indicates whether it is allowed\n# to send metrics to Stackdriver using \"global\" resource type and custom\n# metric type. Setting this flag to \"true\" could cause extra Stackdriver\n# charge.  If metrics.backend-destination is not Stackdriver, this is\n# ignored.\nmetrics.allow-stackdriver-custom-metrics: \"false\"\nmetrics.taskrun.level: \"taskrun\"\nmetrics.taskrun.duration-type: \"histogram\"\nmetrics.pipelinerun.level: \"pipelinerun\"\nmetrics.pipelinerun.duration-type: \"histogram\"\n"` |  |
| controller.ports[0].name | string | `"http-metrics"` |  |
| controller.ports[0].port | int | `9090` |  |
| controller.ports[0].protocol | string | `"TCP"` |  |
| controller.ports[0].targetPort | int | `9090` |  |
| controller.ports[1].name | string | `"http-profiling"` |  |
| controller.ports[1].port | int | `8008` |  |
| controller.ports[1].targetPort | int | `8008` |  |
| controller.ports[2].name | string | `"probes"` |  |
| controller.ports[2].port | int | `8080` |  |
| controller.ports[2].targetPort | int | `0` |  |
| controller.replicas | int | `1` |  |
| controller.tektonPipelinesController.entrypointImage.digest | string | `"sha256:34ee7658bb8a657584e1ada8e84121758cc5d067c1f0740873d614d07423886f"` |  |
| controller.tektonPipelinesController.entrypointImage.repository | string | `"gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/entrypoint"` |  |
| controller.tektonPipelinesController.entrypointImage.tag | string | `"v0.30.0"` |  |
| controller.tektonPipelinesController.gitImage.digest | string | `"sha256:3637bac1e233696a3671155c77de9ed8e02cacbec454d314125a5f1f458effa3"` |  |
| controller.tektonPipelinesController.gitImage.repository | string | `"gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init"` |  |
| controller.tektonPipelinesController.gitImage.tag | string | `"v0.30.0"` |  |
| controller.tektonPipelinesController.gsutilImage.digest | string | `"sha256:27b2c22bf259d9bc1a291e99c63791ba0c27a04d2db0a43241ba0f1f20f4067f"` |  |
| controller.tektonPipelinesController.gsutilImage.repository | string | `"gcr.io/google.com/cloudsdktool/cloud-sdk"` |  |
| controller.tektonPipelinesController.gsutilImage.tag | string | `nil` |  |
| controller.tektonPipelinesController.image.digest | string | `"sha256:ecb7567431d9c2b899be7b04cd5a72722655e36fd58f69ed695e469daab9009b"` |  |
| controller.tektonPipelinesController.image.repository | string | `"gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/controller"` |  |
| controller.tektonPipelinesController.image.tag | string | `"v0.30.0"` |  |
| controller.tektonPipelinesController.imagedigestExporterImage.digest | string | `"sha256:2a6dec9e6d66b2198d9bc3bcf1f03a662e4eb274b66563c5d499e9f29dadcc10"` |  |
| controller.tektonPipelinesController.imagedigestExporterImage.repository | string | `"gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/imagedigestexporter"` |  |
| controller.tektonPipelinesController.imagedigestExporterImage.tag | string | `"v0.30.0"` |  |
| controller.tektonPipelinesController.kubeconfigWriterImage.digest | string | `"sha256:5292621d97834592c983a341e6e8759a8437dd208448a0226459c91e7b273f8c"` |  |
| controller.tektonPipelinesController.kubeconfigWriterImage.repository | string | `"gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/kubeconfigwriter"` |  |
| controller.tektonPipelinesController.kubeconfigWriterImage.tag | string | `"v0.30.0"` |  |
| controller.tektonPipelinesController.nopImage.digest | string | `"sha256:89cb4d5572372c7ade6b20b59bf35dc9dcd5e4cde2fa77f14888d4f7059cd767"` |  |
| controller.tektonPipelinesController.nopImage.repository | string | `"gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/nop"` |  |
| controller.tektonPipelinesController.nopImage.tag | string | `"v0.30.0"` |  |
| controller.tektonPipelinesController.prImage.digest | string | `"sha256:d321d1888a203be9fab57aa528bcf378da6984778c38f015c0a9287fc489602f"` |  |
| controller.tektonPipelinesController.prImage.repository | string | `"gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/pullrequest-init"` |  |
| controller.tektonPipelinesController.prImage.tag | string | `"v0.30.0"` |  |
| controller.tektonPipelinesController.shellImage.digest | string | `"sha256:cfdc553400d41b47fd231b028403469811fcdbc0e69d66ea8030c5a0b5fbac2b"` |  |
| controller.tektonPipelinesController.shellImage.repository | string | `"ghcr.io/distroless/busybox"` |  |
| controller.tektonPipelinesController.shellImage.tag | string | `nil` |  |
| controller.tektonPipelinesController.shellImageWin.digest | string | `"sha256:b6d5ff841b78bdf2dfed7550000fd4f3437385b8fa686ec0f010be24777654d6"` |  |
| controller.tektonPipelinesController.shellImageWin.repository | string | `"mcr.microsoft.com/powershell:nanoserver"` |  |
| controller.tektonPipelinesController.shellImageWin.tag | string | `nil` |  |
| controller.type | string | `"ClusterIP"` |  |
| featureFlags.disableAffinityAssistant | string | `"false"` |  |
| featureFlags.disableCredsInit | string | `"false"` |  |
| featureFlags.disableHomeEnvOverwrite | string | `"true"` |  |
| featureFlags.disableWorkingDirectoryOverwrite | string | `"true"` |  |
| featureFlags.enableApiFields | string | `"stable"` |  |
| featureFlags.enableCustomTasks | string | `"false"` |  |
| featureFlags.enableTektonOciBundles | string | `"false"` |  |
| featureFlags.requireGitSshSecretKnownHosts | string | `"false"` |  |
| featureFlags.runningInEnvironmentWithInjectedSidecars | string | `"true"` |  |
| featureFlags.scopeWhenExpressionsToTask | string | `"false"` |  |
| kubernetesClusterDomain | string | `"cluster.local"` |  |
| pipelinesInfo.version | string | `"v0.30.0"` |  |
| webhook.ports[0].name | string | `"http-metrics"` |  |
| webhook.ports[0].port | int | `9090` |  |
| webhook.ports[0].targetPort | int | `9090` |  |
| webhook.ports[1].name | string | `"http-profiling"` |  |
| webhook.ports[1].port | int | `8008` |  |
| webhook.ports[1].targetPort | int | `8008` |  |
| webhook.ports[2].name | string | `"https-webhook"` |  |
| webhook.ports[2].port | int | `443` |  |
| webhook.ports[2].targetPort | int | `8443` |  |
| webhook.ports[3].name | string | `"probes"` |  |
| webhook.ports[3].port | int | `8080` |  |
| webhook.ports[3].targetPort | int | `0` |  |
| webhook.replicas | int | `1` |  |
| webhook.type | string | `"ClusterIP"` |  |
| webhook.webhook.image.digest | string | `"sha256:b93422365865e7b6fbe96e92cac7494626257165021fa36f71fae22bdfbd3e6e"` |  |
| webhook.webhook.image.repository | string | `"gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/webhook"` |  |
| webhook.webhook.image.tag | string | `"v0.30.0"` |  |
| webhook.webhook.resources.limits.cpu | string | `"500m"` |  |
| webhook.webhook.resources.limits.memory | string | `"500Mi"` |  |
| webhook.webhook.resources.requests.cpu | string | `"100m"` |  |
| webhook.webhook.resources.requests.memory | string | `"100Mi"` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.10.0](https://github.com/norwoodj/helm-docs/releases/v1.10.0)
