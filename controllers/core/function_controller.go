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
	"strings"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openfunction "github.com/openfunction/apis/core/v1alpha2"
	"github.com/openfunction/pkg/util"
)

const (
	functionLabel = "openfunction.io/function"
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
	log := r.Log.WithValues("function", req.NamespacedName)

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
	log := r.Log.WithName("CreateBuilder")

	if !r.needToCreateBuilder(fn) {
		if err := r.updateFuncWithBuilderStatus(fn); err != nil {
			return err
		}

		log.V(1).Info("No need to create builder", "namespace", fn.Namespace, "name", fn.Name)
		return nil
	}

	// Reset function build status.
	if fn.Status.Build == nil {
		fn.Status.Build = &openfunction.Condition{}
	}
	fn.Status.Build.State = ""
	fn.Status.Build.ResourceRef = ""
	if err := r.Status().Update(r.ctx, fn); err != nil {
		log.Error(err, "Failed to reset function build status", "namespace", fn.Namespace, "name", fn.Name)
		return err
	}

	if err := r.deleteOldBuilder(fn); err != nil {
		log.Error(err, "Failed to clean builder", "name", fn.Name, "namespace", fn.Namespace)
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
			log.Error(err, "Failed to update function build status", "namespace", fn.Namespace, "name", fn.Name)
			return err
		}

		log.V(1).Info("Skip build", "namespace", fn.Namespace, "name", fn.Name)
		return nil
	}

	builder := &openfunction.Builder{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-builder-", fn.Name),
			Namespace:    fn.Namespace,
			Labels: map[string]string{
				functionLabel: fn.Name,
			},
		},
		Spec: r.createBuilderSpec(fn),
	}
	builder.SetOwnerReferences(nil)
	if err := ctrl.SetControllerReference(fn, builder, r.Scheme); err != nil {
		log.Error(err, "Failed to SetOwnerReferences for builder", "namespace", builder.Namespace, "name", builder.Name)
		return err
	}

	if err := r.Create(r.ctx, builder); err != nil {
		log.Error(err, "Failed to create builder", "namespace", builder.Namespace, "name", builder.Name)
		return err
	}

	fn.Status.Build = &openfunction.Condition{
		State:        openfunction.Created,
		ResourceRef:  builder.Name,
		ResourceHash: util.Hash(builder.Spec),
	}
	if err := r.Status().Update(r.ctx, fn); err != nil {
		log.Error(err, "Failed to update function build status", "namespace", fn.Namespace, "name", fn.Name)
		return err
	}

	log.V(1).Info("Builder created", "namespace", builder.Namespace, "name", builder.Name)
	return nil
}

// Update the status of the function with the result of the build.
func (r *FunctionReconciler) updateFuncWithBuilderStatus(fn *openfunction.Function) error {
	log := r.Log.WithName("UpdateFuncWithBuilderStatus")

	// Build had not created or not completed, no need to update function status.
	if fn.Status.Build == nil || fn.Status.Build.State == "" {
		return nil
	}

	var builder openfunction.Builder
	builder.Name = fn.Status.Build.ResourceRef
	builder.Namespace = fn.Namespace

	if builder.Name == "" {
		log.V(1).Info("Function has no builder", "namespace", fn.Namespace, "name", fn.Name)
		return nil
	}

	if err := r.Get(r.ctx, client.ObjectKey{Namespace: builder.Namespace, Name: builder.Name}, &builder); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get builder", "namespace", builder.Namespace, "name", builder.Name)
		return util.IgnoreNotFound(err)
	}

	// The build is still in process.
	if builder.Status.Phase != openfunction.BuildPhase || builder.Status.State == openfunction.Building {
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
		log.Error(err, "Failed to update function status", "namespace", fn.Namespace, "name", fn.Name)
		return err
	}

	if builder.Status.State == openfunction.Succeeded {
		return r.cleanupBuilder(fn)
	}

	return nil
}

func (r *FunctionReconciler) cleanupBuilder(fn *openfunction.Function) error {
	log := r.Log.WithName("CleanupBuilder")

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

	log.V(1).Info("Function builder cleanup", "namespace", builder.Namespace, "name", builder.Name)
	return nil
}

// Delete old builders which created by function.
func (r *FunctionReconciler) deleteOldBuilder(fn *openfunction.Function) error {
	log := r.Log.WithName("DeleteOldBuilder")

	builders := &openfunction.BuilderList{}
	if err := r.List(r.ctx, builders, client.InNamespace(fn.Namespace), client.MatchingLabels{functionLabel: fn.Name}); err != nil {
		return err
	}

	for _, item := range builders.Items {
		if strings.HasPrefix(item.Name, fmt.Sprintf("%s-builder", fn.Name)) {
			if err := r.Delete(context.Background(), &item); util.IgnoreNotFound(err) != nil {
				return err
			}
			log.V(1).Info("Delete Builder", "namespace", item.Namespace, "name", item.Name)
		}
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
	}

	spec.SrcRepo = &openfunction.GitRepo{}
	spec.SrcRepo.Init()
	fn.Spec.Build.SrcRepo.DeepCopyInto(spec.SrcRepo)

	return spec
}

func (r *FunctionReconciler) createServing(fn *openfunction.Function) error {
	log := r.Log.WithName("CreateServing")

	if !r.needToCreateServing(fn) {
		log.V(1).Info("No need to create serving", "namespace", fn.Namespace, "name", fn.Name)

		if running, err := r.isServingRunning(fn); err != nil {
			log.Error(err, "Failed to get serving status", "namespace", fn.Namespace, "name", fn.Name)
			return err
		} else if running {
			if err := r.deleteOldServing(fn); err != nil {
				log.Error(err, "Failed to clean old serving", "namespace", fn.Namespace, "name", fn.Name)
				return err
			}

			if fn.Status.Serving.State != openfunction.Running {
				fn.Status.Serving.State = openfunction.Running
				if err := r.Status().Update(r.ctx, fn); err != nil {
					log.Error(err, "Failed to update function serving status", "namespace", fn.Namespace, "name", fn.Name)
					return err
				}
			}
		}

		return nil
	}

	// Reset function serving status.
	if fn.Status.Serving == nil {
		fn.Status.Serving = &openfunction.Condition{}
	}
	fn.Status.Serving.State = ""
	fn.Status.Serving.ResourceHash = ""
	if err := r.Status().Update(r.ctx, fn); err != nil {
		log.Error(err, "Failed to update function serving status", "namespace", fn.Namespace, "name", fn.Name)
		return err
	}

	if fn.Spec.Serving == nil {
		fn.Status.Serving = &openfunction.Condition{
			State:        openfunction.Skipped,
			ResourceHash: util.Hash(openfunction.ServingSpec{}),
		}
		if err := r.Status().Update(r.ctx, fn); err != nil {
			log.Error(err, "Failed to update function serving status", "namespace", fn.Namespace, "name", fn.Name)
			return err
		}

		log.V(1).Info("Skip serving", "namespace", fn.Namespace, "name", fn.Name)
		return nil
	}

	serving := &openfunction.Serving{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-serving-", fn.Name),
			Namespace:    fn.Namespace,
			Labels: map[string]string{
				functionLabel: fn.Name,
			},
		},
		Spec: r.createServingSpec(fn),
	}
	serving.SetOwnerReferences(nil)
	if err := ctrl.SetControllerReference(fn, serving, r.Scheme); err != nil {
		log.Error(err, "Failed to SetOwnerReferences for serving", "namespace", serving.Namespace, "name", serving.Name)
		return err
	}

	if err := r.Create(r.ctx, serving); err != nil {
		log.Error(err, "Failed to create serving", "namespace", serving.Namespace, "name", serving.Name)
		return err
	}

	fn.Status.Serving = &openfunction.Condition{
		State:        openfunction.Created,
		ResourceRef:  serving.Name,
		ResourceHash: util.Hash(serving.Spec),
	}
	if err := r.Status().Update(r.ctx, fn); err != nil {
		log.Error(err, "Failed to update function serving status", "namespace", fn.Namespace, "name", fn.Name)
		return err
	}

	log.V(1).Info("Function serving created", "namespace", serving.Namespace, "name", serving.Name)
	return nil
}

func (r *FunctionReconciler) isServingRunning(fn *openfunction.Function) (bool, error) {

	if fn.Status.Serving == nil || fn.Status.Serving.State == "" {
		return false, nil
	}

	var serving openfunction.Serving
	serving.Name = fn.Status.Serving.ResourceRef
	serving.Namespace = fn.Namespace

	if serving.Name == "" {
		return false, nil
	}

	if err := r.Get(r.ctx, client.ObjectKey{Namespace: serving.Namespace, Name: serving.Name}, &serving); util.IgnoreNotFound(err) != nil {
		return false, util.IgnoreNotFound(err)
	}

	return serving.Status.State == openfunction.Running, nil
}

// Clean up redundant servings caused by the `createOrUpdateBuilder` function failed.
func (r *FunctionReconciler) deleteOldServing(fn *openfunction.Function) error {
	log := r.Log.WithName("DeleteOldServing")

	name := ""
	if fn.Status.Serving != nil {
		name = fn.Status.Serving.ResourceRef
	}

	servings := &openfunction.ServingList{}
	if err := r.List(r.ctx, servings, client.InNamespace(fn.Namespace), client.MatchingLabels{functionLabel: fn.Name}); err != nil {
		return err
	}

	for _, item := range servings.Items {
		if strings.HasPrefix(item.Name, fmt.Sprintf("%s-serving", fn.Name)) && item.Name != name {
			if err := r.Delete(context.Background(), &item); util.IgnoreNotFound(err) != nil {
				return err
			}
			log.V(1).Info("Delete Serving", "namespace", item.Namespace, "name", item.Name)
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
	log := r.Log.WithName("NeedToCreateBuilder")

	// Builder had not created, need to create.
	if fn.Status.Build == nil ||
		fn.Status.Build.ResourceHash == "" ||
		(fn.Status.Build.ResourceRef == "" && fn.Spec.Build != nil) {
		log.V(1).Info("Builder not created", "namespace", fn.Namespace, "name", fn.Name)
		return true
	}

	newHash := util.Hash(r.createBuilderSpec(fn))
	// Builder changed, need to create.
	if newHash != fn.Status.Build.ResourceHash {
		log.V(1).Info("builder changed", "namespace", fn.Namespace, "name", fn.Name, "old", fn.Status.Build.ResourceHash, "new", newHash)
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
			log.V(1).Info("Builder had been deleted", "namespace", fn.Namespace, "name", fn.Name)
			return true
		}
	}

	return false
}

func (r *FunctionReconciler) needToCreateServing(fn *openfunction.Function) bool {

	log := r.Log.WithName("NeedToCreateServing")

	// The build is still in process, no need to create or update serving.
	if fn.Status.Serving == nil {
		log.V(1).Info("Build not completed", "namespace", fn.Namespace, "name", fn.Name)
		return false
	}

	oldHash := fn.Status.Serving.ResourceHash
	oldName := fn.Status.Serving.ResourceRef
	// Serving had not created, need to create.
	if fn.Status.Serving.State == "" || oldHash == "" || (oldName == "" && fn.Spec.Serving != nil) {
		log.V(1).Info("Serving not created", "namespace", fn.Namespace, "name", fn.Name)
		return true
	}

	newHash := util.Hash(r.createServingSpec(fn))
	// Serving changed, need to update.
	if newHash != oldHash {
		log.V(1).Info("Serving changed", "namespace", fn.Namespace, "name", fn.Name, "old", oldHash, "new", newHash)
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
		log.V(1).Info("Serving had been deleted", "namespace", fn.Namespace, "name", fn.Name)
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
