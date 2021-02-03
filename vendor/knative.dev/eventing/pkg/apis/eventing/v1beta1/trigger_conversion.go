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

package v1beta1

import (
	"context"
	"fmt"

	v1 "knative.dev/eventing/pkg/apis/eventing/v1"
	"knative.dev/pkg/apis"
)

// ConvertTo implements apis.Convertible
func (source *Trigger) ConvertTo(_ context.Context, to apis.Convertible) error {
	switch sink := to.(type) {
	case *v1.Trigger:
		sink.ObjectMeta = source.ObjectMeta
		sink.Spec.Broker = source.Spec.Broker
		sink.Spec.Subscriber = source.Spec.Subscriber
		if source.Spec.Filter != nil {
			sink.Spec.Filter = &v1.TriggerFilter{
				Attributes: make(v1.TriggerFilterAttributes),
			}
			for k, v := range source.Spec.Filter.Attributes {
				sink.Spec.Filter.Attributes[k] = v
			}
		}
		sink.Status.Status = source.Status.Status
		sink.Status.SubscriberURI = source.Status.SubscriberURI
		return nil
	default:
		return fmt.Errorf("unknown version, got: %T", sink)
	}
}

// ConvertFrom implements apis.Convertible
func (sink *Trigger) ConvertFrom(_ context.Context, from apis.Convertible) error {
	switch source := from.(type) {
	case *v1.Trigger:
		sink.ObjectMeta = source.ObjectMeta
		sink.Spec.Broker = source.Spec.Broker
		sink.Spec.Subscriber = source.Spec.Subscriber
		if source.Spec.Filter != nil {
			attributes := TriggerFilterAttributes{}
			for k, v := range source.Spec.Filter.Attributes {
				attributes[k] = v
			}
			sink.Spec.Filter = &TriggerFilter{
				Attributes: attributes,
			}
		}
		sink.Status.Status = source.Status.Status
		sink.Status.SubscriberURI = source.Status.SubscriberURI
		return nil
	default:
		return fmt.Errorf("unknown version, got: %T", source)
	}
}
