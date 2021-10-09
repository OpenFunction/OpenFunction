package shipwright

import (
	"context"
	"fmt"
	"strings"

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

	envVars  = "ENV_VARS"
	appImage = "APP_IMAGE"
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

	log := r.log.WithName("Start")

	// Clean up redundant builds and buildruns caused by the `Start` function failed.
	if err := r.clean(builder); err != nil {
		log.Error(err, "Clean failed", "name", builder.Name, "namespace", builder.Namespace)
		return err
	}

	shipwrightBuild := r.createShipwrightBuild(builder)
	if err := ctrl.SetControllerReference(builder, shipwrightBuild, r.scheme); err != nil {
		log.Error(err, "Failed to SetControllerReference", "name", builder.Name, "namespace", builder.Namespace)
		return err
	}

	if err := r.Create(r.ctx, shipwrightBuild); err != nil {
		log.Error(err, "Failed to create shipwright Build", "name", shipwrightBuild.Name, "namespace", shipwrightBuild.Namespace)
		return err
	}

	log.V(1).Info("Shipwright Build created", "namespace", shipwrightBuild.Namespace, "name", shipwrightBuild.Name)

	shipwrightBuildRun := r.createShipwrightBuildRun(builder, shipwrightBuild.Name)
	if err := ctrl.SetControllerReference(builder, shipwrightBuildRun, r.scheme); err != nil {
		log.Error(err, "Failed to SetControllerReference", "name", builder.Name, "namespace", builder.Namespace)
		return err
	}

	if err := r.Create(r.ctx, shipwrightBuildRun); err != nil {
		log.Error(err, "Failed to create shipwright BuildRun", "name", shipwrightBuildRun.Name, "namespace", shipwrightBuildRun.Namespace)
		return err
	}

	log.V(1).Info("Shipwright BuildRun created", "namespace", shipwrightBuildRun.Namespace, "name", shipwrightBuildRun.Name)

	builder.Status.ResourceRef = map[string]string{
		shipwrightBuildName:    shipwrightBuild.Name,
		shipwrightBuildRunName: shipwrightBuildRun.Name,
	}

	return nil
}

func (r *builderRun) Result(builder *openfunction.Builder) (string, error) {
	log := r.log.WithName("Result")

	shipwrightBuild := &shipwrightv1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getName(builder, shipwrightBuildName),
			Namespace: builder.Namespace,
		},
	}
	if err := r.Get(r.ctx, client.ObjectKeyFromObject(shipwrightBuild), shipwrightBuild); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get build", "name", shipwrightBuild.Name, "namespace", shipwrightBuild.Namespace)
		return "", util.IgnoreNotFound(err)
	}

	if shipwrightBuild.Status.Registered != corev1.ConditionTrue {
		return string(shipwrightBuild.Status.Reason), nil
	}

	shipwrightBuildRun := &shipwrightv1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getName(builder, shipwrightBuildRunName),
			Namespace: builder.Namespace,
		},
	}
	if err := r.Get(r.ctx, client.ObjectKeyFromObject(shipwrightBuildRun), shipwrightBuildRun); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get buildRun", "name", shipwrightBuildRun.Name, "namespace", shipwrightBuildRun.Namespace)
		return "", util.IgnoreNotFound(err)
	}

	if shipwrightBuildRun.Status.CompletionTime == nil {
		return "", nil
	}

	if shipwrightBuildRun.Status.IsFailed(shipwrightv1alpha1.Succeeded) {
		return openfunction.Failed, nil
	} else {
		return openfunction.Succeeded, nil
	}
}

// Clean up redundant builds and buildruns caused by the `Start` function failed.
func (r *builderRun) clean(builder *openfunction.Builder) error {
	log := r.log.WithName("Clean")

	builds := &shipwrightv1alpha1.BuildList{}
	if err := r.List(r.ctx, builds, client.InNamespace(builder.Namespace), client.MatchingLabels{builderLabel: builder.Name}); err != nil {
		return err
	}

	for _, item := range builds.Items {
		if strings.HasPrefix(item.Name, builder.Name) {
			if err := r.Delete(context.Background(), &item); util.IgnoreNotFound(err) != nil {
				return err
			}
			log.V(1).Info("Delete shipwright Build", "namespace", item.Namespace, "name", item.Name)
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
			log.V(1).Info("Delete shipwright BuildRun", "namespace", item.Namespace, "name", item.Name)
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

	for k, v := range builder.Spec.Params {
		shipwrightBuild.Spec.ParamValues = append(shipwrightBuild.Spec.ParamValues, shipwrightv1alpha1.ParamValue{
			Name:  k,
			Value: v,
		})
	}

	env := ""
	for k, v := range builder.Spec.Env {
		env = fmt.Sprintf("%s%s=%s,", env, k, v)
	}
	if builder.Spec.Port != nil {
		env = fmt.Sprintf("%sPORT=%d", env, *builder.Spec.Port)
	}

	shipwrightBuild.Spec.ParamValues = append(shipwrightBuild.Spec.ParamValues, shipwrightv1alpha1.ParamValue{
		Name:  envVars,
		Value: env,
	})
	shipwrightBuild.Spec.ParamValues = append(shipwrightBuild.Spec.ParamValues, shipwrightv1alpha1.ParamValue{
		Name:  appImage,
		Value: builder.Spec.Image,
	})

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

		shipwrightBuild.Spec.Timeout = builder.Spec.Shipwright.Timeout
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

func getName(builder *openfunction.Builder, key string) string {
	if builder.Status.ResourceRef == nil {
		return ""
	}

	return builder.Status.ResourceRef[key]
}
