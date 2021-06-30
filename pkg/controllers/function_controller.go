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

	openfunctioncontext "github.com/OpenFunction/functions-framework-go/openfunction-context"
	"github.com/go-logr/logr"
	jsoniter "github.com/json-iterator/go"
	openfunction "github.com/openfunction/pkg/apis/v1alpha1"
	"github.com/openfunction/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	kcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	builderHash = "builder-hash"
	servingHash = "serving-hash"
)

// FunctionReconciler reconciles a Function object
type FunctionReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	ctx         context.Context
	tektonCache kcache.Cache
}

// +kubebuilder:rbac:groups=core.openfunction.io,resources=functions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.openfunction.io,resources=functions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tekton.dev,resources=tasks;pipelineresources;pipelines;pipelineruns,verbs=get;list;watch;create;update;patch;delete

func (r *FunctionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	r.ctx = ctx
	log := r.Log.WithValues("function", req.NamespacedName)

	var fn openfunction.Function

	if err := r.Get(ctx, req.NamespacedName, &fn); err != nil {

		if util.IsNotFound(err) {
			log.V(1).Info("Function deleted", "name", req.Name, "namespace", req.Namespace)
		}

		return ctrl.Result{}, util.IgnoreNotFound(err)
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

	if fn.Spec.Build == nil {
		log.V(1).Info("skip build", "namespace", fn.Namespace, "name", fn.Name)
		return ctrl.Result{}, nil
	}

	var builder openfunction.Builder
	builder.Name = fmt.Sprintf("%s-%s", fn.Name, "builder")
	builder.Namespace = fn.Namespace

	if !r.needToCreateOrUpdateBuilder(fn) {
		log.V(1).Info("No need to update builder", "namespace", builder.Namespace, "name", builder.Name)
		return ctrl.Result{}, nil
	}

	if err := r.Delete(r.ctx, &builder); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to delete builder", "namespace", builder.Namespace, "name", builder.Name)
		return ctrl.Result{}, err
	}

	status := openfunction.FunctionStatus{Phase: "", State: ""}
	if err := r.updateStatus(fn, &status); err != nil {
		log.Error(err, "Failed to update function build status", "namespace", fn.Namespace, "name", fn.Name)
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

	status = openfunction.FunctionStatus{Phase: openfunction.BuildPhase, State: openfunction.Created}
	if err := r.updateStatus(fn, &status); err != nil {
		log.Error(err, "Failed to update function build status", "namespace", fn.Namespace, "name", fn.Name)
		return ctrl.Result{}, err
	}

	if err := r.updateHash(fn, builderHash, r.createBuilderSpec(fn)); err != nil {
		log.Error(err, "Failed to update function build hash", "namespace", fn.Namespace, "name", fn.Name)
		return ctrl.Result{}, err
	}

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

	if err := r.Delete(r.ctx, &serving); util.IgnoreNotFound(err) != nil {
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

	if err := r.updateHash(fn, servingHash, r.createServingSpec(fn)); err != nil {
		log.Error(err, "Failed to update function serving hash", "namespace", fn.Namespace, "name", fn.Name)
		return ctrl.Result{}, err
	}

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
		if spec.Params == nil {
			spec.Params = make(map[string]string)
		}
		spec.Params["FUNC_CONTEXT"] = r.createFunctionContext(fn)
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

func (r *FunctionReconciler) createFunctionContext(fn *openfunction.Function) string {

	rt := openfunctioncontext.Knative
	if fn.Spec.Serving != nil && fn.Spec.Serving.Runtime != nil {
		rt = openfunctioncontext.Runtime(*fn.Spec.Serving.Runtime)
	}

	var port int32 = 8080
	if fn.Spec.Port != nil {
		port = *fn.Spec.Port
	}

	version := ""
	if fn.Spec.Version != nil {
		version = *fn.Spec.Version
	}

	fc := openfunctioncontext.OpenFunctionContext{
		Name:    fn.Name,
		Version: version,
		Runtime: rt,
		Port:    fmt.Sprintf("%d", port),
	}

	if fn.Spec.Serving != nil && fn.Spec.Serving.OpenFuncAsync != nil && fn.Spec.Serving.OpenFuncAsync.Dapr != nil {
		dapr := fn.Spec.Serving.OpenFuncAsync.Dapr

		if dapr.Inputs != nil && len(dapr.Inputs) > 0 {
			input := dapr.Inputs[0]
			fc.Input = openfunctioncontext.Input{
				Name:   input.Name,
				Uri:    getUri(input),
				Params: input.Params,
			}

			if fc.Input.Params == nil {
				fc.Input.Params = map[string]string{
					"type": input.Type,
				}
			}
		}

		if dapr.Outputs != nil && len(dapr.Outputs) > 0 {
			fc.Outputs = make(map[string]*openfunctioncontext.Output)

			for _, o := range dapr.Outputs {
				output := openfunctioncontext.Output{
					Uri:    getUri(o),
					Params: o.Params,
				}

				if output.Params == nil {
					output.Params = map[string]string{
						"type": o.Type,
					}
				}

				fc.Outputs[o.Name] = &output
			}
		}
	}

	bs, _ := jsoniter.Marshal(fc)
	return string(bs)
}

func getUri(io *openfunction.DaprIO) string {
	switch io.Type {
	case string(openfunctioncontext.OpenFuncBinding):
		return io.Name
	case string(openfunctioncontext.OpenFuncTopic):
		return io.Topic
	case string(openfunctioncontext.OpenFuncService):
		return io.MethodName
	default:
		return ""
	}
}

func (r *FunctionReconciler) cleanupBuilder(fn *openfunction.Function) (ctrl.Result, error) {
	log := r.Log.WithName("cleanupBuilder")
	var builder openfunction.Builder
	builder.Name = fmt.Sprintf("%s-%s", fn.Name, "builder")
	builder.Namespace = fn.Namespace

	if err := r.Delete(r.ctx, &builder); err != nil {
		if util.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
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

func (r *FunctionReconciler) updateHash(fn *openfunction.Function, key string, val interface{}) error {

	if fn.Annotations == nil {
		fn.Annotations = make(map[string]string)
	}

	fn.Annotations[key] = util.Hash(val)

	if err := r.Update(r.ctx, fn); err != nil {
		return err
	}

	return nil
}

func (r *FunctionReconciler) getHash(fn *openfunction.Function, key string) string {

	if fn.Annotations == nil {
		return ""
	}

	return fn.Annotations[key]
}

func (r *FunctionReconciler) needToCreateOrUpdateBuilder(fn *openfunction.Function) bool {

	log := r.Log.WithName("needToCreateOrUpdateBuilder")

	oldHash := r.getHash(fn, builderHash)
	// Builder had not created, need to create.
	if oldHash == "" {
		log.V(1).Info("builder hash is nil", "namespace", fn.Namespace, "name", fn.Name)
		return true
	}

	needToCreateOrUpdate := false

	newHash := util.Hash(r.createBuilderSpec(fn))
	// Builder changed, need to update.
	if newHash != oldHash {
		log.V(1).Info("builder changed", "namespace", fn.Namespace, "name", fn.Name, "old", oldHash, "new", newHash)
		needToCreateOrUpdate = true
	}

	var builder openfunction.Builder
	key := client.ObjectKey{Namespace: fn.Namespace, Name: fmt.Sprintf("%s-%s", fn.Name, "builder")}
	if err := r.Get(r.ctx, key, &builder); util.IsNotFound(err) {
		// If the builder is deleted before the build is completed, the builder needs to be recreated.
		if fn.Status.Phase != openfunction.ServingPhase {
			needToCreateOrUpdate = true
			log.V(1).Info("builder does not exist", "namespace", fn.Namespace, "name", fn.Name)
		}
	}

	return needToCreateOrUpdate
}

func (r *FunctionReconciler) needToCreateOrUpdateServing(fn *openfunction.Function) bool {

	log := r.Log.WithName("needToCreateOrUpdateServing")

	// The build is not completed, no need to create or update serving.
	if fn.Status.Phase != openfunction.ServingPhase {
		log.V(1).Info("build not completed", "namespace", fn.Namespace, "name", fn.Name)
		return false
	}

	oldHash := r.getHash(fn, servingHash)
	// Serving had not created, need to create.
	if oldHash == "" {
		log.V(1).Info("Serving not create", "namespace", fn.Namespace, "name", fn.Name)
		return true
	}

	// Build had completed, need to create serving.
	if fn.Status.State == "" {
		return true
	}

	needToCreateOrUpdate := false

	newHash := util.Hash(r.createServingSpec(fn))
	// Serving changed, need to update.
	if newHash != oldHash {
		needToCreateOrUpdate = true
		log.V(1).Info("serving changed", "namespace", fn.Namespace, "name", fn.Name, "old", oldHash, "new", newHash)
	}

	var serving openfunction.Serving
	key := client.ObjectKey{Namespace: fn.Namespace, Name: fmt.Sprintf("%s-%s", fn.Name, "serving")}
	if err := r.Get(r.ctx, key, &serving); util.IsNotFound(err) {
		// If the serving is deleted, need to be recreated.
		needToCreateOrUpdate = true
		log.V(1).Info("serving does not exist", "namespace", fn.Namespace, "name", fn.Name)
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
