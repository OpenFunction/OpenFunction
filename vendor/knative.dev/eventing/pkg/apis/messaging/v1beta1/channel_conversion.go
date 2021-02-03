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
	"knative.dev/eventing/pkg/apis/duck/v1beta1"
	"knative.dev/eventing/pkg/apis/messaging"
	v1 "knative.dev/eventing/pkg/apis/messaging/v1"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmeta"
)

// ConvertTo implements apis.Convertible
// Converts source (from v1beta1.Channel) into v1.Channel
func (source *Channel) ConvertTo(ctx context.Context, obj apis.Convertible) error {
	switch sink := obj.(type) {
	case *v1.Channel:
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
func (source *ChannelSpec) ConvertTo(ctx context.Context, sink *v1.ChannelSpec) error {
	if source.ChannelTemplate != nil {
		sink.ChannelTemplate = &v1.ChannelTemplateSpec{
			TypeMeta: source.ChannelTemplate.TypeMeta,
			Spec:     source.ChannelTemplate.Spec,
		}
	}
	sink.ChannelableSpec = eventingduckv1.ChannelableSpec{}
	source.SubscribableSpec.ConvertTo(ctx, &sink.SubscribableSpec)
	if source.Delivery != nil {
		sink.Delivery = &eventingduckv1.DeliverySpec{}
		return source.Delivery.ConvertTo(ctx, sink.Delivery)
	}
	return nil
}

// ConvertTo helps implement apis.Convertible
func (source *ChannelStatus) ConvertTo(ctx context.Context, sink *v1.ChannelStatus) {
	sink.Status = source.Status
	sink.AddressStatus.Address = source.AddressStatus.Address
	source.SubscribableStatus.ConvertTo(ctx, &sink.SubscribableStatus)
	sink.Channel = source.Channel
	sink.DeadLetterChannel = source.DeadLetterChannel
}

// ConvertFrom implements apis.Convertible.
// Converts obj v1.Channel into v1beta1.Channel
func (sink *Channel) ConvertFrom(ctx context.Context, obj apis.Convertible) error {
	switch source := obj.(type) {
	case *v1.Channel:
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
func (sink *ChannelSpec) ConvertFrom(ctx context.Context, source v1.ChannelSpec) {
	if source.ChannelTemplate != nil {
		sink.ChannelTemplate = &ChannelTemplateSpec{
			TypeMeta: source.ChannelTemplate.TypeMeta,
			Spec:     source.ChannelTemplate.Spec,
		}
	}
	if source.Delivery != nil {
		sink.Delivery = &v1beta1.DeliverySpec{}
		sink.Delivery.ConvertFrom(ctx, source.Delivery)
	}
	sink.ChannelableSpec.SubscribableSpec.ConvertFrom(ctx, &source.ChannelableSpec.SubscribableSpec)
}

// ConvertFrom helps implement apis.Convertible
func (sink *ChannelStatus) ConvertFrom(ctx context.Context, source v1.ChannelStatus) {
	sink.Status = source.Status
	sink.Channel = source.Channel
	sink.SubscribableStatus.ConvertFrom(ctx, &source.SubscribableStatus)
	sink.AddressStatus.Address = source.AddressStatus.Address
	sink.DeadLetterChannel = source.DeadLetterChannel
}
