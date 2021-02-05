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
	"os"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/cache"

	ttv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	kcache "k8s.io/client-go/tools/cache"
	kneventing "knative.dev/eventing/pkg/client/clientset/versioned/scheme"
	knapis "knative.dev/pkg/apis"
	knserving "knative.dev/serving/pkg/client/clientset/versioned/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	openfunctioniov1alpha1 "github.com/openfunction/api/v1alpha1"
	"github.com/openfunction/controllers"
	openfunction "github.com/openfunction/pkg/apis/v1alpha1"
	"github.com/openfunction/pkg/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme      = runtime.NewScheme()
	setupLog    = ctrl.Log.WithName("setup")
	tektonCache cache.Cache
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = knserving.AddToScheme(scheme)
	_ = kneventing.AddToScheme(scheme)
	_ = tekton.AddToScheme(scheme)
	_ = openfunction.AddToScheme(scheme)
	_ = openfunctioniov1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func watchBuildStatus() error {
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

	// Setup informer for PipelineRun
	plrInf, err := tektonCache.GetInformer(ctx, &ttv1beta1.PipelineRun{})
	if err != nil {
		setupLog.Error(err, "Failed to get informer for PipelineRun")
		return err
	}
	plrInf.AddEventHandler(kcache.ResourceEventHandlerFuncs{
		AddFunc: onFuncBuildUpdate,
		UpdateFunc: func(oldObj, newObj interface{}) {
			onFuncBuildUpdate(newObj)
		},
	})

	if ok := tektonCache.WaitForCacheSync(ctx.Done()); !ok {
		err := fmt.Errorf("Tekton cache failed")
		setupLog.Error(err, "Failed to get informer for PipelineRun")
		return err
	}

	return ctx.Err()
}

func onFuncBuildUpdate(obj interface{}) {
	if plr, ok := obj.(*ttv1beta1.PipelineRun); ok {
		if ok := strings.Contains(plr.Name, "-"+controllers.BuildPipelineRun); !ok {
			return
		}
		if plr.Status.CompletionTime != nil {
			var plrResult ttv1beta1.PipelineRunResult
			for _, plrResult = range plr.Status.PipelineResults {
				setupLog.V(1).Info("PipelineResult", "PipelineRun Name", plr.Name,
					"Result Name", plrResult.Name, "Result Value", plrResult.Value)
			}

			var condition knapis.Condition
			for _, condition = range plr.Status.Conditions {
				setupLog.V(1).Info("PipelineRun condition", "PipelineRun Name", plr.Name,
					"Status", condition.Status, "Type", condition.Type, "LastTransitionTime", condition.LastTransitionTime,
					"Message", condition.Message, "Reason", condition.Reason, "Severity", condition.Severity)
			}

			switch {
			case plr.IsDone():
				setupLog.V(1).Info("PipelineRun finished!")
			case plr.IsCancelled():
				setupLog.V(1).Info("PipelineRun cancelled!")
			case plr.IsTimedOut():
				setupLog.V(1).Info("PipelineRun timeout!")
			default:
				setupLog.V(1).Info("PipelineRun status unknown!")
			}
		}
	}
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

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
	if err = (&controllers.FunctionBuildReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("FunctionBuild"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "FunctionBuild")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := watchBuildStatus(); err != nil {
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
