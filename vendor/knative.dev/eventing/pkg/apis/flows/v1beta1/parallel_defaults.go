/*
 * Copyright 2020 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1beta1

import (
	"context"

	"knative.dev/eventing/pkg/apis/messaging/config"
	messagingv1beta1 "knative.dev/eventing/pkg/apis/messaging/v1beta1"
	"knative.dev/pkg/apis"
)

func (p *Parallel) SetDefaults(ctx context.Context) {
	if p == nil {
		return
	}

	withNS := apis.WithinParent(ctx, p.ObjectMeta)
	if p.Spec.ChannelTemplate == nil {
		cfg := config.FromContextOrDefaults(ctx)
		c, err := cfg.ChannelDefaults.GetChannelConfig(apis.ParentMeta(ctx).Namespace)

		if err == nil {
			p.Spec.ChannelTemplate = &messagingv1beta1.ChannelTemplateSpec{
				TypeMeta: c.TypeMeta,
				Spec:     c.Spec,
			}
		}
	}
	p.Spec.SetDefaults(withNS)
}

func (ps *ParallelSpec) SetDefaults(ctx context.Context) {
	for _, branch := range ps.Branches {
		if branch.Filter != nil {
			branch.Filter.SetDefaults(ctx)
		}
		branch.Subscriber.SetDefaults(ctx)
		if branch.Reply != nil {
			branch.Reply.SetDefaults(ctx)
		}
	}
	if ps.Reply != nil {
		ps.Reply.SetDefaults(ctx)
	}
}
