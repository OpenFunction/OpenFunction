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

package v1alpha2

import (
	"context"
	"fmt"
	"strings"

	"knative.dev/eventing/pkg/apis/sources/v1beta1"
	"knative.dev/pkg/apis"
)

// ConvertTo implements apis.Convertible
// Converts source from v1alpha2.PingSource into a higher version.
func (source *PingSource) ConvertTo(ctx context.Context, obj apis.Convertible) error {
	switch sink := obj.(type) {
	case *v1beta1.PingSource:
		sink.ObjectMeta = source.ObjectMeta
		sink.Spec = v1beta1.PingSourceSpec{
			JsonData:   source.Spec.JsonData,
			SourceSpec: source.Spec.SourceSpec,
		}
		sink.Status = v1beta1.PingSourceStatus{
			SourceStatus: source.Status.SourceStatus,
		}

		// in v1beta1, timezone has its own field
		schedule := source.Spec.Schedule
		if strings.HasPrefix(schedule, "TZ=") || strings.HasPrefix(schedule, "CRON_TZ=") {
			i := strings.Index(schedule, " ")
			eq := strings.Index(schedule, "=")
			sink.Spec.Timezone = schedule[eq+1 : i]
			sink.Spec.Schedule = strings.TrimSpace(schedule[i:])
		} else {
			sink.Spec.Schedule = schedule
		}

		return nil
	default:
		return apis.ConvertToViaProxy(ctx, source, &v1beta1.PingSource{}, sink)
	}
}

// ConvertFrom implements apis.Convertible
// Converts obj from a higher version into v1alpha2.PingSource.
func (sink *PingSource) ConvertFrom(ctx context.Context, obj apis.Convertible) error {
	switch source := obj.(type) {
	case *v1beta1.PingSource:
		sink.ObjectMeta = source.ObjectMeta
		sink.Spec = PingSourceSpec{
			JsonData:   source.Spec.JsonData,
			SourceSpec: source.Spec.SourceSpec,
		}
		sink.Status = PingSourceStatus{
			SourceStatus: source.Status.SourceStatus,
		}

		if source.Spec.Timezone != "" {
			sink.Spec.Schedule = fmt.Sprintf("CRON_TZ=%s %s", source.Spec.Timezone, source.Spec.Schedule)
		} else {
			sink.Spec.Schedule = source.Spec.Schedule
		}
		return nil
	default:
		return apis.ConvertFromViaProxy(ctx, source, &v1beta1.PingSource{}, sink)
	}
}
