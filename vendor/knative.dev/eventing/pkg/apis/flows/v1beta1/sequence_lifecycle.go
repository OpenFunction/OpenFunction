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
	"k8s.io/apimachinery/pkg/runtime/schema"
	eventingduckv1beta1 "knative.dev/eventing/pkg/apis/duck/v1beta1"
	messagingv1beta1 "knative.dev/eventing/pkg/apis/messaging/v1beta1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

var sCondSet = apis.NewLivingConditionSet(SequenceConditionReady, SequenceConditionChannelsReady, SequenceConditionSubscriptionsReady, SequenceConditionAddressable)

const (
	// SequenceConditionReady has status True when all subconditions below have been set to True.
	SequenceConditionReady = apis.ConditionReady

	// SequenceConditionChannelsReady has status True when all the channels created as part of
	// this sequence are ready.
	SequenceConditionChannelsReady apis.ConditionType = "ChannelsReady"

	// SequenceConditionSubscriptionsReady has status True when all the subscriptions created as part of
	// this sequence are ready.
	SequenceConditionSubscriptionsReady apis.ConditionType = "SubscriptionsReady"

	// SequenceConditionAddressable has status true when this Sequence meets
	// the Addressable contract and has a non-empty hostname.
	SequenceConditionAddressable apis.ConditionType = "Addressable"
)

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (*Sequence) GetConditionSet() apis.ConditionSet {
	return sCondSet
}

// GetGroupVersionKind returns GroupVersionKind for InMemoryChannels
func (*Sequence) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Sequence")
}

// GetUntypedSpec returns the spec of the Sequence.
func (s *Sequence) GetUntypedSpec() interface{} {
	return s.Spec
}

// GetCondition returns the condition currently associated with the given type, or nil.
func (ss *SequenceStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return sCondSet.Manage(ss).GetCondition(t)
}

// IsReady returns true if the resource is ready overall.
func (ss *SequenceStatus) IsReady() bool {
	return sCondSet.Manage(ss).IsHappy()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (ss *SequenceStatus) InitializeConditions() {
	sCondSet.Manage(ss).InitializeConditions()
}

// PropagateSubscriptionStatuses sets the SubscriptionStatuses and SequenceConditionSubscriptionsReady based on
// the status of the incoming subscriptions.
func (ss *SequenceStatus) PropagateSubscriptionStatuses(subscriptions []*messagingv1beta1.Subscription) {
	ss.SubscriptionStatuses = make([]SequenceSubscriptionStatus, len(subscriptions))
	allReady := true
	// If there are no subscriptions, treat that as a False case. Could go either way, but this seems right.
	if len(subscriptions) == 0 {
		allReady = false

	}
	for i, s := range subscriptions {
		ss.SubscriptionStatuses[i] = SequenceSubscriptionStatus{
			Subscription: corev1.ObjectReference{
				APIVersion: s.APIVersion,
				Kind:       s.Kind,
				Name:       s.Name,
				Namespace:  s.Namespace,
			},
		}
		readyCondition := s.Status.GetCondition(messagingv1beta1.SubscriptionConditionReady)
		if readyCondition != nil {
			ss.SubscriptionStatuses[i].ReadyCondition = *readyCondition
			if readyCondition.Status != corev1.ConditionTrue {
				allReady = false
			}
		} else {
			allReady = false
		}

	}
	if allReady {
		sCondSet.Manage(ss).MarkTrue(SequenceConditionSubscriptionsReady)
	} else {
		ss.MarkSubscriptionsNotReady("SubscriptionsNotReady", "Subscriptions are not ready yet, or there are none")
	}
}

// PropagateChannelStatuses sets the ChannelStatuses and SequenceConditionChannelsReady based on the
// status of the incoming channels.
func (ss *SequenceStatus) PropagateChannelStatuses(channels []*eventingduckv1beta1.Channelable) {
	ss.ChannelStatuses = make([]SequenceChannelStatus, len(channels))
	allReady := true
	// If there are no channels, treat that as a False case. Could go either way, but this seems right.
	if len(channels) == 0 {
		allReady = false

	}
	for i, c := range channels {
		ss.ChannelStatuses[i] = SequenceChannelStatus{
			Channel: corev1.ObjectReference{
				APIVersion: c.APIVersion,
				Kind:       c.Kind,
				Name:       c.Name,
				Namespace:  c.Namespace,
			},
		}
		// TODO: Once the addressable has a real status to dig through, use that here instead of
		// addressable, because it might be addressable but not ready.
		address := c.Status.AddressStatus.Address
		if address != nil {
			ss.ChannelStatuses[i].ReadyCondition = apis.Condition{Type: apis.ConditionReady, Status: corev1.ConditionTrue}
		} else {
			ss.ChannelStatuses[i].ReadyCondition = apis.Condition{Type: apis.ConditionReady, Status: corev1.ConditionFalse, Reason: "NotAddressable", Message: "Channel is not addressable"}
			allReady = false
		}

		// Mark the Sequence address as the Address of the first channel.
		if i == 0 {
			ss.setAddress(address)
		}
	}
	if allReady {
		sCondSet.Manage(ss).MarkTrue(SequenceConditionChannelsReady)
	} else {
		ss.MarkChannelsNotReady("ChannelsNotReady", "Channels are not ready yet, or there are none")
	}
}

func (ss *SequenceStatus) MarkChannelsNotReady(reason, messageFormat string, messageA ...interface{}) {
	sCondSet.Manage(ss).MarkFalse(SequenceConditionChannelsReady, reason, messageFormat, messageA...)
}

func (ss *SequenceStatus) MarkSubscriptionsNotReady(reason, messageFormat string, messageA ...interface{}) {
	sCondSet.Manage(ss).MarkFalse(SequenceConditionSubscriptionsReady, reason, messageFormat, messageA...)
}

func (ss *SequenceStatus) MarkAddressableNotReady(reason, messageFormat string, messageA ...interface{}) {
	sCondSet.Manage(ss).MarkFalse(SequenceConditionAddressable, reason, messageFormat, messageA...)
}

func (ss *SequenceStatus) setAddress(address *duckv1.Addressable) {
	if address == nil || address.URL == nil {
		sCondSet.Manage(ss).MarkFalse(SequenceConditionAddressable, "emptyAddress", "addressable is nil")
	} else {
		ss.AddressStatus.Address = &duckv1.Addressable{URL: address.URL}
		sCondSet.Manage(ss).MarkTrue(SequenceConditionAddressable)
	}
}
