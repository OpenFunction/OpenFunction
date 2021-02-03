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
	"knative.dev/pkg/apis"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=true

// PingSource is the Schema for the PingSources API.
type PingSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PingSourceSpec   `json:"spec,omitempty"`
	Status PingSourceStatus `json:"status,omitempty"`
}

// Check the interfaces that PingSource should be implementing.
var (
	_ runtime.Object     = (*PingSource)(nil)
	_ kmeta.OwnerRefable = (*PingSource)(nil)
	_ apis.Validatable   = (*PingSource)(nil)
	_ apis.Defaultable   = (*PingSource)(nil)
	_ apis.HasSpec       = (*PingSource)(nil)
	_ duckv1.KRShaped    = (*PingSource)(nil)
)

// PingSourceSpec defines the desired state of the PingSource.
type PingSourceSpec struct {
	// inherits duck/v1 SourceSpec, which currently provides:
	// * Sink - a reference to an object that will resolve to a domain name or
	//   a URI directly to use as the sink.
	// * CloudEventOverrides - defines overrides to control the output format
	//   and modifications of the event sent to the sink.
	duckv1.SourceSpec `json:",inline"`

	// Schedule is the cronjob schedule. Defaults to `* * * * *`.
	// +optional
	Schedule string `json:"schedule,omitempty"`

	// Timezone modifies the actual time relative to the specified timezone.
	// Defaults to the system time zone.
	// More general information about time zones: https://www.iana.org/time-zones
	// List of valid timezone values: https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
	Timezone string `json:"timezone,omitempty"`

	// JsonData is json encoded data used as the body of the event posted to
	// the sink. Default is empty. If set, datacontenttype will also be set
	// to "application/json".
	// +optional
	JsonData string `json:"jsonData,omitempty"`
}

// PingSourceStatus defines the observed state of PingSource.
type PingSourceStatus struct {
	// inherits duck/v1 SourceStatus, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last
	//   processed by the controller.
	// * Conditions - the latest available observations of a resource's current
	//   state.
	// * SinkURI - the current active sink URI that has been configured for the
	//   Source.
	duckv1.SourceStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PingSourceList contains a list of PingSources.
type PingSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PingSource `json:"items"`
}

// GetStatus retrieves the status of the PingSource. Implements the KRShaped interface.
func (p *PingSource) GetStatus() *duckv1.Status {
	return &p.Status.Status
}
