package adapters

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	kservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openfunction "github.com/openfunction/apis/event/v1alpha1"
	"github.com/openfunction/controllers/event/adapters/http"
	"github.com/openfunction/controllers/event/adapters/kafka"
	"github.com/openfunction/controllers/event/adapters/nats"
	"github.com/openfunction/controllers/event/connector"
)

const (
	// EventSourceComponentNameTemp => eventsource-{eventSourceName}-{eventName}
	EventSourceComponentNameTemp = "eventsource-%s-%s"
	// EventSourceSinkComponentNameTemp => eventsource-sink-{eventSourceName}
	EventSourceSinkComponentNameTemp = "eventsource-sink-%s"
	// EventBusComponentNameTemp => eventbus-{eventBusName}-{kindName}
	EventBusComponentNameTemp = "eventbus-%s-%s"
	// TriggerBusComponentNameTemp => trigger-eventbus-{triggerName}-{eventBusName}
	TriggerBusComponentNameTemp = "trigger-eventbus-%s-%s"
	// TriggerSinkComponentNameTemp => trigger-sink-{triggerName}-{sinkNamespace}-{sinkName}
	TriggerSinkComponentNameTemp = "trigger-sink-%s-%s-%s"
)

var (
	adapterMap = map[string]func(name string, namespace string, spec *componentsv1alpha1.ComponentSpec) (*connector.Connector, error){
		"kafka": kafka.NewKafkaAdapter,
		"http":  http.NewHttpAdapter,
		"nats":  nats.NewNatsAdapter,
	}
)

type EventSourceInterface interface {
	AdaptEventSource() error
}

type triggerEnvConfig struct {
	BusComponentName string               `json:"busComponentName"`
	BusTopic         string               `json:"busTopic,omitempty"`
	Subscribers      []*subscriberConfigs `json:"subscribers,omitempty"`
	Port             string               `json:"port,omitempty"`
}

type subscriberConfigs struct {
	SinkComponentName           string `json:"sinkComponentName,omitempty"`
	DeadLetterSinkComponentName string `json:"deadLetterSinkComponentName,omitempty"`
	TopicName                   string `json:"topicName,omitempty"`
	DeadLetterTopicName         string `json:"deadLetterTopicName,omitempty"`
}

func newConnector(name string, namespace string, kind string, spec *componentsv1alpha1.ComponentSpec) (*connector.Connector, error) {
	adapter := adapterMap[kind]
	c, err := adapter(name, namespace, spec)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func NewEventSourceConnectors(es *openfunction.EventSource) ([]*connector.Connector, error) {
	var connectors []*connector.Connector

	if es.Spec.Kafka != nil {
		for eventName, spec := range es.Spec.Kafka {
			componentName := fmt.Sprintf(EventSourceComponentNameTemp, es.Name, eventName)
			c, err := newConnector(componentName, es.Namespace, "kafka", &spec)
			if err != nil {
				continue
			}
			c.EventSource = es
			connectors = append(connectors, c)
		}
	}

	if connectors == nil {
		return nil, fmt.Errorf("connectors is empty")
	}

	return connectors, nil
}

func NewEventSourceSinkConnectors(client client.Client, log logr.Logger, es *openfunction.EventSource) (*connector.Connector, error) {
	spec, err := newSinkPendingConnector(client, log, es.Spec.Sink.Ref)
	if err != nil {
		return nil, err
	}

	componentName := fmt.Sprintf(EventSourceSinkComponentNameTemp, es.Name)
	c, err := newConnector(componentName, es.Namespace, "http", spec)
	if err != nil {
		return nil, err
	}
	c.EventSource = es

	return c, nil
}

func newSinkPendingConnector(client client.Client, log logr.Logger, ref *openfunction.Reference) (*componentsv1alpha1.ComponentSpec, error) {
	ctx := context.Background()
	sinkNamespace := ref.Namespace
	sinkName := ref.Name
	var ksvc kservingv1.Service
	if err := client.Get(ctx, types.NamespacedName{Namespace: sinkNamespace, Name: sinkName}, &ksvc); err != nil {
		log.Error(err, "Failed to find Knative Service", "namespace", sinkNamespace, "name", sinkName)
		return nil, err
	}
	var spec componentsv1alpha1.ComponentSpec
	specMap := map[string]interface{}{
		"version": "v1",
		"type":    "bindings.http",
		"metadata": []map[string]string{
			{"name": "url", "value": ksvc.Status.URL.String()},
		},
	}
	specBytes, err := json.Marshal(specMap)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(specBytes, &spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

func NewEventBusConnectors(eb *openfunction.EventBus) (*connector.Connector, error) {
	if eb.Spec.Nats != nil {
		for name, spec := range eb.Spec.Nats {
			componentName := fmt.Sprintf(EventBusComponentNameTemp, eb.Name, name)
			c, err := newConnector(componentName, eb.Namespace, "nats", &spec)
			if err != nil {
				continue
			}
			c.EventBus = eb
			return c, nil
		}
	}

	return nil, errors.New("no eventbus configuration found")
}

func NewTriggerConnectors(client client.Client, log logr.Logger, t *openfunction.Trigger) ([]*connector.Connector, []byte, error) {
	var connectors []*connector.Connector
	var tc triggerEnvConfig

	eventBusComponent, topic, err := getEventBusComponent(client, log, t)
	if err != nil {
		return nil, nil, err
	}
	triggerEventBusComponentName := fmt.Sprintf(TriggerBusComponentNameTemp, t.Name, t.Spec.EventBusName)
	switch eventBusComponent.Spec.Type {
	case "pubsub.natsstreaming":
		c, err := newConnector(triggerEventBusComponentName, t.Namespace, "nats", &eventBusComponent.Spec)
		if err != nil {
			return nil, nil, err
		}
		connectors = append(connectors, c)
	default:
		return nil, nil, errors.New("invalid component")
	}
	tc.BusComponentName = triggerEventBusComponentName
	tc.BusTopic = topic

	// The subscribers are classified as "normal", "dead letter" and "total" for both sink and topic types,
	// and the duplicate elements are removed from the arrays for each of these three categories.
	var errList []error
	var sinks []*openfunction.SinkSpec
	var deadLetterSinks []*openfunction.SinkSpec
	var totalSinks []*openfunction.SinkSpec
	var topics []string
	var deadLetterTopics []string
	var totalTopics []string

	fmt.Println(t.Spec.Subscribers)
	if t.Spec.Subscribers != nil {
		for _, sub := range t.Spec.Subscribers {
			ts := subscriberConfigs{}
			sub := sub
			if sub.Sink != nil && !sinkInSlice(sub.Sink, sinks) {
				componentName := fmt.Sprintf(TriggerSinkComponentNameTemp, t.Name, sub.Sink.Ref.Namespace, sub.Sink.Ref.Name)
				sinks = append(sinks, sub.Sink)
				ts.SinkComponentName = componentName
				if !sinkInSlice(sub.Sink, totalSinks) {
					totalSinks = append(totalSinks, sub.Sink)
					// Create sink subscriber pending connector
					sinkSpec, err := newSinkPendingConnector(client, log, sub.Sink.Ref)
					if err != nil {
						errList = append(errList, err)
						log.Info("error")
						continue
					}
					// Create sink subscriber connector
					c, err := newConnector(componentName, t.Namespace, "http", sinkSpec)
					if err != nil {
						errList = append(errList, err)
						continue
					}
					c.Trigger = t
					connectors = append(connectors, c)
				}
			}
			if sub.DeadLetterSink != nil && !sinkInSlice(sub.DeadLetterSink, deadLetterSinks) {
				componentName := fmt.Sprintf(TriggerSinkComponentNameTemp, t.Name, sub.DeadLetterSink.Ref.Namespace, sub.DeadLetterSink.Ref.Name)
				deadLetterSinks = append(deadLetterSinks, sub.DeadLetterSink)
				ts.DeadLetterSinkComponentName = componentName
				if !sinkInSlice(sub.DeadLetterSink, totalSinks) {
					totalSinks = append(totalSinks, sub.DeadLetterSink)
					// Create deadLetterSink subscriber pending connector
					sinkSpec, err := newSinkPendingConnector(client, log, sub.DeadLetterSink.Ref)
					if err != nil {
						errList = append(errList, err)
						log.Info("error")
						continue
					}
					// Create deadLetterSink subscriber connector
					c, err := newConnector(componentName, t.Namespace, "http", sinkSpec)
					if err != nil {
						errList = append(errList, err)
						continue
					}
					c.Trigger = t
					connectors = append(connectors, c)
				}
			}
			if sub.Topic != "" && !topicInSlice(sub.Topic, topics) {
				topics = append(topics, sub.Topic)
				ts.TopicName = sub.Topic
				if !topicInSlice(sub.Topic, totalTopics) {
					totalTopics = append(totalTopics, sub.Topic)
				}
			}
			if sub.DeadLetterTopic != "" && !topicInSlice(sub.DeadLetterTopic, deadLetterTopics) {
				deadLetterTopics = append(deadLetterTopics, sub.DeadLetterTopic)
				ts.DeadLetterTopicName = sub.DeadLetterTopic
				if !topicInSlice(sub.Topic, totalTopics) {
					totalTopics = append(totalTopics, sub.DeadLetterTopic)
				}
			}
			tc.Subscribers = append(tc.Subscribers, &ts)
		}
	}

	tcBytes, err := json.Marshal(tc)
	if err != nil {
		errList = append(errList, err)
	}

	if errList != nil {
		return connectors, nil, errList[0]
	} else {
		return connectors, tcBytes, nil
	}
}

func topicInSlice(topic string, list []string) bool {
	for _, b := range list {
		if reflect.DeepEqual(topic, b) {
			return true
		}
	}
	return false
}

func sinkInSlice(sink *openfunction.SinkSpec, list []*openfunction.SinkSpec) bool {
	for _, b := range list {
		if reflect.DeepEqual(sink, b) {
			return true
		}
	}
	return false
}

func getEventBusComponent(client client.Client, log logr.Logger, t *openfunction.Trigger) (*componentsv1alpha1.Component, string, error) {
	ctx := context.Background()

	var eventbus openfunction.EventBus
	if err := client.Get(ctx, types.NamespacedName{Namespace: t.Namespace, Name: t.Spec.EventBusName}, &eventbus); err != nil {
		log.Error(err, "Failed to find eventbus", "namespace", t.Namespace, "name", t.Spec.EventBusName)
		return nil, "", err
	}
	if componentName, ok := eventbus.GetAnnotations()["component-name"]; ok {
		var eventBusComponent componentsv1alpha1.Component
		if err := client.Get(ctx, types.NamespacedName{Namespace: t.Namespace, Name: componentName}, &eventBusComponent); err != nil {
			log.Error(err, "Failed to find eventbus component", "namespace", t.Namespace, "name", componentName)
			return nil, "", err
		}
		return &eventBusComponent, eventbus.Spec.Topic, nil
	}
	return nil, "", errors.New("failed to get 'component-name' key in eventbus annotation")
}
