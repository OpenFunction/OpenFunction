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

package v1

import (
	"context"

	eventingduckv1 "knative.dev/eventing/pkg/apis/duck/v1"

	"knative.dev/eventing/pkg/apis/config"
	"knative.dev/eventing/pkg/apis/eventing"
	"knative.dev/pkg/apis"
)

func (b *Broker) SetDefaults(ctx context.Context) {
	// Default Spec fields.
	withNS := apis.WithinParent(ctx, b.ObjectMeta)
	b.Spec.SetDefaults(withNS)
	eventing.DefaultBrokerClassIfUnset(withNS, &b.ObjectMeta)
}

func (bs *BrokerSpec) SetDefaults(ctx context.Context) {
	cfg := config.FromContextOrDefaults(ctx)
	c, err := cfg.Defaults.GetBrokerConfig(apis.ParentMeta(ctx).Namespace)
	if err == nil {
		if bs.Config == nil {
			bs.Config = c.KReference
		}
		if bs.Delivery == nil && c.Delivery != nil {
			bs.Delivery = &eventingduckv1.DeliverySpec{
				DeadLetterSink: c.Delivery.DeadLetterSink,
				Retry:          c.Delivery.Retry,
				BackoffPolicy:  c.Delivery.BackoffPolicy,
				BackoffDelay:   c.Delivery.BackoffDelay,
			}
		}
	}
	// Default the namespace if not given
	if bs.Config != nil {
		bs.Config.SetDefaults(ctx)
	}
}
