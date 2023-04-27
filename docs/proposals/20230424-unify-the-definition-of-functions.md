# Unify the definition of sync and async functions

## Motivation
Currently, we use `runtime: knative` and `runtime: async` to distinguish sync and async functions which increases the learning curve. Actually the difference between sync and async functions is trigger type:
- Sync functions are triggered by `HTTP` event, which are defined by specifying `runtime: knative`
- Async functions are triggered by events from components of `Dapr bindings` or `Dapr pubsub`. `runtime: async` and `inputs` have to be used together to specify triggers for async functions

So we can use `triggers` to replace `runtime` and `inputs`.

## Goals
Use `triggers` to unify the definition of sync and async functions.

## Proposal

### OpenFunction

- Use `spec.serving.triggers` to replace the original `spec.serving.inputs` field.
- The original `spec.serving.runtime` field will be replaced by `spec.serving.trigger.type`.
- The `spec.serving.route` field will be moved to `spec.serving.trigger.params.route` (only required for functions with http trigger)
- The `spec.serving.triggers` definition is identical or similar to the origial `spec.serving.inputs`
- The `spec.serving.triggers.type` of sync functions is `http`, `http.knative` or `http.keda`. `http` is default to `http.knative` for now.
- The `spec.serving.triggers.params.name` and `spec.serving.triggers.params.components` is not used for sync functions
- User can only specify one trigger if the `spec.serving.trigger.type` is `http`/`http.knative`/`http.keda`
- Supported `spec.serving.trigger.type` for async functions is the same as Dapr pubsub and binding such as `pubsub.rabbitmq` and `binding.kafka`
- Currently the triggers types include `http` and Dapr `binding` and `pubsub`, we may add other trigger types in the future such as `S3 events trigger` or `github event trigger`
- Multiple triggers can be specified together if they're non-http triggers
- The API version can be upgrade from v1beta1 to v1beta2
- We may add `spec.serving.inputs` back in the future for input bindings that're not triggers, or replace the `spec.serving.inputs` and `spec.serving.outputs` with a unified `ioBinding` field.

Add a `Trigger` type in `OpenFunction/apis/core/v1beta1/serving_types.go`:
```go
type Trigger struct {
  Type string `json:"type"`
  Metadata *DaprIO `json:"metadata,omitempty"`
  Route *RouteImpl `json:"route,omitempty"`
  // import https://github.com/shipwright-io/build/blob/main/pkg/apis/build/v1alpha1/parameter.go
  Params []*ParamValue `json:"params,omitempty"`
}
```

Remove the `Inputs` and add the `Triggers` instead in `OpenFunction/apis/core/v1beta1/function_types.go`:
```go
type ServingImpl struct {
  Triggers []*Trigger `json:"inputs"`
  // Function outputs from Dapr components including binding, pubsub
  // +optional
  Outputs []*DaprIO `json:"outputs,omitempty"`
}
```

```yaml=
apiVersion: core.openfunction.io/v1beta1
kind: Function
metadata:
  name: logs-async-handler
spec:
  version: "v2.0.0"
  image: openfunctiondev/logs-async-handler:v1
  imageCredentials:
    name: push-secret
  build:
    builder: openfunction/builder-go:latest
    env:
      FUNC_NAME: "LogsHandler"
      FUNC_CLEAR_SOURCE: "true"
#     # Use FUNC_GOPROXY to set the goproxy if failed to fetch go modules
#     FUNC_GOPROXY: "https://goproxy.cn"
    srcRepo:
      url: "https://github.com/OpenFunction/samples.git"
      sourceSubPath: "functions/async/logs-handler-function/"
      revision: "main"
  serving:
    triggers:
      # type: http 
      # type: http.knative 
      # type: http.keda
      # type: pubsub.rabbitmq
      - type: binding.kafka 
        metadata:
          name: kafka
          component: kafka-receiver
        ## route is only required for functions with http trigger
        #route:
        #  rules:
        #    - matches:
        #        - path:
        #            type: PathPrefix
        #            value: /echo
    outputs:
      - name: notify
        component: notification-manager
        operation: "post"
    bindings:
      kafka-receiver:
        type: bindings.kafka
        version: v1
        metadata:
          - name: brokers
            value: "kafka-server-kafka-brokers:9092"
          - name: authRequired
            value: "false"
          - name: publishTopic
            value: "logs"
          - name: topics
            value: "logs"
          - name: consumerGroup
            value: "logs-handler"
      notification-manager:
        type: bindings.http
        version: v1
        metadata:
          - name: url
            value: http://notification-manager-svc.kubesphere-monitoring-system.svc.cluster.local:19093/api/v2/alerts
    scaleOptions:
      keda:
        scaledObject:
          pollingInterval: 15
          minReplicaCount: 0
          maxReplicaCount: 10
          cooldownPeriod: 60
          advanced:
            horizontalPodAutoscalerConfig:
              behavior:
                scaleDown:
                  stabilizationWindowSeconds: 45
                  policies:
                  - type: Percent
                    value: 50
                    periodSeconds: 15
                scaleUp:
                  stabilizationWindowSeconds: 0
        ## The triggers definition for KEDA should be moved back to keda
        triggers:
          - type: kafka
            metadata:
              topic: logs
              bootstrapServers: kafka-server-kafka-brokers.default.svc.cluster.local:9092
              consumerGroup: logs-handler
              lagThreshold: "20"
    template:
      containers:
        - name: function # DO NOT change this
          imagePullPolicy: IfNotPresent
```


### functions-framework

We can stil use the original `Inputs` in function context which means no changes is needed for functions-frameworks.

## Action items
- [ ] Adjust OpenFunction CRD and Controller
- [ ] Add conversion between v1beta1 and v1
- [ ] Adjust samples
- [ ] Adjust docs
- [ ] Release OpenFunction v1.1