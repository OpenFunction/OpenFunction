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
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	openfunction "github.com/openfunction/apis/core/v1beta2"
	"github.com/openfunction/pkg/constants"
	"github.com/openfunction/pkg/core"
	"github.com/openfunction/pkg/core/serving/kedahttp"
	"github.com/openfunction/pkg/core/serving/knative"
	"github.com/openfunction/pkg/core/serving/openfuncasync"
	"github.com/openfunction/pkg/util"
)

var doOnce sync.Once

// ServingReconciler reconciles a Serving object
type ServingReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	ctx           context.Context
	timers        map[string]*time.Timer
	defaultConfig map[string]string

	eventRecorder events.EventRecorder
}

func NewServingReconciler(mgr manager.Manager, eventRecorder events.EventRecorder) *ServingReconciler {

	r := &ServingReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		Log:           ctrl.Log.WithName("controllers").WithName("Serving"),
		timers:        make(map[string]*time.Timer),
		eventRecorder: eventRecorder,
	}

	return r
}

//+kubebuilder:rbac:groups=core.openfunction.io,resources=servings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.openfunction.io,resources=servings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.openfunction.io,resources=servings/finalizers,verbs=update
//+kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dapr.io,resources=components;subscriptions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keda.sh,resources=scaledjobs;scaledobjects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=http.keda.sh,resources=httpscaledobjects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=list;get;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Serving object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ServingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ctx = ctx
	log := r.Log.WithValues("Serving", req.NamespacedName)

	var s openfunction.Serving

	if err := r.Get(ctx, req.NamespacedName, &s); err != nil {
		if util.IsNotFound(err) {
			log.V(1).Info("Serving deleted")
			r.stopTimer(req.NamespacedName.String())
		}
		return ctrl.Result{}, util.IgnoreNotFound(err)
	}

	servingRun := r.getServingRun(&s)

	// Serving start timeout, update serving status.
	if s.Spec.Timeout != nil &&
		time.Since(s.CreationTimestamp.Time) > s.Spec.Timeout.Duration {
		if s.Status.IsStarting() {
			s.Status.State = openfunction.Timeout
			if err := r.Status().Update(r.ctx, &s); err != nil {
				log.Error(err, "Failed to update serving status")
				return ctrl.Result{}, err
			}

			r.recordEvent(&s)
		}
		return ctrl.Result{}, nil
	}

	// Get default global configuration from ConfigMap
	r.defaultConfig = util.GetDefaultConfig(r.ctx, r.Client, r.Log)

	// Start timer if serving is starting.
	if s.Status.IsStarting() {
		r.startTimer(&s)
	}

	// Serving is running, no need to create.
	if s.Status.Phase != "" && s.Status.State != "" {
		// Update the status of the serving according to the result of the serving.
		if err := r.getServingResult(&s, servingRun); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if s.Spec.Timeout != nil &&
		time.Since(s.CreationTimestamp.Time) > s.Spec.Timeout.Duration {
		log.Error(nil, "Serving start timeout")

		if err := servingRun.Clean(&s); err != nil {
			log.Error(err, "Failed to clean serving")
			return ctrl.Result{}, err
		}

		s.Status.Phase = openfunction.ServingPhase
		s.Status.State = openfunction.Timeout
		if err := r.Status().Update(r.ctx, &s); err != nil {
			log.Error(err, "Failed to update serving status")
			return ctrl.Result{}, err
		}

		r.recordEvent(&s)
		return ctrl.Result{}, nil
	}

	// Reset serving status.
	s.Status = openfunction.ServingStatus{}
	if err := r.Status().Update(r.ctx, &s); err != nil {
		log.Error(err, "Failed to reset serving status")
		return ctrl.Result{}, err
	}

	if err := servingRun.Run(&s, r.defaultConfig); err != nil {
		doOnce.Do(func() {
			if strings.Contains(err.Error(), "valueFrom.fieldRef") {
				log.Info("In order to use the Kubernetes Downward API, " +
					"we need to enable the 'kubernetes.podspec-fieldref' configuration for Knative, refer to: " +
					"https://knative.dev/development/serving/configuration/feature-flags/#kubernetes-downward-api")
				log.Info("Now we will update the 'config-features' ConfigMap resource under the 'knative-serving' Namespace")

				cm := &corev1.ConfigMap{}

				ns := util.GetConfigOrDefault(
					r.defaultConfig,
					"knative-serving.namespace",
					constants.DefaultKnativeServingNamespace,
				)
				name := util.GetConfigOrDefault(r.defaultConfig,
					"knative-serving.config-features.name",
					constants.DefaultKnativeServingFeaturesCMName,
				)

				if err := r.Client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, cm); err == nil {
					if d, ok := cm.Data["kubernetes.podspec-fieldref"]; !ok || d != "enabled" {
						cm.Data["kubernetes.podspec-fieldref"] = "enabled"
						if err := r.Client.Update(ctx, cm); err != nil {
							log.Error(err, "Failed to update 'config-features' ConfigMap")
						}
					}
				} else {
					log.Error(err, "Failed to get 'config-features' ConfigMap")
				}
			} else {
				log.Error(err, "Failed to start serving")
			}
		})
		return ctrl.Result{}, err
	}

	s.Status.Phase = openfunction.ServingPhase
	s.Status.State = openfunction.Starting
	if err := r.Status().Update(r.ctx, &s); err != nil {
		log.Error(err, "Failed to update serving status")
		return ctrl.Result{}, err
	}

	r.recordEvent(&s)

	log.V(1).Info("Serving is starting")

	return ctrl.Result{}, nil
}

func (r *ServingReconciler) getServingRun(s *openfunction.Serving) core.ServingRun {
	if s.Spec.Triggers.Http != nil {
		if s.Spec.Triggers.Http.Engine != nil && *s.Spec.Triggers.Http.Engine == openfunction.HttpEngineKeda {
			return kedahttp.NewServingRun(r.ctx, r.Client, r.Scheme, r.Log)
		} else {
			return knative.NewServingRun(r.ctx, r.Client, r.Scheme, r.Log)
		}
	} else {
		return openfuncasync.NewServingRun(r.ctx, r.Client, r.Scheme, r.Log)
	}
}

// Update the status of the serving according to the result of the serving.
func (r *ServingReconciler) getServingResult(s *openfunction.Serving, servingRun core.ServingRun) error {
	log := r.Log.WithName("GetServingResult").
		WithValues("Serving", fmt.Sprintf("%s/%s", s.Namespace, s.Name))

	res, reason, message, err := servingRun.Result(s)
	if err != nil {
		log.Error(err, "Get serving result error")
		return err
	}

	// Serving is starting.
	if res == "" {
		return nil
	}

	if res != s.Status.State ||
		reason != s.Status.Reason ||
		message != s.Status.Message {
		s.Status.State = res
		s.Status.Reason = reason
		s.Status.Message = message
		if err := r.Status().Update(r.ctx, s); err != nil {
			return err
		}

		r.recordEvent(s)

		r.stopTimer(fmt.Sprintf("%s/%s", s.Namespace, s.Name))
		log.V(1).Info("Update serving status", "state", res)
	}

	return nil
}

func (r *ServingReconciler) startTimer(serving *openfunction.Serving) {
	namespacedName := fmt.Sprintf("%s/%s", serving.Namespace, serving.Name)
	log := r.Log.WithName("Timer").WithValues("Serving", namespacedName)

	if serving.Spec.Timeout == nil ||
		time.Since(serving.CreationTimestamp.Time) > serving.Spec.Timeout.Duration {
		return
	}

	// Skipped when timer had started.
	t, ok := r.timers[namespacedName]
	if ok {
		return
	}

	t = time.NewTimer(serving.Spec.Timeout.Duration - time.Since(serving.CreationTimestamp.Time))
	r.timers[namespacedName] = t

	go func() {

		defer r.stopTimer(namespacedName)

		select {
		case <-t.C:
			s := &openfunction.Serving{}
			if err := r.Get(r.ctx, client.ObjectKeyFromObject(serving), s); err != nil {
				if util.IsNotFound(err) {
					log.Info("Serving had delete")
					return
				}

				log.Error(err, "Failed to get Serving")
				return
			}

			if s.Status.IsStarting() {
				log.Error(nil, "Serving start timeout")
				s.Status.State = openfunction.Timeout
				if err := r.Status().Update(r.ctx, s); err != nil {
					log.Error(err, "Failed to update serving status")
				} else {
					r.recordEvent(s)
				}
			}
		}
	}()

	log.V(1).Info("Timer started")
}

func (r *ServingReconciler) stopTimer(key string) {
	log := r.Log.WithName("Timer").WithValues("Serving", key)
	t := r.timers[key]
	if t != nil {
		t.Stop()
		delete(r.timers, key)
		log.Info("Timer stopped")
	}
}

func (r *ServingReconciler) recordEvent(serving *openfunction.Serving) {
	log := r.Log.WithName("RecordEvent").
		WithValues("Serving", fmt.Sprintf("%s/%s", serving.Namespace, serving.Name))

	eventType := corev1.EventTypeNormal
	if serving.Status.State == openfunction.Failed {
		eventType = corev1.EventTypeWarning
	}

	note := ""
	switch serving.Status.State {
	case openfunction.Starting:
		note = "Serving is starting"
	case openfunction.Running:
		note = "Serving is running"
	case openfunction.Failed:
		note = fmt.Sprintf("Serving start failed: %s", serving.Status.Message)
	}

	r.eventRecorder.Eventf(serving, nil, eventType, serving.Status.State, servingAction, note)
	log.V(1).Info("Record Event", "Reason", serving.Status.State)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServingReconciler) SetupWithManager(mgr ctrl.Manager, owns []client.Object) error {

	b := ctrl.NewControllerManagedBy(mgr).
		For(&openfunction.Serving{})

	for _, own := range owns {
		b.Owns(own)
	}

	return b.Complete(r)
}
