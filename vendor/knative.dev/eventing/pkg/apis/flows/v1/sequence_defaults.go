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

package v1

import (
	"context"

	"knative.dev/eventing/pkg/apis/messaging/config"
	messagingv1 "knative.dev/eventing/pkg/apis/messaging/v1"
	"knative.dev/pkg/apis"
)

func (s *Sequence) SetDefaults(ctx context.Context) {
	if s == nil {
		return
	}

	withNS := apis.WithinParent(ctx, s.ObjectMeta)
	if s.Spec.ChannelTemplate == nil {
		cfg := config.FromContextOrDefaults(ctx)
		c, err := cfg.ChannelDefaults.GetChannelConfig(apis.ParentMeta(ctx).Namespace)

		if err == nil {
			s.Spec.ChannelTemplate = &messagingv1.ChannelTemplateSpec{
				TypeMeta: c.TypeMeta,
				Spec:     c.Spec,
			}
		}
	}
	s.Spec.SetDefaults(withNS)
}

func (ss *SequenceSpec) SetDefaults(ctx context.Context) {
	// Default the namespace for all the steps.
	for _, s := range ss.Steps {
		s.SetDefaults(ctx)
	}
	// Default the reply
	if ss.Reply != nil {
		ss.Reply.SetDefaults(ctx)
	}
}

func (ss *SequenceStep) SetDefaults(ctx context.Context) {
	ss.Destination.SetDefaults(ctx)

	// No delivery defaults.
}
