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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"knative.dev/pkg/apis"
)

const (
	// ApiServerConditionReady has status True when the ApiServerSource is ready to send events.
	ApiServerConditionReady = apis.ConditionReady

	// ApiServerConditionSinkProvided has status True when the ApiServerSource has been configured with a sink target.
	ApiServerConditionSinkProvided apis.ConditionType = "SinkProvided"

	// ApiServerConditionDeployed has status True when the ApiServerSource has had it's deployment created.
	ApiServerConditionDeployed apis.ConditionType = "Deployed"

	// ApiServerConditionSufficientPermissions has status True when the ApiServerSource has sufficient permissions to access resources.
	ApiServerConditionSufficientPermissions apis.ConditionType = "SufficientPermissions"
)

var apiserverCondSet = apis.NewLivingConditionSet(
	ApiServerConditionSinkProvided,
	ApiServerConditionDeployed,
	ApiServerConditionSufficientPermissions,
)

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (*ApiServerSource) GetConditionSet() apis.ConditionSet {
	return apiserverCondSet
}

// GetGroupVersionKind returns the GroupVersionKind.
func (*ApiServerSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("ApiServerSource")
}

// GetUntypedSpec returns the spec of the ApiServerSource.
func (s *ApiServerSource) GetUntypedSpec() interface{} {
	return s.Spec
}

// GetCondition returns the condition currently associated with the given type, or nil.
func (s *ApiServerSourceStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return apiserverCondSet.Manage(s).GetCondition(t)
}

// GetTopLevelCondition returns the top level condition.
func (s *ApiServerSourceStatus) GetTopLevelCondition() *apis.Condition {
	return apiserverCondSet.Manage(s).GetTopLevelCondition()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *ApiServerSourceStatus) InitializeConditions() {
	apiserverCondSet.Manage(s).InitializeConditions()
}

// MarkSink sets the condition that the source has a sink configured.
func (s *ApiServerSourceStatus) MarkSink(uri *apis.URL) {
	s.SinkURI = uri
	if uri != nil {
		apiserverCondSet.Manage(s).MarkTrue(ApiServerConditionSinkProvided)
	} else {
		apiserverCondSet.Manage(s).MarkFalse(ApiServerConditionSinkProvided, "SinkEmpty", "Sink has resolved to empty.%s", "")
	}
}

// MarkNoSink sets the condition that the source does not have a sink configured.
func (s *ApiServerSourceStatus) MarkNoSink(reason, messageFormat string, messageA ...interface{}) {
	apiserverCondSet.Manage(s).MarkFalse(ApiServerConditionSinkProvided, reason, messageFormat, messageA...)
}

// PropagateDeploymentAvailability uses the availability of the provided Deployment to determine if
// ApiServerConditionDeployed should be marked as true or false.
func (s *ApiServerSourceStatus) PropagateDeploymentAvailability(d *appsv1.Deployment) {
	deploymentAvailableFound := false
	for _, cond := range d.Status.Conditions {
		if cond.Type == appsv1.DeploymentAvailable {
			deploymentAvailableFound = true
			if cond.Status == corev1.ConditionTrue {
				apiserverCondSet.Manage(s).MarkTrue(ApiServerConditionDeployed)
			} else if cond.Status == corev1.ConditionFalse {
				apiserverCondSet.Manage(s).MarkFalse(ApiServerConditionDeployed, cond.Reason, cond.Message)
			} else if cond.Status == corev1.ConditionUnknown {
				apiserverCondSet.Manage(s).MarkUnknown(ApiServerConditionDeployed, cond.Reason, cond.Message)
			}
		}
	}
	if !deploymentAvailableFound {
		apiserverCondSet.Manage(s).MarkUnknown(ApiServerConditionDeployed, "DeploymentUnavailable", "The Deployment '%s' is unavailable.", d.Name)
	}
}

// MarkSufficientPermissions sets the condition that the source has enough permissions to access the resources.
func (s *ApiServerSourceStatus) MarkSufficientPermissions() {
	apiserverCondSet.Manage(s).MarkTrue(ApiServerConditionSufficientPermissions)
}

// MarkNoSufficientPermissions sets the condition that the source does not have enough permissions to access the resources
func (s *ApiServerSourceStatus) MarkNoSufficientPermissions(reason, messageFormat string, messageA ...interface{}) {
	apiserverCondSet.Manage(s).MarkFalse(ApiServerConditionSufficientPermissions, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *ApiServerSourceStatus) IsReady() bool {
	return apiserverCondSet.Manage(s).IsHappy()
}
