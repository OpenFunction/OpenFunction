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
	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// +genduck
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Channelable is a skeleton type wrapping Subscribable and Addressable in the manner we expect resource writers
// defining compatible resources to embed it. We will typically use this type to deserialize
// Channelable ObjectReferences and access their subscription and address data.  This is not a real resource.
type Channelable struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the part where the Channelable fulfills the Subscribable contract.
	Spec ChannelableSpec `json:"spec,omitempty"`

	Status ChannelableStatus `json:"status,omitempty"`
}

// ChannelableSpec contains Spec of the Channelable object
type ChannelableSpec struct {
	SubscribableSpec `json:",inline"`

	// DeliverySpec contains options controlling the event delivery
	// +optional
	Delivery *DeliverySpec `json:"delivery,omitempty"`
}

// ChannelableStatus contains the Status of a Channelable object.
type ChannelableStatus struct {
	// inherits duck/v1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last processed by the controller.
	// * Conditions - the latest available observations of a resource's current state.
	duckv1.Status `json:",inline"`
	// AddressStatus is the part where the Channelable fulfills the Addressable contract.
	duckv1.AddressStatus `json:",inline"`
	// Subscribers is populated with the statuses of each of the Channelable's subscribers.
	SubscribableStatus `json:",inline"`
	// DeadLetterChannel is a KReference and is set by the channel when it supports native error handling via a channel
	// Failed messages are delivered here.
	// +optional
	DeadLetterChannel *duckv1.KReference `json:"deadLetterChannel,omitempty"`
}

var (
	// Verify Channelable resources meet duck contracts.
	_ duck.Populatable   = (*Channelable)(nil)
	_ duck.Implementable = (*Channelable)(nil)
	_ apis.Listable      = (*Channelable)(nil)
)

// Populate implements duck.Populatable
func (c *Channelable) Populate() {
	c.Spec.SubscribableSpec = SubscribableSpec{
		// Populate ALL fields
		Subscribers: []SubscriberSpec{{
			UID:           "2f9b5e8e-deb6-11e8-9f32-f2801f1b9fd1",
			Generation:    1,
			SubscriberURI: apis.HTTP("call1"),
			ReplyURI:      apis.HTTP("sink2"),
		}, {
			UID:           "34c5aec8-deb6-11e8-9f32-f2801f1b9fd1",
			Generation:    2,
			SubscriberURI: apis.HTTP("call2"),
			ReplyURI:      apis.HTTP("sink2"),
		}},
	}
	retry := int32(5)
	linear := BackoffPolicyLinear
	delay := "5s"
	c.Spec.Delivery = &DeliverySpec{
		DeadLetterSink: &duckv1.Destination{
			Ref: &duckv1.KReference{
				Name: "aname",
			},
			URI: &apis.URL{
				Scheme: "http",
				Host:   "test-error-domain",
			},
		},
		Retry:         &retry,
		BackoffPolicy: &linear,
		BackoffDelay:  &delay,
	}
	c.Status = ChannelableStatus{
		AddressStatus: duckv1.AddressStatus{
			Address: &duckv1.Addressable{
				URL: &apis.URL{
					Scheme: "http",
					Host:   "test-domain",
				},
			},
		},
		SubscribableStatus: SubscribableStatus{
			Subscribers: []SubscriberStatus{{
				UID:                "2f9b5e8e-deb6-11e8-9f32-f2801f1b9fd1",
				ObservedGeneration: 1,
				Ready:              corev1.ConditionTrue,
				Message:            "Some message",
			}, {
				UID:                "34c5aec8-deb6-11e8-9f32-f2801f1b9fd1",
				ObservedGeneration: 2,
				Ready:              corev1.ConditionFalse,
				Message:            "Some message",
			}},
		},
	}
}

// GetFullType implements duck.Implementable
func (s *Channelable) GetFullType() duck.Populatable {
	return &Channelable{}
}

// GetListType implements apis.Listable
func (c *Channelable) GetListType() runtime.Object {
	return &ChannelableList{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ChannelableList is a list of Channelable resources.
type ChannelableList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Channelable `json:"items"`
}
