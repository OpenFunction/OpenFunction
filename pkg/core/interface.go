package core

import (
	openfunction "github.com/openfunction/apis/core/v1alpha2"
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
	Result(builder *openfunction.Builder) (string, error)
	// Clean all resources which created by builder.
	Clean(builder *openfunction.Builder) error
}

type ServingRun interface {
	Run(s *openfunction.Serving) error
	// Result get the serving result.
	// '' means serving is starting.
	// `Running` means serving is running.
	// Other means serving failed.
	Result(s *openfunction.Serving) (string, error)
	// Clean all resources which created by serving.
	Clean(s *openfunction.Serving) error
}
