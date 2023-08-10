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

package kafka

import (
	"sync"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ofcore "github.com/openfunction/apis/core/v1beta1"
	ofevent "github.com/openfunction/apis/events/v1alpha1"
	"github.com/openfunction/pkg/event"
)

const (
	ComponentType    = "bindings.kafka"
	ComponentVersion = "v1"
	ScaledObjectType = "kafka"
)

type EventSource struct {
	mu       sync.Mutex
	log      logr.Logger
	Spec     *ofevent.KafkaSpec
	Metadata map[string]interface{}
}

func NewKafkaEventSource(log logr.Logger, spec *ofevent.KafkaSpec) event.OpenFunctionEventSource {
	es := &EventSource{}

	es.log = log
	es.log.WithName("KafkaEventSource")

	es.Spec = spec
	es.init()
	return es
}

func (es *EventSource) init() {
	m := map[string]interface{}{}

	// handle mandatory parameters
	m["brokers"] = es.Spec.Brokers
	m["publishTopic"] = es.Spec.Topic
	m["topics"] = es.Spec.Topic
	m["authRequired"] = es.Spec.AuthRequired

	// handle optional parameters
	if es.Spec.SaslUsername != nil {
		m["saslUsername"] = *es.Spec.SaslUsername
	}
	if es.Spec.SaslPassword != nil {
		m["saslPassword"] = *es.Spec.SaslPassword
	}
	if es.Spec.MaxMessageBytes != nil {
		m["maxMessageBytes"] = *es.Spec.MaxMessageBytes
	}

	es.Metadata = m
}

func (es *EventSource) SetMetadata(key string, value interface{}) {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.Metadata[key] = value
}

func (es *EventSource) GetMetadata() map[string]interface{} {
	es.mu.Lock()
	defer es.mu.Unlock()
	return es.Metadata
}

func (es *EventSource) GenComponent(namespace string, name string) (*componentsv1alpha1.Component, error) {
	component := &componentsv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	component.Spec.Type = ComponentType
	component.Spec.Version = ComponentVersion

	metadataItems, err := event.ConvertMetadata(es.GetMetadata())
	if err != nil {
		es.log.Error(err, "failed to generate component", "namespace", namespace, "name", name)
		return nil, err
	}
	component.Spec.Metadata = metadataItems
	return component, nil
}

func (es *EventSource) GenScaleOptions() (*ofcore.KedaScaledObject, *kedav1alpha1.ScaleTriggers) {
	if es.Spec.ScaleOption == nil {
		return nil, nil
	}
	scaledObject := &ofcore.KedaScaledObject{}
	trigger := &kedav1alpha1.ScaleTriggers{}

	// handle scaledObject
	scaledObject.MinReplicaCount = es.Spec.ScaleOption.MinReplicaCount
	scaledObject.MaxReplicaCount = es.Spec.ScaleOption.MaxReplicaCount
	scaledObject.CooldownPeriod = es.Spec.ScaleOption.CooldownPeriod
	scaledObject.PollingInterval = es.Spec.ScaleOption.PollingInterval

	// handle trigger
	trigger.Type = ScaledObjectType
	if es.Spec.ScaleOption.Metadata != nil {
		trigger.Metadata = es.Spec.ScaleOption.Metadata
	} else {
		trigger.Metadata = map[string]string{}
	}
	trigger.Metadata["bootstrapServers"] = es.Spec.Brokers
	trigger.Metadata["topic"] = es.Spec.Topic
	md := es.GetMetadata()
	if consumerGroup, exist := md["consumerGroup"]; exist {
		trigger.Metadata["consumerGroup"] = consumerGroup.(string)
	}
	return scaledObject, trigger
}
