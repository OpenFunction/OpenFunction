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

	messagingv1beta1 "knative.dev/eventing/pkg/apis/messaging/v1beta1"
	"knative.dev/pkg/apis"
)

func (p *Sequence) Validate(ctx context.Context) *apis.FieldError {
	return p.Spec.Validate(ctx).ViaField("spec")
}

func (ps *SequenceSpec) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError

	if len(ps.Steps) == 0 {
		errs = errs.Also(apis.ErrMissingField("steps"))
	}

	for i, s := range ps.Steps {
		if e := s.Validate(ctx); e != nil {
			errs = errs.Also(apis.ErrInvalidArrayValue(s, "steps", i))
		}
	}

	if ps.ChannelTemplate == nil {
		errs = errs.Also(apis.ErrMissingField("channelTemplate"))
	} else {
		if ce := messagingv1beta1.IsValidChannelTemplate(ps.ChannelTemplate); ce != nil {
			errs = errs.Also(ce.ViaField("channelTemplate"))
		}
	}

	if err := ps.Reply.Validate(ctx); err != nil {
		errs = errs.Also(err.ViaField("reply"))
	}

	return errs
}

func (ss *SequenceStep) Validate(ctx context.Context) *apis.FieldError {
	errs := ss.Destination.Validate(ctx)

	if ss.Delivery != nil {
		if de := ss.Delivery.Validate(ctx); de != nil {
			errs = errs.Also(de.ViaField("delivery"))
		}
	}

	return errs
}
