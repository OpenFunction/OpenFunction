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

	eventingduckv1 "knative.dev/eventing/pkg/apis/duck/v1"
	eventingduckv1beta1 "knative.dev/eventing/pkg/apis/duck/v1beta1"
	"knative.dev/eventing/pkg/apis/messaging"
	v1 "knative.dev/eventing/pkg/apis/messaging/v1"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"
)

// ConvertTo implements apis.Convertible
// Converts source (from v1beta1.InMemoryChannel) into v1.InMemoryChannel
func (source *InMemoryChannel) ConvertTo(ctx context.Context, obj apis.Convertible) error {
	switch sink := obj.(type) {
	case *v1.InMemoryChannel:
		sink.ObjectMeta = source.ObjectMeta
		// Does a deep copy, adds our duck version.
		sink.Annotations = kmeta.UnionMaps(source.Annotations, map[string]string{messaging.SubscribableDuckVersionAnnotation: "v1"})
		source.Status.ConvertTo(ctx, &sink.Status)
		return source.Spec.ConvertTo(ctx, &sink.Spec)
	default:
		return fmt.Errorf("unknown version, got: %T", sink)
	}
}

// ConvertTo helps implement apis.Convertible
func (source *InMemoryChannelSpec) ConvertTo(ctx context.Context, sink *v1.InMemoryChannelSpec) error {
	sink.SubscribableSpec = eventingduckv1.SubscribableSpec{}
	source.SubscribableSpec.ConvertTo(ctx, &sink.SubscribableSpec)
	if source.Delivery != nil {
		sink.Delivery = &eventingduckv1.DeliverySpec{}
		return source.Delivery.ConvertTo(ctx, sink.Delivery)
	}
	return nil
}

// ConvertTo helps implement apis.Convertible
func (source *InMemoryChannelStatus) ConvertTo(ctx context.Context, sink *v1.InMemoryChannelStatus) {
	sink.Status = source.Status
	sink.AddressStatus = source.AddressStatus
	sink.SubscribableStatus = eventingduckv1.SubscribableStatus{}
	source.SubscribableStatus.ConvertTo(ctx, &sink.SubscribableStatus)
	sink.DeadLetterChannel = source.DeadLetterChannel
}

// ConvertFrom implements apis.Convertible.
// Converts obj v1.InMemoryChannel into v1beta1.InMemoryChannel
func (sink *InMemoryChannel) ConvertFrom(ctx context.Context, obj apis.Convertible) error {
	switch source := obj.(type) {
	case *v1.InMemoryChannel:
		sink.ObjectMeta = source.ObjectMeta
		sink.Status.ConvertFrom(ctx, source.Status)
		sink.Spec.ConvertFrom(ctx, source.Spec)
		// Does a deep copy, adds our duck version.
		sink.Annotations = kmeta.UnionMaps(source.Annotations, map[string]string{messaging.SubscribableDuckVersionAnnotation: "v1beta1"})
		return nil
	default:
		return fmt.Errorf("unknown version, got: %T", source)
	}
}

// ConvertFrom helps implement apis.Convertible
func (sink *InMemoryChannelSpec) ConvertFrom(ctx context.Context, source v1.InMemoryChannelSpec) error {
	if source.Delivery != nil {
		sink.Delivery = &eventingduckv1beta1.DeliverySpec{}
		if err := sink.Delivery.ConvertFrom(ctx, source.Delivery); err != nil {
			return err
		}
	}
	sink.SubscribableSpec = eventingduckv1beta1.SubscribableSpec{}
	sink.SubscribableSpec.ConvertFrom(ctx, &source.SubscribableSpec)
	return nil
}

// ConvertFrom helps implement apis.Convertible
func (sink *InMemoryChannelStatus) ConvertFrom(ctx context.Context, source v1.InMemoryChannelStatus) {
	sink.Status = source.Status
	sink.AddressStatus = source.AddressStatus
	sink.SubscribableStatus = eventingduckv1beta1.SubscribableStatus{}
	sink.SubscribableStatus.ConvertFrom(ctx, &source.SubscribableStatus)
	sink.DeadLetterChannel = source.DeadLetterChannel
}
