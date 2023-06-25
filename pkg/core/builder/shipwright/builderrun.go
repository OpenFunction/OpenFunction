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

package shipwright

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	shipwrightv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openfunction "github.com/openfunction/apis/core/v1beta2"
	"github.com/openfunction/pkg/core"
	"github.com/openfunction/pkg/util"
)

const (
	shipwrightBuildName    = "shipwright.io/build"
	shipwrightBuildRunName = "shipwright.io/buildRun"
	builderLabel           = "openfunction.io/builder"
	defaultStrategy        = "openfunction"
	shipwrightGenerateSA   = true

	waitBuildInterval = time.Second
	waitBuildTimeout  = time.Minute

	envVars = "ENV_VARS"
)

type builderRun struct {
	client.Client
	ctx    context.Context
	log    logr.Logger
	scheme *runtime.Scheme
}

func NewBuildRun(ctx context.Context, c client.Client, scheme *runtime.Scheme, log logr.Logger) core.BuilderRun {

	return &builderRun{
		c,
		ctx,
		log.WithName("Shipwright"),
		scheme,
	}
}

func Registry(rm meta.RESTMapper) []client.Object {
	var objs = []client.Object{}

	if _, err := rm.ResourcesFor(schema.GroupVersionResource{Group: "shipwright.io", Version: "v1alpha1", Resource: "buildruns"}); err == nil {
		objs = append(objs, &shipwrightv1alpha1.BuildRun{})
	}

	if _, err := rm.ResourcesFor(schema.GroupVersionResource{Group: "shipwright.io", Version: "v1alpha1", Resource: "builds"}); err == nil {
		objs = append(objs, &shipwrightv1alpha1.Build{})
	}

	return objs
}

func (r *builderRun) Start(builder *openfunction.Builder) error {

	log := r.log.WithName("Start").
		WithValues("Builder", fmt.Sprintf("%s/%s", builder.Namespace, builder.Name))

	// Clean up redundant builds and buildruns caused by the `Start` function failed.
	if err := r.Clean(builder); err != nil {
		log.Error(err, "Clean failed")
		return err
	}

	shipwrightBuild := r.createShipwrightBuild(builder)
	if err := ctrl.SetControllerReference(builder, shipwrightBuild, r.scheme); err != nil {
		log.Error(err, "Failed to SetControllerReference for Build", "Build", shipwrightBuild.Name)
		return err
	}

	if err := r.Create(r.ctx, shipwrightBuild); err != nil {
		log.Error(err, "Failed to create Build", "Build", shipwrightBuild.Name)
		return err
	}

	if err := r.waitForBuildReady(shipwrightBuild); err != nil {
		log.Error(err, "Failed to wait for the Build to be ready", "Build", shipwrightBuild.Name)
		return err
	}

	log.V(1).Info("Build created", "Build", shipwrightBuild.Name)

	shipwrightBuildRun := r.createShipwrightBuildRun(builder, shipwrightBuild.Name)
	if err := ctrl.SetControllerReference(builder, shipwrightBuildRun, r.scheme); err != nil {
		log.Error(err, "Failed to SetControllerReference for BuildRun", "BuildRun", shipwrightBuildRun.Name)
		return err
	}

	if err := r.Create(r.ctx, shipwrightBuildRun); err != nil {
		log.Error(err, "Failed to create BuildRun", "BuildRun", shipwrightBuildRun.Name)
		return err
	}

	log.V(1).Info("BuildRun created", "BuildRun", shipwrightBuildRun.Name)

	builder.Status.ResourceRef = map[string]string{
		shipwrightBuildName:    shipwrightBuild.Name,
		shipwrightBuildRunName: shipwrightBuildRun.Name,
	}

	return nil
}

func (r *builderRun) Result(builder *openfunction.Builder) (string, string, string, error) {
	log := r.log.WithName("Result").
		WithValues("Builder", fmt.Sprintf("%s/%s", builder.Namespace, builder.Name))

	shipwrightBuildName := getName(builder, shipwrightBuildName)
	if shipwrightBuildName == "" {
		return "", "", "", nil
	}

	shipwrightBuild := &shipwrightv1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      shipwrightBuildName,
			Namespace: builder.Namespace,
		},
	}
	if err := r.Get(r.ctx, client.ObjectKeyFromObject(shipwrightBuild), shipwrightBuild); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get Build", "Build", shipwrightBuild.Name)
		return "", "", "", util.IgnoreNotFound(err)
	}

	if shipwrightBuild.Status.Registered == nil {
		return "", "", "", nil
	}

	if shipwrightBuild.Status.Registered == shipwrightv1alpha1.ConditionStatusPtr(corev1.ConditionFalse) {
		return openfunction.Failed, string(*shipwrightBuild.Status.Reason), *shipwrightBuild.Status.Message, nil
	} else if shipwrightBuild.Status.Registered == shipwrightv1alpha1.ConditionStatusPtr(corev1.ConditionUnknown) {
		return "", "", "", nil
	}

	shipwrightBuildRunName := getName(builder, shipwrightBuildRunName)
	if shipwrightBuildRunName == "" {
		return "", "", "", nil
	}

	shipwrightBuildRun := &shipwrightv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      shipwrightBuildRunName,
			Namespace: builder.Namespace,
		},
	}
	if err := r.Get(r.ctx, client.ObjectKeyFromObject(shipwrightBuildRun), shipwrightBuildRun); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get BuildRun", "BuildRun", shipwrightBuildRun.Name)
		return "", "", "", util.IgnoreNotFound(err)
	}

	if shipwrightBuildRun.Status.CompletionTime == nil {
		return "", "", "", nil
	}

	for _, c := range shipwrightBuildRun.Status.Conditions {
		if c.Type == shipwrightv1alpha1.Succeeded {
			if c.Status == corev1.ConditionFalse {
				switch c.Reason {
				case "BuildRunTimeout":
					return openfunction.Timeout, c.Reason, c.Message, nil
				case shipwrightv1alpha1.BuildRunStateCancel:
					return openfunction.Canceled, c.Reason, c.Message, nil
				default:
					return openfunction.Failed, c.Reason, c.Message, nil
				}
			} else if c.Status == corev1.ConditionTrue {
				if shipwrightBuildRun.Status.Output != nil {
					builder.Status.Output = &openfunction.BuilderOutput{
						Digest: shipwrightBuildRun.Status.Output.Digest,
						Size:   shipwrightBuildRun.Status.Output.Size,
					}
				}

				builder.Status.Sources = []openfunction.SourceResult{}
				for _, source := range shipwrightBuildRun.Status.Sources {
					sr := openfunction.SourceResult{
						Name: source.Name,
					}

					if source.Git != nil {
						sr.Git = &openfunction.GitSourceResult{
							CommitSha:    source.Git.CommitSha,
							CommitAuthor: source.Git.CommitAuthor,
						}
					}

					if source.Bundle != nil {
						sr.Bundle = &openfunction.BundleSourceResult{
							Digest: source.Bundle.Digest,
						}
					}

					builder.Status.Sources = append(builder.Status.Sources, sr)
				}

				return openfunction.Succeeded, openfunction.Succeeded, openfunction.Succeeded, nil
			} else {
				return "", "", "", nil
			}
		}
	}

	return "", "", "", nil
}

// Clean up redundant builds and buildruns caused by the `Start` function failed.
func (r *builderRun) Clean(builder *openfunction.Builder) error {
	log := r.log.WithName("Clean").
		WithValues("Builder", fmt.Sprintf("%s/%s", builder.Namespace, builder.Name))

	builds := &shipwrightv1alpha1.BuildList{}
	if err := r.List(r.ctx, builds, client.InNamespace(builder.Namespace), client.MatchingLabels{builderLabel: builder.Name}); err != nil {
		return err
	}

	for _, item := range builds.Items {
		if strings.HasPrefix(item.Name, builder.Name) {
			if err := r.Delete(context.Background(), &item); util.IgnoreNotFound(err) != nil {
				return err
			}
			log.V(1).Info("Delete Build", "Build", item.Name)
		}
	}

	buildRuns := &shipwrightv1alpha1.BuildRunList{}
	if err := r.List(r.ctx, buildRuns, client.InNamespace(builder.Namespace), client.MatchingLabels{builderLabel: builder.Name}); err != nil {
		return err
	}

	for _, item := range buildRuns.Items {
		if strings.HasPrefix(item.Name, builder.Name) {
			if err := r.Delete(context.Background(), &item); util.IgnoreNotFound(err) != nil {
				return err
			}
			log.V(1).Info("Delete BuildRun", "BuildRun", item.Name)
		}
	}

	return nil
}

// Cancel the running builder.
func (r *builderRun) Cancel(builder *openfunction.Builder) error {
	log := r.log.WithName("Cancel").
		WithValues("Builder", fmt.Sprintf("%s/%s", builder.Namespace, builder.Name))

	shipwrightBuildRunName := getName(builder, shipwrightBuildRunName)
	if shipwrightBuildRunName == "" {
		return nil
	}

	shipwrightBuildRun := &shipwrightv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      shipwrightBuildRunName,
			Namespace: builder.Namespace,
		},
	}

	if err := r.Get(r.ctx, client.ObjectKeyFromObject(shipwrightBuildRun), shipwrightBuildRun); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get BuildRun", "BuildRun", shipwrightBuildRun.Name)
		return util.IgnoreNotFound(err)
	}

	if shipwrightBuildRun.Spec.State != shipwrightv1alpha1.BuildRunRequestedStatePtr(shipwrightv1alpha1.BuildRunStateCancel) {
		shipwrightBuildRun.Spec.State = shipwrightv1alpha1.BuildRunRequestedStatePtr(shipwrightv1alpha1.BuildRunStateCancel)
		if err := r.Update(r.ctx, shipwrightBuildRun); util.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to cancel BuildRun", "BuildRun", shipwrightBuildRun.Name)
			return err
		}
	}

	return nil
}

func (r *builderRun) createShipwrightBuild(builder *openfunction.Builder) *shipwrightv1alpha1.Build {
	shipwrightBuild := &shipwrightv1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-build-", builder.Name),
			Namespace:    builder.Namespace,
			Labels: map[string]string{
				builderLabel: builder.Name,
			},
		},
		Spec: shipwrightv1alpha1.BuildSpec{
			Source: shipwrightv1alpha1.Source{
				Revision:    builder.Spec.SrcRepo.Revision,
				ContextDir:  builder.Spec.SrcRepo.SourceSubPath,
				Credentials: builder.Spec.SrcRepo.Credentials,
			},
			Dockerfile: builder.Spec.Dockerfile,
			Output: shipwrightv1alpha1.Image{
				Image:       builder.Spec.Image,
				Credentials: builder.Spec.ImageCredentials,
			},
		},
	}

	if builder.Spec.Builder != nil {
		shipwrightBuild.Spec.Builder = &shipwrightv1alpha1.Image{
			Image:       *builder.Spec.Builder,
			Credentials: builder.Spec.BuilderCredentials,
		}
	}

	switch {
	case builder.Spec.SrcRepo.BundleContainer != nil:
		shipwrightBuild.Spec.Source.BundleContainer = &shipwrightv1alpha1.BundleContainer{
			Image: builder.Spec.SrcRepo.BundleContainer.Image,
		}
	case builder.Spec.SrcRepo.Url != "":
		url := builder.Spec.SrcRepo.Url
		shipwrightBuild.Spec.Source.URL = &url
	}

	if builder.Spec.Timeout != nil {
		shipwrightBuild.Spec.Timeout = &metav1.Duration{
			Duration: builder.Spec.Timeout.Duration - time.Since(builder.CreationTimestamp.Time),
		}
	}

	if builder.Spec.Shipwright != nil {
		appendParams(shipwrightBuild, builder)
	}

	for k, v := range builder.Labels {
		shipwrightBuild.Labels[k] = v
	}

	env := ""
	for k, v := range builder.Spec.Env {
		env = fmt.Sprintf("%s%s=%s#", env, k, v)
	}

	if len(env) > 0 {
		shipwrightBuild.Spec.ParamValues = append(shipwrightBuild.Spec.ParamValues, shipwrightv1alpha1.ParamValue{
			Name: envVars,
			SingleValue: &shipwrightv1alpha1.SingleValue{
				Value: &env,
			},
		})
	}

	if builder.Spec.Shipwright == nil || builder.Spec.Shipwright.Strategy == nil {
		kind := shipwrightv1alpha1.ClusterBuildStrategyKind
		shipwrightBuild.Spec.Strategy = shipwrightv1alpha1.Strategy{
			Name: defaultStrategy,
			Kind: &kind,
		}
	}

	if builder.Spec.Shipwright != nil {
		if builder.Spec.Shipwright.Strategy != nil {
			shipwrightBuild.Spec.Strategy = shipwrightv1alpha1.Strategy{
				Name: builder.Spec.Shipwright.Strategy.Name,
			}

			if builder.Spec.Shipwright.Strategy.Kind != nil {
				kind := shipwrightv1alpha1.BuildStrategyKind(*builder.Spec.Shipwright.Strategy.Kind)
				shipwrightBuild.Spec.Strategy.Kind = &kind
			}
		}
	}

	shipwrightBuild.SetOwnerReferences(nil)
	return shipwrightBuild
}

func appendParams(shipwrightBuild *shipwrightv1alpha1.Build, b *openfunction.Builder) {
	for _, p := range b.Spec.Shipwright.Params {
		if p == nil {
			continue
		}

		param := shipwrightv1alpha1.ParamValue{
			SingleValue: nil,
			Name:        p.Name,
		}

		if p.SingleValue != nil {
			param.SingleValue = &shipwrightv1alpha1.SingleValue{
				Value:          p.Value,
				ConfigMapValue: p.ConfigMapValue,
				SecretValue:    p.SecretValue,
			}
		}

		for _, v := range p.Values {
			param.Values = append(param.Values, shipwrightv1alpha1.SingleValue{
				Value:          v.Value,
				ConfigMapValue: v.ConfigMapValue,
				SecretValue:    v.SecretValue,
			})
		}

		shipwrightBuild.Spec.ParamValues = append(shipwrightBuild.Spec.ParamValues, param)
	}
}

func (r *builderRun) createShipwrightBuildRun(builder *openfunction.Builder, name string) *shipwrightv1alpha1.BuildRun {
	generateSA := shipwrightGenerateSA
	shipwrightBuildRun := &shipwrightv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-buildrun-", builder.Name),
			Namespace:    builder.Namespace,
			Labels: map[string]string{
				builderLabel: builder.Name,
			},
		},
		Spec: shipwrightv1alpha1.BuildRunSpec{
			BuildRef: &shipwrightv1alpha1.BuildRef{
				Name: name,
			},
			// Generate a temporal service account for each buildrun to avoid build failed when previous used secret was deleted
			ServiceAccount: &shipwrightv1alpha1.ServiceAccount{
				Generate: &generateSA,
			},
		},
	}
	for k, v := range builder.Labels {
		shipwrightBuildRun.Labels[k] = v
	}

	shipwrightBuildRun.SetOwnerReferences(nil)
	return shipwrightBuildRun
}

func (r *builderRun) waitForBuildReady(build *shipwrightv1alpha1.Build) error {

	ticker := time.NewTicker(waitBuildInterval)
	timer := time.NewTimer(waitBuildTimeout)

	for {
		select {
		case <-ticker.C:
			b := &shipwrightv1alpha1.Build{}
			if err := r.Get(r.ctx, client.ObjectKeyFromObject(build), b); err != nil {
				continue
			}

			if b.Status.Registered == shipwrightv1alpha1.ConditionStatusPtr(corev1.ConditionUnknown) {
				continue
			} else {
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("wait for build ready timeout")
		}
	}
}

func getName(builder *openfunction.Builder, key string) string {
	if builder.Status.ResourceRef == nil {
		return ""
	}

	return builder.Status.ResourceRef[key]
}
