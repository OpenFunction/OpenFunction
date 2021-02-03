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

const (
	// ApiServerSourceAddEventType is the ApiServerSource CloudEvent type for adds.
	ApiServerSourceAddEventType = "dev.knative.apiserver.resource.add"
	// ApiServerSourceUpdateEventType is the ApiServerSource CloudEvent type for updates.
	ApiServerSourceUpdateEventType = "dev.knative.apiserver.resource.update"
	// ApiServerSourceDeleteEventType is the ApiServerSource CloudEvent type for deletions.
	ApiServerSourceDeleteEventType = "dev.knative.apiserver.resource.delete"

	// ApiServerSourceAddRefEventType is the ApiServerSource CloudEvent type for ref adds.
	ApiServerSourceAddRefEventType = "dev.knative.apiserver.ref.add"
	// ApiServerSourceUpdateRefEventType is the ApiServerSource CloudEvent type for ref updates.
	ApiServerSourceUpdateRefEventType = "dev.knative.apiserver.ref.update"
	// ApiServerSourceDeleteRefEventType is the ApiServerSource CloudEvent type for ref deletions.
	ApiServerSourceDeleteRefEventType = "dev.knative.apiserver.ref.delete"
)

// ApiServerSourceEventReferenceModeTypes is the list of CloudEvent types the ApiServerSource with EventMode of ReferenceMode emits.
var ApiServerSourceEventReferenceModeTypes = []string{
	ApiServerSourceAddRefEventType,
	ApiServerSourceDeleteRefEventType,
	ApiServerSourceUpdateRefEventType,
}

// ApiServerSourceEventResourceModeTypes is the list of CloudEvent types the ApiServerSource with EventMode of ResourceMode emits.
var ApiServerSourceEventResourceModeTypes = []string{
	ApiServerSourceAddEventType,
	ApiServerSourceDeleteEventType,
	ApiServerSourceUpdateEventType,
}
