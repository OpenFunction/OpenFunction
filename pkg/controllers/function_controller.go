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
	"fmt"
	"github.com/go-logr/logr"
	openfunction "github.com/openfunction/pkg/apis/v1alpha1"
	"github.com/openfunction/pkg/util"
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
		log.V(10).Info("Function deleted", "error", err)
		return ctrl.Result{}, util.IgnoreNotFound(client.IgnoreNotFound(err))
	}

	if _, err := r.createOrUpdateFunc(&fn); err != nil {
		log.Error(err, "Failed to create function", "name", req.NamespacedName.String())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *FunctionReconciler) createOrUpdateFunc(fn *openfunction.Function) (ctrl.Result, error) {
	switch {
	case fn.Status.Phase == "" && fn.Status.State == "":
		if result, err := r.createOrUpdateBuilder(fn); err != nil {
			return result, err
		}
	case fn.Status.Phase == openfunction.ServingPhase && fn.Status.State == "":
		if result, err := r.createOrUpdateServing(fn); err != nil {
			return result, err
		}
	case fn.Status.Phase == openfunction.ServingPhase && fn.Status.State == openfunction.Created:
		if result, err := r.cleanupBuilder(fn); err != nil {
			return result, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *FunctionReconciler) createOrUpdateBuilder(fn *openfunction.Function) (ctrl.Result, error) {
	log := r.Log.WithName("createOrUpdateBuilder")

	var builder openfunction.Builder
	builder.Name = fmt.Sprintf("%s-%s", fn.Name, "builder")
	builder.Namespace = fn.Namespace

	if err := r.Delete(r.ctx, &builder); util.IgnoreNotFound(client.IgnoreNotFound(err)) != nil {
		log.Error(err, "Failed to delete builder", "namespace", builder.Namespace, "name", builder.Name)
		return ctrl.Result{}, err
	}

	builder.Spec.FuncName = fn.Spec.FuncName
	builder.Spec.FuncType = fn.Spec.FuncType
	builder.Spec.FuncVersion = fn.Spec.FuncVersion
	builder.Spec.Builder = fn.Spec.Builder
	builder.Spec.Image = fn.Spec.Image

	gitRepo := openfunction.GitRepo{}
	gitRepo.Init()
	builder.Spec.Source = &gitRepo
	fn.Spec.Source.DeepCopyInto(builder.Spec.Source)

	registry := openfunction.Registry{}
	registry.Init()
	builder.Spec.Registry = &registry
	fn.Spec.Registry.DeepCopyInto(builder.Spec.Registry)

	builder.SetOwnerReferences(nil)
	if err := ctrl.SetControllerReference(fn, &builder, r.Scheme); err != nil {
		log.Error(err, "Failed to SetOwnerReferences for builder", "namespace", builder.Namespace, "name", builder.Name)
		return ctrl.Result{}, err
	}

	if err := r.Create(r.ctx, &builder); err != nil {
		log.Error(err, "Failed to create builder", "namespace", builder.Namespace, "name", builder.Name)
		return ctrl.Result{}, err
	}

	status := openfunction.FunctionStatus{Phase: openfunction.BuildPhase, State: openfunction.Created}
	if err := r.updateStatus(fn, &status); err != nil {
		log.Error(err, "Failed to update builder status", "namespace", builder.Namespace, "name", builder.Name)
		return ctrl.Result{}, err
	}

	log.V(1).Info("Builder created", "namespace", builder.Namespace, "name", builder.Name)
	return ctrl.Result{}, nil
}

func (r *FunctionReconciler) createOrUpdateServing(fn *openfunction.Function) (ctrl.Result, error) {
	log := r.Log.WithName("createOrUpdateServing")
	var serving openfunction.Serving
	serving.Name = fmt.Sprintf("%s-%s", fn.Name, "serving")
	serving.Namespace = fn.Namespace

	if err := r.Delete(r.ctx, &serving); util.IgnoreNotFound(client.IgnoreNotFound(err)) != nil {
		log.Error(err, "Failed to delete serving", "namespace", serving.Namespace, "name", serving.Name)
		return ctrl.Result{}, err
	}

	serving.Spec.Image = fn.Spec.Image
	if fn.Spec.Port != nil {
		port := *fn.Spec.Port
		serving.Spec.Port = &port
	}

	serving.SetOwnerReferences(nil)
	if err := ctrl.SetControllerReference(fn, &serving, r.Scheme); err != nil {
		log.Error(err, "Failed to SetOwnerReferences for serving", "namespace", serving.Namespace, "name", serving.Name)
		return ctrl.Result{}, err
	}

	if err := r.Create(r.ctx, &serving); err != nil {
		log.Error(err, "Failed to create serving", "namespace", serving.Namespace, "name", serving.Name)
		return ctrl.Result{}, err
	}

	status := openfunction.FunctionStatus{Phase: openfunction.ServingPhase, State: openfunction.Created}
	if err := r.updateStatus(fn, &status); err != nil {
		log.Error(err, "Failed to update function serving status", "namespace", serving.Namespace, "name", serving.Name)
		return ctrl.Result{}, err
	}

	log.V(1).Info("Function serving created", "namespace", serving.Namespace, "name", serving.Name)
	return ctrl.Result{}, nil
}

func (r *FunctionReconciler) cleanupBuilder(fn *openfunction.Function) (ctrl.Result, error) {
	log := r.Log.WithName("cleanupBuilder")
	var builder openfunction.Builder
	builder.Name = fmt.Sprintf("%s-%s", fn.Name, "builder")
	builder.Namespace = fn.Namespace

	if err := r.Delete(r.ctx, &builder); err != nil {
		log.V(1).Info("Failed to delete builder", "namespace", builder.Namespace, "name", builder.Name, "error", err)
		return ctrl.Result{}, util.IgnoreNotFound(client.IgnoreNotFound(err))
	}

	log.V(1).Info("Function builder deleted", "namespace", builder.Namespace, "name", builder.Name)
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
		Complete(r)
}
