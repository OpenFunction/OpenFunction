package serving

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openfunction/pkg/core/serving/knative"
	"github.com/openfunction/pkg/core/serving/openfuncasync"
)

func Registry() []client.Object {
	var objs []client.Object
	objs = append(objs, knative.Registry()...)
	objs = append(objs, openfuncasync.Registry()...)
	return objs
}
