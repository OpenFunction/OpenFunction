/*
Copyright 2018 The Knative Authors

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

package eventing

import "k8s.io/apimachinery/pkg/runtime/schema"

const (
	GroupName = "eventing.knative.dev"

	// BrokerLabelKey is the label key on Triggers and Subscriptions
	// to indicate to which Broker they belong to.
	BrokerLabelKey = GroupName + "/broker"

	// BrokerClassKey is the annotation key on Brokers to indicate
	// which Controller is responsible for them.
	BrokerClassKey = GroupName + "/broker.class"

	// MTChannelBrokerClassValue is the value we use to specify the
	// Broker using channels, but the resources (ingress,filter) run
	// in the system namespace. As in Broker from this repository
	// pkg/reconciler/mtbroker
	MTChannelBrokerClassValue = "MTChannelBasedBroker"

	// ScopeAnnotationKey is the annotation key to indicate
	// the scope of the component handling a given resource.
	// Valid values are: cluster, namespace, resource.
	ScopeAnnotationKey = GroupName + "/scope"

	// ScopeResource indicates that the resource
	// must be handled by a dedicated component
	ScopeResource = "resource"

	// ScopeNamespace indicates that the resource
	// must be handled by the namespace-scoped component
	ScopeNamespace = "namespace"

	// ScopeCluster indicates the resource must be
	// handled by the cluster-scoped component
	ScopeCluster = "cluster"

	// EventTypesAnnotationKey is the annotation key to specify
	// if a Source has event types defines in its CRD.
	EventTypesAnnotationKey = "registry.knative.dev/eventTypes"

	// BrokerChannelAddressStatusAnnotationKey is the broker status
	// annotation key used to specify the address of its channel.
	BrokerChannelAddressStatusAnnotationKey = "knative.dev/channelAddress"

	// BrokerChannelAPIVersionStatusAnnotationKey is the broker status
	// annotation key used to specify the APIVersion of the channel for
	// the triggers to subscribe to.
	BrokerChannelAPIVersionStatusAnnotationKey = "knative.dev/channelAPIVersion"

	// BrokerChannelKindStatusAnnotationKey is the broker status
	// annotation key used to specify the Kind of the channel for
	// the triggers to subscribe to.
	BrokerChannelKindStatusAnnotationKey = "knative.dev/channelKind"

	// BrokerChannelNameStatusAnnotationKey is the broker status
	// annotation key used to specify the name of the channel for
	// the triggers to subscribe to.
	BrokerChannelNameStatusAnnotationKey = "knative.dev/channelName"
)

var (
	// TriggersResource represents a Knative Trigger
	TriggersResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "triggers",
	}
	// BrokersResource represents a Knative Broker
	BrokersResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "brokers",
	}
	// EventTypesResource represents a Knative EventType
	EventTypesResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "eventtypes",
	}
)
