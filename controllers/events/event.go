/*
Copyright 2022 The OpenFunction Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package events

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	kservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ofcore "github.com/openfunction/apis/core/v1beta1"
	ofevent "github.com/openfunction/apis/events/v1alpha1"
)

const (
	EventSourceControlledLabel = "controlled-by-eventsource"
	TriggerControlledLabel     = "controlled-by-trigger"
	EventBusNameLabel          = "eventbus-name"
	EventBusTopicName          = "eventbus-topic-name"

	DefaultLogLevel = "1"

	// Component Name Template

	// EventSourceComponentNameTmpl => esc(EventSource Component)-{sourceKind}-{eventName}
	EventSourceComponentNameTmpl = "esc-%s-%s"
	// EventSourceBusComponentNameTmpl => ebfes(EventBus for EventSource)-{eventSourceName}
	EventSourceBusComponentNameTmpl = "ebfes-%s"
	// TriggerBusComponentNameTmpl => ebft(EventBus for Trigger)-{triggerName}
	TriggerBusComponentNameTmpl = "ebft-%s"
	// SinkComponentNameTmpl => ts-{resourceName}-{sinkNamespace}
	SinkComponentNameTmpl = "ts-%s-%s"

	// EventSourceWorkloadsNameTmpl => esw(EventSource Workloads)-{eventSourceName}-{sourceKind}-{eventName}
	EventSourceWorkloadsNameTmpl = "esw-%s-%s-%s"
	// TriggerWorkloadsNameTmpl => t(Trigger)-{triggerName}
	TriggerWorkloadsNameTmpl = "t-%s"
	// EventBusTopicNameTmpl => {namespace}-{eventSourceName}-{eventName}
	EventBusTopicNameTmpl = "%s-%s-%s"

	// DaprIO Name Template

	// EventBusOutputNameTmpl => ebo(EventBus output)-{topicName}
	EventBusOutputNameTmpl = "ebo-%s"
	// SinkOutputNameTmpl => so(Sink output)-{resourceType}-{resourceName}-{index}
	SinkOutputNameTmpl = "so-%s-%s-%s"
	// EventSourceInputNameTmpl => esi(EventSource input)-{eventName}
	EventSourceInputNameTmpl = "esi-%s"
	// TriggerInputNameTmpl => ti(Trigger input)-{topicName}
	TriggerInputNameTmpl = "ti-%s"

	// SourceKindKafka indicates kafka event source
	SourceKindKafka = "kafka"
	// SourceKindCron indicates cron (scheduler) event source
	SourceKindCron = "cron"
	// SourceKindMQTT indicates mqtt event source
	SourceKindMQTT = "mqtt"
	// SourceKindRedis indicates redis event source
	SourceKindRedis = "redis"
)

var (
	knativeServiceGVK = schema.FromAPIVersionAndKind(kservingv1.SchemeGroupVersion.String(), "Service")
	ofFunctionGVK     = schema.FromAPIVersionAndKind(ofcore.SchemeGroupVersion.String(), "Function")
)

type EventSourceConfig struct {
	EventBusComponent  string `json:"eventBusComponent,omitempty"`
	EventBusOutputName string `json:"eventBusOutputName,omitempty"`
	EventBusTopic      string `json:"eventBusTopic,omitempty"`
	SinkOutputName     string `json:"sinkOutputName,omitempty"`
	LogLevel           string `json:"logLevel,omitempty"`
}

type TriggerConfig struct {
	EventBusComponent string                 `json:"eventBusComponent,omitempty"`
	Inputs            []*Input               `json:"inputs,omitempty"`
	Subscribers       map[string]*Subscriber `json:"subscribers,omitempty"`
	LogLevel          string                 `json:"logLevel,omitempty"`
}

type Input struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	EventSource string `json:"eventSource"`
	Event       string `json:"event"`
}

type Subscriber struct {
	SinkOutputName       string `json:"sinkOutputName,omitempty"`
	DLSinkOutputName     string `json:"dlSinkOutputName,omitempty"`
	EventBusOutputName   string `json:"eventBusOutputName,omitempty"`
	DLEventBusOutputName string `json:"dlEventBusOutputName,omitempty"`
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

func newSinkComponentSpec(c client.Client, log logr.Logger, ref *ofevent.Reference) (*componentsv1alpha1.ComponentSpec, error) {
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

func createSinkComponent(ctx context.Context, c client.Client, log logr.Logger, resource client.Object, sink *ofevent.SinkSpec) (*componentsv1alpha1.Component, error) {
	if sink.Uri == nil && sink.Ref == nil {
		return nil, errors.New("at least one uri or ref must be set in sink")
	}

	var namespace, url string
	if sink.Uri != nil {
		url = *sink.Uri
		// when setting the Uri, use resource.GetNamespace()
		namespace = resource.GetNamespace()
	} else {
		gvk := sink.Ref.GroupVersionKind()
		switch gvk {
		case knativeServiceGVK:
			var ksvc kservingv1.Service
			if err := c.Get(ctx, types.NamespacedName{Namespace: sink.Ref.Namespace, Name: sink.Ref.Name}, &ksvc); err != nil {
				log.Error(err, "Failed to find Knative Service", "namespace", sink.Ref.Namespace, "name", sink.Ref.Name)
				return nil, err
			}
			url = ksvc.Status.URL.String()
		case ofFunctionGVK:
			var of ofcore.Function
			if err := c.Get(ctx, types.NamespacedName{Namespace: sink.Ref.Namespace, Name: sink.Ref.Name}, &of); err != nil {
				log.Error(err, "Failed to find OpenFunction", "namespace", sink.Ref.Namespace, "name", sink.Ref.Name)
				return nil, err
			}
			for _, address := range of.Status.Addresses {
				if *address.Type == ofcore.InternalAddressType {
					url = address.Value
					break
				}
			}
		default:
			return nil, fmt.Errorf("unsupported reference %s", gvk.String())
		}
		namespace = sink.Ref.Namespace
	}

	sink.Uri = &url

	component := &componentsv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf(SinkComponentNameTmpl, resource.GetName(), namespace),
			Namespace: resource.GetNamespace(),
		},
	}

	var spec componentsv1alpha1.ComponentSpec
	specMap := map[string]interface{}{
		"version": "v1",
		"type":    "bindings.http",
		"metadata": []map[string]string{
			{"name": "url", "value": url},
		},
	}
	specBytes, err := json.Marshal(specMap)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(specBytes, &spec); err != nil {
		return nil, err
	}
	component.Spec = spec
	return component, nil
}

func retrieveEventBus(ctx context.Context, c client.Client, eventBusNamespace string, eventBusName string) *ofevent.EventBus {
	var eventBus ofevent.EventBus
	if err := c.Get(ctx, types.NamespacedName{Namespace: eventBusNamespace, Name: eventBusName}, &eventBus); err != nil {
		return nil
	}
	return &eventBus
}

func retrieveClusterEventBus(ctx context.Context, c client.Client, eventBusName string) *ofevent.ClusterEventBus {
	var clusterEventBus ofevent.ClusterEventBus
	if err := c.Get(ctx, types.NamespacedName{Name: eventBusName}, &clusterEventBus); err != nil {
		return nil
	}
	return &clusterEventBus
}

func addSinkForFunction(name string, function *ofcore.Function, component *componentsv1alpha1.Component) *ofcore.Function {
	// add sink bindings component
	function.Spec.Serving.Bindings[component.Name] = &component.Spec

	// add sink output
	output := &ofcore.DaprIO{
		Name:      name,
		Component: component.Name,
		Operation: "post",
	}
	function.Spec.Serving.Outputs = append(function.Spec.Serving.Outputs, output)

	return function
}

func InitFunction(image string) *ofcore.Function {
	function := &ofcore.Function{
		Spec: ofcore.FunctionSpec{
			Image:   image,
			Serving: &ofcore.ServingImpl{},
		},
	}

	version := "v1.0.0"
	function.Spec.Version = &version
	function.Spec.Serving.Runtime = ofcore.Async
	function.Spec.Serving.Inputs = []*ofcore.DaprIO{}
	function.Spec.Serving.Outputs = []*ofcore.DaprIO{}
	function.Spec.Serving.Bindings = map[string]*componentsv1alpha1.ComponentSpec{}
	function.Spec.Serving.Pubsub = map[string]*componentsv1alpha1.ComponentSpec{}
	function.Spec.Serving.ScaleOptions = &ofcore.ScaleOptions{}
	function.Spec.Serving.ScaleOptions.Keda = &ofcore.KedaScaleOptions{}
	function.Spec.Serving.ScaleOptions.Keda.ScaledObject = &ofcore.KedaScaledObject{}
	function.Spec.Serving.Triggers = []ofcore.Triggers{}

	return function
}
