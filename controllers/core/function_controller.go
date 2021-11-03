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
	"fmt"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openfunction "github.com/openfunction/apis/core/v1alpha2"
	"github.com/openfunction/pkg/constants"
	"github.com/openfunction/pkg/util"
)

// FunctionReconciler reconciles a Function object
type FunctionReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	ctx    context.Context
}

//+kubebuilder:rbac:groups=core.openfunction.io,resources=functions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.openfunction.io,resources=functions/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Function object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *FunctionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ctx = ctx
	log := r.Log.WithValues("Function", req.NamespacedName)

	var fn openfunction.Function

	if err := r.Get(ctx, req.NamespacedName, &fn); err != nil {

		if util.IsNotFound(err) {
			log.V(1).Info("Function deleted")
		}

		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	if err := r.createBuilder(&fn); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.createServing(&fn); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *FunctionReconciler) createBuilder(fn *openfunction.Function) error {
	log := r.Log.WithName("CreateBuilder").
		WithValues("Function", fmt.Sprintf("%s/%s", fn.Namespace, fn.Name))

	if !r.needToCreateBuilder(fn) {
		if err := r.updateFuncWithBuilderStatus(fn); err != nil {
			return err
		}

		log.V(1).Info("No need to create Builder")
		return nil
	}

	// Reset function build status.
	if fn.Status.Build == nil {
		fn.Status.Build = &openfunction.Condition{}
	}
	fn.Status.Build.State = ""
	fn.Status.Build.ResourceRef = ""
	if err := r.Status().Update(r.ctx, fn); err != nil {
		log.Error(err, "Failed to reset function build status")
		return err
	}

	if err := r.deleteOldBuilder(fn); err != nil {
		log.Error(err, "Failed to clean builder")
		return err
	}

	// If `spec.Build` is nil, skip build, else create a new builder.
	if fn.Spec.Build == nil {
		fn.Status.Build = &openfunction.Condition{
			State:        openfunction.Skipped,
			ResourceHash: util.Hash(openfunction.BuilderSpec{}),
		}
		fn.Status.Serving = &openfunction.Condition{}
		if err := r.Status().Update(r.ctx, fn); err != nil {
			log.Error(err, "Failed to update function build status")
			return err
		}

		log.V(1).Info("Skip build")
		return nil
	}

	builder := &openfunction.Builder{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "builder-",
			Namespace:    fn.Namespace,
			Labels: map[string]string{
				constants.FunctionLabel: fn.Name,
			},
		},
		Spec: r.createBuilderSpec(fn),
	}
	builder.SetOwnerReferences(nil)
	if err := ctrl.SetControllerReference(fn, builder, r.Scheme); err != nil {
		log.Error(err, "Failed to SetOwnerReferences for builder")
		return err
	}

	if err := r.Create(r.ctx, builder); err != nil {
		log.Error(err, "Failed to create builder")
		return err
	}

	fn.Status.Build = &openfunction.Condition{
		State:        openfunction.Created,
		ResourceRef:  builder.Name,
		ResourceHash: util.Hash(builder.Spec),
	}
	if err := r.Status().Update(r.ctx, fn); err != nil {
		log.Error(err, "Failed to update function build status")
		return err
	}

	log.V(1).Info("Builder created", "Builder", builder.Name)
	return nil
}

// Update the status of the function with the result of the build.
func (r *FunctionReconciler) updateFuncWithBuilderStatus(fn *openfunction.Function) error {
	log := r.Log.WithName("UpdateFuncWithBuilderStatus").
		WithValues("Function", fmt.Sprintf("%s/%s", fn.Namespace, fn.Name))

	// Build had not created or not completed, no need to update function status.
	if fn.Status.Build == nil || fn.Status.Build.State == "" {
		return nil
	}

	var builder openfunction.Builder
	builder.Name = fn.Status.Build.ResourceRef
	builder.Namespace = fn.Namespace

	if builder.Name == "" {
		log.V(1).Info("Function has no builder")
		return nil
	}

	if err := r.Get(r.ctx, client.ObjectKey{Namespace: builder.Namespace, Name: builder.Name}, &builder); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get builder", "Builder", builder.Name)
		return util.IgnoreNotFound(err)
	}

	// The build does not start.
	if builder.Status.Phase != openfunction.BuildPhase {
		return nil
	}

	// If builder status changed, update function build status.
	if fn.Status.Build.State != builder.Status.State {
		fn.Status.Build.State = builder.Status.State
		// If build had complete, update function serving status.
		if builder.Status.State == openfunction.Succeeded {
			if fn.Status.Serving == nil {
				fn.Status.Serving = &openfunction.Condition{}
			}

			fn.Status.Serving.State = ""
		}
	}

	if err := r.Status().Update(r.ctx, fn); err != nil {
		log.Error(err, "Failed to update function status")
		return err
	}

	if builder.Status.State == openfunction.Succeeded {
		return r.cleanupBuilder(fn)
	}

	return nil
}

func (r *FunctionReconciler) cleanupBuilder(fn *openfunction.Function) error {
	log := r.Log.WithName("CleanupBuilder").
		WithValues("Function", fmt.Sprintf("%s/%s", fn.Namespace, fn.Name))

	if fn.Status.Build == nil {
		return nil
	}
	var builder openfunction.Builder
	builder.Name = fn.Status.Build.ResourceRef
	builder.Namespace = fn.Namespace

	if builder.Name == "" {
		return nil
	}

	if err := r.Delete(r.ctx, &builder); err != nil {
		return util.IgnoreNotFound(err)
	}

	log.V(1).Info("Function builder cleanup", "Builder", builder.Name)
	return nil
}

// Delete old builders which created by function.
func (r *FunctionReconciler) deleteOldBuilder(fn *openfunction.Function) error {
	log := r.Log.WithName("DeleteOldBuilder").
		WithValues("Function", fmt.Sprintf("%s/%s", fn.Namespace, fn.Name))

	builders := &openfunction.BuilderList{}
	if err := r.List(r.ctx, builders, client.InNamespace(fn.Namespace), client.MatchingLabels{constants.FunctionLabel: fn.Name}); err != nil {
		return err
	}

	for _, item := range builders.Items {
		if err := r.Delete(context.Background(), &item); util.IgnoreNotFound(err) != nil {
			return err
		}
		log.V(1).Info("Delete Builder", "builder", item.Name)
	}

	return nil
}

func (r *FunctionReconciler) createBuilderSpec(fn *openfunction.Function) openfunction.BuilderSpec {

	if fn.Spec.Build == nil {
		return openfunction.BuilderSpec{}
	}

	spec := openfunction.BuilderSpec{
		Params:             fn.Spec.Build.Params,
		Env:                fn.Spec.Build.Env,
		Builder:            fn.Spec.Build.Builder,
		BuilderCredentials: fn.Spec.Build.BuilderCredentials,
		Image:              fn.Spec.Image,
		ImageCredentials:   fn.Spec.ImageCredentials,
		Port:               fn.Spec.Port,
		Timeout:            fn.Spec.Build.Timeout,
	}

	spec.SrcRepo = &openfunction.GitRepo{}
	spec.SrcRepo.Init()
	fn.Spec.Build.SrcRepo.DeepCopyInto(spec.SrcRepo)

	return spec
}

func (r *FunctionReconciler) createServing(fn *openfunction.Function) error {
	log := r.Log.WithName("CreateServing").
		WithValues("Function", fmt.Sprintf("%s/%s", fn.Namespace, fn.Name))

	if !r.needToCreateServing(fn) {
		log.V(1).Info("No need to create Serving")

		if err := r.updateFuncWithServingStatus(fn); err != nil {
			return err
		}

		return nil
	}

	// Reset function serving status.
	if fn.Status.Serving == nil {
		fn.Status.Serving = &openfunction.Condition{}
	}
	fn.Status.Serving.State = ""
	fn.Status.Serving.ResourceRef = ""
	fn.Status.Serving.ResourceHash = ""
	if err := r.Status().Update(r.ctx, fn); err != nil {
		log.Error(err, "Failed to update function serving status")
		return err
	}

	if err := r.cleanServing(fn); err != nil {
		log.Error(err, "Failed to clean Serving")
		return err
	}

	if fn.Spec.Serving == nil {
		fn.Status.Serving = &openfunction.Condition{
			State:        openfunction.Skipped,
			ResourceHash: util.Hash(openfunction.ServingSpec{}),
		}
		if err := r.Status().Update(r.ctx, fn); err != nil {
			log.Error(err, "Failed to update function serving status")
			return err
		}

		log.V(1).Info("Skip serving")
		return nil
	}

	serving := &openfunction.Serving{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "serving-",
			Namespace:    fn.Namespace,
			Labels: map[string]string{
				constants.FunctionLabel: fn.Name,
			},
		},
		Spec: r.createServingSpec(fn),
	}
	serving.SetOwnerReferences(nil)
	if err := ctrl.SetControllerReference(fn, serving, r.Scheme); err != nil {
		log.Error(err, "Failed to SetOwnerReferences for serving")
		return err
	}

	if err := r.Create(r.ctx, serving); err != nil {
		log.Error(err, "Failed to create serving")
		return err
	}

	fn.Status.Serving = &openfunction.Condition{
		State:          openfunction.Created,
		ResourceRef:    serving.Name,
		ResourceHash:   util.Hash(serving.Spec),
		OldResourceRef: fn.Status.Serving.OldResourceRef,
	}
	if err := r.Status().Update(r.ctx, fn); err != nil {
		log.Error(err, "Failed to update function serving status")
		return err
	}

	log.V(1).Info("Serving created", "Serving", serving.Name)
	return nil
}

// Update the status of the function with the result of the serving.
func (r *FunctionReconciler) updateFuncWithServingStatus(fn *openfunction.Function) error {
	log := r.Log.WithName("UpdateFuncWithServingStatus").
		WithValues("Function", fmt.Sprintf("%s/%s", fn.Namespace, fn.Name))

	// Serving had not created, no need to update function status.
	if fn.Status.Serving == nil || fn.Status.Serving.State == "" {
		return nil
	}

	var serving openfunction.Serving
	serving.Name = fn.Status.Serving.ResourceRef
	serving.Namespace = fn.Namespace

	if serving.Name == "" {
		log.V(1).Info("Function has no serving")
		return nil
	}

	if err := r.Get(r.ctx, client.ObjectKeyFromObject(&serving), &serving); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get serving", "Serving", serving.Name)
		return util.IgnoreNotFound(err)
	}

	// Serving does not start.
	if serving.Status.Phase != openfunction.ServingPhase {
		return nil
	}

	// If serving status changed, update function serving status.
	if fn.Status.Serving.State != serving.Status.State {
		fn.Status.Serving.State = serving.Status.State

		// If new serving is running, clean old serving.
		if serving.Status.State == openfunction.Running {
			fn.Status.Serving.OldResourceRef = fn.Status.Serving.ResourceRef
			if err := r.cleanServing(fn); err != nil {
				log.Error(err, "Failed to clean Serving")
				return err
			}
			log.V(1).Info("Serving is running", "serving", serving.Name)
		}
	}

	if err := r.Status().Update(r.ctx, fn); err != nil {
		log.Error(err, "Failed to update function status")
		return err
	}

	return nil
}

// Clean up redundant servings caused by the `createOrUpdateBuilder` function failed.
func (r *FunctionReconciler) cleanServing(fn *openfunction.Function) error {
	log := r.Log.WithName("CleanServing").
		WithValues("Function", fmt.Sprintf("%s/%s", fn.Namespace, fn.Name))

	name := ""
	oldName := ""
	if fn.Status.Serving != nil {
		name = fn.Status.Serving.ResourceRef
		oldName = fn.Status.Serving.OldResourceRef
	}

	servings := &openfunction.ServingList{}
	if err := r.List(r.ctx, servings, client.InNamespace(fn.Namespace), client.MatchingLabels{constants.FunctionLabel: fn.Name}); err != nil {
		return err
	}

	for _, item := range servings.Items {
		if item.Name != name && item.Name != oldName {
			if err := r.Delete(context.Background(), &item); util.IgnoreNotFound(err) != nil {
				return err
			}
			log.V(1).Info("Delete Serving", "Serving", item.Name)
		}
	}

	return nil
}

func (r *FunctionReconciler) createServingSpec(fn *openfunction.Function) openfunction.ServingSpec {

	if fn.Spec.Serving == nil {
		return openfunction.ServingSpec{}
	}

	spec := openfunction.ServingSpec{
		Version:          fn.Spec.Version,
		Image:            fn.Spec.Image,
		ImageCredentials: fn.Spec.ImageCredentials,
		Timeout:          fn.Spec.Serving.Timeout,
	}

	if fn.Spec.Port != nil {
		port := *fn.Spec.Port
		spec.Port = &port
	}

	if fn.Spec.Serving != nil {
		spec.Params = fn.Spec.Serving.Params
		spec.OpenFuncAsync = fn.Spec.Serving.OpenFuncAsync
		spec.Template = fn.Spec.Serving.Template
	}

	if fn.Spec.Serving != nil && fn.Spec.Serving.Runtime != nil {
		spec.Runtime = fn.Spec.Serving.Runtime
	} else {
		runt := openfunction.Knative
		spec.Runtime = &runt
	}

	return spec
}

func (r *FunctionReconciler) needToCreateBuilder(fn *openfunction.Function) bool {
	log := r.Log.WithName("NeedToCreateBuilder").
		WithValues("Function", fmt.Sprintf("%s/%s", fn.Namespace, fn.Name))

	// Builder had not created, need to create.
	if fn.Status.Build == nil ||
		fn.Status.Build.ResourceHash == "" ||
		(fn.Status.Build.ResourceRef == "" && fn.Spec.Build != nil) {
		log.V(1).Info("Builder not created")
		return true
	}

	newHash := util.Hash(r.createBuilderSpec(fn))
	// Builder changed, need to create.
	if newHash != fn.Status.Build.ResourceHash {
		log.V(1).Info("builder changed", "old", fn.Status.Build.ResourceHash, "new", newHash)
		return true
	}

	// It will skip build, no need to create builder.
	if fn.Spec.Build == nil {
		return false
	}

	var builder openfunction.Builder
	key := client.ObjectKey{Namespace: fn.Namespace, Name: fn.Status.Build.ResourceRef}
	if err := r.Get(r.ctx, key, &builder); util.IsNotFound(err) {
		// If the builder is deleted before the build is completed, the builder needs to be recreated.
		if fn.Status.Build.State != openfunction.Succeeded {
			log.V(1).Info("Builder had been deleted")
			return true
		}
	}

	return false
}

func (r *FunctionReconciler) needToCreateServing(fn *openfunction.Function) bool {

	log := r.Log.WithName("NeedToCreateServing").
		WithValues("Function", fmt.Sprintf("%s/%s", fn.Namespace, fn.Name))

	// The build is still in process, no need to create or update serving.
	if fn.Status.Serving == nil {
		log.V(1).Info("Build not completed")
		return false
	}

	oldHash := fn.Status.Serving.ResourceHash
	oldName := fn.Status.Serving.ResourceRef
	// Serving had not created, need to create.
	if fn.Status.Serving.State == "" || oldHash == "" || (oldName == "" && fn.Spec.Serving != nil) {
		log.V(1).Info("Serving not created")
		return true
	}

	newHash := util.Hash(r.createServingSpec(fn))
	// Serving changed, need to update.
	if newHash != oldHash {
		log.V(1).Info("Serving changed", "old", oldHash, "new", newHash)
		return true
	}

	// It will skip serving, no need to create serving.
	if fn.Spec.Serving == nil {
		return false
	}

	var serving openfunction.Serving
	key := client.ObjectKey{Namespace: fn.Namespace, Name: oldName}
	if err := r.Get(r.ctx, key, &serving); util.IsNotFound(err) {
		// If the serving is deleted, need to be recreated.
		log.V(1).Info("Serving had been deleted")
		return true
	}

	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *FunctionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunction.Function{}).
		Owns(&openfunction.Builder{}).
		Owns(&openfunction.Serving{}).
		Complete(r)
}
