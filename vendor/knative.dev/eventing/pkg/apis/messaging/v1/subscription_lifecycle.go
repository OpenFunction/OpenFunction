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
	"knative.dev/pkg/apis"
)

// SubCondSet is a condition set with Ready as the happy condition and
// ReferencesResolved and ChannelReady as the dependent conditions.
var SubCondSet = apis.NewLivingConditionSet(SubscriptionConditionReferencesResolved, SubscriptionConditionAddedToChannel, SubscriptionConditionChannelReady)

const (
	// SubscriptionConditionReady has status True when all subconditions below have been set to True.
	SubscriptionConditionReady = apis.ConditionReady
	// SubscriptionConditionReferencesResolved has status True when all the specified references have been successfully
	// resolved.
	SubscriptionConditionReferencesResolved apis.ConditionType = "ReferencesResolved"

	// SubscriptionConditionAddedToChannel has status True when controller has successfully added a
	// subscription to the spec.channel resource.
	SubscriptionConditionAddedToChannel apis.ConditionType = "AddedToChannel"

	// SubscriptionConditionChannelReady has status True when the channel has marked the subscriber as 'ready'
	SubscriptionConditionChannelReady apis.ConditionType = "ChannelReady"
)

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (*Subscription) GetConditionSet() apis.ConditionSet {
	return SubCondSet
}

// GetCondition returns the condition currently associated with the given type, or nil.
func (ss *SubscriptionStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return SubCondSet.Manage(ss).GetCondition(t)
}

// GetTopLevelCondition returns the top level Condition.
func (ss *SubscriptionStatus) GetTopLevelCondition() *apis.Condition {
	return SubCondSet.Manage(ss).GetTopLevelCondition()
}

// IsReady returns true if the resource is ready overall.
func (ss *SubscriptionStatus) IsReady() bool {
	return SubCondSet.Manage(ss).IsHappy()
}

// IsAddedToChannel returns true if SubscriptionConditionAddedToChannel is true
func (ss *SubscriptionStatus) IsAddedToChannel() bool {
	return ss.GetCondition(SubscriptionConditionAddedToChannel).IsTrue()
}

// AreReferencesResolved returns true if SubscriptionConditionReferencesResolved is true
func (ss *SubscriptionStatus) AreReferencesResolved() bool {
	return ss.GetCondition(SubscriptionConditionReferencesResolved).IsTrue()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (ss *SubscriptionStatus) InitializeConditions() {
	SubCondSet.Manage(ss).InitializeConditions()
}

// MarkReferencesResolved sets the ReferencesResolved condition to True state.
func (ss *SubscriptionStatus) MarkReferencesResolved() {
	SubCondSet.Manage(ss).MarkTrue(SubscriptionConditionReferencesResolved)
}

// MarkChannelReady sets the ChannelReady condition to True state.
func (ss *SubscriptionStatus) MarkChannelReady() {
	SubCondSet.Manage(ss).MarkTrue(SubscriptionConditionChannelReady)
}

// MarkAddedToChannel sets the AddedToChannel condition to True state.
func (ss *SubscriptionStatus) MarkAddedToChannel() {
	SubCondSet.Manage(ss).MarkTrue(SubscriptionConditionAddedToChannel)
}

// MarkReferencesNotResolved sets the ReferencesResolved condition to False state.
func (ss *SubscriptionStatus) MarkReferencesNotResolved(reason, messageFormat string, messageA ...interface{}) {
	SubCondSet.Manage(ss).MarkFalse(SubscriptionConditionReferencesResolved, reason, messageFormat, messageA...)
}

// MarkReferencesResolvedUnknown sets the ReferencesResolved condition to Unknown state.
func (ss *SubscriptionStatus) MarkReferencesResolvedUnknown(reason, messageFormat string, messageA ...interface{}) {
	SubCondSet.Manage(ss).MarkUnknown(SubscriptionConditionReferencesResolved, reason, messageFormat, messageA...)
}

// MarkChannelFailed sets the ChannelReady condition to False state.
func (ss *SubscriptionStatus) MarkChannelFailed(reason, messageFormat string, messageA ...interface{}) {
	SubCondSet.Manage(ss).MarkFalse(SubscriptionConditionChannelReady, reason, messageFormat, messageA...)
}

// MarkChannelUnknown sets the ChannelReady condition to Unknown state.
func (ss *SubscriptionStatus) MarkChannelUnknown(reason, messageFormat string, messageA ...interface{}) {
	SubCondSet.Manage(ss).MarkUnknown(SubscriptionConditionChannelReady, reason, messageFormat, messageA...)
}

// MarkNotAddedToChannel sets the AddedToChannel condition to False state.
func (ss *SubscriptionStatus) MarkNotAddedToChannel(reason, messageFormat string, messageA ...interface{}) {
	SubCondSet.Manage(ss).MarkFalse(SubscriptionConditionAddedToChannel, reason, messageFormat, messageA...)
}
