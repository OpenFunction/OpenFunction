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
	//"fmt"
	//"reflect"
	"testing"
	"time"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
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
	targetPendingRequests := int32(-1)
	pollingInterval := int32(-1)
	cooldownPeriod := int32(-1)
	stabilizationWindowSecondsNegative := int32(-1)
	stabilizationWindowSecondsLimit := int32(3601)
	var selectPolicy autoscalingv2.ScalingPolicySelect = "test"

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
			name: "function.spec.serving.scaleOptions.minReplicas",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Triggers: &Triggers{Http: &HttpTrigger{}},
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
						Triggers: &Triggers{Http: &HttpTrigger{}},
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
						Triggers: &Triggers{Http: &HttpTrigger{}},
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
			name: "function.spec.serving.scaleOptions.keda.ScaledJob and ScaledObject and HTTPScaledObject",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.httpScaledObject.targetPendingRequests",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								HTTPScaledObject: &HTTPScaledObject{
									TargetPendingRequests: &targetPendingRequests,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "function.spec.serving.scaleOptions.keda.httpScaledObject.cooldownPeriod",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								HTTPScaledObject: &HTTPScaledObject{
									CooldownPeriod: &cooldownPeriod,
								},
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
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									PollingInterval: &pollingInterval,
								},
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
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									CooldownPeriod: &cooldownPeriod,
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
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
												ScaleUp: &autoscalingv2.HPAScalingRules{
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
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
												ScaleUp: &autoscalingv2.HPAScalingRules{
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
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
												ScaleUp: &autoscalingv2.HPAScalingRules{
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
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
												ScaleUp: &autoscalingv2.HPAScalingRules{
													Policies: []autoscalingv2.HPAScalingPolicy{
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
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
												ScaleUp: &autoscalingv2.HPAScalingRules{
													Policies: []autoscalingv2.HPAScalingPolicy{
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
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
												ScaleDown: &autoscalingv2.HPAScalingRules{
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
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
												ScaleDown: &autoscalingv2.HPAScalingRules{
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
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
												ScaleDown: &autoscalingv2.HPAScalingRules{
													Policies: []autoscalingv2.HPAScalingPolicy{
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
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{
									Advanced: &kedav1alpha1.AdvancedConfig{
										HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{
											Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
												ScaleDown: &autoscalingv2.HPAScalingRules{
													Policies: []autoscalingv2.HPAScalingPolicy{
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
			// TODO: add validation for this case
			name: "function.spec.serving.triggers.[0].type",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{},
								Triggers: []kedav1alpha1.ScaleTriggers{
									{
										Type: "",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			// TODO: add validation for this case
			name: "function.spec.serving.triggers.[0].metadata",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{},
								Triggers: []kedav1alpha1.ScaleTriggers{
									{
										Type: "activemq",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			// TODO: add validation for this case
			name: "function.spec.serving.triggers.[0].authenticationRef.kind",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								ScaledObject: &KedaScaledObject{},
								Triggers: []kedav1alpha1.ScaleTriggers{
									{
										Type:              "activemq",
										Metadata:          map[string]string{"key": "value"},
										AuthenticationRef: &kedav1alpha1.ScaledObjectAuthRef{Kind: "test"},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "function.spec.serving.ScaledObject.fallback.replicas",
			r: Function{
				Spec: FunctionSpec{
					Image:            "test",
					ImageCredentials: &v1.LocalObjectReference{Name: "secret"},
					Serving: &ServingImpl{
						Triggers: &Triggers{Http: &HttpTrigger{}},
						ScaleOptions: &ScaleOptions{
							Keda: &KedaScaleOptions{
								Triggers: []kedav1alpha1.ScaleTriggers{
									{
										Type:     "activemq",
										Metadata: map[string]string{"key": "value"},
									},
								},
								ScaledObject: &KedaScaledObject{
									Fallback: &kedav1alpha1.Fallback{
										Replicas: minReplicasNegative,
									},
								},
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
