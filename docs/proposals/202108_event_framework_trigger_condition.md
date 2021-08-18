## Motivation

In [events framework proposal](https://github.com/OpenFunction/OpenFunction/blob/main/docs/proposals/202107_add_event_framework.md#filter) we mentioned the need to add conditional judgement capabilities to the event framework's triggers, and the **condition** is an implementation of such capabilities.

## Design

### Struct

**TriggerEnvConfig** is used to receive and process events passed by the OpenFunction events framework.

**TriggerMgr** is used to store informations during the event triggering process, providing the ability for condition determination.

```go
type TriggerEnvConfig struct {
	EventBusComponent string                 `json:"eventBusComponent"`
	Inputs            []*Input               `json:"inputs,omitempty"`
	Subscribers       map[string]*Subscriber `json:"subscribers,omitempty"`
	Port              string                 `json:"port,omitempty"`
}

type Input struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	EventSource string `json:"eventSource"`
	Event       string `json:"event"`
}

type Subscriber struct {
	SinkComponent   string `json:"sinkComponent,omitempty"`
	DLSinkComponent string `json:"deadLetterSinkComponent,omitempty"`
	Topic           string `json:"topic,omitempty"`
	DLTopic         string `json:"deadLetterTopic,omitempty"`
}

type TriggerManager struct {
	// key: condition, value: *Subscriber
	ConditionSubscriberMap *sync.Map
	// key: topic, value: *InputStatus
	TopicInputMap *sync.Map
	TopicEventMap map[string]chan *common.TopicEvent
	CelEnv        *cel.Env
}

type InputStatus struct {
	Name        string
	LastMsgTime int64
	LastEvent   *common.TopicEvent
	Status      bool
}
```

### Logic

1. **topic** is associated with **input** on a one-to-one basis
2. Create a goroutine and a channel for each **input** (i.e. **topic**)
3. Incoming events are sent to the channel corresponding to the **topic**
4. When the channel receives an event, it will:
    1. Reset timer ticker (Default 60s Timeout)
    2. Update the status of the **input** to true
    3. Check if any **condition** matches at this point
        1. If there is a matched **condition**, then get the **subscriber** configuration corresponding to the **condition**
        2. Sending event to the final function by the **subscriber** configuration
5. Reset **input** status to false when timer ticker is end up

## Demonstration

Suppose we have defined a Trigger according to the following configuration:

```yaml
apiVersion: events.openfunction.io/v1alpha1
kind: Trigger
metadata:
  name: trigger-a
spec:
  eventBus: "default"
  inputs:
    - name: "A"
      eventSourceName: "my-es-a"
      eventName: "event-a"
    - name: "B"
      eventSourceName: "my-es-b"
      eventName: "event-b"
  subscribers:
  - condition: A || B
    sink:
      ref:
        apiVersion: serving.knative.dev/v1
        kind: Service
        name: function-sample-serving-ksvc
        namespace: default
  - condition: A && B
    topic: "metrics"
```

According to the topic naming rules in the event bus ({namespace}-{eventSourceName}-{eventName}), the topic names used for the two inputs are as follows:

Input name: A -> Topic name: default-my-es-a-event-a

Input name: B -> Topic name: default-my-es-b-event-b

### Initilization

triggerManager.TopicInputMap:

```go
&sync.map{
    "default-my-es-a-event-a": &InputStatus{
        LastMsgTime: 0, 
        LastEvent: nil, 
        Status: false, 
        Name: "A",
    },
    "default-my-es-b-event-b": &InputStatus{
        LastMsgTime: 0, 
        LastEvent: nil, 
        Status: false, 
        Name: "B",
    }
}
```

triggerManager.ConditionSubscriberMap:

```go
&sync.map{
    "A || B": &Subscriber{
        SinkComponent: "http-sink", 
    },
    "A && B": &Subscriber{
        Topic: "metrics",
    }
}
```

### When events incoming

#### Input A receives an event

triggerManager.TopicInputMap:

```go
&sync.map{
    "default-my-es-a-event-a": &InputStatus{
        LastMsgTime: time.Now().Unix(), 
        LastEvent: event1, 
        Status: true, 
        Name: "A",
    },
    "default-my-es-b-event-b": &InputStatus{
        LastMsgTime: 0, 
        LastEvent: nil, 
        Status: false, 
        Name: "B",
    }
}
```

The condition "A || B" will be matched by **cel** and the event **event1** will be sent to "http-sink".

#### Input B receives an event in 60s

triggerManager.TopicInputMap:

```go
&sync.map{
    "default-my-es-a-event-a": &InputStatus{
        LastMsgTime: <Time when event1 is received>, 
        LastEvent: event1, 
        Status: true, 
        Name: "A",
    },
    "default-my-es-b-event-b": &InputStatus{
        LastMsgTime: time.Now().Unix(), 
        LastEvent: event2, 
        Status: true, 
        Name: "B",
    }
}
```

After **cel** has determined the condition, the "A || B" condition and "A && B" will both be matched and the events event1 and event2 will be sent to "http-sink" and "metrics"

#### Input B receives an event after 60s but Input A does not

triggerManager.TopicInputMap:

```go
&sync.map{
    "default-my-es-a-event-a": &InputStatus{
        LastMsgTime: 0, 
        LastEvent: nil, 
        Status: false, 
        Name: "A",
    },
    "default-my-es-b-event-b": &InputStatus{
        LastMsgTime: time.Now().Unix(), 
        LastEvent: event2, 
        Status: true, 
        Name: "B",
    }
}
```

The condition "A || B" will be matched by **cel** and the event2 will be sent to "http-sink"

## Performance

> You need to [install dapr](https://docs.dapr.io/getting-started/) first.

Add the domain name of the nats server to the ip address mapping in `/etc/hosts` on the node:

```shell
# nats
<svc address> nats.default nats-0.nats.default.svc.cluster.local
```

Import the configuration:

```shell
export CONFIG="eyJidXNDb21wb25lbnQiOiJ0cmlnZ2VyIiwiYnVzVG9waWMiOlt7Im5hbWUiOiJBIiwibmFtZXNwYWNlIjoiZGVmYXVsdCIsImV2ZW50U291cmNlIjoibXktZXZlbnRzb3VyY2UiLCJldmVudCI6InNhbXBsZS1vbmUifSx7Im5hbWUiOiJCIiwibmFtZXNwYWNlIjoiZGVmYXVsdCIsImV2ZW50U291cmNlIjoibXktZXZlbnRzb3VyY2UiLCJldmVudCI6InNhbXBsZS10d28ifV0sInN1YnNjcmliZXJzIjp7IkEgXHUwMDI2XHUwMDI2IEIiOnsidG9waWMiOiJtZXRyaWNzIn0sIkEgfHwgQiI6eyJzaW5rQ29tcG9uZW50IjoiaHR0cC1zaW5rIn19LCJwb3J0IjoiNTA1MCJ9"
```

Clone [OpenFunction/events-handlers](https://github.com/OpenFunction/events-handlers) to local and go to `trigger/handler`.

Start the program using the dapr command line, specifying the `--profile-port` and `--enable-profiling`:

```shell
dapr run --app-id trigger-handler --enable-profiling --profile-port 7777 --app-protocol grpc --app-port 5050 --components-path ../example/deploy/ go run ./main.go
```

Now that the connection has been established, we can use `pprof` to profile the Dapr runtime.

The following example will create a `cpu.pprof` file containing samples from a profile session that lasts 120 seconds:

```shell
curl "http://localhost:7777/debug/pprof/profile?seconds=120" > cpu.pprof
```

Use the following command to display the profile (You need to install graphviz first) :

```shell
go tool pprof -http=":8081" cpu.pprof
```

You can refer to [Profiling & Debugging](https://docs.dapr.io/operations/troubleshooting/profiling-debugging/) to learn more.

