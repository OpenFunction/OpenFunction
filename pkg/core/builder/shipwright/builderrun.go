package shipwright

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	shipwrightv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	openfunction "github.com/openfunction/apis/core/v1alpha2"
	"github.com/openfunction/pkg/core"
	"github.com/openfunction/pkg/util"
)

const (
	shipwrightBuildName    = "shipwright.io/build"
	shipwrightBuildRunName = "shipwright.io/buildRun"
	builderLabel           = "openfunction.io/builder"
	defaultStrategy        = "openfunction"

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

func Registry() []client.Object {

	return []client.Object{&shipwrightv1alpha1.Build{}, &shipwrightv1alpha1.BuildRun{}}
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

func (r *builderRun) Result(builder *openfunction.Builder) (string, string, error) {
	log := r.log.WithName("Result").
		WithValues("Builder", fmt.Sprintf("%s/%s", builder.Namespace, builder.Name))

	shipwrightBuild := &shipwrightv1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getName(builder, shipwrightBuildName),
			Namespace: builder.Namespace,
		},
	}
	if err := r.Get(r.ctx, client.ObjectKeyFromObject(shipwrightBuild), shipwrightBuild); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get Build", "Build", shipwrightBuild.Name)
		return "", "", util.IgnoreNotFound(err)
	}

	if shipwrightBuild.Status.Registered == corev1.ConditionFalse {
		return openfunction.Failed, string(shipwrightBuild.Status.Reason), nil
	} else if shipwrightBuild.Status.Registered == corev1.ConditionUnknown {
		return "", "", nil
	}

	shipwrightBuildRun := &shipwrightv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getName(builder, shipwrightBuildRunName),
			Namespace: builder.Namespace,
		},
	}
	if err := r.Get(r.ctx, client.ObjectKeyFromObject(shipwrightBuildRun), shipwrightBuildRun); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get BuildRun", "BuildRun", shipwrightBuildRun.Name)
		return "", "", util.IgnoreNotFound(err)
	}

	if shipwrightBuildRun.Status.CompletionTime == nil {
		return "", "", nil
	}

	for _, c := range shipwrightBuildRun.Status.Conditions {
		if c.Type == shipwrightv1alpha1.Succeeded {
			if c.Status == corev1.ConditionFalse {
				switch c.Reason {
				case "BuildRunTimeout":
					return openfunction.Timeout, c.Reason, nil
				case shipwrightv1alpha1.BuildRunStateCancel:
					return openfunction.Canceled, c.Reason, nil
				default:
					return openfunction.Failed, c.Reason, nil
				}
			} else if c.Status == corev1.ConditionTrue {
				if shipwrightBuildRun.Status.Output != nil {
					builder.Status.Output = &openfunction.Output{
						Digest: shipwrightBuildRun.Status.Output.Digest,
						Size:   shipwrightBuildRun.Status.Output.Size,
					}
				}

				return openfunction.Succeeded, openfunction.Succeeded, nil
			} else {
				return "", "", nil
			}
		}
	}

	return "", "", nil
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

	shipwrightBuildRun := &shipwrightv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getName(builder, shipwrightBuildRunName),
			Namespace: builder.Namespace,
		},
	}

	if err := r.Get(r.ctx, client.ObjectKeyFromObject(shipwrightBuildRun), shipwrightBuildRun); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get BuildRun", "BuildRun", shipwrightBuildRun.Name)
		return util.IgnoreNotFound(err)
	}

	if shipwrightBuildRun.Spec.State != shipwrightv1alpha1.BuildRunStateCancel {
		shipwrightBuildRun.Spec.State = shipwrightv1alpha1.BuildRunStateCancel
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
				URL:         builder.Spec.SrcRepo.Url,
				Revision:    builder.Spec.SrcRepo.Revision,
				ContextDir:  builder.Spec.SrcRepo.SourceSubPath,
				Credentials: builder.Spec.SrcRepo.Credentials,
			},
			Dockerfile: builder.Spec.Dockerfile,
			Output: shipwrightv1alpha1.Image{
				Image:       builder.Spec.Image,
				Credentials: builder.Spec.ImageCredentials,
			},
			Builder: &shipwrightv1alpha1.Image{
				Image:       *builder.Spec.Builder,
				Credentials: builder.Spec.BuilderCredentials,
			},
		},
	}

	if builder.Spec.Timeout != nil {
		shipwrightBuild.Spec.Timeout = &metav1.Duration{
			Duration: builder.Spec.Timeout.Duration - time.Since(builder.CreationTimestamp.Time),
		}
	}

	for k, v := range builder.Spec.Params {
		shipwrightBuild.Spec.ParamValues = append(shipwrightBuild.Spec.ParamValues, shipwrightv1alpha1.ParamValue{
			Name:  k,
			Value: v,
		})
	}

	env := ""
	for k, v := range builder.Spec.Env {
		env = fmt.Sprintf("%s%s=%s#", env, k, v)
	}
	if builder.Spec.Port != nil {
		env = fmt.Sprintf("%sPORT=%d", env, *builder.Spec.Port)
	}

	if len(env) > 0 {
		shipwrightBuild.Spec.ParamValues = append(shipwrightBuild.Spec.ParamValues, shipwrightv1alpha1.ParamValue{
			Name:  envVars,
			Value: env,
		})
	}

	if builder.Spec.Shipwright == nil || builder.Spec.Shipwright.Strategy == nil {
		kind := shipwrightv1alpha1.ClusterBuildStrategyKind
		shipwrightBuild.Spec.Strategy = &shipwrightv1alpha1.Strategy{
			Name: defaultStrategy,
			Kind: &kind,
		}
	}

	if builder.Spec.Shipwright != nil {
		if builder.Spec.Shipwright.Strategy != nil {
			shipwrightBuild.Spec.Strategy = &shipwrightv1alpha1.Strategy{
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

func (r *builderRun) createShipwrightBuildRun(builder *openfunction.Builder, name string) *shipwrightv1alpha1.BuildRun {

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
		},
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

			if b.Status.Registered == corev1.ConditionUnknown {
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
