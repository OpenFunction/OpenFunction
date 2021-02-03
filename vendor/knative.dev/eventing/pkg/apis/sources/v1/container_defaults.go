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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"knative.dev/pkg/apis"
)

func (s *ContainerSource) SetDefaults(ctx context.Context) {
	withName := apis.WithinParent(ctx, s.ObjectMeta)
	s.Spec.SetDefaults(withName)
}

func (ss *ContainerSourceSpec) SetDefaults(ctx context.Context) {
	containers := make([]corev1.Container, 0, len(ss.Template.Spec.Containers))
	for i, c := range ss.Template.Spec.Containers {
		// If the Container specified has no name, then default to "<source_name>_<i>".
		if c.Name == "" {
			c.Name = fmt.Sprintf("%s-%d", apis.ParentMeta(ctx).Name, i)
		}
		containers = append(containers, c)
	}
	ss.Template.Spec.Containers = containers
}
