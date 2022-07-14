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

package gateway

import "sigs.k8s.io/gateway-api/apis/v1alpha2"

func ConvertListenersListToMapping(listeners []v1alpha2.Listener) map[v1alpha2.SectionName]v1alpha2.Listener {
	mapping := make(map[v1alpha2.SectionName]v1alpha2.Listener)
	for _, listener := range listeners {
		mapping[listener.Name] = listener
	}
	return mapping
}

func ConvertListenersMappingToList(mapping map[v1alpha2.SectionName]v1alpha2.Listener) []v1alpha2.Listener {
	var listeners []v1alpha2.Listener
	for _, listener := range mapping {
		listeners = append(listeners, listener)
	}
	return listeners
}
