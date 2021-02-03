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
)

// ConvertTo implements apis.Convertible
func (source *DeliverySpec) ConvertTo(ctx context.Context, to apis.Convertible) error {
	switch sink := to.(type) {
	case *eventingduckv1.DeliverySpec:
		sink.Retry = source.Retry
		sink.BackoffDelay = source.BackoffDelay
		if source.BackoffPolicy != nil {
			if *source.BackoffPolicy == BackoffPolicyLinear {
				linear := eventingduckv1.BackoffPolicyLinear
				sink.BackoffPolicy = &linear
			} else if *source.BackoffPolicy == BackoffPolicyExponential {
				exponential := eventingduckv1.BackoffPolicyExponential
				sink.BackoffPolicy = &exponential
			} else {
				return fmt.Errorf("unknown BackoffPolicy, got: %q", *source.BackoffPolicy)
			}
		}
		sink.DeadLetterSink = source.DeadLetterSink
		return nil
	default:
		return fmt.Errorf("unknown version, got: %T", sink)
	}
}

// ConvertFrom implements apis.Convertible
func (sink *DeliverySpec) ConvertFrom(ctx context.Context, from apis.Convertible) error {
	switch source := from.(type) {
	case *eventingduckv1.DeliverySpec:
		sink.Retry = source.Retry
		sink.BackoffDelay = source.BackoffDelay
		if source.BackoffPolicy != nil {
			if *source.BackoffPolicy == eventingduckv1.BackoffPolicyLinear {
				linear := BackoffPolicyLinear
				sink.BackoffPolicy = &linear
			} else if *source.BackoffPolicy == eventingduckv1.BackoffPolicyExponential {
				exponential := BackoffPolicyExponential
				sink.BackoffPolicy = &exponential
			} else {
				return fmt.Errorf("unknown BackoffPolicy, got: %q", *source.BackoffPolicy)
			}

		}
		sink.DeadLetterSink = source.DeadLetterSink
		return nil
	default:
		return fmt.Errorf("unknown version, got: %T", source)
	}
}

// ConvertTo implements apis.Convertible
func (source *DeliveryStatus) ConvertTo(ctx context.Context, to apis.Convertible) error {
	switch sink := to.(type) {
	case *eventingduckv1.DeliveryStatus:
		sink.DeadLetterChannel = source.DeadLetterChannel
		return nil
	default:
		return fmt.Errorf("unknown version, got: %T", sink)
	}
}

// ConvertFrom implements apis.Convertible
func (sink *DeliveryStatus) ConvertFrom(ctx context.Context, from apis.Convertible) error {
	switch source := from.(type) {
	case *eventingduckv1.DeliveryStatus:
		sink.DeadLetterChannel = source.DeadLetterChannel
		return nil
	default:
		return fmt.Errorf("unknown version, got: %T", source)
	}
}
