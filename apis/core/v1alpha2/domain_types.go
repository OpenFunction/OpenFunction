/*
Copyright 2022 The OpenFunction Authors.

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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IngressControllerService struct {
	// Name of the Ingress controller service.
	Name string `json:"name"`
	// Namespace of the Ingress controller service.
	Namespace string `json:"namespace"`
	// Port of the Ingress controller service, default is 80.
	Port int32 `json:"port,omitempty"`
}
type IngressConfig struct {
	// Annotations for Ingress.
	//
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// Ingress controller service.
	Service IngressControllerService `json:"service"`
	// IngressClassName is the name of the IngressClass cluster resource. The
	// associated IngressClass defines which controller will implement the resource.
	IngressClassName string `json:"ingressClassName"`
}

// DomainSpec defines the desired state of a Domain
type DomainSpec struct {
	// Ingress configuration.
	Ingress IngressConfig `json:"ingress"`
}

// DomainStatus defines the observed state of Domain
type DomainStatus struct {
}

//+genclient
//+genclient:noStatus
//+kubebuilder:object:root=true
//+kubebuilder:storageversion
//+kubebuilder:subresource:status

// Domain define a unified entry for function, user can access function through it.
type Domain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DomainSpec   `json:"spec,omitempty"`
	Status DomainStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DomainList contains a list of Domain
type DomainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Domain `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Domain{}, &DomainList{})
}
