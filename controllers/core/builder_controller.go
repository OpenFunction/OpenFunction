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
	"math"
	"time"

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
	timers map[string]*time.Timer
}

func NewBuilderReconciler(mgr manager.Manager) *BuilderReconciler {

	r := &BuilderReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("Builder"),
		timers: make(map[string]*time.Timer),
	}

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
			r.stopTimer(req.NamespacedName.String())
		}
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	builderRun := r.createBuilderRun()

	if builder.Spec.State == openfunction.BuilderStateCancel {
		if err := builderRun.Cancel(builder); err != nil {
			log.Error(err, "Failed to cancel builder")
			return ctrl.Result{}, err
		}
	}

	if builder.Status.IsCompleted() {
		log.V(1).Info("Build had completed")
		return ctrl.Result{}, nil
	}

	// Build timeout, update builder status.
	// when the operator restarts, update the status of builder that has timed out, without creating a timer.
	if builder.Spec.Timeout != nil &&
		time.Since(builder.CreationTimestamp.Time) > builder.Spec.Timeout.Duration {
		builder.Status.State = openfunction.Timeout
		builder.Status.Reason = openfunction.Timeout
		if err := r.Status().Update(r.ctx, builder); err != nil {
			log.Error(err, "Failed to update builder status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Start timer if build is not completed.
	r.startTimer(builder)

	// If Builder had created, Update the status of the builder according to the result of the build.
	if builder.Status.Phase != "" && builder.Status.State != "" {
		if err := r.getBuilderResult(builder, builderRun); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Reset builder status.
	builder.Status = openfunction.BuilderStatus{}
	if err := r.Status().Update(r.ctx, builder); err != nil {
		log.Error(err, "Failed to reset builder status")
		return ctrl.Result{}, err
	}

	if err := builderRun.Start(builder); err != nil {
		log.Error(err, "Failed to start builder")
		return ctrl.Result{}, err
	}

	builder.Status.Phase = openfunction.BuildPhase
	builder.Status.State = openfunction.Building
	if err := r.Status().Update(r.ctx, builder); err != nil {
		log.Error(err, "Failed to update builder status")
		return ctrl.Result{}, err
	}

	log.V(1).Info("Builder is running")

	return ctrl.Result{}, nil
}

func (r *BuilderReconciler) createBuilderRun() core.BuilderRun {

	return shipwright.NewBuildRun(r.ctx, r.Client, r.Scheme, r.Log)
}

// Update the status of the builder according to the result of the build.
func (r *BuilderReconciler) getBuilderResult(builder *openfunction.Builder, builderRun core.BuilderRun) error {
	log := r.Log.WithName("GetBuilderResult").
		WithValues("Builder", fmt.Sprintf("%s/%s", builder.Namespace, builder.Name))

	res, reason, err := builderRun.Result(builder)
	if err != nil {
		log.Error(err, "Get build result error")
		return err
	}

	// Build did not complete.
	if res == "" {
		return nil
	}

	if res != builder.Status.State {
		builder.Status.State = res
		builder.Status.Reason = reason
		if err := r.Status().Update(r.ctx, builder); err != nil {
			return err
		}

		r.stopTimer(fmt.Sprintf("%s/%s", builder.Namespace, builder.Name))
		log.V(1).Info("Update builder status", "state", res)
	}

	return nil
}

func (r *BuilderReconciler) startTimer(builder *openfunction.Builder) {
	namespacedName := fmt.Sprintf("%s/%s", builder.Namespace, builder.Name)
	log := r.Log.WithName("Timer").WithValues("Builder", namespacedName)

	if builder.Spec.Timeout == nil ||
		time.Since(builder.CreationTimestamp.Time) > builder.Spec.Timeout.Duration {
		return
	}

	// Skipped when timer had started.
	t, ok := r.timers[namespacedName]
	if ok {
		return
	}

	t = time.NewTimer(builder.Spec.Timeout.Duration - time.Since(builder.CreationTimestamp.Time))
	r.timers[namespacedName] = t

	go r.waitBuildTimeout(builder, t, 0)

	log.V(1).Info("Timer started")
}

func (r *BuilderReconciler) waitBuildTimeout(builder *openfunction.Builder, t *time.Timer, retries int) {
	namespacedName := fmt.Sprintf("%s/%s", builder.Namespace, builder.Name)
	log := r.Log.WithName("BuildTimeout").WithValues("Builder", namespacedName)

	select {
	case <-t.C:
		log.V(1).Info("Build timeout", "Retries", retries)
		if err := r.buildTimeout(builder); err != nil {
			log.Error(err, "Failed to update builder status")
			// Reset the timer, the time of the timer is 2 to the power of retries times,
			// and the maximum is 5 minutes.
			if retries <= 8 {
				t.Reset(time.Second * time.Duration(math.Pow(2, float64(retries))))
			} else {
				t.Reset(5 * time.Minute)
			}
			retries++
			go r.waitBuildTimeout(builder, t, retries)
			log.V(1).Info("Retry to update builder status", "Retries", retries)
			return
		}

		r.stopTimer(namespacedName)
	}
}

func (r *BuilderReconciler) buildTimeout(builder *openfunction.Builder) error {
	b := &openfunction.Builder{}
	if err := r.Get(r.ctx, client.ObjectKeyFromObject(builder), b); err != nil {
		if util.IsNotFound(err) {
			return nil
		}

		return err
	}

	if !b.Status.IsCompleted() {
		b.Status.State = openfunction.Timeout
		b.Status.Reason = openfunction.Timeout
		return r.Status().Update(r.ctx, b)
	}

	return nil
}

func (r *BuilderReconciler) stopTimer(key string) {
	log := r.Log.WithName("Timer").WithValues("Builder", key)
	t := r.timers[key]
	if t != nil {
		t.Stop()
		delete(r.timers, key)
		log.Info("Timer stopped")
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *BuilderReconciler) SetupWithManager(mgr ctrl.Manager, owns []client.Object) error {

	b := ctrl.NewControllerManagedBy(mgr).
		For(&openfunction.Builder{})

	for _, own := range owns {
		b.Owns(own)
	}

	return b.Complete(r)
}
