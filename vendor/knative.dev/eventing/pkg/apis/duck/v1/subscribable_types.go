/*
Copyright 2020 The Knative Authors

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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
)

// +genduck

var _ duck.Implementable = (*Subscribable)(nil)

// SubscriberSpec defines a single subscriber to a Subscribable.
//
// At least one of SubscriberURI and ReplyURI must be present
type SubscriberSpec struct {
	// UID is used to understand the origin of the subscriber.
	// +optional
	UID types.UID `json:"uid,omitempty"`
	// Generation of the origin of the subscriber with uid:UID.
	// +optional
	Generation int64 `json:"generation,omitempty"`
	// SubscriberURI is the endpoint for the subscriber
	// +optional
	SubscriberURI *apis.URL `json:"subscriberUri,omitempty"`
	// ReplyURI is the endpoint for the reply
	// +optional
	ReplyURI *apis.URL `json:"replyUri,omitempty"`
	// +optional
	// DeliverySpec contains options controlling the event delivery
	// +optional
	Delivery *DeliverySpec `json:"delivery,omitempty"`
}

// SubscriberStatus defines the status of a single subscriber to a Channel.
type SubscriberStatus struct {
	// UID is used to understand the origin of the subscriber.
	// +optional
	UID types.UID `json:"uid,omitempty"`
	// Generation of the origin of the subscriber with uid:UID.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// Status of the subscriber.
	Ready corev1.ConditionStatus `json:"ready,omitempty"`
	// A human readable message indicating details of Ready status.
	// +optional
	Message string `json:"message,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Subscribable is a skeleton type wrapping Subscribable in the manner we expect resource writers
// defining compatible resources to embed it. We will typically use this type to deserialize
// SubscribableType ObjectReferences and access the Subscription data.  This is not a real resource.
type Subscribable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// SubscribableSpec is the part where Subscribable object is
	// configured as to be compatible with Subscribable contract.
	Spec SubscribableSpec `json:"spec"`

	// SubscribableStatus is the part where SubscribableStatus object is
	// configured as to be compatible with Subscribable contract.
	Status SubscribableStatus `json:"status"`
}

// SubscribableSpec shows how we expect folks to embed Subscribable in their Spec field.
type SubscribableSpec struct {
	// This is the list of subscriptions for this subscribable.
	// +patchMergeKey=uid
	// +patchStrategy=merge
	Subscribers []SubscriberSpec `json:"subscribers,omitempty" patchStrategy:"merge" patchMergeKey:"uid"`
}

// SubscribableStatus is the schema for the subscribable's status portion of the status
// section of the resource.
type SubscribableStatus struct {
	// This is the list of subscription's statuses for this channel.
	// +patchMergeKey=uid
	// +patchStrategy=merge
	Subscribers []SubscriberStatus `json:"subscribers,omitempty" patchStrategy:"merge" patchMergeKey:"uid"`
}

var (
	// Verify SubscribableType resources meet duck contracts.
	_ duck.Populatable = (*Subscribable)(nil)
	_ apis.Listable    = (*Subscribable)(nil)

	_ apis.Convertible = (*Subscribable)(nil)
	_ apis.Convertible = (*SubscribableSpec)(nil)
	_ apis.Convertible = (*SubscribableStatus)(nil)

	_ apis.Convertible = (*SubscriberSpec)(nil)
	_ apis.Convertible = (*SubscriberStatus)(nil)
)

// GetFullType implements duck.Implementable
func (s *Subscribable) GetFullType() duck.Populatable {
	return &Subscribable{}
}

// Populate implements duck.Populatable
func (c *Subscribable) Populate() {
	c.Spec.Subscribers = []SubscriberSpec{{
		UID:           "2f9b5e8e-deb6-11e8-9f32-f2801f1b9fd1",
		Generation:    1,
		SubscriberURI: apis.HTTP("call1"),
		ReplyURI:      apis.HTTP("sink2"),
	}, {
		UID:           "34c5aec8-deb6-11e8-9f32-f2801f1b9fd1",
		Generation:    2,
		SubscriberURI: apis.HTTP("call2"),
		ReplyURI:      apis.HTTP("sink2"),
	}}
	c.Status.Subscribers = // Populate ALL fields
		[]SubscriberStatus{{
			UID:                "2f9b5e8e-deb6-11e8-9f32-f2801f1b9fd1",
			ObservedGeneration: 1,
			Ready:              corev1.ConditionTrue,
			Message:            "Some message",
		}, {
			UID:                "34c5aec8-deb6-11e8-9f32-f2801f1b9fd1",
			ObservedGeneration: 2,
			Ready:              corev1.ConditionFalse,
			Message:            "Some message",
		}}
}

// GetListType implements apis.Listable
func (c *Subscribable) GetListType() runtime.Object {
	return &SubscribableList{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SubscribableTypeList is a list of SubscribableType resources
type SubscribableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Subscribable `json:"items"`
}
