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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuilderSpec defines the desired state of Builder
type BuilderSpec struct {
	// The name of the exported function to be invoked in function code
	Name string `json:"name"`
	// Function type such as HTTP or CloudEvent
	Type string `json:"type"`
	// Function version in format like v1.0.0
	Version *string `json:"version,omitempty"`
	// Function source file such as main.py for python
	Source string `json:"source,omitempty"`
	// Cloud Native Buildpacks builders
	Builder string `json:"builder"`
	// Git repository info of a function
	GitRepo *GitRepo `json:"gitRepo"`
	// Function image name
	Image string `json:"image"`
	// Image registry of the function image
	Registry *Registry `json:"registry"`
	// The port on which the function will be invoked
	Port *int32 `json:"port,omitempty"`
}

// BuilderStatus defines the observed state of Builder
type BuilderStatus struct {
	Phase string `json:"phase,omitempty"`
	State string `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=fb
// +kubebuilder:subresource:status
// Builder is the Schema for the builders API
type Builder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuilderSpec   `json:"spec,omitempty"`
	Status BuilderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BuilderList contains a list of Builder
type BuilderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Builder `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Builder{}, &BuilderList{})
}
