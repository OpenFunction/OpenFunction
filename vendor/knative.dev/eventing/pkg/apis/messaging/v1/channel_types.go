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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	eventingduckv1 "knative.dev/eventing/pkg/apis/duck/v1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Channel represents a generic Channel. It is normally used when we want a Channel, but don't need a specific Channel implementation.
type Channel struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the Channel.
	Spec ChannelSpec `json:"spec,omitempty"`

	// Status represents the current state of the Channel. This data may be out of
	// date.
	// +optional
	Status ChannelStatus `json:"status,omitempty"`
}

var (
	// Check that Channel can be validated and defaulted.
	_ apis.Validatable = (*Channel)(nil)
	_ apis.Defaultable = (*Channel)(nil)

	// Check that Channel can return its spec untyped.
	_ apis.HasSpec = (*Channel)(nil)

	_ runtime.Object = (*Channel)(nil)

	// Check that we can create OwnerReferences to a Channel.
	_ kmeta.OwnerRefable = (*Channel)(nil)

	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*Channel)(nil)
)

// ChannelSpec defines which subscribers have expressed interest in receiving events from this Channel.
// It also defines the ChannelTemplate to use in order to create the CRD Channel backing this Channel.
type ChannelSpec struct {
	// ChannelTemplate specifies which Channel CRD to use to create the CRD Channel backing this Channel.
	// This is immutable after creation. Normally this is set by the Channel defaulter, not directly by the user.
	ChannelTemplate *ChannelTemplateSpec `json:"channelTemplate"`

	// Channel conforms to ChannelableSpec
	eventingduckv1.ChannelableSpec `json:",inline"`
}

// ChannelStatus represents the current state of a Channel.
type ChannelStatus struct {
	// Channel conforms to ChannelableStatus
	eventingduckv1.ChannelableStatus `json:",inline"`

	// Channel is an KReference to the Channel CRD backing this Channel.
	Channel *duckv1.KReference `json:"channel,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ChannelList is a collection of Channels.
type ChannelList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Channel `json:"items"`
}

// GetStatus retrieves the status of the Channel. Implements the KRShaped interface.
func (t *Channel) GetStatus() *duckv1.Status {
	return &t.Status.Status
}
