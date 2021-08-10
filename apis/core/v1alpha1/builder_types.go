/*
Copyright 2021.

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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BuilderSpec defines the desired state of Builder
type BuilderSpec struct {
	// Params is a list of key/value that could be used to set strategy parameters.
	Params map[string]string `json:"params,omitempty"`
	// Environment params to pass to the builder.
	Env map[string]string `json:"env,omitempty"`
	// Builder refers to the image containing the build tools inside which
	// the source code would be built.
	//
	// +optional
	Builder *string `json:"builder"`
	// BuilderCredentials references a Secret that contains credentials to access
	// the builder image repository.
	//
	// +optional
	BuilderCredentials *v1.LocalObjectReference `json:"builderCredentials,omitempty"`
	// The configuration for `Shipwright` build engine.
	Shipwright *ShipwrightEngine `json:"shipwright,omitempty"`
	// Git repository info of a function
	SrcRepo *GitRepo `json:"srcRepo"`
	// Function image name
	Image string `json:"image"`
	// ImageCredentials references a Secret that contains credentials to access
	// the image repository.
	//
	// +optional
	ImageCredentials *v1.LocalObjectReference `json:"imageCredentials,omitempty"`
	// The port on which the function will be invoked
	Port *int32 `json:"port,omitempty"`
	// Dockerfile is the path to the Dockerfile to be used for
	// build strategies that rely on the Dockerfile for building an image.
	//
	// +optional
	Dockerfile *string `json:"dockerfile,omitempty"`
}

// BuilderStatus defines the observed state of Builder
type BuilderStatus struct {
	Phase string `json:"phase,omitempty"`
	State string `json:"state,omitempty"`
	// Associate resources.
	ResourceRef map[string]string `json:"resourceRef,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=fb
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Builder is the Schema for the builders API
type Builder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuilderSpec   `json:"spec,omitempty"`
	Status BuilderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BuilderList contains a list of Builder
type BuilderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Builder `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Builder{}, &BuilderList{})
}
