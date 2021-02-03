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

package sources

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis/duck"
)

const (
	GroupName = "sources.knative.dev"

	// SourceDuckLabelKey is the label key to indicate
	// whether the CRD is a Source duck type.
	// Valid values: "true" or "false"
	SourceDuckLabelKey = duck.GroupName + "/source"

	// SourceDuckLabelValue is the label value to indicate
	// the CRD is a Source duck type.
	SourceDuckLabelValue = "true"
)

var (
	// ApiServerSourceResource respresents a Knative Eventing Sources ApiServerSource
	ApiServerSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "apiserversources",
	}
	// PingSourceResource respresents a Knative Eventing Sources PingSource
	PingSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "pingsources",
	}
	// SinkBindingResource respresents a Knative Eventing Sources SinkBinding
	SinkBindingResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "sinkbindings",
	}

	// ContainerSourceResource respresents a Knative Eventing Sources ContainerSource
	ContainerSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "containersources",
	}
)
