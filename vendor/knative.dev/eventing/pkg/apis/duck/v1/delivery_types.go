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

	"github.com/rickb777/date/period"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

// DeliverySpec contains the delivery options for event senders,
// such as channelable and source.
type DeliverySpec struct {
	// DeadLetterSink is the sink receiving event that could not be sent to
	// a destination.
	// +optional
	DeadLetterSink *duckv1.Destination `json:"deadLetterSink,omitempty"`

	// Retry is the minimum number of retries the sender should attempt when
	// sending an event before moving it to the dead letter sink.
	// +optional
	Retry *int32 `json:"retry,omitempty"`

	// BackoffPolicy is the retry backoff policy (linear, exponential).
	// +optional
	BackoffPolicy *BackoffPolicyType `json:"backoffPolicy,omitempty"`

	// BackoffDelay is the delay before retrying.
	// More information on Duration format:
	//  - https://www.iso.org/iso-8601-date-and-time-format.html
	//  - https://en.wikipedia.org/wiki/ISO_8601
	//
	// For linear policy, backoff delay is backoffDelay*<numberOfRetries>.
	// For exponential policy, backoff delay is backoffDelay*2^<numberOfRetries>.
	// +optional
	BackoffDelay *string `json:"backoffDelay,omitempty"`
}

func (ds *DeliverySpec) Validate(ctx context.Context) *apis.FieldError {
	if ds == nil {
		return nil
	}
	var errs *apis.FieldError
	if dlse := ds.DeadLetterSink.Validate(ctx); dlse != nil {
		errs = errs.Also(dlse).ViaField("deadLetterSink")
	}

	if ds.Retry != nil && *ds.Retry < 0 {
		errs = errs.Also(apis.ErrInvalidValue(*ds.Retry, "retry"))
	}

	if ds.BackoffPolicy != nil {
		switch *ds.BackoffPolicy {
		case BackoffPolicyExponential, BackoffPolicyLinear:
			// nothing
		default:
			errs = errs.Also(apis.ErrInvalidValue(*ds.BackoffPolicy, "backoffPolicy"))
		}
	}

	if ds.BackoffDelay != nil {
		_, te := period.Parse(*ds.BackoffDelay)
		if te != nil {
			errs = errs.Also(apis.ErrInvalidValue(*ds.BackoffDelay, "backoffDelay"))
		}
	}
	return errs
}

// BackoffPolicyType is the type for backoff policies
type BackoffPolicyType string

const (
	// Linear backoff policy
	BackoffPolicyLinear BackoffPolicyType = "linear"

	// Exponential backoff policy
	BackoffPolicyExponential BackoffPolicyType = "exponential"
)

// DeliveryStatus contains the Status of an object supporting delivery options.
type DeliveryStatus struct {
	// DeadLetterChannel is a KReference that is the reference to the native, platform specific channel
	// where failed events are sent to.
	// +optional
	DeadLetterChannel *duckv1.KReference `json:"deadLetterChannel,omitempty"`
}
