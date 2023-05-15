/*
Copyright 2022 The OpenFunction Authors.

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

package core

import (
	openfunction "github.com/openfunction/apis/core/v1beta2"
)

const (
	FunctionContainer = "function"
	FunctionPort      = "function-port"
)

type BuilderRun interface {

	// Start to build image.
	Start(builder *openfunction.Builder) error
	// Result get the build result.
	// "" means build has not been completed.
	// `Succeeded` means build completed.
	// Other means build failed.
	Result(builder *openfunction.Builder) (string, string, string, error)
	// Clean all resources which created by builder.
	Clean(builder *openfunction.Builder) error
	// Cancel the builder.
	Cancel(builder *openfunction.Builder) error
}

type ServingRun interface {
	Run(s *openfunction.Serving, cm map[string]string) error
	// Result get the serving result.
	// '' means serving is starting.
	// `Running` means serving is running.
	// Other means serving failed.
	Result(s *openfunction.Serving) (string, string, string, error)
	// Clean all resources which created by serving.
	Clean(s *openfunction.Serving) error
}
