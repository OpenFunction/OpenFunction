# Overview

**OpenFunction Events** is the event management framework for OpenFunction.

## Features

- Support for triggering target functions by synchronous and asynchronous calls
- User-defined trigger judgment logic
- The components of OpenFunction Events can be driven by OpenFunction itself

# Concepts

## Architecture

![openfunction events](../images/OpenFunction-events-architecture.png)

## EventSource

Represents the producer of an event, such as a Kafka service, an object storage service, and can also be a function.

The **EventSource** contains a description of these event producers  and is also responsible for directing where the events they generate should go.

### Available:

- Kafka
- Cron (scheduler)
- Redis

## EventBus(ClusterEventBus)

The event bus is responsible for aggregating events and persisting them.

The **EventBus** contains a description of the event bus backend service (usually a MQ server such as NATS, Kafka, etc.) and then provides these configurations for EventSource and Trigger.

EventBus will take care of event bus adaptation for namespaced scope by default, while we provide an event bus adapter **ClusterEventBus** for clustered scope. **ClusterEventBus** will take effect when other components do not find an EventBus under the namespace.

### Available:

- NATS Streaming

## Trigger

An abstraction of the purpose of the event, such as what needs to be done when a message is received.

The **Trigger** contains the user's description of the purpose of the event, which guides the trigger on which event sources it should fetch the event from and subsequently determine whether to trigger the target function according to the given conditions

# Getting Started

## Sample 1: Event source trigger synchronization function

In this sample, the event source is a Kafka server and the target function is a Knative Service. We will define an EventSource for synchronous invocation, whose role is to use the event source (Kafka server) as an input bindings of function (Knative service) , and when the event source generates an event, it will invoke the function and get a synchronous return through the `EventSource.Sink` configuration.

### Prerequisites

- A Knative runtime function (target function)

  You can refer to [Sample Function Go](https://github.com/OpenFunction/samples/tree/main/functions/Knative/hello-world-go) to create a Knative runtime function.

  Here I assume that the name of this function (Knative Service) is `function-sample-serving-ksvc` . You should be able to see this value with the `kubectl get ksvc` command

- A Kafka server (event source)

  You can refer to [Setting up a Kafka in Kubernetes](https://github.com/dapr/quickstarts/tree/master/bindings#setting-up-a-kafka-in-kubernetes) to deploy a Kafka server.

  Here I assume that the access address of this Kafka server is `dapr-kafka.kafka:9092` .

Create an EventSource configuration `eventsource-sink.yaml` :

> We define an event source named "my-eventsource" and mark the events producered by the specified Kafka server as "sample-one".
>
> `EventSource.Sink` references the target function (Knative Service) we created above.

```yaml
apiVersion: events.openfunction.io/v1alpha1
kind: EventSource
metadata:
  name: my-eventsource
spec:
  kafka:
    sample-one:
      version: v1
      type: bindings.kafka
      metadata:
        - name: brokers
          value: dapr-kafka.kafka:9092
        - name: topics
          value: sample
        - name: consumerGroup
          value: group1
        - name: publishTopic
          value: sample
        - name: authRequired
          value: "false"
  sink:
    ref:
      apiVersion: serving.knative.dev/v1
      kind: Service
      name: function-sample-serving-ksvc
      namespace: default
```

Apply it :

```shell
kubectl apply -f eventsource-sink.yaml
```

You will observe the following changes:

> In the synchronous sample, the workflow of the EventSource controller is as follows :
>
> 1. Create the EventSource CR called "my-eventsource"
> 2. Create a Dapr Component called "eventsource-my-eventsource-sample-one" for associating the event source
> 3. Create a Dapr Component called "eventsource-sink-my-eventsource" for associating the target function
> 4. Create a Deployments called "eventsource-my-eventsource-kafka-sample-one" for processing events

```shell
~# kubectl get eventsources.events.openfunction.io
NAME             EVENTBUS   SINK
my-eventsource              function-sample-serving-ksvc

~# kubectl get components
NAME                                          AGE
eventsource-my-eventsource-kafka-sample-one   3m45s
eventsource-sink-my-eventsource               3m45s

~# kubectl get deployments.apps
NAME                                           READY   UP-TO-DATE   AVAILABLE   AGE
eventsource-my-eventsource-kafka-sample-one    1/1     1            1           4m14s
```

At this point we see that the target function is not started (because there is no event input) and we can create some events to trigger the function.

Create an event producer `events-producer.yaml` :

> You can choose the producer image `openfunctiondev/events-producer:latest`, which will publish events to the event source at a rate of one event per 5 seconds.
>
> And if you are using this image as an event producer, then you need to set an environment variable that sets the value of `TARGET_NAME` to the name of the dapr component of the EventSource deployed above, i.e. `eventsource-my-eventsource-kafka-sample-one` .

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: events-producer
  labels:
    app: eventsproducer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: eventsproducer
  template:
    metadata:
      labels:
        app: eventsproducer
      annotations:
        dapr.io/enabled: "true"
        dapr.io/app-id: "events-producer"
        dapr.io/log-as-json: "true"
    spec:
      containers:
        - name: producer
          image: openfunctiondev/events-producer:latest
          imagePullPolicy: Always
          env:
            - name: TARGET_NAME
              value: "eventsource-my-eventsource-kafka-sample-one"
```

Apply it :

```shell
kubectl apply -f events-producer.yaml
```

We can observe the change in Pod resources :

```shell
~# kubectl get po --watch
NAME                                                           READY   STATUS              RESTARTS   AGE
events-producer-86b49654-8stj6                                 0/2     ContainerCreating   0          1s
eventsource-my-eventsource-kafka-sample-one-789b767c79-45bdf   2/2     Running             0          23m
events-producer-86b49654-8stj6                                 0/2     ContainerCreating   0          1s
events-producer-86b49654-8stj6                                 1/2     Running             0          5s
events-producer-86b49654-8stj6                                 2/2     Running             0          8s
function-sample-serving-ksvc-v100-deployment-5df4f559db-2nbp4   0/2     Pending             0          0s
function-sample-serving-ksvc-v100-deployment-5df4f559db-2nbp4   0/2     Pending             0          0s
function-sample-serving-ksvc-v100-deployment-5df4f559db-2nbp4   0/2     ContainerCreating   0          0s
function-sample-serving-ksvc-v100-deployment-5df4f559db-2nbp4   0/2     ContainerCreating   0          2s
function-sample-serving-ksvc-v100-deployment-5df4f559db-2nbp4   1/2     Running             0          4s
function-sample-serving-ksvc-v100-deployment-5df4f559db-2nbp4   1/2     Running             0          4s
function-sample-serving-ksvc-v100-deployment-5df4f559db-2nbp4   2/2     Running             0          4s
```

## Sample 2: Use of event bus and triggers

### Prerequisites

- A Knative runtime function (target function)

  You can refer to [Sample Function Go](https://github.com/OpenFunction/samples/tree/main/functions/Knative/hello-world-go) to create a Knative runtime function.

  Here I assume that the name of this function (Knative Service) is `function-sample-serving-ksvc` . You should be able to see this value with the `kubectl get ksvc` command

- A Kafka server (event source)

  You can refer to [Setting up a Kafka in Kubernetes](https://github.com/dapr/quickstarts/tree/master/bindings#setting-up-a-kafka-in-kubernetes) to deploy a Kafka server.

  Here I assume that the access address of this Kafka server is `dapr-kafka.kafka:9092` .

- A Nats streaming server (event bus)

  You can refer to [Deploy NATS on Kubernetes with Helm Charts](https://nats-io.github.io/k8s/) to deploy a Nats streaming server.

  Here I assume that the access address of this NATS Streaming server is `nats://nats.default:4222` and the cluster ID is `stan` .

Create an EventBus configuration `eventbus-default.yaml` ：

```yaml
apiVersion: events.openfunction.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  nats:
    version: v1
    type: pubsub.natsstreaming
    metadata:
      - name: natsURL
        value: "nats://nats.default:4222"
      - name: natsStreamingClusterID
        value: "stan"
      - name: subscriptionType
        value: queue
```

Create an EventSource configuration `eventsource-eventbus.yaml` ：

>  We need to set the name of the event bus via `eventBus`

```yaml
apiVersion: events.openfunction.io/v1alpha1
kind: EventSource
metadata:
  name: my-eventsource
spec:
  eventBus: "default"
  kafka:
    sample-two:
      version: v1
      type: bindings.kafka
      metadata:
        - name: brokers
          value: dapr-kafka.kafka:9092
        - name: topics
          value: sample
        - name: consumerGroup
          value: group1
        - name: publishTopic
          value: sample
        - name: authRequired
          value: "false"
```

Apply them :

```shell
kubectl apply -f eventbus-default.yaml
kubectl apply -f eventsource-eventbus.yaml
```

You will observe the following changes:

> In the case of using the event bus, the workflow of the EventSource controller is as follows :
>
> 1. Create EventSource CR called "my-eventsource"
> 2. Retrieve and reorganize the configuration of the EventBus (used to pass in the Deployments in step 5), including:
     >    1. The EventBus name ("default" in this sample)
>    2. The name of the Dapr Component associated with the EventBus ("eventsource-eventbus-my-eventsource" in this sample)
> 3. Create a Dapr Component called "eventsource-eventbus-my-eventsource" for associating the event bus
> 4. Create a Dapr Component called "eventsource-my-eventsource-kafka-sample-two" for associating the event source
> 5. Create a Deployments called "eventsource-my-eventsource-kafka-sample-two" for processing events

```shell
~# kubectl get eventsources.events.openfunction.io
NAME             EVENTBUS   SINK
my-eventsource   default

~# kubectl get eventbus.events.openfunction.io
NAME      AGE
default   10m

~# kubectl get components
NAME                                          AGE
eventsource-eventbus-my-eventsource           28s
eventsource-my-eventsource-kafka-sample-two   28s

~# kubectl get deployments.apps
NAME                                           READY   UP-TO-DATE   AVAILABLE   AGE
eventsource-my-eventsource-kafka-sample-two    1/1     1            1           4m53s
```

At this point we also need a trigger to guide what the event should do.

Create a Trigger configuration `trigger.yaml` :

> Set the event bus associated with the Trigger via `spec.eventBus` .
>
> `spec.inputs` is used to set the event input source.
>
> In `spec.subscribers` , `subscriber.condition` will perform a logical operation on `input.name` in `spec.inputs`. When the result is true, the event will be processed according to the `subscriber.sink` or `subscriber.topic` configuration. (In Development ...)
>
> Here we set up a very simple trigger that will collect events from the "default" EventBus. When it retrieves a "sample-two" event from the "my-eventsource" EventSource, it will trigger a Knative Service called "function-sample-serving-ksvc" and send the event to the "metrics" topic of the event bus at the same time.

```yaml
apiVersion: events.openfunction.io/v1alpha1
kind: Trigger
metadata:
  name: my-trigger
spec:
  eventBus: "default"
  inputs:
    - name: "input-demo"
      eventSourceName: "my-eventsource"
      eventName: "sample-two"
  subscribers:
  - condition: input-demo
    sink:
      ref:
        apiVersion: serving.knative.dev/v1
        kind: Service
        name: function-sample-serving-ksvc
        namespace: default
    topic: "metrics"
```

Apply it :

```yaml
kubectl apply -f trigger.yaml
```

You will observe the following changes :

> In the case of using the event bus, the workflow of the Trigger controller is as follows :
>
> 1. Create a Trigger CR called "my-trigger"
> 2. Retrieve and reorganize the configuration of the EventBus (used to pass in the Deployments in step 5), including:
     >    1. The EventBus name ("default" in this sample)
>    2. The name of the Dapr Component associated with the EventBus ("trigger-eventbus-my-trigger" in this sample)
> 3. Create a Dapr Component called "trigger-eventbus-my-trigger" for associating the event bus
> 4. Create a Dapr Component called "trigger-sink-my-trigger-default-function-sample-serving-ksvc" for associating the target function
> 5. Create a Deployments called "trigger-my-trigger" to handle trigger tasks

```shell
~# kubectl get triggers.events.openfunction.io
NAME         AGE
my-trigger   34m

~# kubectl get eventbus.events.openfunction.io
NAME      AGE
default   62m

~# kubectl get components
NAME                                                           AGE
trigger-eventbus-my-trigger                                    34m
trigger-sink-my-trigger-default-function-sample-serving-ksvc   34m

~# kubectl get deployments.apps
NAME                    READY   UP-TO-DATE   AVAILABLE   AGE
trigger-my-trigger      1/1     1            1           2m52s
```

At this point we see that the target function is not started (because there is no event input) and we can create some events to trigger the function.

Create an event producer `events-producer.yaml` :

> You can choose the producer image `openfunctiondev/events-producer:latest`, which will publish events to the event source at a rate of one event per 5 seconds.
>
> And if you are using this image as an event producer, then you need to set an environment variable that sets the value of `TARGET_NAME` to the name of the dapr component of the EventSource deployed above, i.e. `eventsource-my-eventsource-kafka-sample-two` .

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: events-producer
  labels:
    app: eventsproducer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: eventsproducer
  template:
    metadata:
      labels:
        app: eventsproducer
      annotations:
        dapr.io/enabled: "true"
        dapr.io/app-id: "events-producer"
        dapr.io/log-as-json: "true"
    spec:
      containers:
        - name: producer
          image: openfunctiondev/events-producer:latest
          imagePullPolicy: Always
          env:
            - name: TARGET_NAME
              value: "eventsource-my-eventsource-kafka-sample-two"
```

Apply it :

```shell
kubectl apply -f events-producer.yaml
```

We can observe the change in Pod resources :

```shell
~# kubectl get po --watch
NAME                                                           READY   STATUS              RESTARTS   AGE
events-producer-86679d99fb-4tlbt                               0/2     ContainerCreating   0          2s
eventsource-my-eventsource-kafka-sample-two-7695c6cfdd-j2g5b   2/2     Running             0          58m
trigger-my-trigger-7b799c7f7d-4ph77                            2/2     Running             0          37m
events-producer-86679d99fb-4tlbt                               0/2     ContainerCreating   0          2s
events-producer-86679d99fb-4tlbt                               1/2     Running             0          6s
events-producer-86679d99fb-4tlbt                               2/2     Running             0          10s
function-sample-serving-ksvc-v100-deployment-5df4f559db-h69xj   0/2     Pending             0          0s
function-sample-serving-ksvc-v100-deployment-5df4f559db-h69xj   0/2     Pending             0          0s
function-sample-serving-ksvc-v100-deployment-5df4f559db-h69xj   0/2     ContainerCreating   0          0s
function-sample-serving-ksvc-v100-deployment-5df4f559db-h69xj   0/2     ContainerCreating   0          2s
function-sample-serving-ksvc-v100-deployment-5df4f559db-h69xj   1/2     Running             0          3s
function-sample-serving-ksvc-v100-deployment-5df4f559db-h69xj   1/2     Running             0          4s
function-sample-serving-ksvc-v100-deployment-5df4f559db-h69xj   2/2     Running             0          4s
```

## Sample 3: Multi sources in one EventSource

We add an event source configuration to the EventSource based on [Sample 1](#Sample 1:Event source trigger synchronization function) .

Create an EventSource configuration `eventsource-multi.yaml` :

> We define an event source named "my-eventsource" and mark the events producered by the specified Kafka server as "sample-one".
>
> `EventSource.Sink` references the target function (Knative Service) we created above.

```yaml
apiVersion: events.openfunction.io/v1alpha1
kind: EventSource
metadata:
  name: my-eventsource
spec:
  kafka:
    sample-three:
      version: v1
      type: bindings.kafka
      metadata:
        - name: brokers
          value: dapr-kafka.kafka:9092
        - name: topics
          value: sample
        - name: consumerGroup
          value: group1
        - name: publishTopic
          value: sample
        - name: authRequired
          value: "false"
  cron:
    sample-three:
      version: v1
      type: bindings.cron
      metadata:
        - name: schedule
          value: "@every 5s"  
  sink:
    ref:
      apiVersion: serving.knative.dev/v1
      kind: Service
      name: function-sample-serving-ksvc
      namespace: default
```

Apply it :

```shell
kubectl apply -f eventsource-sink.yaml
```

You will observe the following changes:

```shell
~# kubectl get eventsources.events.openfunction.io
NAME             EVENTBUS   SINK
my-eventsource              function-sample-serving-ksvc

~# kubectl get components
NAME                                            AGE
eventsource-my-eventsource-cron-sample-three    96s
eventsource-my-eventsource-kafka-sample-three   96s
eventsource-sink-my-eventsource                 96s

~# kubectl get deployments.apps
NAME                                            READY   UP-TO-DATE   AVAILABLE   AGE
eventsource-my-eventsource-cron-sample-three    1/1     1            1           109s
eventsource-my-eventsource-kafka-sample-three   1/1     1            1           109s
```

The role of the `cron` event is to trigger the function in sink every 5 seconds.

## Sample 4: EventBus and ClusterEventBus

Based on [Sample 2](#Sample 2: Use of event bus and triggers), we try to use a ClusterEventBus instead of an EventBus in the namespace.

Create a ClusterEventBus configuration `clustereventbus-default.yaml` ：

```yaml
apiVersion: events.openfunction.io/v1alpha1
kind: ClusterEventBus
metadata:
  name: default
spec:
  nats:
    version: v1
    type: pubsub.natsstreaming
    metadata:
      - name: natsURL
        value: "nats://nats.default:4222"
      - name: natsStreamingClusterID
        value: "stan"
      - name: subscriptionType
        value: queue
```

Delete EventBus:

```shell
kubectl delete eventbus.events.openfunction.io default
```

And apply ClusterEventBus:

```shell
kubectl apply -f clustereventbus-default.yaml
```

You will observe the following changes:

```shell
~# kubectl get eventbus.events.openfunction.io
No resources found in default namespace.

~# kubectl get clustereventbus.events.openfunction.io
NAME      AGE
default   21s
```

If there are no other changes, you can see that the event bus is still working properly in the whole sample.


