/*
Copyright 2022 The OpenFunction Authors.

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
	"bytes"
	"context"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/events"
	kservingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlbuilder "sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	k8sgatewayapiv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	openfunction "github.com/openfunction/apis/core/v1beta2"
	networkingv1alpha1 "github.com/openfunction/apis/networking/v1alpha1"
	"github.com/openfunction/pkg/constants"
	"github.com/openfunction/pkg/core/serving/common"
	"github.com/openfunction/pkg/util"
)

const (
	GatewayField = ".spec.route.gatewayRef"

	buildAction   = "Build"
	servingAction = "Serving"
)

// FunctionReconciler reconciles a Function object
type FunctionReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	ctx      context.Context
	interval time.Duration

	eventRecorder events.EventRecorder
}

func NewFunctionReconciler(mgr manager.Manager, interval time.Duration, eventRecorder events.EventRecorder) *FunctionReconciler {

	r := &FunctionReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		Log:           ctrl.Log.WithName("controllers").WithName("Function"),
		interval:      interval,
		eventRecorder: eventRecorder,
	}

	r.startFunctionWatcher()

	return r
}

//+kubebuilder:rbac:groups=core.openfunction.io,resources=functions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.openfunction.io,resources=functions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=list;get;watch;update;patch
//+kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch
//+kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.openfunction.io,resources=gateways,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=get;list;watch;create;update;patch;delete

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

	fn := openfunction.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
	}

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

	if err := r.createOrUpdateHTTPRoute(&fn); err != nil {
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
	fn.Status.Build.Reason = ""
	fn.Status.Build.Message = ""
	fn.Status.Build.BuildDuration = nil
	fn.Status.Build.ResourceRef = ""
	if err := r.Status().Update(r.ctx, fn); err != nil {
		log.Error(err, "Failed to reset function build status")
		return err
	}

	if waitBuilderCancel, err := r.cancelOldBuilder(fn); err != nil {
		log.Error(err, "Failed to cancel builder")
		return err
	} else if waitBuilderCancel {
		return nil
	}

	if err := r.pruneBuilder(fn); err != nil {
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
		ResourceHash: getBuilderHash(builder.Spec),
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
	if fn.Status.Build.State != builder.Status.State ||
		fn.Status.Build.Reason != builder.Status.Reason ||
		fn.Status.Build.Message != builder.Status.Message {
		fn.Status.Build.State = builder.Status.State
		fn.Status.Build.Reason = builder.Status.Reason
		fn.Status.Build.Message = builder.Status.Message
		fn.Status.Build.BuildDuration = builder.Status.BuildDuration
		// If build had complete, update function serving status.
		if builder.Status.State == openfunction.Succeeded {
			if builder.Status.Output != nil {
				fn.Status.Revision = &openfunction.Revision{
					ImageDigest: builder.Status.Output.Digest,
				}
				fn.Status.Sources = builder.Status.Sources
			}
			if fn.Status.Serving == nil {
				fn.Status.Serving = &openfunction.Condition{}
			}

			fn.Status.Serving.State = ""
			fn.Status.Serving.Reason = ""
			fn.Status.Serving.Message = ""
		}

		if err := r.Status().Update(r.ctx, fn); err != nil {
			log.Error(err, "Failed to update function status")
			return err
		}

		r.recordEvent(fn, &builder, buildAction, fn.Status.Build.State, fn.Status.Build.Message)
	}

	return r.pruneBuilder(fn)
}

// Only one builder can run at the same time, cancel old builders.
func (r *FunctionReconciler) cancelOldBuilder(fn *openfunction.Function) (bool, error) {
	log := r.Log.WithName("CancelOldBuilder").
		WithValues("Function", fmt.Sprintf("%s/%s", fn.Namespace, fn.Name))

	builders := &openfunction.BuilderList{}
	if err := r.List(r.ctx, builders, client.InNamespace(fn.Namespace), client.MatchingLabels{constants.FunctionLabel: fn.Name}); err != nil {
		return false, err
	}

	waitBuilderCancel := false
	for _, item := range builders.Items {
		builder := item
		if !builder.Status.IsCompleted() {
			log.V(1).Info("Builder is still running, cancel it", "builder", builder.Name)

			builder.Spec.State = openfunction.BuilderStateCancelled
			if err := r.Update(r.ctx, &builder); err != nil {
				return false, err
			}

			waitBuilderCancel = true
		}
	}

	return waitBuilderCancel, nil
}

// Delete old builders which created by function.
func (r *FunctionReconciler) pruneBuilder(fn *openfunction.Function) error {
	log := r.Log.WithName("PruneBuilder").
		WithValues("Function", fmt.Sprintf("%s/%s", fn.Namespace, fn.Name))

	builders := &openfunction.BuilderList{}
	if err := r.List(r.ctx, builders, client.InNamespace(fn.Namespace), client.MatchingLabels{constants.FunctionLabel: fn.Name}); err != nil {
		return err
	}

	sort.SliceStable(builders.Items, func(i, j int) bool {
		return builders.Items[i].CreationTimestamp.Time.After(builders.Items[j].CreationTimestamp.Time)
	})

	var failedBuildsHistoryLimit int32 = 1
	if fn.Spec.Build != nil && fn.Spec.Build.FailedBuildsHistoryLimit != nil {
		failedBuildsHistoryLimit = *fn.Spec.Build.FailedBuildsHistoryLimit
	}

	var successfulBuildsHistoryLimit int32 = 0
	if fn.Spec.Build != nil && fn.Spec.Build.SuccessfulBuildsHistoryLimit != nil {
		successfulBuildsHistoryLimit = *fn.Spec.Build.SuccessfulBuildsHistoryLimit
	}

	var failedBuildsHistoryNum int32 = 0
	var successfulBuildsHistoryNum int32 = 0
	for _, item := range builders.Items {
		builder := item
		if !builder.Status.IsCompleted() {
			continue
		}

		if builder.Status.IsSucceeded() {
			successfulBuildsHistoryNum++
			if successfulBuildsHistoryNum <= successfulBuildsHistoryLimit {
				continue
			}
		} else {
			failedBuildsHistoryNum++
			if failedBuildsHistoryNum <= failedBuildsHistoryLimit {
				continue
			}
		}

		if err := r.Delete(context.Background(), &builder); util.IgnoreNotFound(err) != nil {
			return err
		}
		log.V(1).Info("Delete Builder", "builder", builder.Name)
	}

	return nil
}

func (r *FunctionReconciler) createBuilderSpec(fn *openfunction.Function) openfunction.BuilderSpec {
	if fn.Spec.Build == nil {
		return openfunction.BuilderSpec{}
	}

	spec := openfunction.BuilderSpec{
		BuildImpl:        *fn.Spec.Build.DeepCopy(),
		Image:            fn.Spec.Image,
		ImageCredentials: fn.Spec.ImageCredentials,
	}

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
	fn.Status.Serving.Reason = ""
	fn.Status.Serving.Message = ""
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
			Annotations: fn.Annotations,
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

	if common.NeedCreateDaprProxy(serving) {
		if err := r.CreateOrUpdateServiceForAsyncFunc(fn, serving); err != nil {
			return err
		}
	}

	fn.Status.Serving = &openfunction.Condition{
		State:                     openfunction.Created,
		ResourceRef:               serving.Name,
		ResourceHash:              util.Hash(serving.Spec),
		LastSuccessfulResourceRef: fn.Status.Serving.LastSuccessfulResourceRef,
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
	if fn.Status.Serving.State != serving.Status.State ||
		fn.Status.Serving.Reason != serving.Status.Reason ||
		fn.Status.Serving.Message != serving.Status.Message {
		fn.Status.Serving.State = serving.Status.State
		fn.Status.Serving.Reason = serving.Status.Reason
		fn.Status.Serving.Message = serving.Status.Message

		// If new serving is running, clean old serving.
		if serving.Status.State == openfunction.Running {
			fn.Status.Serving.LastSuccessfulResourceRef = fn.Status.Serving.ResourceRef
			fn.Status.Serving.Service = serving.Status.Service
			if err := r.cleanServing(fn); err != nil {
				log.Error(err, "Failed to clean Serving")
				return err
			}
			log.V(1).Info("Serving is running", "serving", serving.Name)
		}

		if err := r.Status().Update(r.ctx, fn); err != nil {
			log.Error(err, "Failed to update function status")
			return err
		}

		r.recordEvent(fn, &serving, servingAction, fn.Status.Serving.State, fn.Status.Serving.Message)
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
		oldName = fn.Status.Serving.LastSuccessfulResourceRef
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
		Image:            getServingImage(fn),
		ImageCredentials: fn.Spec.ImageCredentials,
		ServingImpl:      *fn.Spec.Serving.DeepCopy(),
	}

	return spec
}

func getServingImage(fn *openfunction.Function) string {
	if fn.Status.Revision == nil ||
		fn.Status.Revision.ImageDigest == "" {
		return fn.Spec.Image
	}

	array := strings.Split(fn.Spec.Image, "@")
	repo := fn.Spec.Image
	if len(array) > 1 {
		repo = array[0]
	}

	return repo + "@" + fn.Status.Revision.ImageDigest
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

	newHash := getBuilderHash(r.createBuilderSpec(fn))
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

func getBuilderHash(spec openfunction.BuilderSpec) string {
	newSpec := spec.DeepCopy()
	newSpec.SuccessfulBuildsHistoryLimit = nil
	newSpec.FailedBuildsHistoryLimit = nil
	newSpec.BuilderMaxAge = nil
	newSpec.Timeout = nil
	newSpec.State = ""

	return util.Hash(newSpec)
}

func (r *FunctionReconciler) needToCreateServing(fn *openfunction.Function) bool {

	log := r.Log.WithName("NeedToCreateServing").
		WithValues("Function", fmt.Sprintf("%s/%s", fn.Namespace, fn.Name))

	// The build is still in process, no need to create or update serving.
	if fn.Status.Serving == nil ||
		(fn.Status.Build != nil &&
			fn.Status.Build.State != openfunction.Succeeded &&
			fn.Status.Build.State != openfunction.Skipped) {
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

func (r *FunctionReconciler) createOrUpdateHTTPRoute(fn *openfunction.Function) error {
	log := r.Log.WithName("createOrUpdateHTTPRoute")

	if fn.Status.Serving == nil ||
		fn.Status.Serving.State != openfunction.Running ||
		fn.Status.Serving.Service == "" {
		return nil
	}

	if fn.Spec.Serving.Triggers == nil || fn.Spec.Serving.Triggers.Http == nil {
		return nil
	}

	namespace := constants.DefaultGatewayNamespace
	if fn.Spec.Serving.Triggers.Http.Route == nil {
		route := openfunction.RouteImpl{
			CommonRouteSpec: openfunction.CommonRouteSpec{
				GatewayRef: &openfunction.GatewayRef{
					Name:      constants.DefaultGatewayName,
					Namespace: &namespace,
				},
			},
		}
		fn.Spec.Serving.Triggers.Http.Route = &route
	} else if fn.Spec.Serving.Triggers.Http.Route.GatewayRef == nil {
		fn.Spec.Serving.Triggers.Http.Route.GatewayRef = &openfunction.GatewayRef{Name: constants.DefaultGatewayName, Namespace: &namespace}
	}

	route := fn.Spec.Serving.Triggers.Http.Route
	gateway := &networkingv1alpha1.Gateway{}
	key := client.ObjectKey{
		Namespace: string(*route.GatewayRef.Namespace),
		Name:      string(route.GatewayRef.Name),
	}
	if err := r.Get(r.ctx, key, gateway); err != nil {
		log.Error(err, "Failed to get gateway",
			"namespace", route.GatewayRef.Namespace, "name", route.GatewayRef.Name)
		return err
	}

	if fn.Spec.Serving.Triggers.Http.Engine == nil || *fn.Spec.Serving.Triggers.Http.Engine == "" ||
		*fn.Spec.Serving.Triggers.Http.Engine == openfunction.HttpEngineKnative {
		knativeService := &kservingv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fn.Status.Serving.Service,
				Namespace: fn.Namespace,
			},
		}
		if err := r.Get(r.ctx, client.ObjectKeyFromObject(knativeService), knativeService); err != nil {
			log.Error(err, "Failed to get knative service",
				"namespace", fn.Namespace, "name", fn.Status.Serving.Service)
			return err
		}

		httpRoute := &k8sgatewayapiv1alpha2.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{Namespace: fn.Namespace, Name: fn.Name},
		}
		op, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, httpRoute, r.mutateHTTPRoute(fn, knativeService, nil, gateway, httpRoute))
		if err != nil {
			log.Error(err, "Failed to CreateOrUpdate HTTPRoute")
			return err
		}
		log.V(1).Info(fmt.Sprintf("HTTPRoute %s", op))

		if err := r.updateFuncWithHTTPRouteStatus(fn, gateway, httpRoute); err != nil {
			return err
		}
	} else if *fn.Spec.Serving.Triggers.Http.Engine == openfunction.HttpEngineKeda {
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fn.Status.Serving.Service,
				Namespace: fn.Namespace,
			},
		}
		if err := r.Get(r.ctx, client.ObjectKeyFromObject(service), service); err != nil {
			log.Error(err, "Failed to get keda service",
				"namespace", fn.Namespace, "name", fn.Status.Serving.Service)
			return err
		}

		httpRoute := &k8sgatewayapiv1alpha2.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{Namespace: fn.Namespace, Name: fn.Name},
		}

		op, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, httpRoute, r.mutateHTTPRoute(fn, nil, service, gateway, httpRoute))
		if err != nil {
			log.Error(err, "Failed to CreateOrUpdate HTTPRoute")
			return err
		}
		log.V(1).Info(fmt.Sprintf("HTTPRoute %s", op))

		if err := r.updateFuncWithHTTPRouteStatus(fn, gateway, httpRoute); err != nil {
			return err
		}
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: fn.Namespace, Name: fn.Name},
	}
	op, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, service, r.mutateService(fn, gateway, service))
	if err != nil {
		log.Error(err, "Failed to CreateOrUpdate service")
		return err
	}
	log.V(1).Info(fmt.Sprintf("service %s", op))

	return nil
}

func (r *FunctionReconciler) mutateHTTPRoute(
	fn *openfunction.Function,
	knativeService *kservingv1.Service,
	service *corev1.Service,
	gateway *networkingv1alpha1.Gateway,
	httpRoute *k8sgatewayapiv1alpha2.HTTPRoute) controllerutil.MutateFn {
	return func() error {
		var clusterHostname = k8sgatewayapiv1alpha2.Hostname(
			fmt.Sprintf("%s.%s.svc.%s", fn.Name, fn.Namespace, gateway.Spec.ClusterDomain))
		var hostnames []k8sgatewayapiv1alpha2.Hostname
		var rules []k8sgatewayapiv1alpha2.HTTPRouteRule
		var port = constants.DefaultFunctionServicePort
		if service != nil {
			port = constants.DefaultInterceptorPort
		}
		var namespace = k8sgatewayapiv1alpha2.Namespace(fn.Namespace)
		var httpHeaders []k8sgatewayapiv1alpha2.HTTPHeader
		if knativeService != nil {
			httpHeaders = []k8sgatewayapiv1alpha2.HTTPHeader{{
				Name:  "Host",
				Value: fmt.Sprintf("%s.%s.svc.%s", knativeService.Status.LatestReadyRevisionName, fn.Namespace, gateway.Spec.ClusterDomain),
			}}
		} else if service != nil {
			for _, hostname := range fn.Spec.Serving.Triggers.Http.Route.Hostnames {
				httpHeaders = append(httpHeaders, k8sgatewayapiv1alpha2.HTTPHeader{
					Name:  "Host",
					Value: string(hostname),
				})
			}
		}
		var filter = k8sgatewayapiv1alpha2.HTTPRouteFilter{
			Type: k8sgatewayapiv1alpha2.HTTPRouteFilterRequestHeaderModifier,
			RequestHeaderModifier: &k8sgatewayapiv1alpha2.HTTPRequestHeaderFilter{
				Add: httpHeaders,
			},
		}
		var parentRefName k8sgatewayapiv1alpha2.ObjectName
		var parentRefNamespace k8sgatewayapiv1alpha2.Namespace
		if gateway.Spec.GatewayRef != nil {
			parentRefName = k8sgatewayapiv1alpha2.ObjectName(gateway.Spec.GatewayRef.Name)
			parentRefNamespace = k8sgatewayapiv1alpha2.Namespace(gateway.Spec.GatewayRef.Namespace)
		}
		if gateway.Spec.GatewayDef != nil {
			parentRefName = k8sgatewayapiv1alpha2.ObjectName(gateway.Spec.GatewayDef.Name)
			parentRefNamespace = k8sgatewayapiv1alpha2.Namespace(gateway.Spec.GatewayDef.Namespace)
		}

		if fn.Spec.Serving.Triggers.Http.Route.Hostnames == nil {
			var hostnameBuffer bytes.Buffer

			hostTemplate := template.Must(template.New("host").Parse(gateway.Spec.HostTemplate))
			hostInfoObj := struct {
				Name      string
				Namespace string
				Domain    string
			}{Name: fn.Name, Namespace: fn.Namespace, Domain: gateway.Spec.Domain}
			if err := hostTemplate.Execute(&hostnameBuffer, hostInfoObj); err != nil {
				return err
			}
			hostname := k8sgatewayapiv1alpha2.Hostname(hostnameBuffer.String())
			hostnames = append(hostnames, hostname)
		} else {
			hostnames = fn.Spec.Serving.Triggers.Http.Route.Hostnames
		}
		if !containsHTTPHostname(fn.Spec.Serving.Triggers.Http.Route.Hostnames, clusterHostname) {
			hostnames = append(hostnames, clusterHostname)
		}

		var backendGroup k8sgatewayapiv1alpha2.Group = ""
		var backendKind k8sgatewayapiv1alpha2.Kind = "Service"
		var backendWeight int32 = 1
		var backendRefName k8sgatewayapiv1alpha2.ObjectName
		if knativeService != nil {
			backendRefName = k8sgatewayapiv1alpha2.ObjectName(knativeService.Status.LatestReadyRevisionName)
		} else if service != nil {
			backendRefName = constants.DefaultKedaInterceptorProxyName
		}
		if fn.Spec.Serving.Triggers.Http.Route.Rules == nil {
			var path string
			if fn.Spec.Serving.Triggers.Http.Route.Hostnames == nil {
				path = "/"
			} else {
				var pathBuffer bytes.Buffer
				pathTemplate := template.Must(template.New("path").Parse(gateway.Spec.PathTemplate))
				pathInfoObj := struct {
					Name      string
					Namespace string
				}{Name: fn.Name, Namespace: fn.Namespace}
				if err := pathTemplate.Execute(&pathBuffer, pathInfoObj); err != nil {
					return err
				}
				path = pathBuffer.String()
				if !strings.HasPrefix(path, "/") {
					path = fmt.Sprintf("/%s", path)
				}
			}
			matchType := k8sgatewayapiv1alpha2.PathMatchPathPrefix
			rule := k8sgatewayapiv1alpha2.HTTPRouteRule{
				Matches: []k8sgatewayapiv1alpha2.HTTPRouteMatch{{
					Path: &k8sgatewayapiv1alpha2.HTTPPathMatch{Type: &matchType, Value: &path},
				}},
				BackendRefs: []k8sgatewayapiv1alpha2.HTTPBackendRef{
					{
						BackendRef: k8sgatewayapiv1alpha2.BackendRef{
							BackendObjectReference: k8sgatewayapiv1alpha2.BackendObjectReference{
								Group:     &backendGroup,
								Kind:      &backendKind,
								Name:      backendRefName,
								Namespace: &namespace,
								Port:      &port,
							},
							Weight: &backendWeight,
						},
					},
				},
				Filters: []k8sgatewayapiv1alpha2.HTTPRouteFilter{filter},
			}
			rules = append(rules, rule)
		} else {
			for _, rule := range fn.Spec.Serving.Triggers.Http.Route.Rules {
				rule.BackendRefs = []k8sgatewayapiv1alpha2.HTTPBackendRef{{
					BackendRef: k8sgatewayapiv1alpha2.BackendRef{
						BackendObjectReference: k8sgatewayapiv1alpha2.BackendObjectReference{
							Group:     &backendGroup,
							Kind:      &backendKind,
							Name:      backendRefName,
							Namespace: &namespace,
							Port:      &port,
						},
						Weight: &backendWeight,
					}}}
				rule.Filters = append(rule.Filters, filter)
				rules = append(rules, rule)
			}
		}
		httpRouteLabelValue := fmt.Sprintf("%s.%s", gateway.Namespace, gateway.Name)
		if httpRoute.Labels == nil {
			httpRoute.Labels = map[string]string{gateway.Spec.HttpRouteLabelKey: httpRouteLabelValue}
		} else {
			httpRoute.Labels[gateway.Spec.HttpRouteLabelKey] = httpRouteLabelValue
		}
		var parentGroup k8sgatewayapiv1alpha2.Group = "gateway.networking.k8s.io"
		var parentKind k8sgatewayapiv1alpha2.Kind = "Gateway"
		httpRoute.Spec.ParentRefs = []k8sgatewayapiv1alpha2.ParentRef{
			{
				Group:     &parentGroup,
				Kind:      &parentKind,
				Namespace: &parentRefNamespace,
				Name:      parentRefName,
			},
		}
		httpRoute.Spec.Hostnames = hostnames
		httpRoute.Spec.Rules = rules
		return ctrl.SetControllerReference(fn, httpRoute, r.Scheme)
	}
}

func (r *FunctionReconciler) mutateService(
	fn *openfunction.Function,
	gateway *networkingv1alpha1.Gateway,
	service *corev1.Service) controllerutil.MutateFn {
	return func() error {
		var servicePorts []corev1.ServicePort
		var externalName = fmt.Sprintf("%s.%s.svc.%s",
			networkingv1alpha1.DefaultGatewayServiceName,
			gateway.Namespace,
			gateway.Spec.ClusterDomain)
		for _, listener := range gateway.Spec.GatewaySpec.Listeners {
			if strings.HasSuffix(string(*listener.Hostname), gateway.Spec.ClusterDomain) {
				servicePort := corev1.ServicePort{
					Name:       string(listener.Name),
					Protocol:   corev1.ProtocolTCP,
					Port:       int32(listener.Port),
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(listener.Port)},
				}
				servicePorts = append(servicePorts, servicePort)
			}
		}
		service.Spec.Type = corev1.ServiceTypeExternalName
		service.Spec.Ports = servicePorts
		service.Spec.ExternalName = externalName
		return ctrl.SetControllerReference(fn, service, r.Scheme)
	}
}

func (r *FunctionReconciler) CreateOrUpdateServiceForAsyncFunc(fn *openfunction.Function, s *openfunction.Serving) error {
	log := r.Log.WithName("CreateOrUpdateServiceForAsyncFunc")

	if fn.Spec.Serving.Triggers.Dapr == nil {
		return nil
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: fn.Namespace, Name: fn.Name},
	}
	op, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, service, func() error {
		var port = int32(constants.DefaultFuncPort)
		funcPort := corev1.ServicePort{
			Name:       "http",
			Protocol:   corev1.ProtocolTCP,
			Port:       port,
			TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: port},
		}
		if fn.Spec.Serving.Annotations[common.DaprAppProtocol] != "http" {
			service.Spec.ClusterIP = corev1.ClusterIPNone
		}
		selector := map[string]string{common.ServingLabel: s.Name}
		service.Spec.Ports = []corev1.ServicePort{funcPort}
		service.Spec.Selector = selector
		return ctrl.SetControllerReference(fn, service, r.Scheme)
	})
	if err != nil {
		log.Error(err, "Failed to CreateOrUpdate service")
		return err
	}

	log.V(1).Info(fmt.Sprintf("Service %s", op))
	return nil
}

func (r *FunctionReconciler) updateFuncWithHTTPRouteStatus(
	fn *openfunction.Function,
	gateway *networkingv1alpha1.Gateway,
	httpRoute *k8sgatewayapiv1alpha2.HTTPRoute) error {
	log := r.Log.WithName("updateFuncWithHTTPRouteStatus")
	var addresses []openfunction.FunctionAddress
	var paths []k8sgatewayapiv1alpha2.HTTPPathMatch
	var oldRouteStatus = fn.Status.Route.DeepCopy()
	if fn.Status.Route == nil {
		fn.Status.Route = &openfunction.RouteStatus{}
	}
	if len(httpRoute.Status.RouteStatus.Parents) != 0 {
		fn.Status.Route.Conditions = httpRoute.Status.Parents[0].Conditions
	}
	// Set a fixed value to prevent the Status of the Function from being updated frequently when the traffic is heavy.
	for index := 0; index < len(fn.Status.Route.Conditions); index++ {
		fn.Status.Route.Conditions[index].LastTransitionTime = fn.CreationTimestamp
	}
	fn.Status.Route.Hosts = httpRoute.Spec.Hostnames
	for _, httpRule := range httpRoute.Spec.Rules {
		for _, match := range httpRule.Matches {
			paths = append(paths, *match.Path)
		}
	}
	fn.Status.Route.Paths = paths
	for _, hostname := range httpRoute.Spec.Hostnames {
		var addressType openfunction.AddressType
		if strings.HasSuffix(string(hostname), gateway.Spec.ClusterDomain) {
			addressType = openfunction.InternalAddressType
		} else {
			addressType = openfunction.ExternalAddressType
		}
		for _, path := range paths {
			addressValue := url.URL{
				Scheme: "http",
				Host:   string(hostname),
				Path:   *path.Value,
			}
			address := openfunction.FunctionAddress{
				Type:  &addressType,
				Value: addressValue.String(),
			}
			addresses = append(addresses, address)
		}
	}
	fn.Status.Addresses = addresses
	if !equality.Semantic.DeepEqual(oldRouteStatus, fn.Status.Route.DeepCopy()) {
		if err := r.Status().Update(r.ctx, fn); err != nil {
			log.Error(err, "Failed to update status on function", "namespace", fn.Namespace, "name", fn.Name)
			return err
		} else {
			log.Info("Updated status on function", "namespace", fn.Namespace,
				"name", fn.Name, "resource version", fn.ResourceVersion)
		}
	}
	return nil
}

func containsHTTPHostname(hostnames []k8sgatewayapiv1alpha2.Hostname, hostname k8sgatewayapiv1alpha2.Hostname) bool {
	for _, item := range hostnames {
		if item == hostname {
			return true
		}
	}
	return false
}

func (r *FunctionReconciler) startFunctionWatcher() {

	ticker := time.NewTicker(r.interval)

	go func() {
		for {
			select {
			case <-ticker.C:
				r.cleanExpiredBuilder()
			}
		}
	}()
}

func (r *FunctionReconciler) cleanExpiredBuilder() {

	log := ctrl.Log.WithName("FunctionWatcher")

	fnList := &openfunction.FunctionList{}
	if err := r.List(context.Background(), fnList); err != nil {
		log.Error(err, "Failed to list function")
		return
	}

	for _, fn := range fnList.Items {
		if fn.Spec.Build == nil ||
			fn.Spec.Build.BuilderMaxAge == nil ||
			(*fn.Spec.Build.BuilderMaxAge).Duration == 0 {
			continue
		}

		builders := &openfunction.BuilderList{}
		if err := r.List(r.ctx, builders, client.InNamespace(fn.Namespace), client.MatchingLabels{constants.FunctionLabel: fn.Name}); err != nil {
			log.Error(err, "Failed to list builder", "Function", fmt.Sprintf("%s/%s", fn.Name, fn.Namespace))
			return
		}

		for _, item := range builders.Items {
			builder := item
			if !builder.Status.IsCompleted() {
				continue
			}

			if time.Since(builder.CreationTimestamp.Time) > (*fn.Spec.Build.BuilderMaxAge).Duration {
				if err := r.Delete(r.ctx, &builder); err != nil {
					log.Error(err, "Failed to delete expired builder",
						"Function", fmt.Sprintf("%s/%s", fn.Name, fn.Namespace),
						"Builder", fmt.Sprintf("%s/%s", builder.Name, builder.Namespace))
				} else {
					log.V(1).Info("Delete expired builder",
						"Function", fmt.Sprintf("%s/%s", fn.Name, fn.Namespace),
						"Builder", fmt.Sprintf("%s/%s", builder.Name, builder.Namespace))
				}
			}
		}
	}
}

func (r *FunctionReconciler) recordEvent(fn *openfunction.Function, related runtime.Object, action, state, message string) {
	log := r.Log.WithName("RecordEvent").
		WithValues("Function", fmt.Sprintf("%s/%s", fn.Namespace, fn.Name))

	eventType := corev1.EventTypeNormal
	if state == openfunction.Timeout ||
		state == openfunction.Failed ||
		state == openfunction.Canceled {
		eventType = corev1.EventTypeWarning
	}

	reason := ""
	note := ""
	switch action {
	case buildAction:
		switch state {
		case openfunction.Building:
			reason = "BuildStarted"
			note = "Build started"
		case openfunction.Succeeded:
			reason = "BuildCompleted"
			note = "Build completed"
		case openfunction.Failed:
			reason = "BuildFailed"
			note = fmt.Sprintf("Build failed: %s", message)
		case openfunction.Timeout:
			reason = "BuildTimeout"
			note = "Build timeout"
		case openfunction.Canceled:
			reason = "BuildCanceled"
			note = "Build cancelled"
		}
	case servingAction:
		switch state {
		case openfunction.Starting:
			reason = "Starting"
			note = "Serving is starting"
		case openfunction.Running:
			reason = "Running"
			note = "Serving is running"
		case openfunction.Failed:
			reason = "ServingFailed"
			note = fmt.Sprintf("Serving start failed: %s", message)
		}
	}

	r.eventRecorder.Eventf(fn, related, eventType, reason, action, note)
	log.V(1).Info("Record Event", "Reason", reason)
}

// SetupWithManager sets up the controller with the Manager.
func (r *FunctionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &openfunction.Function{}, GatewayField, func(rawObj client.Object) []string {
		fn := rawObj.(*openfunction.Function)
		if fn.Spec.Serving != nil && fn.Spec.Serving.Triggers != nil &&
			fn.Spec.Serving.Triggers.Http != nil &&
			fn.Spec.Serving.Triggers.Http.Route != nil {
			return []string{fmt.Sprintf("%s,%s", *fn.Spec.Serving.Triggers.Http.Route.GatewayRef.Namespace, fn.Spec.Serving.Triggers.Http.Route.GatewayRef.Name)}
		} else {
			return []string{fmt.Sprintf("%s,%s", constants.DefaultGatewayNamespace, constants.DefaultGatewayName)}
		}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&openfunction.Function{}).
		Owns(&openfunction.Builder{}).
		Owns(&openfunction.Serving{}).
		Owns(&corev1.Service{}).
		Owns(&k8sgatewayapiv1alpha2.HTTPRoute{}, ctrlbuilder.WithPredicates(predicate.Funcs{UpdateFunc: r.filterHttpRouteUpdateEvent})).
		Watches(
			&source.Kind{Type: &networkingv1alpha1.Gateway{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForGateway),
			ctrlbuilder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		Complete(r)
}

func (r *FunctionReconciler) filterHttpRouteUpdateEvent(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldRoute := e.ObjectOld.(*k8sgatewayapiv1alpha2.HTTPRoute).DeepCopy()
	newRoute := e.ObjectNew.(*k8sgatewayapiv1alpha2.HTTPRoute).DeepCopy()

	if !reflect.DeepEqual(oldRoute.Spec, newRoute.Spec) {
		return true
	}

	oldRoute.ManagedFields = make([]metav1.ManagedFieldsEntry, 0)
	newRoute.ManagedFields = make([]metav1.ManagedFieldsEntry, 0)
	newRoute.ResourceVersion = ""
	oldRoute.ResourceVersion = ""
	if !reflect.DeepEqual(oldRoute.ObjectMeta, newRoute.ObjectMeta) {
		return true
	}

	return false
}

func (r *FunctionReconciler) findObjectsForGateway(gateway client.Object) []reconcile.Request {
	attachedFunctions := &openfunction.FunctionList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(GatewayField, fmt.Sprintf("%s,%s", gateway.GetNamespace(), gateway.GetName())),
	}
	err := r.List(context.TODO(), attachedFunctions, listOps)
	if err != nil {
		return []reconcile.Request{}
	}
	requests := make([]reconcile.Request, len(attachedFunctions.Items))
	for i, item := range attachedFunctions.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}
