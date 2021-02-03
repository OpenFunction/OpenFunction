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

package v1beta1

import (
	"context"

	"github.com/google/go-cmp/cmp/cmpopts"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmp"
)

func (c *Channel) Validate(ctx context.Context) *apis.FieldError {
	withNS := apis.WithinParent(ctx, c.ObjectMeta)
	return c.Spec.Validate(withNS).ViaField("spec")
}

func (cs *ChannelSpec) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError

	if cs.ChannelTemplate == nil {
		// The Channel defaulter is expected to set this, not the users.
		errs = errs.Also(apis.ErrMissingField("channelTemplate"))
	} else {
		if cte := IsValidChannelTemplate(cs.ChannelTemplate); cte != nil {
			errs = errs.Also(cte.ViaField("channelTemplate"))
		}
	}

	if len(cs.SubscribableSpec.Subscribers) > 0 {
		errs = errs.Also(apis.ErrDisallowedFields("subscribers").ViaField("subscribable"))
	}
	return errs
}

func IsValidChannelTemplate(ct *ChannelTemplateSpec) *apis.FieldError {
	var errs *apis.FieldError
	if ct.Kind == "" {
		errs = errs.Also(apis.ErrMissingField("kind"))
	}
	if ct.APIVersion == "" {
		errs = errs.Also(apis.ErrMissingField("apiVersion"))
	}
	return errs
}

func (c *Channel) CheckImmutableFields(ctx context.Context, original *Channel) *apis.FieldError {
	if original == nil {
		return nil
	}

	ignoreArguments := cmpopts.IgnoreFields(ChannelSpec{}, "SubscribableSpec")
	if diff, err := kmp.ShortDiff(original.Spec, c.Spec, ignoreArguments); err != nil {
		return &apis.FieldError{
			Message: "Failed to diff Channel",
			Paths:   []string{"spec"},
			Details: err.Error(),
		}
	} else if diff != "" {
		return &apis.FieldError{
			Message: "Immutable fields changed (-old +new)",
			Paths:   []string{"spec"},
			Details: diff,
		}
	}
	return nil
}
