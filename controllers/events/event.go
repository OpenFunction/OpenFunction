package events

import (
	"context"
	"encoding/base64"
	"errors"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	kservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	openfunction "github.com/openfunction/apis/events/v1alpha1"
	openfunctionevent "github.com/openfunction/apis/events/v1alpha1"
)

const (
	EventSourceControlledLabel = "controlled-by-eventsource"
	TriggerControlledLabel     = "controlled-by-trigger"

	// EventSourceComponentNameTmpl => eventsource-{eventSourceName}-{sourceKind}-{eventName}
	EventSourceComponentNameTmpl = "eventsource-%s-%s-%s"
	// EventSourceSinkComponentNameTmpl => eventsource-sink-{eventSourceName}
	EventSourceSinkComponentNameTmpl = "eventsource-sink-%s"
	// EventSourceBusComponentNameTmpl => eventsource-eventbus-{eventSourceName}
	EventSourceBusComponentNameTmpl = "eventsource-eventbus-%s"
	// TriggerBusComponentNameTmpl => trigger-eventbus-{triggerName}
	TriggerBusComponentNameTmpl = "trigger-eventbus-%s"
	// TriggerSinkComponentNameTmpl => trigger-sink-{triggerName}-{sinkNamespace}-{sinkName}
	TriggerSinkComponentNameTmpl = "trigger-sink-%s-%s-%s"
	// EventSourceWorkloadsNameTmpl => eventsource-{eventSourceName}-{sourceKind}-{eventName}
	EventSourceWorkloadsNameTmpl = "eventsource-%s-%s-%s"
	// TriggerWorkloadsNameTmpl => trigger-{triggerName}
	TriggerWorkloadsNameTmpl = "trigger-%s"
	// EventBusTopicNameTmpl => {namespace}-{eventSourceName}-{eventName}
	EventBusTopicNameTmpl = "%s-%s-%s"

	// SourceKindKafka indicates kafka event source
	SourceKindKafka = "kafka"
	// SourceKindCron indicates cron (scheduler) event source
	SourceKindCron = "cron"
	// SourceKindRedis indicates redis event source
	SourceKindRedis = "redis"
)

type EventSourceConfig struct {
	EventSourceComponentName string `json:"eventSourceComponentName"`
	EventSourceTopic         string `json:"eventSourceTopic,omitempty"`
	EventBusComponentName    string `json:"eventBusComponentName,omitempty"`
	EventBusTopic            string `json:"eventBusTopic,omitempty"`
	SinkComponentName        string `json:"sinkComponentName,omitempty"`
	EventSourceSpecEncode    string `json:"eventSourceSpecEncode,omitempty"`
	EventBusSpecEncode       string `json:"eventBusSpecEncode,omitempty"`
	SinkSpecEncode           string `json:"sinkSpecEncode,omitempty"`
}

type TriggerConfig struct {
	EventBusComponentName string               `json:"eventBusComponentName"`
	EventBusTopics        []string             `json:"eventBusTopics,omitempty"`
	Subscribers           []*SubscriberConfigs `json:"subscribers,omitempty"`
	EventBusSpecEncode    string               `json:"eventBusSpecEncode,omitempty"`
	SinkSpecEncode        string               `json:"sinkSpecEncode,omitempty"`
}

type SubscriberConfigs struct {
	SinkComponentName           string `json:"sinkComponentName,omitempty"`
	DeadLetterSinkComponentName string `json:"deadLetterSinkComponentName,omitempty"`
	TopicName                   string `json:"topicName,omitempty"`
	DeadLetterTopicName         string `json:"deadLetterTopicName,omitempty"`
}

func (e *EventSourceConfig) EncodeConfig() (string, error) {
	return encodeEnv(e)
}

func (e *TriggerConfig) EncodeConfig() (string, error) {
	return encodeEnv(e)
}

func encodeEnv(config interface{}) (string, error) {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	configEncode := base64.StdEncoding.EncodeToString(configBytes)
	return configEncode, nil
}

func (e *EventSourceConfig) DecodeEnv(encodedConfig string) (*EventSourceConfig, error) {
	var config *EventSourceConfig
	configSpec, err := decodeConfig(encodedConfig)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(configSpec, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func (e *TriggerConfig) DecodeEnv(encodedConfig string) (*TriggerConfig, error) {
	var config *TriggerConfig
	configSpec, err := decodeConfig(encodedConfig)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(configSpec, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func decodeConfig(encodedConfig string) ([]byte, error) {
	if len(encodedConfig) > 0 {
		configSpec, err := base64.StdEncoding.DecodeString(encodedConfig)
		if err != nil {
			return nil, err
		}
		return configSpec, nil
	}
	return nil, errors.New("string length is zero")
}

func mutateDaprComponent(scheme *runtime.Scheme, component *componentsv1alpha1.Component, object v1.Object) controllerutil.MutateFn {
	return func() error {
		var controlledByLabel string
		switch object.(type) {
		case *openfunctionevent.EventSource:
			controlledByLabel = EventSourceControlledLabel
		case *openfunctionevent.Trigger:
			controlledByLabel = TriggerControlledLabel
		}
		component.SetLabels(map[string]string{controlledByLabel: object.GetName()})
		component.SetOwnerReferences(nil)
		return ctrl.SetControllerReference(object, component, scheme)
	}
}

func newSinkComponentSpec(c client.Client, log logr.Logger, ref *openfunction.Reference) (*componentsv1alpha1.ComponentSpec, error) {
	ctx := context.Background()
	sinkNamespace := ref.Namespace
	sinkName := ref.Name
	var ksvc kservingv1.Service
	if err := c.Get(ctx, types.NamespacedName{Namespace: sinkNamespace, Name: sinkName}, &ksvc); err != nil {
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

func retrieveEventBus(ctx context.Context, c client.Client, eventBusNamespace string, eventBusName string) *openfunctionevent.EventBus {
	var eventBus openfunctionevent.EventBus
	if err := c.Get(ctx, types.NamespacedName{Namespace: eventBusNamespace, Name: eventBusName}, &eventBus); err != nil {
		return nil
	}
	return &eventBus
}

func retrieveClusterEventBus(ctx context.Context, c client.Client, eventBusName string) *openfunctionevent.ClusterEventBus {
	var clusterEventBus openfunctionevent.ClusterEventBus
	if err := c.Get(ctx, types.NamespacedName{Name: eventBusName}, &clusterEventBus); err != nil {
		return nil
	}
	return &clusterEventBus
}
