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
func (source *Subscribable) ConvertTo(ctx context.Context, to apis.Convertible) error {
	switch sink := to.(type) {
	case *eventingduckv1.Subscribable:
		sink.ObjectMeta = source.ObjectMeta
		if err := source.Status.ConvertTo(ctx, &sink.Status); err != nil {
			return err
		}
		if err := source.Spec.ConvertTo(ctx, &sink.Spec); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown version, got: %T", sink)
	}
	return nil
}

// ConvertTo helps implement apis.Convertible
func (source *SubscribableSpec) ConvertTo(ctx context.Context, obj apis.Convertible) error {
	switch sink := obj.(type) {
	case *eventingduckv1.SubscribableSpec:
		if len(source.Subscribers) > 0 {
			sink.Subscribers = make([]eventingduckv1.SubscriberSpec, len(source.Subscribers))
			for i, s := range source.Subscribers {
				if err := s.ConvertTo(ctx, &sink.Subscribers[i]); err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("unknown version, got: %T", sink)
	}
	return nil
}

// ConvertTo implements apis.Convertible
func (source *SubscriberSpec) ConvertTo(ctx context.Context, obj apis.Convertible) error {
	switch sink := obj.(type) {
	case *eventingduckv1.SubscriberSpec:
		sink.UID = source.UID
		sink.Generation = source.Generation
		sink.SubscriberURI = source.SubscriberURI
		if source.Delivery != nil {
			sink.Delivery = &eventingduckv1.DeliverySpec{}
			if err := source.Delivery.ConvertTo(ctx, sink.Delivery); err != nil {
				return err
			}
		}
		sink.ReplyURI = source.ReplyURI
	default:
		return fmt.Errorf("unknown version, got: %T", sink)
	}
	return nil
}

// ConvertTo implements apis.Convertible
func (source *SubscribableStatus) ConvertTo(ctx context.Context, obj apis.Convertible) error {
	switch sink := obj.(type) {
	case *eventingduckv1.SubscribableStatus:
		if len(source.Subscribers) > 0 {
			sink.Subscribers = make([]eventingduckv1.SubscriberStatus, len(source.Subscribers))
			for i, ss := range source.Subscribers {
				sink.Subscribers[i] = eventingduckv1.SubscriberStatus{}
				if err := ss.ConvertTo(ctx, &sink.Subscribers[i]); err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("unknown version, got: %T", sink)
	}
	return nil
}

// ConvertTo implements apis.Convertible
func (source *SubscriberStatus) ConvertTo(ctx context.Context, obj apis.Convertible) error {
	switch sink := obj.(type) {
	case *eventingduckv1.SubscriberStatus:
		sink.UID = source.UID
		sink.ObservedGeneration = source.ObservedGeneration
		sink.Ready = source.Ready
		sink.Message = source.Message
	default:
		return fmt.Errorf("unknown version, got: %T", sink)
	}
	return nil
}

// ConvertFrom implements apis.Convertible.
func (sink *Subscribable) ConvertFrom(ctx context.Context, from apis.Convertible) error {
	switch source := from.(type) {
	case *eventingduckv1.Subscribable:
		sink.ObjectMeta = source.ObjectMeta
		if err := sink.Status.ConvertFrom(ctx, &source.Status); err != nil {
			return err
		}
		if err := sink.Spec.ConvertFrom(ctx, &source.Spec); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown version, got: %T", source)
	}
	return nil
}

// ConvertFrom helps implement apis.Convertible
func (sink *SubscribableSpec) ConvertFrom(ctx context.Context, obj apis.Convertible) error {
	switch source := obj.(type) {
	case *eventingduckv1.SubscribableSpec:
		if len(source.Subscribers) > 0 {
			sink.Subscribers = make([]SubscriberSpec, len(source.Subscribers))
			for i := range source.Subscribers {
				if err := sink.Subscribers[i].ConvertFrom(ctx, &source.Subscribers[i]); err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("unknown version, got: %T", source)
	}
	return nil
}

// ConvertFrom helps implement apis.Convertible
func (sink *SubscriberSpec) ConvertFrom(ctx context.Context, obj apis.Convertible) error {
	switch source := obj.(type) {
	case *eventingduckv1.SubscriberSpec:
		sink.UID = source.UID
		sink.Generation = source.Generation
		sink.SubscriberURI = source.SubscriberURI
		sink.ReplyURI = source.ReplyURI
		if source.Delivery != nil {
			sink.Delivery = &DeliverySpec{}
			return sink.Delivery.ConvertFrom(ctx, source.Delivery)
		}
	default:
		return fmt.Errorf("unknown version, got: %T", source)
	}
	return nil
}

// ConvertFrom implements apis.Convertible
func (sink *SubscribableStatus) ConvertFrom(ctx context.Context, obj apis.Convertible) error {
	switch source := obj.(type) {
	case *eventingduckv1.SubscribableStatus:
		if len(source.Subscribers) > 0 {
			sink.Subscribers = make([]SubscriberStatus, len(source.Subscribers))
			for i := range source.Subscribers {
				sink.Subscribers[i] = SubscriberStatus{}
				if err := sink.Subscribers[i].ConvertFrom(ctx, &source.Subscribers[i]); err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("unknown version, got: %T", sink)
	}
	return nil
}

// ConvertFrom implements apis.Convertible
func (sink *SubscriberStatus) ConvertFrom(ctx context.Context, obj apis.Convertible) error {
	switch source := obj.(type) {
	case *eventingduckv1.SubscriberStatus:
		sink.UID = source.UID
		sink.ObservedGeneration = source.ObservedGeneration
		sink.Ready = source.Ready
		sink.Message = source.Message
	default:
		return fmt.Errorf("unknown version, got: %T", sink)
	}
	return nil
}
