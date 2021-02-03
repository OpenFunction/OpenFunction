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

	"knative.dev/pkg/apis"

	eventingduckv1 "knative.dev/eventing/pkg/apis/duck/v1"
	eventingduckv1beta1 "knative.dev/eventing/pkg/apis/duck/v1beta1"
	v1 "knative.dev/eventing/pkg/apis/flows/v1"
	messagingv1 "knative.dev/eventing/pkg/apis/messaging/v1"
	messagingv1beta1 "knative.dev/eventing/pkg/apis/messaging/v1beta1"
)

// ConvertTo implements apis.Convertible
// Converts obj from v1beta1.Sequence into v1.Sequence
func (source *Sequence) ConvertTo(ctx context.Context, obj apis.Convertible) error {
	switch sink := obj.(type) {
	case *v1.Sequence:
		sink.ObjectMeta = source.ObjectMeta

		sink.Spec.Steps = make([]v1.SequenceStep, len(source.Spec.Steps))
		for i, s := range source.Spec.Steps {
			sink.Spec.Steps[i] = v1.SequenceStep{
				Destination: s.Destination,
			}

			if s.Delivery != nil {
				sink.Spec.Steps[i].Delivery = &eventingduckv1.DeliverySpec{}
				if err := s.Delivery.ConvertTo(ctx, sink.Spec.Steps[i].Delivery); err != nil {
					return err
				}
			}
		}

		if source.Spec.ChannelTemplate != nil {
			sink.Spec.ChannelTemplate = &messagingv1.ChannelTemplateSpec{
				TypeMeta: source.Spec.ChannelTemplate.TypeMeta,
				Spec:     source.Spec.ChannelTemplate.Spec,
			}
		}
		sink.Spec.Reply = source.Spec.Reply

		sink.Status.Status = source.Status.Status
		sink.Status.AddressStatus = source.Status.AddressStatus

		if source.Status.SubscriptionStatuses != nil {
			sink.Status.SubscriptionStatuses = make([]v1.SequenceSubscriptionStatus, len(source.Status.SubscriptionStatuses))
			for i, s := range source.Status.SubscriptionStatuses {
				sink.Status.SubscriptionStatuses[i] = v1.SequenceSubscriptionStatus{
					Subscription:   s.Subscription,
					ReadyCondition: s.ReadyCondition,
				}
			}
		}

		if source.Status.ChannelStatuses != nil {
			sink.Status.ChannelStatuses = make([]v1.SequenceChannelStatus, len(source.Status.ChannelStatuses))
			for i, s := range source.Status.ChannelStatuses {
				sink.Status.ChannelStatuses[i] = v1.SequenceChannelStatus{
					Channel:        s.Channel,
					ReadyCondition: s.ReadyCondition,
				}
			}
		}

		return nil
	default:
		return fmt.Errorf("Unknown conversion, got: %T", sink)
	}
}

// ConvertFrom implements apis.Convertible
func (sink *Sequence) ConvertFrom(ctx context.Context, obj apis.Convertible) error {
	switch source := obj.(type) {
	case *v1.Sequence:
		sink.ObjectMeta = source.ObjectMeta

		sink.Spec.Steps = make([]SequenceStep, len(source.Spec.Steps))
		for i, s := range source.Spec.Steps {
			sink.Spec.Steps[i] = SequenceStep{
				Destination: s.Destination,
			}
			if s.Delivery != nil {
				sink.Spec.Steps[i].Delivery = &eventingduckv1beta1.DeliverySpec{}
				if err := sink.Spec.Steps[i].Delivery.ConvertFrom(ctx, s.Delivery); err != nil {
					return err
				}
			}
		}

		if source.Spec.ChannelTemplate != nil {
			sink.Spec.ChannelTemplate = &messagingv1beta1.ChannelTemplateSpec{
				TypeMeta: source.Spec.ChannelTemplate.TypeMeta,
				Spec:     source.Spec.ChannelTemplate.Spec,
			}
		}

		sink.Spec.Reply = source.Spec.Reply

		sink.Status.Status = source.Status.Status
		sink.Status.AddressStatus = source.Status.AddressStatus

		if source.Status.SubscriptionStatuses != nil {
			sink.Status.SubscriptionStatuses = make([]SequenceSubscriptionStatus, len(source.Status.SubscriptionStatuses))
			for i, s := range source.Status.SubscriptionStatuses {
				sink.Status.SubscriptionStatuses[i] = SequenceSubscriptionStatus{
					Subscription:   s.Subscription,
					ReadyCondition: s.ReadyCondition,
				}
			}
		}

		if source.Status.ChannelStatuses != nil {
			sink.Status.ChannelStatuses = make([]SequenceChannelStatus, len(source.Status.ChannelStatuses))
			for i, s := range source.Status.ChannelStatuses {
				sink.Status.ChannelStatuses[i] = SequenceChannelStatus{
					Channel:        s.Channel,
					ReadyCondition: s.ReadyCondition,
				}
			}
		}

		return nil
	default:
		return fmt.Errorf("unknown version, got: %T", source)
	}
}
