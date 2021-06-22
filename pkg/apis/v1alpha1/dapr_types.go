/*
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
	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	subscriptionsv1alpha1 "github.com/dapr/dapr/pkg/apis/subscriptions/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
)

type KedaScaledObject struct {
	// How to run the function, known values are Deployment or StatefulSet, default is Deployment.
	WorkloadType string `json:"workloadType,omitempty"`
	// +optional
	PollingInterval *int32 `json:"pollingInterval,omitempty"`
	// +optional
	CooldownPeriod *int32 `json:"cooldownPeriod,omitempty"`
	// +optional
	MinReplicaCount *int32 `json:"minReplicaCount,omitempty"`
	// +optional
	MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty"`
	// +optional
	Advanced *kedav1alpha1.AdvancedConfig `json:"advanced,omitempty"`

	Triggers []kedav1alpha1.ScaleTriggers `json:"triggers"`
}

type KedaScaledJob struct {
	// +optional
	PollingInterval *int32 `json:"pollingInterval,omitempty"`
	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`
	// +optional
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty"`
	// +optional
	MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty"`
	// +optional
	ScalingStrategy kedav1alpha1.ScalingStrategy `json:"scalingStrategy,omitempty"`

	Triggers []kedav1alpha1.ScaleTriggers `json:"triggers"`
}

type DaprInput struct {
	// Input name, the name of Dapr component. The component must be created in k8s cluster,
	// or defined in the `components` or `subscriptions`.
	Name string `json:"name"`
	// Input type, known values are bindings, pubsub, invoke.
	// bindings: Indicates that the input is the Dapr bindings component.
	// pubsub: Indicates that the input is the Dapr pubsub component.
	// invoke: Indicates that the input is the Dapr service invocation component.
	Type string `json:"type"`
	// Input serving listening path.
	// if the type is bindings, pattern is the name of component.
	// if the type is pubsub, pattern is the topic of subscription.
	// If the type is invoke, pattern is the name of function.
	Pattern string `json:"pattern"`
}

type DaprOutput struct {
	// Output name, the name of Dapr component. The component must be created in k8s cluster,
	// or defined in the `components` or `subscriptions`.
	Name string `json:"name"`
	// Output type, known values are bindings, pubsub, invoke.
	// bindings: Indicates that the input is the Dapr bindings component.
	// pubsub: Indicates that the input is the Dapr pubsub component.
	// invoke: Indicates that the input is the Dapr service invocation component.
	Type string `json:"type"`
	// Output serving listening path.
	// if the type is bindings, pattern is the name of component.
	// if the type is pubsub, pattern is the topic of subscription.
	// If the type is invoke, pattern is the name of function.
	Pattern string `json:"pattern"`
	// Parameters for output.
	Params map[string]string `json:"params,omitempty"`
}

type DaprComponent struct {
	Name                             string `json:"name"`
	componentsv1alpha1.ComponentSpec `json:",inline"`
}

type DaprSubscription struct {
	Name                                   string `json:"name"`
	subscriptionsv1alpha1.SubscriptionSpec `json:",inline"`
	// +optional
	Scopes []string `json:"scopes,omitempty"`
}

type DaprRuntime struct {
	// Annotations for dapr
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// Function serving kind, known values are HTTP, gRPC default value is http.
	// +optional
	Protocol string `json:"protocol,omitempty"`
	// Components of dapr
	// +optional
	Components []DaprComponent `json:"components,omitempty"`
	// Subscriptions of dapr
	// +optional
	Subscriptions []DaprSubscription `json:"subscriptions,omitempty"`

	// Function input from bindings data
	// +optional
	Input *DaprInput `json:"input,omitempty"`
	// Function output to bindings data
	// +optional
	Output []DaprOutput `json:"output,omitempty"`

	// One of scaledObject and scaledJob can not be nil.
	// +optional
	ScaledObject *KedaScaledObject `json:"scaledObject,omitempty"`
	// +optional
	ScaledJob *KedaScaledJob `json:"scaledJob,omitempty"`
}
