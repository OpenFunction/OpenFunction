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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"

	eventingduckv1 "knative.dev/eventing/pkg/apis/duck/v1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=true

// Subscription routes events received on a Channel to a DNS name and
// corresponds to the subscriptions.channels.knative.dev CRD.
type Subscription struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              SubscriptionSpec   `json:"spec"`
	Status            SubscriptionStatus `json:"status,omitempty"`
}

var (
	// Check that Subscription can be validated, can be defaulted, and has immutable fields.
	_ apis.Validatable = (*Subscription)(nil)
	_ apis.Defaultable = (*Subscription)(nil)

	// Check that Subscription can return its spec untyped.
	_ apis.HasSpec = (*Subscription)(nil)

	_ runtime.Object = (*Subscription)(nil)

	// Check that we can create OwnerReferences to a Subscription.
	_ kmeta.OwnerRefable = (*Subscription)(nil)

	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*Subscription)(nil)
)

// SubscriptionSpec specifies the Channel for incoming events, a Subscriber target
// for processing those events and where to put the result of the processing. Only
// From (where the events are coming from) is always required. You can optionally
// only Process the events (results in no output events) by leaving out the Result.
// You can also perform an identity transformation on the incoming events by leaving
// out the Subscriber and only specifying Result.
//
// The following are all valid specifications:
// channel --[subscriber]--> reply
// Sink, no outgoing events:
// channel -- subscriber
// no-op function (identity transformation):
// channel --> reply
type SubscriptionSpec struct {
	// Reference to a channel that will be used to create the subscription
	// You can specify only the following fields of the ObjectReference:
	//   - Kind
	//   - APIVersion
	//   - Name
	//  The resource pointed by this ObjectReference must meet the
	// contract to the ChannelableSpec duck type. If the resource does not
	// meet this contract it will be reflected in the Subscription's status.
	//
	// This field is immutable. We have no good answer on what happens to
	// the events that are currently in the channel being consumed from
	// and what the semantics there should be. For now, you can always
	// delete the Subscription and recreate it to point to a different
	// channel, giving the user more control over what semantics should
	// be used (drain the channel first, possibly have events dropped,
	// etc.)
	Channel corev1.ObjectReference `json:"channel"`

	// Subscriber is reference to (optional) function for processing events.
	// Events from the Channel will be delivered here and replies are
	// sent to a Destination as specified by the Reply.
	// +optional
	Subscriber *duckv1.Destination `json:"subscriber,omitempty"`

	// Reply specifies (optionally) how to handle events returned from
	// the Subscriber target.
	// +optional
	Reply *duckv1.Destination `json:"reply,omitempty"`

	// Delivery configuration
	// +optional
	Delivery *eventingduckv1.DeliverySpec `json:"delivery,omitempty"`
}

// SubscriptionStatus (computed) for a subscription
type SubscriptionStatus struct {
	// inherits duck/v1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last processed by the controller.
	// * Conditions - the latest available observations of a resource's current state.
	duckv1.Status `json:",inline"`

	// PhysicalSubscription is the fully resolved values that this Subscription represents.
	PhysicalSubscription SubscriptionStatusPhysicalSubscription `json:"physicalSubscription,omitempty"`
}

// SubscriptionStatusPhysicalSubscription represents the fully resolved values for this
// Subscription.
type SubscriptionStatusPhysicalSubscription struct {
	// SubscriberURI is the fully resolved URI for spec.subscriber.
	SubscriberURI *apis.URL `json:"subscriberUri,omitempty"`

	// ReplyURI is the fully resolved URI for the spec.reply.
	ReplyURI *apis.URL `json:"replyUri,omitempty"`

	// ReplyURI is the fully resolved URI for the spec.delivery.deadLetterSink.
	DeadLetterSinkURI *apis.URL `json:"deadLetterSinkUri,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SubscriptionList returned in list operations
type SubscriptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Subscription `json:"items"`
}

// GetGroupVersionKind returns GroupVersionKind for Subscriptions
func (*Subscription) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Subscription")
}

// GetUntypedSpec returns the spec of the Subscription.
func (s *Subscription) GetUntypedSpec() interface{} {
	return s.Spec
}

// GetStatus retrieves the status of the Subscription. Implements the KRShaped interface.
func (s *Subscription) GetStatus() *duckv1.Status {
	return &s.Status.Status
}
