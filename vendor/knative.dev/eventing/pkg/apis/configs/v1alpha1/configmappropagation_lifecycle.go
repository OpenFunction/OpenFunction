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

package v1alpha1

import (
	"knative.dev/pkg/apis"
)

var configMapPropagationCondSet = apis.NewLivingConditionSet(ConfigMapPropagationConditionReady, ConfigMapPropagationConditionPropagated)

const (
	// ConfigMapPropagationConditionReady has status True when all subconditions below have been set to True.
	ConfigMapPropagationConditionReady = apis.ConditionReady
	// ConfigMapPropagationConditionPropagated has status True when the ConfigMaps in original namespace are all propagated to target namespace.
	ConfigMapPropagationConditionPropagated apis.ConditionType = "Propagated"
)

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (*ConfigMapPropagation) GetConditionSet() apis.ConditionSet {
	return configMapPropagationCondSet
}

// GetCondition returns the condition currently associated with the given type, or nil.
func (cmps *ConfigMapPropagationStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return configMapPropagationCondSet.Manage(cmps).GetCondition(t)
}

// IsReady returns true if the resource is ready overall.
func (cmps *ConfigMapPropagationStatus) IsReady() bool {
	return configMapPropagationCondSet.Manage(cmps).IsHappy()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (cmps *ConfigMapPropagationStatus) InitializeConditions() {
	configMapPropagationCondSet.Manage(cmps).InitializeConditions()
}

func (cmps *ConfigMapPropagationStatus) MarkPropagated() {
	configMapPropagationCondSet.Manage(cmps).MarkTrue(ConfigMapPropagationConditionPropagated)
}

func (cmps *ConfigMapPropagationStatus) MarkNotPropagated() {
	configMapPropagationCondSet.Manage(cmps).MarkFalse(ConfigMapPropagationConditionPropagated, "PropagationFailed",
		"ConfigMapPropagation could not fully propagate ConfigMaps from original namespace to current namespace")
}

func (cmpsc *ConfigMapPropagationStatusCopyConfigMap) SetCopyConfigMapStatus(name, source, operation, ready, reason, resourceVersion string) {
	cmpsc.Name = name
	cmpsc.Source = source
	cmpsc.Operation = operation
	cmpsc.Ready = ready
	cmpsc.Reason = reason
	cmpsc.ResourceVersion = resourceVersion
}
