/*
Copyright 2019 The Knative Authors

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

package flows

import "k8s.io/apimachinery/pkg/runtime/schema"

const (
	GroupName = "flows.knative.dev"
)

var (
	// SequenceResource represents a Knative Sequence
	SequenceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "sequences",
	}
	// ParallelResource represents a Knative Parallel
	ParallelResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "parallels",
	}
)
