/*
 * Copyright 2020 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"

	eventingduckv1 "knative.dev/eventing/pkg/apis/duck/v1"
	messagingv1 "knative.dev/eventing/pkg/apis/messaging/v1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Sequence defines a sequence of Subscribers that will be wired in
// series through Channels and Subscriptions.
type Sequence struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the Sequence.
	Spec SequenceSpec `json:"spec,omitempty"`

	// Status represents the current state of the Sequence. This data may be out of
	// date.
	// +optional
	Status SequenceStatus `json:"status,omitempty"`
}

var (
	// Check that Sequence can be validated and defaulted.
	_ apis.Validatable = (*Sequence)(nil)
	_ apis.Defaultable = (*Sequence)(nil)

	// Check that Sequence can return its spec untyped.
	_ apis.HasSpec = (*Sequence)(nil)

	// TODO: make appropriate fields immutable.
	//_ apis.Immutable = (*Sequence)(nil)

	_ runtime.Object = (*Sequence)(nil)

	// Check that we can create OwnerReferences to a Sequence.
	_ kmeta.OwnerRefable = (*Sequence)(nil)

	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*Sequence)(nil)
)

type SequenceSpec struct {
	// Steps is the list of Destinations (processors / functions) that will be called in the order
	// provided. Each step has its own delivery options
	Steps []SequenceStep `json:"steps"`

	// ChannelTemplate specifies which Channel CRD to use. If left unspecified, it is set to the default Channel CRD
	// for the namespace (or cluster, in case there are no defaults for the namespace).
	// +optional
	ChannelTemplate *messagingv1.ChannelTemplateSpec `json:"channelTemplate,omitempty"`

	// Reply is a Reference to where the result of the last Subscriber gets sent to.
	// +optional
	Reply *duckv1.Destination `json:"reply,omitempty"`
}

type SequenceStep struct {
	// Subscriber receiving the step event
	duckv1.Destination `json:",inline"`

	// Delivery is the delivery specification for events to the subscriber
	// This includes things like retries, DLQ, etc.
	// +optional
	Delivery *eventingduckv1.DeliverySpec `json:"delivery,omitempty"`
}

type SequenceChannelStatus struct {
	// Channel is the reference to the underlying channel.
	Channel corev1.ObjectReference `json:"channel"`

	// ReadyCondition indicates whether the Channel is ready or not.
	ReadyCondition apis.Condition `json:"ready"`
}

type SequenceSubscriptionStatus struct {
	// Subscription is the reference to the underlying Subscription.
	Subscription corev1.ObjectReference `json:"subscription"`

	// ReadyCondition indicates whether the Subscription is ready or not.
	ReadyCondition apis.Condition `json:"ready"`
}

// SequenceStatus represents the current state of a Sequence.
type SequenceStatus struct {
	// inherits duck/v1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last processed by the controller.
	// * Conditions - the latest available observations of a resource's current state.
	duckv1.Status `json:",inline"`

	// SubscriptionStatuses is an array of corresponding Subscription statuses.
	// Matches the Spec.Steps array in the order.
	SubscriptionStatuses []SequenceSubscriptionStatus `json:"subscriptionStatuses"`

	// ChannelStatuses is an array of corresponding Channel statuses.
	// Matches the Spec.Steps array in the order.
	ChannelStatuses []SequenceChannelStatus `json:"channelStatuses"`

	// AddressStatus is the starting point to this Sequence. Sending to this
	// will target the first subscriber.
	// It generally has the form {channel}.{namespace}.svc.{cluster domain name}
	duckv1.AddressStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SequenceList is a collection of Sequences.
type SequenceList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Sequence `json:"items"`
}

// GetStatus retrieves the status of the Sequence. Implements the KRShaped interface.
func (p *Sequence) GetStatus() *duckv1.Status {
	return &p.Status.Status
}
