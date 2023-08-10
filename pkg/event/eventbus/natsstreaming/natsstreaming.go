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

package natsstreaming

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
	ComponentType    = "pubsub.natsstreaming"
	ComponentVersion = "v1"
	ScaledObjectType = "stan"
)

type EventBus struct {
	mu       sync.Mutex
	log      logr.Logger
	Spec     *ofevent.NatsStreamingSpec
	Metadata map[string]interface{}
}

func NewNatsStreamingEventBus(log logr.Logger, spec *ofevent.NatsStreamingSpec) event.OpenFunctionEventBus {
	eb := &EventBus{}

	eb.log = log
	eb.log.WithName("NatsStreamingEventBus")

	eb.Spec = spec
	eb.init()
	return eb
}

func (eb *EventBus) init() {
	m := map[string]interface{}{}

	// handle mandatory parameters
	m["natsURL"] = eb.Spec.NatsURL
	m["natsStreamingClusterID"] = eb.Spec.NatsStreamingClusterID
	m["subscriptionType"] = eb.Spec.SubscriptionType
	m["durableSubscriptionName"] = eb.Spec.DurableSubscriptionName

	// handle optional parameters
	if eb.Spec.AckWaitTime != nil {
		m["ackWaitTime"] = *eb.Spec.AckWaitTime
	}
	if eb.Spec.MaxInFlight != nil {
		m["maxInFlight"] = *eb.Spec.MaxInFlight
	}
	if eb.Spec.DeliverNew != nil {
		m["deliverNew"] = *eb.Spec.DeliverNew
	}
	if eb.Spec.StartAtSequence != nil {
		m["startAtSequence"] = *eb.Spec.StartAtSequence
	}
	if eb.Spec.StartWithLastReceived != nil {
		m["startWithLastReceived"] = *eb.Spec.StartWithLastReceived
	}
	if eb.Spec.DeliverAll != nil {
		m["deliverAll"] = *eb.Spec.DeliverAll
	}
	if eb.Spec.StartAtTimeDelta != nil {
		m["startAtTimeDelta"] = *eb.Spec.StartAtTimeDelta
	}
	if eb.Spec.StartAtTime != nil {
		m["startAtTime"] = *eb.Spec.StartAtTime
	}
	if eb.Spec.StartAtTimeFormat != nil {
		m["startAtTimeFormat"] = *eb.Spec.StartAtTimeFormat
	}

	eb.Metadata = m
}

func (eb *EventBus) SetMetadata(key string, value interface{}) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.Metadata[key] = value
}

func (eb *EventBus) GetMetadata() map[string]interface{} {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	return eb.Metadata
}

func (eb *EventBus) GenComponent(namespace string, name string) (*componentsv1alpha1.Component, error) {
	component := &componentsv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	component.Spec.Type = ComponentType
	component.Spec.Version = ComponentVersion

	metadataItems, err := event.ConvertMetadata(eb.GetMetadata())
	if err != nil {
		eb.log.Error(err, "failed to generate component", "namespace", namespace, "name", name)
		return nil, err
	}
	component.Spec.Metadata = metadataItems
	return component, nil
}

func (eb *EventBus) GenScaleOptions(subjects []string) (*ofcore.KedaScaledObject, []*kedav1alpha1.ScaleTriggers) {
	if eb.Spec.ScaleOption == nil {
		return nil, nil
	}
	scaledObject := &ofcore.KedaScaledObject{}
	triggers := []*kedav1alpha1.ScaleTriggers{}

	// handle scaledObject
	scaledObject.MinReplicaCount = eb.Spec.ScaleOption.MinReplicaCount
	scaledObject.MaxReplicaCount = eb.Spec.ScaleOption.MaxReplicaCount
	scaledObject.CooldownPeriod = eb.Spec.ScaleOption.CooldownPeriod
	scaledObject.PollingInterval = eb.Spec.ScaleOption.PollingInterval

	// handle trigger
	triggerMD := map[string]string{}
	if eb.Spec.ScaleOption.Metadata != nil {
		triggerMD = eb.Spec.ScaleOption.Metadata
	}
	for _, subject := range subjects {
		trigger := &kedav1alpha1.ScaleTriggers{}
		trigger.Type = ScaledObjectType
		trigger.Metadata = triggerMD
		trigger.Metadata["natsServerMonitoringEndpoint"] = eb.Spec.ScaleOption.NatsServerMonitoringEndpoint
		trigger.Metadata["lagThreshold"] = eb.Spec.ScaleOption.LagThreshold
		trigger.Metadata["durableName"] = eb.Spec.DurableSubscriptionName
		trigger.Metadata["subject"] = subject
		md := eb.GetMetadata()
		if consumerID, exist := md["consumerID"]; exist {
			trigger.Metadata["queueGroup"] = consumerID.(string)
		}
		triggers = append(triggers, trigger)
	}

	return scaledObject, triggers
}
