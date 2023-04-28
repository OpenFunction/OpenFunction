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
- There have two triggers, httpTrigger and daprTrigger.
- The original `spec.serving.runtime` field will be deprecated.
- The `spec.serving.route` field will be moved to `spec.serving.trigger.http` (only required for functions with http trigger)
- Multiple triggers can be specified together if they're non-http triggers
- The API version can be upgrade from v1beta1 to v1beta2

Add a `Triggers` type in `OpenFunction/apis/core/v1beta1/serving_types.go`:
```go
type Triggers struct {
  Http *HttpTrigger `json:"http,omitempty"`
  Dapr []*DaprTrigger `json:"dapr,omitempty"`
}
```

> A `TimerTrigger` will support later.
> ```go
> type TimerTrigger struct {
>   Scheduler string `json:"scheduler"`
>   Inputs []*Input `json:"inputs,omitempty"`
> }
> ```

The `HttpTrigger` is defined as follows:
```go
type HttpTrigger struct {
  // The port on which the function will be invoked
  Port *int32 `json:"port,omitempty"`
  // Information needed to make HTTPRoute.
  // Will attempt to make HTTPRoute using the default Gateway resource if Route is nil.
  //
  // +optional
  Route *RouteImpl `json:"route,omitempty"`
  Inputs []*Input `json:"inputs,omitempty"`
}
```

> The `port` is moved from `.spec`.

The `DaprTrigger` are triggers that triggered by dapr binding events or topic events, defined as follows:
```go
type DaprTrigger struct {
  *DaprComponent `json:"_inline"`
  Inputs []*Input `json:"inputs,omitempty"`
}

type DaprComponent struct {
  Type string `json:"type,omitempty"`
  Name string `json:"name"`
  Topic string `json:"topic,omitempty"`
}
```

All triggers can define some inputs, when the function is triggered, function can get data from these inputs.
Currently, it only supports dapr state store.
The `Input` define as follows:
```go
type Input struct {
  Dapr DaprComponent `json:"dapr,omitempty"`
}
```

Rename `plugin` to `hook`, and move the hook config to `spec.serving`.
There are global hooks and private hooks, the global hooks define in the config of `OpenFunction Controller`, the private hooks
define in the `Function` cr. 

By default, all hooks will be executed, user can controll this logic by the `policy` field. There have two policy to determine the relation of global hooks and private hooks.
- `Append` All hooks will be executed, the private pre hooks will execute after the global pre hooks , and the private post hooks will execute before the global post hooks. this is the default policy.
- `Override` Only execute the private hooks. 

The `hook` config define as follows:
```go
type HookConfig struct {
  Pre   []string `yaml:"pre,omitempty"`
  Post  []string `yaml:"post,omitempty"`
  Policy string `yaml:"policy,omitempty"`
}
```

The `plugins.tracing` will move from `annotation` to `spec.serving`.

The `Function` yaml will like this:
```yaml=
apiVersion: core.openfunction.io/v1beta2
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
      # http:
      #   port: "8080"
      #   route:
      #     rules:
      #       - matches:
      #           - path:
      #               type: PathPrefix
      #               value: /echo
      #   inputs:
      #     - type: binding.kafka 
      #       name: kafka-input
      #     - type: pubsub.rocketmq
      #       name: rocketmq-input
      #       topic: sample
      #     - type: state.redis
      #       name: redis-input 
      dapr:
        - type: binding.kafka 
          name: kafka-receiver
        - type: pubsub.rocketmq
          name: rocketmq-server
          topic: sample
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
    hooks:
      pre:
        - pre-hook1
        - pre-hook2
      post:
        - post-hook2
        - post-hook1
    tracing:
      enabled: true
      provider:
        name: opentelemetry
        exporter:
          name: jaeger
          endpoint: "http://localhost:14268/api/traces"
      tags:
        func: sample-binding
        layer: faas
      baggage:
        key: opentelemetry
        value: v1.23.0
```

### functions-framework

The controller need to create two function context, one is for v1beta1, and other is for v1beta2, the framework need to get the context by the API version.

## Action items
- [ ] Adjust OpenFunction CRD and Controller
- [ ] Add conversion between v1beta1 and v1
- [ ] Split builder to separate repository
- [ ] Adjust samples
- [ ] Adjust docs
- [ ] Release OpenFunction v1.1