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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	eventingduckv1beta1 "knative.dev/eventing/pkg/apis/duck/v1beta1"
	messagingv1beta1 "knative.dev/eventing/pkg/apis/messaging/v1beta1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Parallel defines conditional branches that will be wired in
// series through Channels and Subscriptions.
type Parallel struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the Parallel.
	Spec ParallelSpec `json:"spec,omitempty"`

	// Status represents the current state of the Parallel. This data may be out of
	// date.
	// +optional
	Status ParallelStatus `json:"status,omitempty"`
}

var (
	// Check that Parallel can be validated and defaulted.
	_ apis.Validatable = (*Parallel)(nil)
	_ apis.Defaultable = (*Parallel)(nil)

	// Check that Parallel can return its spec untyped.
	_ apis.HasSpec = (*Parallel)(nil)

	// TODO: make appropriate fields immutable.
	//_ apis.Immutable = (*Parallel)(nil)

	_ runtime.Object = (*Parallel)(nil)

	// Check that we can create OwnerReferences to a Parallel.
	_ kmeta.OwnerRefable = (*Parallel)(nil)

	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*Parallel)(nil)
)

type ParallelSpec struct {
	// Branches is the list of Filter/Subscribers pairs.
	Branches []ParallelBranch `json:"branches"`

	// ChannelTemplate specifies which Channel CRD to use. If left unspecified, it is set to the default Channel CRD
	// for the namespace (or cluster, in case there are no defaults for the namespace).
	// +optional
	ChannelTemplate *messagingv1beta1.ChannelTemplateSpec `json:"channelTemplate"`

	// Reply is a Reference to where the result of a case Subscriber gets sent to
	// when the case does not have a Reply
	// +optional
	Reply *duckv1.Destination `json:"reply,omitempty"`
}

type ParallelBranch struct {
	// Filter is the expression guarding the branch
	// +optional
	Filter *duckv1.Destination `json:"filter,omitempty"`

	// Subscriber receiving the event when the filter passes
	Subscriber duckv1.Destination `json:"subscriber"`

	// Reply is a Reference to where the result of Subscriber of this case gets sent to.
	// If not specified, sent the result to the Parallel Reply
	// +optional
	Reply *duckv1.Destination `json:"reply,omitempty"`

	// Delivery is the delivery specification for events to the subscriber
	// This includes things like retries, DLQ, etc.
	// Needed for Roundtripping v1alpha1 <-> v1beta1.
	// +optional
	Delivery *eventingduckv1beta1.DeliverySpec `json:"delivery,omitempty"`
}

// ParallelStatus represents the current state of a Parallel.
type ParallelStatus struct {
	// inherits duck/v1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last processed by the controller.
	// * Conditions - the latest available observations of a resource's current state.
	duckv1.Status `json:",inline"`

	// IngressChannelStatus corresponds to the ingress channel status.
	IngressChannelStatus ParallelChannelStatus `json:"ingressChannelStatus"`

	// BranchStatuses is an array of corresponding to branch statuses.
	// Matches the Spec.Branches array in the order.
	BranchStatuses []ParallelBranchStatus `json:"branchStatuses"`

	// AddressStatus is the starting point to this Parallel. Sending to this
	// will target the first subscriber.
	// It generally has the form {channel}.{namespace}.svc.{cluster domain name}
	duckv1.AddressStatus `json:",inline"`
}

// ParallelBranchStatus represents the current state of a Parallel branch
type ParallelBranchStatus struct {
	// FilterSubscriptionStatus corresponds to the filter subscription status.
	FilterSubscriptionStatus ParallelSubscriptionStatus `json:"filterSubscriptionStatus"`

	// FilterChannelStatus corresponds to the filter channel status.
	FilterChannelStatus ParallelChannelStatus `json:"filterChannelStatus"`

	// SubscriptionStatus corresponds to the subscriber subscription status.
	SubscriptionStatus ParallelSubscriptionStatus `json:"subscriberSubscriptionStatus"`
}

type ParallelChannelStatus struct {
	// Channel is the reference to the underlying channel.
	Channel corev1.ObjectReference `json:"channel"`

	// ReadyCondition indicates whether the Channel is ready or not.
	ReadyCondition apis.Condition `json:"ready"`
}

type ParallelSubscriptionStatus struct {
	// Subscription is the reference to the underlying Subscription.
	Subscription corev1.ObjectReference `json:"subscription"`

	// ReadyCondition indicates whether the Subscription is ready or not.
	ReadyCondition apis.Condition `json:"ready"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ParallelList is a collection of Parallels.
type ParallelList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Parallel `json:"items"`
}

// GetStatus retrieves the status of the Parallel. Implements the KRShaped interface.
func (p *Parallel) GetStatus() *duckv1.Status {
	return &p.Status.Status
}
