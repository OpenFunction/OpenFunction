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
// Converts obj from v1beta1.Parallel into v1.Parallel
func (source *Parallel) ConvertTo(ctx context.Context, obj apis.Convertible) error {
	switch sink := obj.(type) {
	case *v1.Parallel:
		sink.ObjectMeta = source.ObjectMeta

		sink.Spec.Branches = make([]v1.ParallelBranch, len(source.Spec.Branches))
		for i, b := range source.Spec.Branches {
			sink.Spec.Branches[i] = v1.ParallelBranch{
				Filter:     b.Filter,
				Subscriber: b.Subscriber,
				Reply:      b.Reply,
			}

			if b.Delivery != nil {
				sink.Spec.Branches[i].Delivery = &eventingduckv1.DeliverySpec{}
				if err := b.Delivery.ConvertTo(ctx, sink.Spec.Branches[i].Delivery); err != nil {
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

		sink.Status.IngressChannelStatus = v1.ParallelChannelStatus{
			Channel:        source.Status.IngressChannelStatus.Channel,
			ReadyCondition: source.Status.IngressChannelStatus.ReadyCondition,
		}

		if source.Status.BranchStatuses != nil {
			sink.Status.BranchStatuses = make([]v1.ParallelBranchStatus, len(source.Status.BranchStatuses))
			for i, b := range source.Status.BranchStatuses {
				sink.Status.BranchStatuses[i] = v1.ParallelBranchStatus{
					FilterSubscriptionStatus: v1.ParallelSubscriptionStatus{
						Subscription:   b.FilterSubscriptionStatus.Subscription,
						ReadyCondition: b.FilterSubscriptionStatus.ReadyCondition,
					},
					FilterChannelStatus: v1.ParallelChannelStatus{
						Channel:        b.FilterChannelStatus.Channel,
						ReadyCondition: b.FilterChannelStatus.ReadyCondition,
					},
					SubscriptionStatus: v1.ParallelSubscriptionStatus{
						Subscription:   b.SubscriptionStatus.Subscription,
						ReadyCondition: b.SubscriptionStatus.ReadyCondition,
					},
				}
			}
		}

		return nil
	default:
		return fmt.Errorf("Unknown conversion, got: %T", sink)
	}
}

// ConvertFrom implements apis.Convertible
// Converts obj from v1.Parallel into v1beta1.Parallel
func (sink *Parallel) ConvertFrom(ctx context.Context, obj apis.Convertible) error {
	switch source := obj.(type) {
	case *v1.Parallel:
		sink.ObjectMeta = source.ObjectMeta

		sink.Spec.Branches = make([]ParallelBranch, len(source.Spec.Branches))
		for i, b := range source.Spec.Branches {
			sink.Spec.Branches[i] = ParallelBranch{
				Filter:     b.Filter,
				Subscriber: b.Subscriber,
				Reply:      b.Reply,
			}
			if b.Delivery != nil {
				sink.Spec.Branches[i].Delivery = &eventingduckv1beta1.DeliverySpec{}
				if err := sink.Spec.Branches[i].Delivery.ConvertFrom(ctx, b.Delivery); err != nil {
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

		sink.Status.IngressChannelStatus = ParallelChannelStatus{
			Channel:        source.Status.IngressChannelStatus.Channel,
			ReadyCondition: source.Status.IngressChannelStatus.ReadyCondition,
		}

		if source.Status.BranchStatuses != nil {
			sink.Status.BranchStatuses = make([]ParallelBranchStatus, len(source.Status.BranchStatuses))
			for i, b := range source.Status.BranchStatuses {
				sink.Status.BranchStatuses[i] = ParallelBranchStatus{
					FilterSubscriptionStatus: ParallelSubscriptionStatus{
						Subscription:   b.FilterSubscriptionStatus.Subscription,
						ReadyCondition: b.FilterSubscriptionStatus.ReadyCondition,
					},
					FilterChannelStatus: ParallelChannelStatus{
						Channel:        b.FilterChannelStatus.Channel,
						ReadyCondition: b.FilterChannelStatus.ReadyCondition,
					},
					SubscriptionStatus: ParallelSubscriptionStatus{
						Subscription:   b.SubscriptionStatus.Subscription,
						ReadyCondition: b.SubscriptionStatus.ReadyCondition,
					},
				}
			}
		}

		return nil
	default:
		return fmt.Errorf("unknown version, got: %T", source)
	}
}
