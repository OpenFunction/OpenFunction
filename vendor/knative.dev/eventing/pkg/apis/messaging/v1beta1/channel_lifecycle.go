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
	eventingduck "knative.dev/eventing/pkg/apis/duck/v1beta1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

var chCondSet = apis.NewLivingConditionSet(ChannelConditionBackingChannelReady, ChannelConditionAddressable)

const (
	// ChannelConditionReady has status True when all subconditions below have been set to True.
	ChannelConditionReady = apis.ConditionReady

	// ChannelConditionBackingChannelReady has status True when the backing Channel CRD is ready.
	ChannelConditionBackingChannelReady apis.ConditionType = "BackingChannelReady"

	// ChannelConditionAddressable has status true when this Channel meets
	// the Addressable contract and has a non-empty hostname.
	ChannelConditionAddressable apis.ConditionType = "Addressable"
)

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (*Channel) GetConditionSet() apis.ConditionSet {
	return chCondSet
}

// GetGroupVersionKind returns GroupVersionKind for Channels.
func (*Channel) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Channel")
}

// GetUntypedSpec returns the spec of the Channel.
func (c *Channel) GetUntypedSpec() interface{} {
	return c.Spec
}

// GetCondition returns the condition currently associated with the given type, or nil.
func (cs *ChannelStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return chCondSet.Manage(cs).GetCondition(t)
}

// GetTopLevelCondition returns the top level Condition.
func (cs *ChannelStatus) GetTopLevelCondition() *apis.Condition {
	return chCondSet.Manage(cs).GetTopLevelCondition()
}

// IsReady returns true if the resource is ready overall.
func (cs *ChannelStatus) IsReady() bool {
	return chCondSet.Manage(cs).IsHappy()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (cs *ChannelStatus) InitializeConditions() {
	chCondSet.Manage(cs).InitializeConditions()
}

func (cs *ChannelStatus) SetAddress(address *duckv1.Addressable) {
	cs.Address = address
	if address == nil || address.URL.IsEmpty() {
		chCondSet.Manage(cs).MarkFalse(ChannelConditionAddressable, "EmptyHostname", "hostname is the empty string")
	} else {
		chCondSet.Manage(cs).MarkTrue(ChannelConditionAddressable)

	}
}

func (cs *ChannelStatus) MarkBackingChannelFailed(reason, messageFormat string, messageA ...interface{}) {
	chCondSet.Manage(cs).MarkFalse(ChannelConditionBackingChannelReady, reason, messageFormat, messageA...)
}

func (cs *ChannelStatus) MarkBackingChannelUnknown(reason, messageFormat string, messageA ...interface{}) {
	chCondSet.Manage(cs).MarkUnknown(ChannelConditionBackingChannelReady, reason, messageFormat, messageA...)
}

func (cs *ChannelStatus) MarkBackingChannelNotConfigured() {
	chCondSet.Manage(cs).MarkUnknown(ChannelConditionBackingChannelReady,
		"BackingChannelNotConfigured", "BackingChannel has not yet been reconciled.")
}

func (cs *ChannelStatus) MarkBackingChannelReady() {
	chCondSet.Manage(cs).MarkTrue(ChannelConditionBackingChannelReady)
}

func (cs *ChannelStatus) PropagateStatuses(chs *eventingduck.ChannelableStatus) {
	// TODO: Once you can get a Ready status from Channelable in a generic way, use it here.
	readyCondition := chs.Status.GetCondition(apis.ConditionReady)
	if readyCondition == nil {
		cs.MarkBackingChannelNotConfigured()
	} else {
		switch {
		case readyCondition.Status == corev1.ConditionUnknown:
			cs.MarkBackingChannelUnknown(readyCondition.Reason, readyCondition.Message)
		case readyCondition.Status == corev1.ConditionTrue:
			cs.MarkBackingChannelReady()
		case readyCondition.Status == corev1.ConditionFalse:
			cs.MarkBackingChannelFailed(readyCondition.Reason, readyCondition.Message)
		default:
			cs.MarkBackingChannelUnknown("BackingChannelUnknown", "The status of BackingChannel is invalid: %v", readyCondition.Status)
		}
	}
	// Set the address and update the Addressable conditions.
	cs.SetAddress(chs.AddressStatus.Address)
	// Set the subscribable status.
	cs.SubscribableStatus = chs.SubscribableStatus
}
