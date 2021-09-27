/*
Copyright 2021.

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
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openfunction "github.com/openfunction/apis/core/v1alpha2"
	"github.com/openfunction/pkg/core"
	"github.com/openfunction/pkg/core/serving/knative"
	"github.com/openfunction/pkg/core/serving/openfuncasync"
	"github.com/openfunction/pkg/util"
)

// ServingReconciler reconciles a Serving object
type ServingReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	ctx    context.Context
}

//+kubebuilder:rbac:groups=core.openfunction.io,resources=servings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.openfunction.io,resources=servings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.openfunction.io,resources=servings/finalizers,verbs=update
//+kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dapr.io,resources=components;subscriptions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keda.sh,resources=scaledjobs;scaledobjects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Serving object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ServingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ctx = ctx
	log := r.Log.WithValues("serving", req.NamespacedName)

	var s openfunction.Serving

	if err := r.Get(ctx, req.NamespacedName, &s); err != nil {
		if util.IsNotFound(err) {
			log.V(1).Info("Serving deleted")
		}
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	// Serving is running, no need to create.
	if s.Status.Phase != "" && s.Status.State != "" {
		return ctrl.Result{}, nil
	}

	servingRun := r.getServingRun(&s)
	if util.InterfaceIsNil(servingRun) {
		log.Error(nil, "Unknown runtime", "runtime", *s.Spec.Runtime)
		return ctrl.Result{}, nil
	}

	if err := servingRun.Run(&s); err != nil {
		log.Error(err, "Failed to run serving", "name", s.Name, "namespace", s.Namespace)
		return ctrl.Result{}, err
	}

	s.Status.Phase = openfunction.ServingPhase
	s.Status.State = openfunction.Running
	if err := r.Status().Update(r.ctx, &s); err != nil {
		log.Error(err, "Failed to update serving status", "name", s.Name, "namespace", s.Namespace)
		return ctrl.Result{}, err
	}

	log.V(1).Info("Serving is running", "name", s.Name, "namespace", s.Namespace)

	return ctrl.Result{}, nil
}

func (r *ServingReconciler) getServingRun(s *openfunction.Serving) core.ServingRun {

	switch *s.Spec.Runtime {
	case openfunction.OpenFuncAsync:
		return openfuncasync.NewServingRun(r.ctx, r.Client, r.Scheme, r.Log)
	case openfunction.Knative:
		return knative.NewServingRun(r.ctx, r.Client, r.Scheme, r.Log)
	default:
		return nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunction.Serving{}).
		Complete(r)
}
