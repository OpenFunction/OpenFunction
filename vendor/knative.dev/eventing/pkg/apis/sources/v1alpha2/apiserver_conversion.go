/*
Copyright 2020 The Knative Authors.

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
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/eventing/pkg/apis/sources/v1beta1"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// ConvertTo implements apis.Convertible
// Converts source (from v1alpha2.ApiServerSource) into into a higher version.
func (source *ApiServerSource) ConvertTo(ctx context.Context, obj apis.Convertible) error {
	switch sink := obj.(type) {
	case *v1beta1.ApiServerSource:
		// Meta
		sink.ObjectMeta = source.ObjectMeta

		// Spec

		if len(source.Spec.Resources) > 0 {
			sink.Spec.Resources = make([]v1beta1.APIVersionKindSelector, len(source.Spec.Resources))
		}
		for i, v := range source.Spec.Resources {
			sink.Spec.Resources[i] = v1beta1.APIVersionKindSelector{
				APIVersion: v.APIVersion,
				Kind:       v.Kind,
			}

			if v.LabelSelector != nil {
				sink.Spec.Resources[i].LabelSelector = &metav1.LabelSelector{}
				v.LabelSelector.DeepCopyInto(sink.Spec.Resources[i].LabelSelector)
			}
		}

		sink.Spec.EventMode = source.Spec.EventMode

		// Optional Spec

		if source.Spec.ResourceOwner != nil {
			sink.Spec.ResourceOwner = &v1beta1.APIVersionKind{
				Kind:       source.Spec.ResourceOwner.Kind,
				APIVersion: source.Spec.ResourceOwner.APIVersion,
			}
		}

		var ref *duckv1.KReference
		if source.Spec.Sink.Ref != nil {
			ref = &duckv1.KReference{
				Kind:       source.Spec.Sink.Ref.Kind,
				Namespace:  source.Spec.Sink.Ref.Namespace,
				Name:       source.Spec.Sink.Ref.Name,
				APIVersion: source.Spec.Sink.Ref.APIVersion,
			}
		}
		sink.Spec.Sink = duckv1.Destination{
			Ref: ref,
			URI: source.Spec.Sink.URI,
		}

		if source.Spec.CloudEventOverrides != nil {
			sink.Spec.CloudEventOverrides = source.Spec.CloudEventOverrides.DeepCopy()
		}

		sink.Spec.ServiceAccountName = source.Spec.ServiceAccountName

		// Status
		source.Status.SourceStatus.DeepCopyInto(&sink.Status.SourceStatus)
		return nil
	default:
		return apis.ConvertToViaProxy(ctx, source, &v1beta1.ApiServerSource{}, sink)
	}
}

// ConvertFrom implements apis.Convertible
// Converts obj from a higher version into v1alpha2.ApiServerSource.
func (sink *ApiServerSource) ConvertFrom(ctx context.Context, obj apis.Convertible) error {
	switch source := obj.(type) {
	case *v1beta1.ApiServerSource:
		// Meta
		sink.ObjectMeta = source.ObjectMeta

		// Spec
		sink.Spec.EventMode = source.Spec.EventMode

		sink.Spec.CloudEventOverrides = source.Spec.CloudEventOverrides

		sink.Spec.Sink = source.Spec.Sink

		if len(source.Spec.Resources) > 0 {
			sink.Spec.Resources = make([]APIVersionKindSelector, len(source.Spec.Resources))
		}
		for i, v := range source.Spec.Resources {
			sink.Spec.Resources[i] = APIVersionKindSelector{}
			sink.Spec.Resources[i].APIVersion = v.APIVersion
			sink.Spec.Resources[i].Kind = v.Kind
			if v.LabelSelector != nil {
				sink.Spec.Resources[i].LabelSelector = v.LabelSelector.DeepCopy()
			}
		}

		// Spec Optionals

		if source.Spec.ResourceOwner != nil {
			sink.Spec.ResourceOwner = &APIVersionKind{
				Kind:       source.Spec.ResourceOwner.Kind,
				APIVersion: source.Spec.ResourceOwner.APIVersion,
			}
		}

		sink.Spec.ServiceAccountName = source.Spec.ServiceAccountName

		// Status
		source.Status.SourceStatus.DeepCopyInto(&sink.Status.SourceStatus)

		return nil
	default:
		return apis.ConvertFromViaProxy(ctx, source, &v1beta1.ApiServerSource{}, sink)
	}
}
