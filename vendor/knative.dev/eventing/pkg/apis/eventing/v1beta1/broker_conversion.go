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

	duckv1 "knative.dev/eventing/pkg/apis/duck/v1"
	duckv1beta1 "knative.dev/eventing/pkg/apis/duck/v1beta1"
	v1 "knative.dev/eventing/pkg/apis/eventing/v1"
	"knative.dev/pkg/apis"
)

// ConvertTo implements apis.Convertible
func (source *Broker) ConvertTo(ctx context.Context, to apis.Convertible) error {
	switch sink := to.(type) {
	case *v1.Broker:
		sink.ObjectMeta = source.ObjectMeta
		sink.Spec.Config = source.Spec.Config
		if source.Spec.Delivery != nil {
			sink.Spec.Delivery = &duckv1.DeliverySpec{}
			if err := source.Spec.Delivery.ConvertTo(ctx, sink.Spec.Delivery); err != nil {
				return err
			}
		}
		sink.Status.Status = source.Status.Status
		sink.Status.Address = source.Status.Address
		return nil
	default:
		return fmt.Errorf("unknown version, got: %T", sink)
	}
}

// ConvertFrom implements apis.Convertible
func (sink *Broker) ConvertFrom(ctx context.Context, from apis.Convertible) error {
	switch source := from.(type) {
	case *v1.Broker:
		sink.ObjectMeta = source.ObjectMeta
		sink.Spec.Config = source.Spec.Config
		if source.Spec.Delivery != nil {
			sink.Spec.Delivery = &duckv1beta1.DeliverySpec{}
			if err := sink.Spec.Delivery.ConvertFrom(ctx, source.Spec.Delivery); err != nil {
				return err
			}
		}
		sink.Status.Status = source.Status.Status
		sink.Status.Address = source.Status.Address
		return nil
	default:
		return fmt.Errorf("unknown version, got: %T", source)
	}
}
