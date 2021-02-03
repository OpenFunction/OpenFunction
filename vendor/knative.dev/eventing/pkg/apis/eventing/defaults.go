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

package eventing

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/eventing/pkg/apis/config"
)

// DefaultBrokerClassIfUnset sets default broker class annotation if unset.
func DefaultBrokerClassIfUnset(ctx context.Context, obj *metav1.ObjectMeta) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}
	if _, present := annotations[BrokerClassKey]; !present {
		cfg := config.FromContextOrDefaults(ctx)
		c, err := cfg.Defaults.GetBrokerClass(obj.Namespace)
		if err == nil {
			annotations[BrokerClassKey] = c
			obj.SetAnnotations(annotations)
		}
	}
}
