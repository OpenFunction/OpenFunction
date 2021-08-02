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

	openfunction "github.com/openfunction/apis/core/v1alpha1"
	"github.com/openfunction/pkg/util"
)

// BuilderReconciler reconciles a Builder object
type BuilderReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	ctx    context.Context
}

//+kubebuilder:rbac:groups=core.openfunction.io,resources=builders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.openfunction.io,resources=builders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tekton.dev,resources=tasks;pipelineresources;pipelines;pipelineruns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps;persistentvolumeclaims;serviceaccounts;secrets;services,verbs=get;list;watch;create;update;patch;delete

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

	var builder openfunction.Builder

	if err := r.Get(ctx, req.NamespacedName, &builder); err != nil {
		log.V(1).Info("Builder deleted", "error", err)
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	if _, err := r.createOrUpdateBuild(&builder); err != nil {
		log.Error(err, "Failed to create build", "Builder", req.NamespacedName.String())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BuilderReconciler) createOrUpdateBuild(builder *openfunction.Builder) (ctrl.Result, error) {
	log := r.Log.WithName("createOrUpdate")

	if builder.Status.Phase == openfunction.BuildPhase && builder.Status.State == openfunction.Launched {
		return ctrl.Result{}, nil
	}

	status := openfunction.BuilderStatus{Phase: openfunction.BuildPhase, State: openfunction.Launching}
	if err := r.updateStatus(builder, &status); err != nil {
		log.Error(err, "Failed to update builder Launching status", "name", builder.Name, "namespace", builder.Namespace)
		return ctrl.Result{}, err
	}

	if err := r.CreateOrUpdateTask(builder, gitCloneTask); err != nil {
		log.Error(err, "Failed to create task", "task", gitCloneTask)
		return ctrl.Result{}, err
	}

	if err := r.CreateOrUpdateTask(builder, buildTask); err != nil {
		log.Error(err, "Failed to create task", "task", buildTask)
		return ctrl.Result{}, err
	}

	if err := r.CreateOrUpdateBuildpackPVCs(builder); err != nil {
		log.Error(err, "Failed to create buildpack pvcs", "namespace", builder.Namespace)
		return ctrl.Result{}, err
	}

	if err := r.CreateOrUpdateRegistryAuth(builder); err != nil {
		log.Error(err, "Failed to create registry auth", "namespace", builder.Namespace)
		return ctrl.Result{}, err
	}

	if err := r.CreateOrUpdatePipeline(builder); err != nil {
		log.Error(err, "Failed to create Pipeline", "namespace", builder.Namespace)
		return ctrl.Result{}, err
	}

	if err := r.CreateOrUpdatePipelineRun(builder); err != nil {
		log.Error(err, "Failed to create PipelineRun", "namespace", builder.Namespace)
		return ctrl.Result{}, err
	}

	status = openfunction.BuilderStatus{Phase: openfunction.BuildPhase, State: openfunction.Launched}
	if err := r.updateStatus(builder, &status); err != nil {
		log.Error(err, "Failed to update builder Launched status", "name", builder.Name, "namespace", builder.Namespace)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *BuilderReconciler) updateStatus(builder *openfunction.Builder, status *openfunction.BuilderStatus) error {

	status.DeepCopyInto(&builder.Status)
	if err := r.Status().Update(r.ctx, builder); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuilderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunction.Builder{}).
		Complete(r)
}
