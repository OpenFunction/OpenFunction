## Motivation

Event-driven is the core of the Function-as-a-service framework (or Serverless).

In order to make OpenFunction an event-driven function service, we need to design an effective and semantic event framework for it.

## Goals

- **Event-driven functions**. It enables the control of function wordloads triggered by events, and the number of function replications increases and decreases according to the amount of events.
- **Event aggregation processing.** It is able to perform logical operations on events and then trigger specific functional workloads by specific events (or event combinations).
- :star2: **Self-sufficiency**. The workloads of the EventSource and Trigger can be driven by OpenFunction itself.

## Proposal

Several concepts are essential in the path from the event to the workload:

- **Event Source**. Represents the producer of an event, such as a Kafka service, an object storage service, and can also be a function.
- **Event**. Instances of event source behavior, such as sending a message to the Kafka server, uploading a file to the object storage service.
- **Sink**. A sink is an Addressable resource that acts as a link between the Eventing mesh and an entity or system.
- **Trigger**. An abstraction of the purpose of the event, such as what needs to be done when a message is received.
- **Subscriber** (Function). The consumer of the event.

We therefore need to generate several corresponding CRDs to manage the above resources:

- **EventSource**. For managing event sources.
- **Trigger**. For handling the triggering of events.
- **EventBus**. To persist events and to aggregate them.

We envision two ways for an event to trigger a function:

- **Sink**

  The event is sent directly to an addressable resource receiver (the so-called Sink), which is responsible for driving the function start to respond and process the event.

  In OpenFunction, we can refer to the following resources as entities of Sink:
    - [Knative Sink](https://knative.dev/docs/eventing/sink/) (Not supported at this time)
    - [Knative Service](https://knative.dev/docs/serving/services/)
    - [KEDA HTTP](https://github.com/kedacore/http-add-on)

- **Trigger**

  Send events to the event bus. In this way, the triggers connected to the event bus aggregate the events and are subsequently responsible for driving the function to react to those events that satisfy the filtering rules.

  In OpenFunction, Trigger can control events to the following entities:

    - Sink as described above
    - Process the event and send it back to the event bus as a new event
    - OpenFuncAsync runtime driven by KEDA

### EventSource

In practice, events from different event sources differ in format and content. Passing the message directly to the consumer increases the cost of processing the event for the consumer.

So an additional step is needed to handle events from the event source and convert the events into a uniform format. Currently CloudEvents has become the mainstream cloud-native event format specification, and the [CloudEvents v1.0.1](https://github.com/cloudevents/spec/blob/v1.0.1/spec.md) specification will be followed here for event format processing.

Goals of the EventSource:

1. Adapt event sources (especially for event sources that do not support persistence)
2. Convert the event format — use CloudEvents format uniformly
3. Sending events to Sink or EventBus

It will be implemented based on Dapr's Pub/Sub Component. The current list of supported Dapr Pub/Sub Components can be found at [Pub/sub brokers component specs](https://docs.dapr.io/reference/components-reference/supported-pubsub/). As you can see, the current Dapr Pub/Sub support is not rich enough, therefore we need to develop new extensions by ourselves if we adopt this approach.

The control flow of the EventSource controller is as follows:

- When using Sink

  ![sync flow](https://i.imgur.com/vG5zE67.png)

    1. Watches the EventSource CRD
    2. Reconcile an EventSource deployments for listening to events from the event source and formatting the events (based on the Dapr Pub/Sub Component)
    3. When using Knative runtime, EventSource deployments send events directly to Knative Service, which are responsible for driving the function to respond to the event
    4. When using OpenFuncAsync runtime, EventSource deployments send events to KEDA HTTP, and KEDA is responsible for driving the function to respond to the event

- When using Trigger:
    1. Watches the EventSource CRD
    2. Reconcile an EventSource deployments for listening to events from the event source and formatting the events (based on the Dapr Pub/Sub Component)
    3. Sending events to the EventBus

An example CRD for EventSource is as follows.

```yaml
apiVersion: events.openfunction.io/v1alpha1
kind: EventSource
metadata:
  name: kafka
spec:
  kafka:
    example:
      # kafka broker url
      url: kafka.argo-events:9092
      # name of the kafka topic
      topic: topic-2
      # jsonBody specifies that all event body payload coming from this
      # source will be JSON
      jsonBody: true
      # partition id
      # optional backoff time for connection retries.
      # if not provided, default connection backoff time will be used.
      connectionBackoff:
        # duration in nanoseconds, or strings like "3s", "2m". following value is 10 seconds
        duration: 10s
        # how many backoffs
        steps: 5
        # factor to increase on each step.
        # setting factor > 1 makes backoff exponential.
        factor: 2
        jitter: 0.2
      consumerGroup:
        groupName: test-group
        oldest: false
        rebalanceStrategy: range
  sink:
    ref:
      apiVersion: serving.knative.dev/v1
      kind: Service
      name: event-display
      namespace: default
```

### Trigger

The simplest approach to an event-driven framework is to associate event sources and consumers directly through triggers. But in order to aggregate events, we need to introduce an event bus.

#### EventBus

We will design a CRD for EventBus that will generate a generic template for a Dapr Component based on a specific implementation of the event bus (usually a messaging server, such as Kafka, Nats, etc.).

```yaml
apiVersion: events.openfunction.io/v1alpha1
kind: EventBus
metadata:
  name: default
spec:
  nats:
    native:
      natsURL: "nats://localhost:4222"
      subscriptionType: topic
      natsStreamingClusterID: "clusterId"
```

This Dapr Component template will help those Triggers connected to the EventBus by allowing them to reuse this information and generate independent Dapr Component instances.

#### Filter

Also, we need to combine CloudEvents' Spec and add filters to `Trigger` for filtering events based on the given conditions. When the filter condition is met, the event will be sent to the subscriber, which means the trigger was successful.

Goals of the Trigger Filter.

1. Filter events according to filtering rules (multiple events can exist)
2. Events that do not match the filtering rules will be discarded
3. Events that match the filter rules are sent to another topic in the event bus that is relevant to the subscriber

![async flow](https://i.imgur.com/SjKVvhC.png)

In order to handle the case that subscribers are unable to handle events, it is also necessary to add **Dead Letter Queues** to handle these events that cannot reach their targets. In Knative runtime, you can refer to the [Event delivery](https://knative.dev/docs/eventing/event-delivery/) document to configure the dead letter queue; in OpenFuncAsync runtime, you can implement the dead letter queue by using the subscriber's PubSub and creating a new topic.

#### Controller

An example of a CRD for Trigger is as follows.

> `spec.eventSources` defines the event sources associated with a Trigger.
>
> `spec.subscribers` defines the subscribers associated with a Trigger.
>
> `spec.subscribers.condition` is the Trigger's filter, which can support expressions such as "!" (not), "&&" (and), "||" (or) logical operations.
>
> The `spec.subscribers.ref` defines the subscriber, such as Sink or Service for Knative, or Dapr Component for OpenFuncAsync.
>
> The `spec.subscribers.deadLetterSink` defines the dead letter queue, the content is the same as the `spec.subscribers.ref`.

```yaml
apiVersion: events.openfunction.io/v1alpha1
kind: Trigger
metadata:
  name: func1
spec:
  eventSources:
    - name: mqtt
      eventSourceName: mqtt
      eventName: example
      filters:
        data:
          - path: body.value
            type: number
            comparator: ">"
            value:
              - "50.0"
    - name: kafka
      eventSourceName: kafka
      eventName: example
      filters:
        data:
          - path: bucket
            type: string
            value:
              - func1-input1
              - func1-input2
  subscribers:
    - condition: kafka
      ref:
        apiVersion: serving.knative.dev/v1
        kind: Service
        name: knative-func1
        namespace: default
      deadLetterSink:
        ref:
          apiVersion: serving.knative.dev/v1
          kind: Service
          name: knative-func2
          namespace: default
    - condition: mqtt
      ref:
        apiVersion: dapr.io/v1alpha1
        kind: Component
        name: func2-kafka-input-binding
        namespace: default
      deadLetterSink:
        ref:
          apiVersion: dapr.io/v1alpha1
          kind: Component
          name: dead-letter-kafka-pubsub
          namespace: default 
    - condition: mqtt || kafka
      ref:
        apiVersion: dapr.io/v1alpha1
        kind: Component
        name: func3-kafka-input-binding
        namespace: default   
```

> Some of the features of Trigger, such as retry, event order, filtering rules, etc., I think can be borrowed from the design pattern of [More About Sensors And Triggers](https://argoproj.github.io/argo-events/sensors/more-about-sensors-and-triggers/).

The Controller flow of Trigger is illustrated as follows:

As mentioned above, the Trigger will create an independent Dapr Component based on the EventBus CRD information, associated with a specific implementation of the event bus (usually a messaging server, such as Kafka, Nats, etc.). After that it will do:

1. Collecting events from the event bus
2. Do what Trigger Filter would do
3. When using OpenFuncAsync runtime, KEDA will fetch the events from the specified topic of the event bus to drive the function to respond to the events
4. When using Knative runtime, the Knative Service will receive events from the event bus to drive the function to respond to the event

## Action Items

- EventSource
    - Improve EventSource CRD specification
    - Complete EventSource controller development
    - Make OpenFunction responsible for driving the EventSource workload
- Trigger
    - Improve Trigger CRD specification
    - Complete Trigger controller development
    - Make OpenFunction responsible for driving the Trigger workload
- EventBus
    - Improve EventBus CRD specification
