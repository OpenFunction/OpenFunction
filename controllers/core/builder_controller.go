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
	"sigs.k8s.io/controller-runtime/pkg/manager"

	openfunction "github.com/openfunction/apis/core/v1alpha2"
	"github.com/openfunction/pkg/core"
	"github.com/openfunction/pkg/core/builder/shipwright"
	"github.com/openfunction/pkg/util"
)

// BuilderReconciler reconciles a Builder object
type BuilderReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	ctx    context.Context
	owns   []client.Object
}

func NewBuilderReconciler(mgr manager.Manager) *BuilderReconciler {

	r := &BuilderReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("Builder"),
	}

	r.owns = append(r.owns, shipwright.Registry()...)
	return r
}

//+kubebuilder:rbac:groups=core.openfunction.io,resources=builders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.openfunction.io,resources=builders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=shipwright.io,resources=builds;buildruns,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Builder object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *BuilderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ctx = ctx
	log := r.Log.WithValues("Builder", req.NamespacedName)

	builder := &openfunction.Builder{}

	if err := r.Get(ctx, req.NamespacedName, builder); err != nil {
		if util.IsNotFound(err) {
			log.V(1).Info("Builder deleted")
		}
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	builderRun := r.createBuilderRun()

	// Builder is running, no need to update.
	if builder.Status.Phase != "" && builder.Status.State != "" {
		// Update the status of the builder according to the result of the build.
		if err := r.getBuilderResult(builder, builderRun); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Reset builder status.
	builder.Status = openfunction.BuilderStatus{}
	if err := r.Status().Update(r.ctx, builder); err != nil {
		log.Error(err, "Failed to reset builder status", "name", builder.Name, "namespace", builder.Namespace)
		return ctrl.Result{}, err
	}

	if err := builderRun.Start(builder); err != nil {
		log.Error(err, "Failed to start builder", "name", builder.Name, "namespace", builder.Namespace)
		return ctrl.Result{}, err
	}

	builder.Status.Phase = openfunction.BuildPhase
	builder.Status.State = openfunction.Building
	if err := r.Status().Update(r.ctx, builder); err != nil {
		log.Error(err, "Failed to update builder status", "name", builder.Name, "namespace", builder.Namespace)
		return ctrl.Result{}, err
	}

	log.V(1).Info("Builder is running", "namespace", builder.Namespace, "name", builder.Name)

	return ctrl.Result{}, nil
}

func (r *BuilderReconciler) createBuilderRun() core.BuilderRun {

	return shipwright.NewBuildRun(r.ctx, r.Client, r.Scheme, r.Log)
}

// Update the status of the builder according to the result of the build.
func (r *BuilderReconciler) getBuilderResult(builder *openfunction.Builder, builderRun core.BuilderRun) error {
	log := r.Log.WithName("GetBuilderResult")

	res, err := builderRun.Result(builder)
	if err != nil {
		log.Error(err, "Get build result error", "name", builder.Name, "namespace", builder.Namespace)
		return err
	}

	// Build did not complete.
	if res == "" {
		return nil
	}

	if res != builder.Status.State {
		builder.Status.State = res
		if err := r.Status().Update(r.ctx, builder); err != nil {
			return err
		}

		log.V(1).Info("Update builder status", "namespace", builder.Namespace, "name", builder.Name, "state", res)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuilderReconciler) SetupWithManager(mgr ctrl.Manager) error {

	b := ctrl.NewControllerManagedBy(mgr).
		For(&openfunction.Builder{})

	for _, own := range r.owns {
		b.Owns(own)
	}

	return b.Complete(r)
}
