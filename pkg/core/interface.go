package core

import openfunction "github.com/openfunction/apis/core/v1alpha2"

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
}

type ServingRun interface {
	Run(s *openfunction.Serving) error
}
