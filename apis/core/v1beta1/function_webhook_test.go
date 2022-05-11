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

package v1beta1

import (
	//"fmt"
	//"reflect"
	"testing"
	"time"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_Validate(t *testing.T) {
	builder := "builder"
	successfulBuildsHistoryLimit := int32(-1)
	failedBuildsHistoryLimit := int32(-1)
	strategyKind := "test"
	minReplicasNegative := int32(-1)
	maxReplicasNegative := int32(-1)
	minReplicas := int32(5)
	maxReplicas := int32(0)
	pollingInterval := int32(-1)
	cooldownPeriod := int32(-1)
	stabilizationWindowSecondsNegative := int32(-1)
	stabilizationWindowSecondsLimit := int32(3601)
	var selectPolicy autoscalingv2beta2.ScalingPolicySelect = "test"
	restartPolicy := v1.RestartPolicy("test")
	var scaleTargetKind ScaleTargetKind = ""

	tests := []struct {
		name    string
		r       Function
		want    error
		wantErr bool
	}{
		{
			name: "function.spec.image",
			r: Function{
				Spec: FunctionSpec{
					Image: "",
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.image.imageCredentials",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: nil,
					Build:            &BuildImpl{},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.image.imageCredentials.name",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{},
					Build:            &BuildImpl{},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.build.builder",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Build:            &BuildImpl{},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.build.srcRepo.url",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Build: &BuildImpl{
						Builder: &builder,
						SrcRepo: &GitRepo{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.build.timeout",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Build: &BuildImpl{
						Builder: &builder,
						SrcRepo: &GitRepo{Url: "test"},
						Timeout: &metav1.Duration{Duration: time.Duration(-1)},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.build.successfulBuildsHistoryLimit",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Build: &BuildImpl{
						Builder:                      &builder,
						SrcRepo:                      &GitRepo{Url: "test"},
						SuccessfulBuildsHistoryLimit: &successfulBuildsHistoryLimit,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.build.failedBuildsHistoryLimit",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Build: &BuildImpl{
						Builder:                  &builder,
						SrcRepo:                  &GitRepo{Url: "test"},
						FailedBuildsHistoryLimit: &failedBuildsHistoryLimit,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.build.builderMaxAge",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Build: &BuildImpl{
						Builder:       &builder,
						SrcRepo:       &GitRepo{Url: "test"},
						BuilderMaxAge: &metav1.Duration{Duration: time.Duration(-1)},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.build.shipwright.strategy.kind",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Build: &BuildImpl{
						Builder:    &builder,
						SrcRepo:    &GitRepo{Url: "test"},
						Shipwright: &ShipwrightEngine{Strategy: &Strategy{Kind: &strategyKind}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.build.shipwright.timeout",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Build: &BuildImpl{
						Builder: &builder,
						SrcRepo: &GitRepo{Url: "test"},
						Shipwright: &ShipwrightEngine{
							Timeout: &metav1.Duration{Duration: time.Duration(-1)},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.runtime",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving:          &ServingImpl{},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.runtime",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving:          &ServingImpl{Runtime: Runtime("keda-http")},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.minReplicas",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							MinReplicas: &minReplicasNegative,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.maxReplicas",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							MaxReplicas: &maxReplicasNegative,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.minReplicas and maxReplicas",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							MinReplicas: &minReplicas,
							MaxReplicas: &maxReplicas,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.ScaledJob and ScaledObject",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledObject.workloadType",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{WorkloadType: "test"},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledObject.pollingInterval",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{PollingInterval: &pollingInterval},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledObject.cooldownPeriod",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{CooldownPeriod: &cooldownPeriod},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledObject.minReplicaCount",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{MinReplicaCount: &minReplicasNegative},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledObject.maxReplicaCount",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{MaxReplicaCount: &maxReplicasNegative},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledObject.minReplicaCount and maxReplicaCount",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									MinReplicaCount: &minReplicas,
									MaxReplicaCount: &maxReplicas,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "spec.serving.scaleOptions.keda.scaleObject.advanced.horizontalPodAutoscalerConfig.behavior.scaleUp.stabilizationWindowSeconds",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2beta2.HorizontalPodAutoscalerBehavior{
												ScaleUp: &autoscalingv2beta2.HPAScalingRules{
													StabilizationWindowSeconds: &stabilizationWindowSecondsNegative,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledObject.advanced.horizontalPodAutoscalerConfig.behavior.scaleUp.stabilizationWindowSeconds",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2beta2.HorizontalPodAutoscalerBehavior{
												ScaleUp: &autoscalingv2beta2.HPAScalingRules{
													StabilizationWindowSeconds: &stabilizationWindowSecondsLimit,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledObject.advanced.horizontalPodAutoscalerConfig.behavior.scaleUp.SelectPolicy",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2beta2.HorizontalPodAutoscalerBehavior{
												ScaleUp: &autoscalingv2beta2.HPAScalingRules{
													SelectPolicy: &selectPolicy,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledObject.advanced.horizontalPodAutoscalerConfig.behavior.scaleUp.policies[0].type",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2beta2.HorizontalPodAutoscalerBehavior{
												ScaleUp: &autoscalingv2beta2.HPAScalingRules{
													Policies: []autoscalingv2beta2.HPAScalingPolicy{
														{Type: "test"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledObject.advanced.horizontalPodAutoscalerConfig.behavior.scaleUp.policies[0].periodSeconds",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2beta2.HorizontalPodAutoscalerBehavior{
												ScaleUp: &autoscalingv2beta2.HPAScalingRules{
													Policies: []autoscalingv2beta2.HPAScalingPolicy{
														{
															Type:          "Pods",
															PeriodSeconds: -1,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledObject.advanced.horizontalPodAutoscalerConfig.behavior.scaleDown.stabilizationWindowSeconds",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2beta2.HorizontalPodAutoscalerBehavior{
												ScaleDown: &autoscalingv2beta2.HPAScalingRules{
													StabilizationWindowSeconds: &stabilizationWindowSecondsLimit,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledObject.advanced.horizontalPodAutoscalerConfig.behavior.scaleDown.SelectPolicy",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2beta2.HorizontalPodAutoscalerBehavior{
												ScaleDown: &autoscalingv2beta2.HPAScalingRules{
													SelectPolicy: &selectPolicy,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledObject.advanced.horizontalPodAutoscalerConfig.behavior.scaleDown.policies[0].type",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2beta2.HorizontalPodAutoscalerBehavior{
												ScaleDown: &autoscalingv2beta2.HPAScalingRules{
													Policies: []autoscalingv2beta2.HPAScalingPolicy{
														{Type: "test"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledObject.advanced.horizontalPodAutoscalerConfig.behavior.scaleDown.policies[0].periodSeconds",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2beta2.HorizontalPodAutoscalerBehavior{
												ScaleDown: &autoscalingv2beta2.HPAScalingRules{
													Policies: []autoscalingv2beta2.HPAScalingPolicy{
														{
															Type:          "Pods",
															PeriodSeconds: -1,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledJob.restartPolicy",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledJob: &KedaScaledJob{RestartPolicy: &restartPolicy},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledJob.pollingInterval",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledJob: &KedaScaledJob{PollingInterval: &pollingInterval},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledJob.successfulJobsHistoryLimit",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledJob: &KedaScaledJob{SuccessfulJobsHistoryLimit: &successfulBuildsHistoryLimit},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledJob.failedBuildsHistoryLimit",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledJob: &KedaScaledJob{FailedJobsHistoryLimit: &failedBuildsHistoryLimit},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaledJob.maxReplicaCount",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledJob: &KedaScaledJob{MaxReplicaCount: &maxReplicasNegative},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaleJob.scalingStrategy.strategy",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledJob: &KedaScaledJob{
									ScalingStrategy: kedav1alpha1.ScalingStrategy{
										Strategy: "test",
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaleJob.scalingStrategy.customScalingQueueLengthDeduction",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledJob: &KedaScaledJob{
									ScalingStrategy: kedav1alpha1.ScalingStrategy{
										Strategy: "custom",
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaleJob.scalingStrategy.customScalingRunningJobPercentage",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledJob: &KedaScaledJob{
									ScalingStrategy: kedav1alpha1.ScalingStrategy{
										Strategy:                          "custom",
										CustomScalingQueueLengthDeduction: &maxReplicas,
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaleJob.scalingStrategy.customScalingQueueLengthDeduction",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledJob: &KedaScaledJob{
									ScalingStrategy: kedav1alpha1.ScalingStrategy{
										CustomScalingQueueLengthDeduction: &maxReplicasNegative,
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.scaleJob.scalingStrategy.customScalingRunningJobPercentage",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledJob: &KedaScaledJob{
									ScalingStrategy: kedav1alpha1.ScalingStrategy{
										CustomScalingRunningJobPercentage: "%t",
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.inputs[0].name",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Inputs: []*DaprIO{
							{Name: ""},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.inputs[0].component",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Inputs: []*DaprIO{
							{Name: "test"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.inputs[0].component",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Inputs: []*DaprIO{
							{
								Name:      "test",
								Component: "test",
							},
						},
						Pubsub: map[string]*componentsv1alpha1.ComponentSpec{
							"test": {Type: "pubsub.kafka"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "function.spec.serving.outputs[0].name",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Outputs: []*DaprIO{
							{Name: ""},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.outputs[0].component",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Outputs: []*DaprIO{
							{Name: "test"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.outputs[0].component",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Outputs: []*DaprIO{
							{
								Name:      "test",
								Component: "test",
							},
						},
						Pubsub: map[string]*componentsv1alpha1.ComponentSpec{
							"test": {Type: "pubsub.kafka"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "function.spec.serving.pubsub.key",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Bindings: map[string]*componentsv1alpha1.ComponentSpec{
							"test": {},
						},
						Pubsub: map[string]*componentsv1alpha1.ComponentSpec{
							"test": {Type: "pubsub.kafka"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.pubsub.test.type",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Pubsub: map[string]*componentsv1alpha1.ComponentSpec{
							"test": {Type: ""},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.pubsub.test.type",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Pubsub: map[string]*componentsv1alpha1.ComponentSpec{
							"test": {Type: "pubsub.test"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.bindings.key",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Bindings: map[string]*componentsv1alpha1.ComponentSpec{
							"test": {},
						},
						Pubsub: map[string]*componentsv1alpha1.ComponentSpec{
							"test": {Type: "pubsub.kafka"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.bindings.test.type",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Bindings: map[string]*componentsv1alpha1.ComponentSpec{
							"test": {Type: ""},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.bindings.test.type",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Bindings: map[string]*componentsv1alpha1.ComponentSpec{
							"test": {Type: "pubsub.test"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.triggers.[0].type",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Triggers: []Triggers{
							{
								ScaleTriggers: kedav1alpha1.ScaleTriggers{Type: ""},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.triggers.[0].metadata",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Triggers: []Triggers{
							{
								ScaleTriggers: kedav1alpha1.ScaleTriggers{Type: "activemq"},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.triggers.[0].authenticationRef.kind",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Triggers: []Triggers{
							{
								ScaleTriggers: kedav1alpha1.ScaleTriggers{
									Type:              "activemq",
									Metadata:          map[string]string{"key": "value"},
									AuthenticationRef: &kedav1alpha1.ScaledObjectAuthRef{Kind: "test"},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.triggers.[0].fallbackReplicas",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Triggers: []Triggers{
							{
								ScaleTriggers: kedav1alpha1.ScaleTriggers{
									Type:             "activemq",
									Metadata:         map[string]string{"key": "value"},
									FallbackReplicas: &minReplicasNegative,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.triggers.[0].targetKind",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Runtime: Runtime("knative"),
						Triggers: []Triggers{
							{
								ScaleTriggers: kedav1alpha1.ScaleTriggers{
									Type:     "activemq",
									Metadata: map[string]string{"key": "value"},
								},
								TargetKind: &scaleTargetKind,
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.Validate()
			if (got != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", got, tt.wantErr)
				return
			}
		})
	}
}
