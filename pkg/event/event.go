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

package event

import (
	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"

	ofcore "github.com/openfunction/apis/core/v1beta1"
)

type OpenFunctionEventSource interface {
	SetMetadata(key string, value interface{})
	GetMetadata() map[string]interface{}
	GenComponent(namespace string, name string) (*componentsv1alpha1.Component, error)
	GenScaleOptions() (*ofcore.KedaScaledObject, *kedav1alpha1.ScaleTriggers)
}

type OpenFunctionEventBus interface {
	SetMetadata(key string, value interface{})
	GetMetadata() map[string]interface{}
	GenComponent(namespace string, name string) (*componentsv1alpha1.Component, error)
	GenScaleOptions(subjects []string) (*ofcore.KedaScaledObject, []*kedav1alpha1.ScaleTriggers)
}
