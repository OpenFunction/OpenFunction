## Motivation

We need to add features that control the number of instances of functions for better security and stability. In short, a unified definition of min/max replicas is needed.

## Goal
Currently, we've pod [template](https://github.com/OpenFunction/OpenFunction/blob/main/apis/core/v1alpha2/serving_types.go#L140) and [keda scaled object](https://github.com/OpenFunction/OpenFunction/blob/main/apis/core/v1alpha2/serving_types.go#L140) defined and users can control min/max replica by setting these params. It works, but it's not graceful.We might need a more direct way to specify the min/max replicas.
## Proposal
Refer to [keda](https://keda.sh/docs/2.5/concepts/scaling-deployments/#scaledobject-spec)，We can define the relevant resources in a knative way, controlling the maximum and minimum quantities through the `spec.serving.minReplicaCount` and `spec.serving.maxReplicaCount` fields.

We can implement the number of control functions in the following way.
```yaml=
minReplicaCount:  1                                
maxReplicaCount:  100
```
As shown below
```yaml=
apiVersion: core.openfunction.io/v1alpha2
kind: Function
metadata:
  name: function-sample
spec:
  version: "v1.0.0"
  image: "openfunctiondev/sample-go-func:latest"
  imageCredentials:
    name: push-secret
  build:
    timeout: 5m
    builder: openfunction/builder:v1
    env:
      FUNC_NAME: "HelloWorld"
      FUNC_TYPE: "http"
    srcRepo:
      url: "https://github.com/OpenFunction/samples.git"
      sourceSubPath: "latest/functions/Knative/hello-world-go"
  serving:
    scaleOptions:
      minReplicaCount: 10
      maxReplicaCount: 100 
    timeout: 1m
    runtime: Knative
    template:
      containers:
        - name: function
          imagePullPolicy: Always

```
A function can only use one runtime, so we need to set different parameters depending on the runtime,like this:

```Fyaml
// for keda
scaleOptions:
  keda:
    minReplicaCount:  10
    maxReplicaCount:  100

// for knative
scaleOptions:
  knative:
    minReplicaCount:  10
    maxReplicaCount:  100
```

For the OpenFuncAsync runtime is defined as below:

```yaml
  serving:
    runtime: "OpenFuncAsync"
    bindings:
      dapr:
        annotations:
          dapr.io/log-level: "debug"
        components:
          kafka-receiver:
            type: bindings.kafka
            version: v1
            metadata:
              - name: brokers
                value: "kafka-logs-receiver-kafka-brokers:9092"
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
      minReplicaCount: 1
      maxReplicaCount:  10
      keda:
        scaledObject:
          pollingInterval: 15
          minReplicaCount: 0 // will overwrite the min/max params above
          maxReplicaCount: 10 // will overwrite the min/max params above
          cooldownPeriod: 30
          triggers:
            - type: kafka
              metadata:
                topic: logs
                bootstrapServers: kafka-logs-receiver-kafka-brokers.default.svc.cluster.local:9092
                consumerGroup: logs-handler
                lagThreshold: "10"
    openFuncAsync:
      inputs:
        - name: kafka
          component: kafka-receiver
          type: bindings
      outputs:
        - name: notify
          type: bindings
          component: notification-manager
          operation: "post"
```

