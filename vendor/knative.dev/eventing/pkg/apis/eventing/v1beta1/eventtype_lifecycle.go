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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"

	"knative.dev/pkg/apis"
)

var eventTypeCondSet = apis.NewLivingConditionSet(EventTypeConditionBrokerExists, EventTypeConditionBrokerReady)

const (
	EventTypeConditionReady                           = apis.ConditionReady
	EventTypeConditionBrokerExists apis.ConditionType = "BrokerExists"
	EventTypeConditionBrokerReady  apis.ConditionType = "BrokerReady"
)

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (*EventType) GetConditionSet() apis.ConditionSet {
	return eventTypeCondSet
}

// GetCondition returns the condition currently associated with the given type, or nil.
func (et *EventTypeStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return eventTypeCondSet.Manage(et).GetCondition(t)
}

// IsReady returns true if the resource is ready overall.
func (et *EventTypeStatus) IsReady() bool {
	return eventTypeCondSet.Manage(et).IsHappy()
}

// GetTopLevelCondition returns the top level Condition.
func (et *EventTypeStatus) GetTopLevelCondition() *apis.Condition {
	return eventTypeCondSet.Manage(et).GetTopLevelCondition()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (et *EventTypeStatus) InitializeConditions() {
	eventTypeCondSet.Manage(et).InitializeConditions()
}

func (et *EventTypeStatus) MarkBrokerExists() {
	eventTypeCondSet.Manage(et).MarkTrue(EventTypeConditionBrokerExists)
}

func (et *EventTypeStatus) MarkBrokerDoesNotExist() {
	eventTypeCondSet.Manage(et).MarkFalse(EventTypeConditionBrokerExists, "BrokerDoesNotExist", "Broker does not exist")
}

func (et *EventTypeStatus) MarkBrokerExistsUnknown(reason, messageFormat string, messageA ...interface{}) {
	eventTypeCondSet.Manage(et).MarkUnknown(EventTypeConditionBrokerExists, reason, messageFormat, messageA...)
}

func (et *EventTypeStatus) MarkBrokerReady() {
	eventTypeCondSet.Manage(et).MarkTrue(EventTypeConditionBrokerReady)
}

func (et *EventTypeStatus) MarkBrokerFailed(reason, messageFormat string, messageA ...interface{}) {
	eventTypeCondSet.Manage(et).MarkFalse(EventTypeConditionBrokerReady, reason, messageFormat, messageA...)
}

func (et *EventTypeStatus) MarkBrokerUnknown(reason, messageFormat string, messageA ...interface{}) {
	eventTypeCondSet.Manage(et).MarkUnknown(EventTypeConditionBrokerReady, reason, messageFormat, messageA...)
}

func (et *EventTypeStatus) MarkBrokerNotConfigured() {
	eventTypeCondSet.Manage(et).MarkUnknown(EventTypeConditionBrokerReady,
		"BrokerNotConfigured", "Broker has not yet been reconciled.")
}

func (et *EventTypeStatus) PropagateBrokerStatus(bs *BrokerStatus) {
	bc := bs.GetConditionSet().Manage(bs).GetTopLevelCondition()
	if bc == nil {
		et.MarkBrokerNotConfigured()
		return
	}
	switch {
	case bc.Status == corev1.ConditionUnknown:
		et.MarkBrokerUnknown(bc.Reason, bc.Message)
	case bc.Status == corev1.ConditionTrue:
		eventTypeCondSet.Manage(et).MarkTrue(EventTypeConditionBrokerReady)
	case bc.Status == corev1.ConditionFalse:
		et.MarkBrokerFailed(bc.Reason, bc.Message)
	default:
		et.MarkBrokerUnknown("BrokerUnknown", "The status of Broker is invalid: %v", bc.Status)
	}
}
