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

package v1beta2

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	shipwrightv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/openfunction/pkg/constants"
)

//var (
//	shipwrightBuildStrategyKinds = map[shipwrightv1alpha1.BuildStrategyKind]bool{
//		shipwrightv1alpha1.NamespacedBuildStrategyKind: true,
//		shipwrightv1alpha1.ClusterBuildStrategyKind:    true}
//	shipwrightBuildStrategyKindsSlice = convertMapKeysToStringSlice(shipwrightBuildStrategyKinds)
//	kedaScaledObjectAuthRefKinds      = map[string]bool{"TriggerAuthentication": true, "ClusterTriggerAuthentication": true}
//	kedaScaledObjectAuthRefKindsSlice = convertMapKeysToStringSlice(kedaScaledObjectAuthRefKinds)
//	scalingPolicySelects              = map[v2beta2.ScalingPolicySelect]bool{
//		v2beta2.MaxPolicySelect:      true,
//		v2beta2.MinPolicySelect:      true,
//		v2beta2.DisabledPolicySelect: true,
//	}
//	scalingPolicySelectsSlice = convertMapKeysToStringSlice(scalingPolicySelects)
//	HPAScalingPolicyTypes     = map[v2beta2.HPAScalingPolicyType]bool{
//		v2beta2.PodsScalingPolicy:    true,
//		v2beta2.PercentScalingPolicy: true,
//	}
//	HPAScalingPolicyTypesSlice = convertMapKeysToStringSlice(HPAScalingPolicyTypes)
//)

var (
	shipwrightBuildStrategyKinds = map[shipwrightv1alpha1.BuildStrategyKind]bool{
		shipwrightv1alpha1.NamespacedBuildStrategyKind: true,
		shipwrightv1alpha1.ClusterBuildStrategyKind:    true}
	shipwrightBuildStrategyKindsSlice = convertMapKeysToStringSlice(shipwrightBuildStrategyKinds)
	kedaScaledJobRestartPolices       = map[v1.RestartPolicy]bool{
		v1.RestartPolicyAlways:    true,
		v1.RestartPolicyOnFailure: true,
		v1.RestartPolicyNever:     true,
	}
	kedaScaledJobRestartPolicesSlice = convertMapKeysToStringSlice(kedaScaledJobRestartPolices)
	scalingPolicySelects             = map[autoscalingv2.ScalingPolicySelect]bool{
		autoscalingv2.MaxChangePolicySelect: true,
		autoscalingv2.MinChangePolicySelect: true,
		autoscalingv2.DisabledPolicySelect:  true,
	}
	scalingPolicySelectsSlice = convertMapKeysToStringSlice(scalingPolicySelects)
	HPAScalingPolicyTypes     = map[autoscalingv2.HPAScalingPolicyType]bool{
		autoscalingv2.PercentScalingPolicy: true,
		autoscalingv2.PodsScalingPolicy:    true,
	}
	HPAScalingPolicyTypesSlice          = convertMapKeysToStringSlice(HPAScalingPolicyTypes)
	kedaScaledJobScalingStrategies      = map[string]bool{"default": true, "custom": true, "accurate": true}
	kedaScaledJobScalingStrategiesSlice = convertMapKeysToStringSlice(kedaScaledJobScalingStrategies)
)

// log is for logging in this package.
var functionlog = logf.Log.WithName("function-resource")

func (r *Function) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// +kubebuilder:webhook:path=/mutate-core-openfunction-io-v1beta2-function,mutating=true,failurePolicy=fail,groups=core.openfunction.io,resources=functions,verbs=create;update,versions=v1beta1,name=mfunctions.of.io,sideEffects=None,admissionReviewVersions=v1
var _ webhook.Defaulter = &Function{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Function) Default() {
	functionlog.Info("default", "name", r.Name)
	if r.Spec.Version == nil || *r.Spec.Version == "" {
		version := "latest"
		r.Spec.Version = &version
	}

	if r.Spec.Serving != nil && r.Spec.Serving.Triggers == nil {
		r.Spec.Serving.Triggers = &Triggers{}
	}

	if r.Spec.Serving != nil && len(r.Spec.Serving.Triggers.Dapr) == 0 {
		if r.Spec.Serving.Triggers.Http == nil {
			r.Spec.Serving.Triggers.Http = &HttpTrigger{}
		}

		if r.Spec.Serving.Triggers.Http.Port == nil || *r.Spec.Serving.Triggers.Http.Port == 0 {
			port := int32(constants.DefaultFuncPort)
			r.Spec.Serving.Triggers.Http.Port = &port
		}
	}

	if r.Spec.Serving != nil &&
		r.Spec.Serving.Triggers != nil &&
		r.Spec.Serving.Triggers.Http != nil {
		if r.Spec.Serving.Triggers.Http.Port == nil {
			port := int32(constants.DefaultFuncPort)
			r.Spec.Serving.Triggers.Http.Port = &port
		}

		namespace := constants.DefaultGatewayNamespace
		if r.Spec.Serving.Triggers.Http.Route == nil {
			route := RouteImpl{
				CommonRouteSpec: CommonRouteSpec{
					GatewayRef: &GatewayRef{
						Name:      constants.DefaultGatewayName,
						Namespace: &namespace,
					},
				},
			}
			r.Spec.Serving.Triggers.Http.Route = &route
		} else if r.Spec.Serving.Triggers.Http.Route.GatewayRef == nil {
			r.Spec.Serving.Triggers.Http.Route.GatewayRef = &GatewayRef{Name: constants.DefaultGatewayName, Namespace: &namespace}
		}
	}

	r.HandleWorkloadRuntime()
}

func (r *Function) HandleWorkloadRuntime() {
	if r.Annotations == nil {
		r.Annotations = make(map[string]string)
	}

	if _, ok := r.Annotations[constants.WasmVariantAnnotation]; !ok {
		if strings.EqualFold(r.Spec.WorkloadRuntime, constants.WasmEdgeWorkloadRuntimeName) {
			r.Annotations[constants.WasmVariantAnnotation] = constants.WasmCompatSmart
		} else {
			return
		}
	}

	if r.Spec.Build != nil {
		kind := string(shipwrightv1alpha1.ClusterBuildStrategyKind)
		strategy := &Strategy{Name: constants.WasmEdgeBuildStrategyName, Kind: &kind}
		if r.Spec.Build.Shipwright == nil {
			r.Spec.Build.Shipwright = &ShipwrightEngine{Strategy: strategy}
		} else if r.Spec.Build.Shipwright.Strategy == nil {
			r.Spec.Build.Shipwright.Strategy = strategy
		}
	}

	if r.Spec.Serving != nil {
		if r.Spec.Serving.Annotations == nil {
			r.Spec.Serving.Annotations = map[string]string{}
		}

		if _, ok := r.Spec.Serving.Annotations[constants.WasmVariantAnnotation]; !ok {
			r.Spec.Serving.Annotations[constants.WasmVariantAnnotation] = constants.WasmCompatSmart
		}

		if r.Spec.Serving.Template != nil && r.Spec.Serving.Template.RuntimeClassName == nil {
			runtimeClassName := constants.WasmEdgeRuntimeClassName
			r.Spec.Serving.Template.RuntimeClassName = &runtimeClassName
		}
	}
}

// +kubebuilder:webhook:path=/validate-core-openfunction-io-v1beta2-function,mutating=false,failurePolicy=fail,groups=core.openfunction.io,resources=functions,verbs=create;update,versions=v1beta1,name=vfunctions.of.io,sideEffects=None,admissionReviewVersions=v1
var _ webhook.Validator = &Function{}

func (r *Function) ValidateCreate() error {
	functionlog.Info("validate create", "name", r.Name)
	return r.Validate()
}

func (r *Function) ValidateUpdate(_ runtime.Object) error {
	functionlog.Info("validate update", "name", r.Name)
	return r.Validate()
}

func (r *Function) ValidateDelete() error {
	functionlog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *Function) Validate() error {
	if r.Spec.Image == "" {
		return field.Required(field.NewPath("spec", "image"), "must be specified")
	}

	if r.Spec.Build != nil {
		if r.Spec.ImageCredentials != nil && r.Spec.ImageCredentials.Name == "" {
			return field.Required(field.NewPath("spec", "imageCredentials", "name"),
				"must be specified when `spec.build.imageCredentials` is enabled")
		}
		if err := r.ValidateBuild(); err != nil {
			return err
		}
	}

	if r.Spec.Serving != nil {
		if err := r.ValidateServing(); err != nil {
			return err
		}
	}

	if r.Spec.Build == nil && r.Spec.Serving == nil {
		return field.Required(field.NewPath("spec", "serving"),
			"must be specified when `spec.build` is not enabled")
	}
	return nil
}

func (r *Function) ValidateBuild() error {
	if r.Spec.Build.Builder == nil && r.Spec.Build.Dockerfile == nil {
		return field.Required(field.NewPath("spec", "build", "builder"),
			"must be specified when `spec.build.dockerfile` is not enabled")
	}

	if r.Spec.Build.SrcRepo == nil {
		return field.Required(field.NewPath("spec", "build", "srcRepo"),
			"must be specified when `spec.build` enabled")
	}

	if r.Spec.Build.SrcRepo.Url == "" && r.Spec.Build.SrcRepo.BundleContainer == nil {
		return field.Required(field.NewPath("spec", "build", "srcRepo"),
			"must specify one of: `url` or `bundleContainer`")
	}

	if r.Spec.Build.Timeout != nil && r.Spec.Build.Timeout.Duration < 0 {
		return field.Invalid(field.NewPath("spec", "build", "timeout"),
			r.Spec.Build.Timeout.Duration, "cannot be less than 0")
	}

	if r.Spec.Build.SuccessfulBuildsHistoryLimit != nil && *r.Spec.Build.SuccessfulBuildsHistoryLimit < 0 {
		return field.Invalid(field.NewPath("spec", "build", "successfulBuildsHistoryLimit"),
			r.Spec.Build.SuccessfulBuildsHistoryLimit, "cannot be less than 0")
	}

	if r.Spec.Build.FailedBuildsHistoryLimit != nil && *r.Spec.Build.FailedBuildsHistoryLimit < 0 {
		return field.Invalid(field.NewPath("spec", "build", "failedBuildsHistoryLimit"),
			r.Spec.Build.FailedBuildsHistoryLimit, "cannot be less than 0")
	}

	if r.Spec.Build.BuilderMaxAge != nil && r.Spec.Build.BuilderMaxAge.Duration < 0 {
		return field.Invalid(field.NewPath("spec", "build", "builderMaxAge"),
			r.Spec.Build.BuilderMaxAge.Duration, "cannot be less than 0")
	}

	if r.Spec.Build.Shipwright != nil {
		if r.Spec.Build.Shipwright.Strategy != nil && r.Spec.Build.Shipwright.Strategy.Kind != nil {
			if _, ok := shipwrightBuildStrategyKinds[shipwrightv1alpha1.BuildStrategyKind(*r.Spec.Build.Shipwright.Strategy.Kind)]; !ok {
				return field.NotSupported(field.NewPath("spec", "build", "shipwright", "strategy", "kind"),
					r.Spec.Build.Shipwright.Strategy.Kind, shipwrightBuildStrategyKindsSlice)
			}
		}

		if r.Spec.Build.Shipwright.Timeout != nil && r.Spec.Build.Shipwright.Timeout.Duration < 0 {
			return field.Invalid(field.NewPath("spec", "build", "shipwright", "timeout"),
				r.Spec.Build.Shipwright.Timeout.Duration, "cannot be less than 0")
		}
	}

	return nil
}

func (r *Function) ValidateServing() error {
	if scaleOptions := r.Spec.Serving.ScaleOptions; scaleOptions != nil {
		minReplicas := int32(0)
		maxReplicas := int32(10)
		if scaleOptions.MaxReplicas != nil {
			maxReplicas = *scaleOptions.MaxReplicas
		}
		if scaleOptions.MinReplicas != nil {
			minReplicas = *scaleOptions.MinReplicas
		}
		if minReplicas < 0 {
			return field.Invalid(field.NewPath("spec", "serving", "scaleOptions", "minReplicas"),
				minReplicas, "cannot be less than 0")
		}
		if maxReplicas < 0 {
			return field.Invalid(field.NewPath("spec", "serving", "scaleOptions", "maxReplicas"),
				maxReplicas, "cannot be less than 0")
		}
		if minReplicas > maxReplicas {
			return field.Invalid(field.NewPath("spec", "serving", "scaleOptions", "minReplicas"),
				minReplicas, "cannot be greater than maxReplicas")
		}

		if keda := scaleOptions.Keda; keda != nil {
			scalerCount := 0
			if keda.ScaledJob != nil {
				scalerCount += 1
			}
			if keda.ScaledObject != nil {
				scalerCount += 1
			}
			if keda.HTTPScaledObject != nil {
				scalerCount += 1
			}
			if scalerCount > 1 {
				return field.Required(
					field.NewPath("spec", "serving", "scaleOptions", "keda"),
					"Exactly one of scaledJob, scaledObject and httpScaledObject should be enabled")
			}
			if httpScaledObject := keda.HTTPScaledObject; httpScaledObject != nil {
				if httpScaledObject.TargetPendingRequests != nil && *httpScaledObject.TargetPendingRequests < 0 {
					return field.Invalid(
						field.NewPath("spec", "serving", "scaleOptions", "keda", "httpScaledObject", "targetPendingRequests"),
						httpScaledObject.TargetPendingRequests,
						"cannot be less than 0")
				}
				if httpScaledObject.CooldownPeriod != nil && *httpScaledObject.CooldownPeriod < 0 {
					return field.Invalid(
						field.NewPath("spec", "serving", "scaleOptions", "keda", "scaledObject", "cooldownPeriod"),
						httpScaledObject.CooldownPeriod,
						"cannot be less than 0")
				}
			}
			if scaledObject := keda.ScaledObject; scaledObject != nil {
				if scaledObject.PollingInterval != nil && *scaledObject.PollingInterval < 0 {
					return field.Invalid(
						field.NewPath("spec", "serving", "scaleOptions", "keda", "scaledObject", "pollingInterval"),
						scaledObject.PollingInterval,
						"cannot be less than 0")
				}
				if scaledObject.CooldownPeriod != nil && *scaledObject.CooldownPeriod < 0 {
					return field.Invalid(
						field.NewPath("spec", "serving", "scaleOptions", "keda", "scaledObject", "cooldownPeriod"),
						scaledObject.CooldownPeriod,
						"cannot be less than 0")
				}

				if scaledObject.Advanced != nil {
					if err := r.ValidateKedaScaledObjectAdvanced(); err != nil {
						return err
					}
				}
				if scaledObject.Fallback != nil {
					if err := r.ValidateKedaScaledObjectFallback(); err != nil {
						return err
					}
				}
			}
			if scaleJob := keda.ScaledJob; scaleJob != nil {
				if scaleJob.RestartPolicy != nil {
					if _, ok := kedaScaledJobRestartPolices[*scaleJob.RestartPolicy]; !ok {
						return field.NotSupported(
							field.NewPath("spec", "serving", "scaleOptions", "keda", "scaleJob", "restartPolicy"),
							scaleJob.RestartPolicy,
							kedaScaledJobRestartPolicesSlice)
					}
				}
				if scaleJob.PollingInterval != nil && *scaleJob.PollingInterval < 0 {
					return field.Invalid(
						field.NewPath("spec", "serving", "scaleOptions", "keda", "scaleJob", "pollingInterval"),
						scaleJob.PollingInterval,
						"must not be less than 0")
				}
				if scaleJob.SuccessfulJobsHistoryLimit != nil && *scaleJob.SuccessfulJobsHistoryLimit < 0 {
					return field.Invalid(
						field.NewPath("spec", "serving", "scaleOptions", "keda", "scaleJob", "successfulJobsHistoryLimit"),
						scaleJob.SuccessfulJobsHistoryLimit,
						"must not be less than 0")
				}
				if scaleJob.FailedJobsHistoryLimit != nil && *scaleJob.FailedJobsHistoryLimit < 0 {
					return field.Invalid(
						field.NewPath("spec", "serving", "scaleOptions", "keda", "scaleJob", "failedJobsHistoryLimit"),
						scaleJob.FailedJobsHistoryLimit,
						"must not be less than 0")
				}

				if err := r.ValidateKedaScaledJobScalingStrategy(); err != nil {
					return err
				}
			}
		}
	}

	if r.Spec.Serving.Triggers.Http != nil && r.Spec.Serving.Triggers.Http.Engine != nil {
		if *r.Spec.Serving.Triggers.Http.Engine != HttpEngineKeda && *r.Spec.Serving.Triggers.Http.Engine != HttpEngineKnative {
			return field.Invalid(
				field.NewPath("spec", "serving", "triggers", "http", "engine"),
				r.Spec.Serving.Triggers.Http.Engine,
				"Unknown engine type, supported engines include knative and keda")
		}
	}

	return nil
}

func (r *Function) ValidateKedaScaledObjectAdvanced() error {
	advanced := r.Spec.Serving.ScaleOptions.Keda.ScaledObject.Advanced
	if advanced.HorizontalPodAutoscalerConfig != nil && advanced.HorizontalPodAutoscalerConfig.Behavior != nil {
		behavior := advanced.HorizontalPodAutoscalerConfig.Behavior
		if behavior.ScaleUp != nil {
			scaleUp := behavior.ScaleUp
			stabilizationWindowSeconds := scaleUp.StabilizationWindowSeconds
			if stabilizationWindowSeconds != nil && (*stabilizationWindowSeconds < 0 || *stabilizationWindowSeconds > 3600) {
				return field.Invalid(field.NewPath("spec", "serving", "scaleOptions",
					"keda", "scaleObject", "advanced", "horizontalPodAutoscalerConfig", "behavior", "scaleUp", "stabilizationWindowSeconds"),
					stabilizationWindowSeconds,
					"must be greater than or equal to 0 less than 3600")
			}
			if scaleUp.SelectPolicy != nil {
				if _, ok := scalingPolicySelects[*scaleUp.SelectPolicy]; !ok {
					return field.NotSupported(field.NewPath("spec", "serving", "scaleOptions",
						"keda", "scaleObject", "advanced", "horizontalPodAutoscalerConfig", "behavior", "scaleUp", "selectPolicy"),
						scaleUp.SelectPolicy,
						scalingPolicySelectsSlice)
				}
			}
			for index, policy := range scaleUp.Policies {
				if _, ok := HPAScalingPolicyTypes[policy.Type]; !ok {
					return field.NotSupported(field.NewPath("spec", "serving", "scaleOptions",
						"keda", "scaleObject", "advanced", "horizontalPodAutoscalerConfig", "behavior", "scaleUp",
						"policies", fmt.Sprintf("[%d]", index), "type"),
						policy.Type,
						HPAScalingPolicyTypesSlice)
				}
				if policy.PeriodSeconds < 0 {
					return field.Invalid(field.NewPath("spec", "serving", "scaleOptions",
						"keda", "scaleObject", "advanced", "horizontalPodAutoscalerConfig", "behavior", "scaleUp",
						"policies", fmt.Sprintf("[%d]", index), "periodSeconds"),
						policy.PeriodSeconds,
						"must be greater than 0")
				}
			}
		}

		if behavior.ScaleDown != nil {
			scaleDown := behavior.ScaleDown
			stabilizationWindowSeconds := scaleDown.StabilizationWindowSeconds
			if stabilizationWindowSeconds != nil && (*stabilizationWindowSeconds < 0 || *stabilizationWindowSeconds > 3600) {
				return field.Invalid(field.NewPath("spec", "serving", "scaleOptions",
					"keda", "scaleObject", "advanced", "horizontalPodAutoscalerConfig", "behavior", "scaleDown", "stabilizationWindowSeconds"),
					stabilizationWindowSeconds,
					"must be greater than or equal to 0 less than 3600")
			}
			if scaleDown.SelectPolicy != nil {
				if _, ok := scalingPolicySelects[*scaleDown.SelectPolicy]; !ok {
					return field.NotSupported(field.NewPath("spec", "serving", "scaleOptions",
						"keda", "scaleObject", "advanced", "horizontalPodAutoscalerConfig", "behavior", "scaleDown", "selectPolicy"),
						scaleDown.SelectPolicy,
						scalingPolicySelectsSlice)
				}
			}
			for index, policy := range scaleDown.Policies {
				if _, ok := HPAScalingPolicyTypes[policy.Type]; !ok {
					return field.NotSupported(field.NewPath("spec", "serving", "scaleOptions",
						"keda", "scaleObject", "advanced", "horizontalPodAutoscalerConfig", "behavior", "scaleDown",
						"policies", fmt.Sprintf("[%d]", index), "type"),
						policy.Type,
						HPAScalingPolicyTypesSlice)
				}
				if policy.PeriodSeconds < 0 {
					return field.Invalid(field.NewPath("spec", "serving", "scaleOptions",
						"keda", "scaleObject", "advanced", "horizontalPodAutoscalerConfig", "behavior", "scaleDown",
						"policies", fmt.Sprintf("[%d]", index), "periodSeconds"),
						policy.PeriodSeconds,
						"must be greater than 0")
				}
			}
		}
	}
	return nil
}

func (r *Function) ValidateKedaScaledObjectFallback() error {
	fallback := r.Spec.Serving.ScaleOptions.Keda.ScaledObject.Fallback
	if fallback != nil {
		if fallback.FailureThreshold < 1 {
			return field.Invalid(field.NewPath("spec", "serving", "scaleOptions",
				"keda", "scaleObject", "fallback", "failureThreshold"),
				fallback.FailureThreshold,
				"must be greater than or equal to 1")
		}
		if fallback.Replicas < 0 {
			return field.Invalid(field.NewPath("spec", "serving", "scaleOptions",
				"keda", "scaleObject", "fallback", "replicas"),
				fallback.Replicas,
				"must be greater than or equal to 0")
		}
	}
	return nil
}

func (r *Function) ValidateKedaScaledJobScalingStrategy() error {
	strategy := r.Spec.Serving.ScaleOptions.Keda.ScaledJob.ScalingStrategy
	if strategy.Strategy != "" {
		if _, ok := kedaScaledJobScalingStrategies[strategy.Strategy]; !ok {
			return field.NotSupported(field.NewPath("spec", "serving", "scaleOptions",
				"keda", "scaleJob", "scalingStrategy", "strategy"),
				strategy.Strategy,
				kedaScaledJobScalingStrategiesSlice)
		}
		if strategy.Strategy == "custom" && strategy.CustomScalingQueueLengthDeduction == nil {
			return field.Required(field.NewPath("spec", "serving", "scaleOptions",
				"keda", "scaleJob", "scalingStrategy", "customScalingQueueLengthDeduction"),
				"must be specified when `strategy.Strategy` is custom")
		}
		if strategy.Strategy == "custom" && strategy.CustomScalingRunningJobPercentage == "" {
			return field.Required(field.NewPath("spec", "serving", "scaleOptions",
				"keda", "scaleJob", "scalingStrategy", "customScalingRunningJobPercentage"),
				"must be specified when `strategy.Strategy` is custom")
		}

	}
	if strategy.CustomScalingQueueLengthDeduction != nil && *strategy.CustomScalingQueueLengthDeduction < 0 {
		return field.Invalid(field.NewPath("spec", "serving", "scaleOptions",
			"keda", "scaleJob", "scalingStrategy", "customScalingQueueLengthDeduction"),
			strategy.CustomScalingQueueLengthDeduction,
			"cannot be less than 0")
	}
	if strategy.CustomScalingRunningJobPercentage != "" {
		reg := regexp.MustCompile(`^([0-9.]+)[ ]*%$`)
		if !reg.MatchString(strategy.CustomScalingRunningJobPercentage) {
			return field.Invalid(field.NewPath("spec", "serving", "scaleOptions",
				"keda", "scaleJob", "scalingStrategy", "customScalingRunningJobPercentage"),
				strategy.CustomScalingRunningJobPercentage,
				"is not an invalid percentage value")
		}
	}
	return nil
}

func convertMapKeysToStringSlice(m interface{}) []string {
	v := reflect.ValueOf(m)
	if v.Kind() == reflect.Map {
		keys := make([]string, 0, len(v.MapKeys()))
		for _, key := range v.MapKeys() {
			switch key.Interface().(type) {
			case string:
				keys = append(keys, key.Interface().(string))
			case shipwrightv1alpha1.BuildStrategyKind:
				keys = append(keys, string(key.Interface().(shipwrightv1alpha1.BuildStrategyKind)))
			case v1.RestartPolicy:
				keys = append(keys, string(key.Interface().(v1.RestartPolicy)))
			case autoscalingv2.ScalingPolicySelect:
				keys = append(keys, string(key.Interface().(autoscalingv2.ScalingPolicySelect)))
			case autoscalingv2.HPAScalingPolicyType:
				keys = append(keys, string(key.Interface().(autoscalingv2.HPAScalingPolicyType)))
			}
		}
		return keys
	}
	return nil
}
