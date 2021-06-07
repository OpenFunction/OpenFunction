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

package main

import (
	"context"
	"flag"
	"fmt"
	"go.uber.org/zap/zapcore"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	ttv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	kcache "k8s.io/client-go/tools/cache"
	kneventing "knative.dev/eventing/pkg/client/clientset/versioned/scheme"
	knserving "knative.dev/serving/pkg/client/clientset/versioned/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	openfunction "github.com/openfunction/pkg/apis/v1alpha1"
	"github.com/openfunction/pkg/controllers"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	// +kubebuilder:scaffold:imports
)

var (
	scheme      = runtime.NewScheme()
	setupLog    = ctrl.Log.WithName("setup")
	tektonCache cache.Cache
	client      crclient.Client
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = knserving.AddToScheme(scheme)
	_ = kneventing.AddToScheme(scheme)
	_ = tekton.AddToScheme(scheme)
	_ = openfunction.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func watchBuilderStatus(mgr manager.Manager) error {
	tektonScheme := runtime.NewScheme()
	_ = tekton.AddToScheme(tektonScheme)
	ctx := context.Background()

	var err error
	tektonCache, err = cache.New(ctrl.GetConfigOrDie(), cache.Options{
		Scheme: tektonScheme,
	})
	if err != nil {
		setupLog.Error(err, "Failed to create tekton cache")
		return err
	}

	go func() {
		_ = tektonCache.Start(ctx.Done())
	}()

	mgr.GetCache().WaitForCacheSync(ctx.Done())

	// Setup informer for PipelineRun
	plrInf, err := tektonCache.GetInformer(ctx, &ttv1beta1.PipelineRun{})
	if err != nil {
		setupLog.Error(err, "Failed to get informer for PipelineRun")
		return err
	}
	plrInf.AddEventHandler(kcache.ResourceEventHandlerFuncs{
		AddFunc: onBuilderUpdate,
		UpdateFunc: func(oldObj, newObj interface{}) {
			onBuilderUpdate(newObj)
		},
	})

	if ok := tektonCache.WaitForCacheSync(ctx.Done()); !ok {
		err := fmt.Errorf("Tekton cache failed")
		setupLog.Error(err, "Failed to get informer for PipelineRun")
		return err
	}

	return ctx.Err()
}

func onBuilderUpdate(obj interface{}) {
	if plr, ok := obj.(*ttv1beta1.PipelineRun); ok {
		if ok := strings.HasSuffix(plr.Name, "-"+controllers.BuildPipelineRun); !ok {
			return
		}

		if plr.Status.CompletionTime != nil {
			fn := strings.TrimSuffix(plr.Name, fmt.Sprintf("-%s-%s", "builder", controllers.BuildPipelineRun))

			cond := plr.Status.GetCondition(apis.ConditionSucceeded)
			// Enter Serving Phase only when the PipelineRun's 'Succeeded' condition is True which means the build is successful
			if cond.Status == corev1.ConditionTrue {
				if err := UpdateFuncStatus(plr.Namespace, fn, openfunction.ServingPhase, ""); err != nil {
					setupLog.Error(err, "Failed to update function status", "namespace", plr.Namespace, "name", fn)
				}
				// Mark build phase failed if the PipelineRun's 'Succeeded' condition is False or Unknown
			} else {
				if err := UpdateFuncStatus(plr.Namespace, fn, openfunction.BuildPhase, openfunction.Failed); err != nil {
					setupLog.Error(err, "Failed to update function status", "namespace", plr.Namespace, "name", fn)
				}
			}

			switch {
			case plr.IsDone():
				setupLog.Info("Function build completed", "Succeeded", cond.Status, "namespace", plr.Namespace, "name", fn)
			case plr.IsCancelled():
				setupLog.Info("PipelineRun cancelled!")
			case plr.IsTimedOut():
				setupLog.Info("PipelineRun timeout!")
			default:
				setupLog.Info("PipelineRun status unknown!")
			}
		}
	}
}

func UpdateFuncStatus(ns string, name string, phase string, state string) error {
	var fn openfunction.Function
	ctx := context.Background()

	nsn := types.NamespacedName{Namespace: ns, Name: name}
	if err := client.Get(ctx, nsn, &fn); err != nil {
		setupLog.Error(err, "Unable to get function", "name", nsn.String())
		return err
	}

	status := openfunction.FunctionStatus{Phase: phase, State: state}
	status.DeepCopyInto(&fn.Status)
	if err := client.Status().Update(ctx, &fn); err != nil {
		return err
	}

	return nil
}

func main() {
	var logLevel string
	var metricsAddr string
	var enableLeaderElection bool

	flag.StringVar(&logLevel, "log-level", "info", "The log level, known values are, debug, info, warn, error")
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	var level zapcore.LevelEnabler
	switch logLevel {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		fmt.Printf("unkonw log level %s", logLevel)
		os.Exit(1)
	}

	ctrl.SetLogger(zap.New(zap.UseDevMode(true), zap.Level(level)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "79f0111e.openfunction.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.FunctionReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Function"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Function")
		os.Exit(1)
	}
	if err = (&controllers.BuilderReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Builder"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Builder")
		os.Exit(1)
	}
	if err = (&controllers.ServingReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Serving"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Serving")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	client = mgr.GetClient()
	go func() {
		if err := watchBuilderStatus(mgr); err != nil {
			os.Exit(1)
		}
	}()

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
