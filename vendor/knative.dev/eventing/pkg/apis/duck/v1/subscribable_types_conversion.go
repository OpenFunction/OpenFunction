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

package v1

import (
	"context"
	"fmt"

	"knative.dev/pkg/apis"
)

// ConvertTo implements apis.Convertible
func (source *Subscribable) ConvertTo(ctx context.Context, sink apis.Convertible) error {
	return fmt.Errorf("v1 is the highest known version, got: %T", sink)
}

// ConvertFrom implements apis.Convertible
func (sink *Subscribable) ConvertFrom(ctx context.Context, source apis.Convertible) error {
	return fmt.Errorf("v1 is the highest known version, got: %T", source)
}

// ConvertTo implements apis.Convertible
func (source *SubscribableSpec) ConvertTo(ctx context.Context, sink apis.Convertible) error {
	return fmt.Errorf("v1 is the highest known version, got: %T", sink)
}

// ConvertFrom implements apis.Convertible
func (sink *SubscribableSpec) ConvertFrom(ctx context.Context, source apis.Convertible) error {
	return fmt.Errorf("v1 is the highest known version, got: %T", source)
}

// ConvertTo implements apis.Convertible
func (source *SubscribableStatus) ConvertTo(ctx context.Context, sink apis.Convertible) error {
	return fmt.Errorf("v1 is the highest known version, got: %T", sink)
}

// ConvertFrom implements apis.Convertible
func (sink *SubscribableStatus) ConvertFrom(ctx context.Context, source apis.Convertible) error {
	return fmt.Errorf("v1 is the highest known version, got: %T", source)
}

// ConvertTo implements apis.Convertible
func (source *SubscriberSpec) ConvertTo(ctx context.Context, sink apis.Convertible) error {
	return fmt.Errorf("v1 is the highest known version, got: %T", sink)
}

// ConvertFrom implements apis.Convertible
func (sink *SubscriberSpec) ConvertFrom(ctx context.Context, source apis.Convertible) error {
	return fmt.Errorf("v1 is the highest known version, got: %T", source)
}

// ConvertTo implements apis.Convertible
func (source *SubscriberStatus) ConvertTo(ctx context.Context, sink apis.Convertible) error {
	return fmt.Errorf("v1 is the highest known version, got: %T", sink)
}

// ConvertFrom implements apis.Convertible
func (sink *SubscriberStatus) ConvertFrom(ctx context.Context, source apis.Convertible) error {
	return fmt.Errorf("v1 is the highest known version, got: %T", source)
}
