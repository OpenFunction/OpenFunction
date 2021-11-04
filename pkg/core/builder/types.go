package builder

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openfunction/pkg/core/builder/shipwright"
)

func Registry() []client.Object {
	return shipwright.Registry()
}
