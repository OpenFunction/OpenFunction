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
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	kcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FunctionReconciler reconciles a Function object
type FunctionReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	ctx         context.Context
	tektonCache kcache.Cache

	builders map[string]*openfunction.Builder
	servings map[string]*openfunction.Serving
}

// +kubebuilder:rbac:groups=core.openfunction.io,resources=functions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.openfunction.io,resources=functions/status,verbs=get;update;patch
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

	if result, err := r.createOrUpdateBuilder(fn); err != nil {
		return result, err
	}

	if result, err := r.createOrUpdateServing(fn); err != nil {
		return result, err
	}

	// Serving is running, clean builder
	if fn.Status.Phase == openfunction.ServingPhase && fn.Status.State == openfunction.Created {
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

	if !r.needToCreateOrUpdateBuilder(fn) {
		log.V(1).Info("No need to update builder", "namespace", builder.Namespace, "name", builder.Name)
		return ctrl.Result{}, nil
	}

	if err := r.Delete(r.ctx, &builder); util.IgnoreNotFound(client.IgnoreNotFound(err)) != nil {
		log.Error(err, "Failed to delete builder", "namespace", builder.Namespace, "name", builder.Name)
		return ctrl.Result{}, err
	}

	builder.Spec = r.createBuilderSpec(fn)

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
		log.Error(err, "Failed to update function build status", "namespace", fn.Namespace, "name", fn.Name)
		return ctrl.Result{}, err
	}

	if r.builders == nil {
		r.builders = make(map[string]*openfunction.Builder)
	}
	r.builders[fmt.Sprintf("%s/%s", builder.Name, builder.Namespace)] = builder.DeepCopy()
	log.V(1).Info("Builder created", "namespace", builder.Namespace, "name", builder.Name)
	return ctrl.Result{}, nil
}

func (r *FunctionReconciler) createBuilderSpec(fn *openfunction.Function) openfunction.BuilderSpec {

	spec := openfunction.BuilderSpec{
		Version:  fn.Spec.Version,
		Params:   fn.Spec.Build.Params,
		Builder:  fn.Spec.Build.Builder,
		SrcRepo:  nil,
		Image:    fn.Spec.Image,
		Registry: nil,
		Port:     fn.Spec.Port,
	}

	gitRepo := openfunction.GitRepo{}
	gitRepo.Init()
	spec.SrcRepo = &gitRepo
	fn.Spec.Build.SrcRepo.DeepCopyInto(spec.SrcRepo)

	registry := openfunction.Registry{}
	registry.Init()
	spec.Registry = &registry
	fn.Spec.Build.Registry.DeepCopyInto(spec.Registry)

	return spec
}

func (r *FunctionReconciler) createOrUpdateServing(fn *openfunction.Function) (ctrl.Result, error) {
	log := r.Log.WithName("createOrUpdateServing")
	var serving openfunction.Serving
	serving.Name = fmt.Sprintf("%s-%s", fn.Name, "serving")
	serving.Namespace = fn.Namespace

	if !r.needToCreateOrUpdateServing(fn) {
		log.V(1).Info("No need to update serving", "namespace", serving.Namespace, "name", serving.Name)
		return ctrl.Result{}, nil
	}

	if err := r.Delete(r.ctx, &serving); util.IgnoreNotFound(client.IgnoreNotFound(err)) != nil {
		log.Error(err, "Failed to delete serving", "namespace", serving.Namespace, "name", serving.Name)
		return ctrl.Result{}, err
	}

	serving.Spec = r.createServingSpec(fn)

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
		log.Error(err, "Failed to update function serving status", "namespace", fn.Namespace, "name", fn.Name)
		return ctrl.Result{}, err
	}

	if r.servings == nil {
		r.servings = make(map[string]*openfunction.Serving)
	}
	r.servings[fmt.Sprintf("%s/%s", serving.Name, serving.Namespace)] = serving.DeepCopy()
	log.V(1).Info("Function serving created", "namespace", serving.Namespace, "name", serving.Name)
	return ctrl.Result{}, nil
}

func (r *FunctionReconciler) createServingSpec(fn *openfunction.Function) openfunction.ServingSpec {

	spec := openfunction.ServingSpec{
		Version: fn.Spec.Version,
		Image:   fn.Spec.Image,
	}

	if fn.Spec.Port != nil {
		port := *fn.Spec.Port
		spec.Port = &port
	}

	if fn.Spec.Serving != nil {
		spec.Params = fn.Spec.Serving.Params
	}

	if fn.Spec.Serving != nil && fn.Spec.Serving.Runtime != nil {
		spec.Runtime = fn.Spec.Serving.Runtime
	} else {
		runt := openfunction.Knative
		spec.Runtime = &runt
	}

	return spec
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
	f := openfunction.Function{}

	if err := r.Get(r.ctx, client.ObjectKey{Name: fn.Name, Namespace: fn.Namespace}, &f); err != nil {
		return err
	}

	status.DeepCopyInto(&f.Status)
	if err := r.Status().Update(r.ctx, &f); err != nil {
		return err
	}
	return nil
}

func (r *FunctionReconciler) needToCreateOrUpdateBuilder(fn *openfunction.Function) bool {

	var old *openfunction.Builder
	if r.builders == nil {
		r.builders = make(map[string]*openfunction.Builder)
	}
	old = r.builders[fmt.Sprintf("%s-builder/%s", fn.Name, fn.Namespace)]

	// Builder had not created, or the operator just startd.
	if old == nil {
		var builder openfunction.Builder
		key := client.ObjectKey{Namespace: fn.Namespace, Name: fmt.Sprintf("%s-%s", fn.Name, "builder")}
		if err := r.Get(context.Background(), key, &builder); err == nil {
			// Get the exsit  builder as old builder.
			old = builder.DeepCopy()
			r.builders[fmt.Sprintf("%s/%s", builder.Name, builder.Namespace)] = old
		} else {
			return fn.Status.Phase != openfunction.ServingPhase
		}
	}

	needToCreateOrUpdate := false

	newSpec := r.createBuilderSpec(fn)
	if !equality.Semantic.DeepEqual(old.Spec, newSpec) {
		needToCreateOrUpdate = true
	}

	var builder openfunction.Builder
	key := client.ObjectKey{Namespace: fn.Namespace, Name: fmt.Sprintf("%s-%s", fn.Name, "builder")}
	if err := r.Get(context.Background(), key, &builder); err != nil {
		if errors.IsNotFound(err) {
			// If the builder is deleted before the build is completed, the builder needs to be recreated.
			if fn.Status.Phase == openfunction.BuildPhase {
				needToCreateOrUpdate = true
			}
		}
	}

	return needToCreateOrUpdate
}

func (r *FunctionReconciler) needToCreateOrUpdateServing(fn *openfunction.Function) bool {

	// The build is not completed, no need to create or update serving.
	if fn.Status.Phase != openfunction.ServingPhase {
		return false
	}

	var old *openfunction.Serving
	if r.servings == nil {
		r.servings = make(map[string]*openfunction.Serving)
	}
	old = r.servings[fmt.Sprintf("%s-serving/%s", fn.Name, fn.Namespace)]

	// Serving had not created, or the operator just startd.
	if old == nil {

		var serving openfunction.Serving
		key := client.ObjectKey{Namespace: fn.Namespace, Name: fmt.Sprintf("%s-%s", fn.Name, "serving")}
		// Get the exsit serving as old serving.
		if err := r.Get(context.Background(), key, &serving); err == nil {
			old = serving.DeepCopy()
			r.servings[fmt.Sprintf("%s/%s", serving.Name, serving.Namespace)] = old
		} else {
			// Serving dose not exist, need to create.
			return true
		}
	}

	// Build had completed, need to create serving.
	if fn.Status.State == "" {
		return true
	}

	needToCreateOrUpdate := false

	var serving openfunction.Serving
	key := client.ObjectKey{Namespace: fn.Namespace, Name: fmt.Sprintf("%s-%s", fn.Name, "serving")}
	if err := r.Get(context.Background(), key, &serving); err != nil {
		if errors.IsNotFound(err) {
			// If the serving is deleted, need to be recreated.
			needToCreateOrUpdate = true
		}
	}

	// Serving changed, need to update.
	newSpec := r.createServingSpec(fn)
	if !equality.Semantic.DeepEqual(old.Spec, newSpec) {
		needToCreateOrUpdate = true
	}

	return needToCreateOrUpdate
}

func (r *FunctionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunction.Function{}).
		Owns(&openfunction.Builder{}).
		Owns(&openfunction.Serving{}).
		Complete(r)
}
