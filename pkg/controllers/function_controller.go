/*


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

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	openfunction "github.com/openfunction/pkg/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FunctionReconciler reconciles a Function object
type FunctionReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	ctx         context.Context
	tektonCache cache.Cache
}

// +kubebuilder:rbac:groups=openfunction.io,resources=functions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=openfunction.io,resources=functions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tekton.dev,resources=tasks;pipelineresources;pipelines;pipelineruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps;persistentvolumeclaims;serviceaccounts;secrets,verbs=get;list;watch;create;update;patch;delete

func (r *FunctionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	r.ctx = ctx
	log := r.Log.WithValues("function", req.NamespacedName)

	var fn openfunction.Function

	if err := r.Get(ctx, req.NamespacedName, &fn); err != nil {
		log.Error(err, "Unable to get Function", "function", req.NamespacedName.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if _, err := r.createOrUpdateBuild(&fn); err != nil {
		log.Error(err, "Failed to create build", "function", req.NamespacedName.String())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *FunctionReconciler) createOrUpdateBuild(fn *openfunction.Function) (ctrl.Result, error) {
	log := r.Log.WithName("createOrUpdate")

	if fn.Status.Phase != "" && fn.Status.State != "" {
		return ctrl.Result{}, nil
	}

	status := openfunction.FunctionStatus{Phase: openfunction.Build, State: openfunction.Launching}
	if err := r.updateStatus(fn, &status); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.CreateOrUpdateTask(fn, gitCloneTask); err != nil {
		log.Error(err, "Failed to create task", "task", gitCloneTask)
		return ctrl.Result{}, err
	}

	if err := r.CreateOrUpdateTask(fn, buildTask); err != nil {
		log.Error(err, "Failed to create task", "task", buildTask)
		return ctrl.Result{}, err
	}

	if err := r.CreateOrUpdateConfigMap(fn); err != nil {
		log.Error(err, "Failed to create configmap", "namaspace", fn.Namespace)
		return ctrl.Result{}, err
	}

	if err := r.CreateOrUpdateBuildpackPVCs(fn); err != nil {
		log.Error(err, "Failed to create buildpack pvcs", "namaspace", fn.Namespace)
		return ctrl.Result{}, err
	}

	if err := r.CreateOrUpdateRegistryAuth(fn); err != nil {
		log.Error(err, "Failed to create registry auth", "namaspace", fn.Namespace)
		return ctrl.Result{}, err
	}

	if err := r.CreateOrUpdatePipelineResource(fn); err != nil {
		log.Error(err, "Failed to create PipelineResource", "namaspace", fn.Namespace)
		return ctrl.Result{}, err
	}

	if err := r.CreateOrUpdatePipeline(fn); err != nil {
		log.Error(err, "Failed to create Pipeline", "namaspace", fn.Namespace)
		return ctrl.Result{}, err
	}

	if err := r.CreateOrUpdatePipelineRun(fn); err != nil {
		log.Error(err, "Failed to create PipelineRun", "namaspace", fn.Namespace)
		return ctrl.Result{}, err
	}

	status = openfunction.FunctionStatus{Phase: openfunction.Build, State: openfunction.Launched}
	if err := r.updateStatus(fn, &status); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *FunctionReconciler) updateStatus(fn *openfunction.Function, status *openfunction.FunctionStatus) error {
	status.DeepCopyInto(&fn.Status)
	if err := r.Status().Update(r.ctx, fn); err != nil {
		return err
	}
	return nil
}

func (r *FunctionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunction.Function{}).
		//		Owns(&pipeline.Task{}).
		//		Owns(&v1.PersistentVolumeClaim{}).
		//		Owns(&v1.ServiceAccount{}).
		//		Owns(&pipelineres.PipelineResource{}).
		//		Owns(&pipeline.Pipeline{}).
		//		Owns(&pipeline.PipelineRun{}).
		Complete(r)
}
