package events

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	openfunctioncore "github.com/openfunction/apis/core/v1alpha1"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
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
	EventBusNameLabel          = "eventbus-name"
	EventBusTopicName          = "eventbus-topic-name"

	// EventSourceComponentNameTmpl => eventsource-{eventSourceName}-{sourceKind}-{eventName}
	EventSourceComponentNameTmpl = "eventsource-%s-%s-%s"
	// EventSourceSinkComponentNameTmpl => eventsource-sink-{eventSourceName}
	EventSourceSinkComponentNameTmpl = "eventsource-sink-%s"
	// EventSourceBusComponentNameTmpl => eventbus-eventsource-{eventSourceName}
	EventSourceBusComponentNameTmpl = "eventbus-eventsource-%s"
	// TriggerBusComponentNameTmpl => eventbus-trigger-{triggerName}
	TriggerBusComponentNameTmpl = "eventbus-trigger-%s"
	// TriggerSinkComponentNameTmpl => trigger-sink-{triggerName}-{sinkNamespace}-{sinkName}
	TriggerSinkComponentNameTmpl = "trigger-sink-%s-%s-%s"
	// EventSourceWorkloadsNameTmpl => eventsource-{eventSourceName}-{sourceKind}-{eventName}
	EventSourceWorkloadsNameTmpl = "es-%s-%s-%s"
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

	Pending = "Pending"
	Running = "Running"
	Error   = "Error"

	ResourceTypeComponent = "Component"
	ResourceTypeWorkload  = "Workload"
)

type EventSourceConfig struct {
	EventSourceComponent string `json:"eventSourceComponent"`
	EventSourceTopic     string `json:"eventSourceTopic,omitempty"`
	EventBusComponent    string `json:"eventBusComponent,omitempty"`
	EventBusTopic        string `json:"eventBusTopic,omitempty"`
	SinkComponent        string `json:"sinkComponent,omitempty"`
}

type TriggerConfig struct {
	EventBusComponent  string                 `json:"eventBusComponent"`
	Inputs             []*Input               `json:"inputs,omitempty"`
	Subscribers        map[string]*Subscriber `json:"subscribers,omitempty"`
	EventBusSpecEncode string                 `json:"eventBusSpecEncode,omitempty"`
	SinkSpecEncode     string                 `json:"sinkSpecEncode,omitempty"`
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

type ControlledResources struct {
	Components map[string]*ControlledComponent `json:"components,omitempty"`
	Workloads  map[string]*ControlledWorkload  `json:"workloads,omitempty"`
}

type ControlledComponent struct {
	IsDeprecated bool                          `json:"isDeprecated"`
	Object       *componentsv1alpha1.Component `json:"object"`
	Status       string                        `json:"status,omitempty"`
}

type ControlledWorkload struct {
	Name         string             `json:"name"`
	IsDeprecated bool               `json:"isDeprecated"`
	Object       *appsv1.Deployment `json:"object"`
	Status       string             `json:"status,omitempty"`
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

func (r *ControlledResources) SetResourceStatusToActive(name string, resourceType string) {
	switch resourceType {
	case ResourceTypeComponent:
		if component, ok := r.Components[name]; ok {
			component.IsDeprecated = false
		}
	case ResourceTypeWorkload:
		if workload, ok := r.Workloads[name]; ok {
			workload.IsDeprecated = false
		}
	}
}

func (r *ControlledResources) SetResourceStatus(name string, resourceType string, status string) {
	switch resourceType {
	case ResourceTypeComponent:
		if component, ok := r.Components[name]; ok {
			component.Status = status
		}
	case ResourceTypeWorkload:
		if workload, ok := r.Workloads[name]; ok {
			workload.Status = status
		}
	}
}

func (r *ControlledResources) GenResourceStatistics(resourceType string) string {
	total := 0
	running := 0
	switch resourceType {
	case ResourceTypeComponent:
		for _, component := range r.Components {
			if !component.IsDeprecated {
				total += 1
				if component.Status == Running {
					running += 1
				}
			}
		}
	case ResourceTypeWorkload:
		for _, workload := range r.Workloads {
			if !workload.IsDeprecated {
				total += 1
				if workload.Status == Running {
					running += 1
				}
			}
		}
	}
	return fmt.Sprintf("%d/%d", running, total)
}

func (r *ControlledResources) GenResourceStatus(resourceType string) []*openfunctionevent.OwnedResourceStatus {
	var statuses []*openfunctionevent.OwnedResourceStatus

	switch resourceType {
	case ResourceTypeComponent:
		for name, component := range r.Components {
			if !component.IsDeprecated {
				statuses = append(statuses, &openfunctionevent.OwnedResourceStatus{State: component.Status, Name: name})
			}
		}
	case ResourceTypeWorkload:
		for name, workload := range r.Workloads {
			if !workload.IsDeprecated {
				statuses = append(statuses, &openfunctionevent.OwnedResourceStatus{State: workload.Status, Name: name})
			}
		}
	}
	return statuses
}

func mutateDaprComponent(scheme *runtime.Scheme, component *componentsv1alpha1.Component, spec *componentsv1alpha1.ComponentSpec, object v1.Object) controllerutil.MutateFn {
	return func() error {
		var controlledByLabel string
		switch object.(type) {
		case *openfunctionevent.EventSource:
			controlledByLabel = EventSourceControlledLabel
		case *openfunctionevent.Trigger:
			controlledByLabel = TriggerControlledLabel
		}
		component.Spec = *spec
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

func addDaprComponent(function *openfunctioncore.Function, component *componentsv1alpha1.Component, componentType string) *openfunctioncore.Function {
	spec := function.Spec.Serving.OpenFuncAsync.Dapr

	obj := &openfunctioncore.DaprIO{
		Name: component.Name,
		Type: strings.Split(component.Spec.Type, ".")[0],
	}
	switch componentType {
	case "input":
		spec.Inputs = append(spec.Inputs, obj)
	case "output":
		spec.Outputs = append(spec.Outputs, obj)
	}

	newComponent := &openfunctioncore.DaprComponent{
		Name:          component.Name,
		ComponentSpec: component.Spec,
	}
	spec.Components = append(spec.Components, *newComponent)
	function.Spec.Serving.OpenFuncAsync.Dapr = spec
	return function
}

func addSinkComponent(function *openfunctioncore.Function, component *componentsv1alpha1.Component) *openfunctioncore.Function {
	spec := function.Spec.Serving.OpenFuncAsync.Dapr

	obj := &openfunctioncore.DaprIO{
		Name: component.Name,
		Type: strings.Split(component.Spec.Type, ".")[0],
		Params: map[string]string{
			"operation": "post",
			"type":      strings.Split(component.Spec.Type, ".")[0],
		},
	}
	spec.Outputs = append(spec.Outputs, obj)

	newComponent := &openfunctioncore.DaprComponent{
		Name:          component.Name,
		ComponentSpec: component.Spec,
	}
	spec.Components = append(spec.Components, *newComponent)
	function.Spec.Serving.OpenFuncAsync.Dapr = spec
	return function
}
