package controllers

import (
	"context"
	goerrors "errors"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/openfunction/pkg/apis/v1alpha1"
	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelineres "github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	buildpacksSa          = "buildpacks-service-account"
	buildpacksAppImage    = "buildpacks-app-image"
	builderImage          = "BUILDER_IMAGE"
	buildpacksCache       = "buildpacks-cache"
	buildpacksPipeline    = "buildpacks-pipeline"
	buildpacksPipelineRun = "buildpacks-pipeline-run"
	buildImage            = "build-image"
	buildpackSourcePvc    = "buildpacks-source-pvc"
	platformEnv           = "platform-env"
	cache                 = "CACHE"
	image                 = "image"
	registryUrlKey        = "tekton.dev/docker-0"
	registryUrl           = "https://index.docker.io/v1/"
	sourceSubpath         = "SOURCE_SUBPATH"
	subDirectory          = "subdirectory"
	taskbuild             = "buildpacks"
	taskGitClone          = "git-clone"
	url                   = "url"
	workspaceShare        = "shared-workspace"
	workspaceOutput       = "output"
	workspaceSource       = "source"
	functionTarget        = "GOOGLE_FUNCTION_TARGET"
	functionSignatureType = "GOOGLE_FUNCTION_SIGNATURE_TYPE"
	platformDir           = "platform-dir"
)

var (
	taskTmplDict = map[string]string{
		taskbuild:    tmplBuild,
		taskGitClone: tmplGitClone,
	}
)

func UnmarshalTask(task string) (*pipeline.Task, error) {
	var t pipeline.Task
	return &t, yaml.Unmarshal([]byte(task), &t)
}

func (r *FunctionReconciler) mutateTask(task *pipeline.Task, owner *v1alpha1.Function, name string) controllerutil.MutateFn {
	return func() error {
		tmpl := ""
		ok := false
		if tmpl, ok = taskTmplDict[name]; !ok {
			err := goerrors.New("Doesn't exist")
			return err
		}

		expected, err := UnmarshalTask(tmpl)
		if err != nil {
			return err
		}

		for i, _ := range expected.Spec.Steps {
			expected.Spec.Steps[i].ImagePullPolicy = v1.PullIfNotPresent
		}

		expected.Spec.DeepCopyInto(&task.Spec)
		task.SetOwnerReferences(nil)
		return ctrl.SetControllerReference(owner, task, r.Scheme)
		//		if len(task.Spec.Steps) == 0 {
		//			task.Spec.Steps = make([]pipeline.Step, len(expected.Spec.Steps))
		//			copy(task.Spec.Steps, expected.Spec.Steps)
		//			task.SetOwnerReferences(nil)
		//			return ctrl.SetControllerReference(owner, task, r.Scheme)
		//		}
		//		return nil
	}
}

func (r *FunctionReconciler) CreateOrUpdateTask(owner *v1alpha1.Function, name string) error {
	log := r.Log.WithName("CreateOrUpdateTask")
	ctx := context.Background()

	task := pipeline.Task{}
	task.Name = fmt.Sprintf("%s-%s", owner.Name, name)
	task.Namespace = owner.Namespace
	if result, err := controllerutil.CreateOrUpdate(ctx, r.Client, &task, r.mutateTask(&task, owner, name)); err != nil {
		log.Error(err, "Failed to CreateOrUpdate Task", "result", result)
		return err
	}
	return nil
}

func (r *FunctionReconciler) mutateConfigMap(cm *v1.ConfigMap, owner *v1alpha1.Function) controllerutil.MutateFn {
	return func() error {
		expected := v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      cm.Name,
				Namespace: owner.Namespace,
			},
			Data: map[string]string{
				functionSignatureType: owner.Spec.FuncType,
				functionTarget:        owner.Spec.FuncName,
			},
		}

		if len(cm.Data) == 0 {
			cm.Data = map[string]string{
				functionSignatureType: expected.Data[functionSignatureType],
				functionTarget:        expected.Data[functionTarget],
			}
			expected.DeepCopyInto(cm)
			cm.SetOwnerReferences(nil)
			return ctrl.SetControllerReference(owner, cm, r.Scheme)
		}
		return nil
	}
}

func (r *FunctionReconciler) CreateOrUpdateConfigMap(owner *v1alpha1.Function) error {
	log := r.Log.WithName("CreateOrUpdateConfigMap")
	ctx := context.Background()

	cm := v1.ConfigMap{}
	cm.Name = fmt.Sprintf("%s-%s", owner.Name, platformEnv)
	cm.Namespace = owner.Namespace
	if result, err := controllerutil.CreateOrUpdate(ctx, r.Client, &cm, r.mutateConfigMap(&cm, owner)); err != nil {
		log.Error(err, "Failed to CreateOrUpdate ConfigMap", "result", result)
		return err
	}
	return nil
}

func (r *FunctionReconciler) mutatePVC(pvc *v1.PersistentVolumeClaim, owner *v1alpha1.Function) controllerutil.MutateFn {
	return func() error {
		expected := v1.PersistentVolumeClaim{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "PersistentVolumeClaim",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvc.Name,
				Namespace: owner.Namespace,
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						"storage": resource.MustParse("500Mi"),
					},
				},
			},
		}

		if pvc.Spec.AccessModes == nil {
			pvc.Spec.AccessModes = expected.Spec.AccessModes
		}
		expected.Spec.Resources.Requests.DeepCopyInto(&pvc.Spec.Resources.Requests)
		pvc.SetOwnerReferences(nil)
		return ctrl.SetControllerReference(owner, pvc, r.Scheme)
	}
}

func (r *FunctionReconciler) CreateOrUpdateBuildpackPVCs(owner *v1alpha1.Function) error {
	log := r.Log.WithName("CreateBuildpackPVCs")
	ctx := context.Background()

	pvcs := []string{fmt.Sprintf("%s-%s", owner.Name, buildpackSourcePvc)}
	for _, v := range pvcs {
		pvc := v1.PersistentVolumeClaim{}
		pvc.Name = v
		pvc.Namespace = owner.Namespace
		if result, err := controllerutil.CreateOrUpdate(ctx, r.Client, &pvc, r.mutatePVC(&pvc, owner)); err != nil {
			log.Error(err, "Failed to CreateOrUpdate PersistentVolumeClaim", "result", result)
			return err
		}
	}
	return nil
}

func (r *FunctionReconciler) mutateRegistryAuth(ctx context.Context, sa *v1.ServiceAccount, owner *v1alpha1.Function) controllerutil.MutateFn {
	return func() error {
		s := v1.Secret{}
		if err := r.Client.Get(ctx, types.NamespacedName{Namespace: owner.Namespace, Name: owner.Spec.Registry.Account.Name}, &s); err != nil {
			return err
		}
		var url string
		if owner.Spec.Registry.Url == nil {
			url = registryUrl
		} else {
			url = *owner.Spec.Registry.Url
		}
		s.Annotations[registryUrlKey] = url
		s.Type = "kubernetes.io/basic-auth"

		if err := r.Client.Update(ctx, &s); err != nil {
			return err
		}

		expected := v1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", owner.Name, buildpacksSa),
				Namespace: owner.Namespace,
			},
			Secrets: []v1.ObjectReference{
				v1.ObjectReference{
					APIVersion: "v1",
					Kind:       "Secret",
					Name:       s.Name,
					Namespace:  owner.Namespace,
				},
			},
		}

		expected.DeepCopyInto(sa)
		sa.SetOwnerReferences(nil)
		return ctrl.SetControllerReference(owner, sa, r.Scheme)
	}
}

func (r *FunctionReconciler) CreateOrUpdateRegistryAuth(owner *v1alpha1.Function) error {
	log := r.Log.WithName("CreateOrUpdateRegistryAuth")
	ctx := context.Background()
	sa := v1.ServiceAccount{}
	sa.Name = fmt.Sprintf("%s-%s", owner.Name, buildpacksSa)
	sa.Namespace = owner.Namespace
	if result, err := controllerutil.CreateOrUpdate(ctx, r.Client, &sa, r.mutateRegistryAuth(ctx, &sa, owner)); err != nil {
		log.Error(err, "Failed to CreateOrUpdate ServiceAccount", "result", result)
		return err
	}
	return nil
}

func (r *FunctionReconciler) mutatePipelineResource(res *pipelineres.PipelineResource, owner *v1alpha1.Function) controllerutil.MutateFn {
	return func() error {
		expected := pipelineres.PipelineResource{
			Spec: pipelineres.PipelineResourceSpec{
				Type: pipelineres.PipelineResourceTypeImage,
				Params: []pipelineres.ResourceParam{
					pipelineres.ResourceParam{
						Name:  url,
						Value: owner.Spec.Image,
					},
				},
			},
		}

		expected.Spec.DeepCopyInto(&res.Spec)
		res.SetOwnerReferences(nil)
		return ctrl.SetControllerReference(owner, res, r.Scheme)
	}
}

func (r *FunctionReconciler) CreateOrUpdatePipelineResource(owner *v1alpha1.Function) error {
	log := r.Log.WithName("CreatePipelineResource")
	ctx := context.Background()

	res := pipelineres.PipelineResource{}
	res.Name = fmt.Sprintf("%s-%s", owner.Name, buildpacksAppImage)
	res.Namespace = owner.Namespace
	if result, err := controllerutil.CreateOrUpdate(ctx, r.Client, &res, r.mutatePipelineResource(&res, owner)); err != nil {
		log.Error(err, "Failed to CreateOrUpdate PipelineResource", "result", result)
		return err
	}
	return nil
}

func (r *FunctionReconciler) mutatePipeline(p *pipeline.Pipeline, owner *v1alpha1.Function) controllerutil.MutateFn {
	return func() error {
		expected := pipeline.Pipeline{
			Spec: pipeline.PipelineSpec{
				Workspaces: []pipeline.PipelineWorkspaceDeclaration{
					pipeline.PipelineWorkspaceDeclaration{
						Name: workspaceShare,
					},
				},
				Resources: []pipeline.PipelineDeclaredResource{
					pipeline.PipelineDeclaredResource{
						Name: buildImage,
						Type: pipeline.PipelineResourceTypeImage,
					},
				},
			},
		}

		taskFetchSrcName := fmt.Sprintf("%s-%s", owner.Name, taskGitClone)
		taskFetchSrc := pipeline.PipelineTask{
			Name:    taskFetchSrcName,
			TaskRef: &pipeline.TaskRef{Name: taskFetchSrcName},
			Workspaces: []pipeline.WorkspacePipelineTaskBinding{
				pipeline.WorkspacePipelineTaskBinding{
					Name:      workspaceOutput,
					Workspace: workspaceShare,
				},
			},
			Params: []pipeline.Param{
				pipeline.Param{
					Name: url,
					Value: pipeline.ArrayOrString{
						Type:      pipeline.ParamTypeString,
						StringVal: owner.Spec.Source.Url,
					},
				},
			},
		}
		if owner.Spec.Source.DeleteExisting != nil {
			param := pipeline.Param{
				Name: subDirectory,
				Value: pipeline.ArrayOrString{
					Type:      pipeline.ParamTypeString,
					StringVal: *owner.Spec.Source.DeleteExisting,
				},
			}
			taskFetchSrc.Params = append(taskFetchSrc.Params, param)
		}

		taskBuildName := fmt.Sprintf("%s-%s", owner.Name, taskbuild)
		taskBuild := pipeline.PipelineTask{
			Name:    taskBuildName,
			TaskRef: &pipeline.TaskRef{Name: taskBuildName},
			RunAfter: []string{
				taskFetchSrcName,
			},
			Workspaces: []pipeline.WorkspacePipelineTaskBinding{
				pipeline.WorkspacePipelineTaskBinding{
					Name:      workspaceSource,
					Workspace: workspaceShare,
				},
			},
			Params: []pipeline.Param{
				pipeline.Param{
					Name: builderImage,
					Value: pipeline.ArrayOrString{
						Type:      pipeline.ParamTypeString,
						StringVal: owner.Spec.Builder,
					},
				},
				pipeline.Param{
					Name: cache,
					Value: pipeline.ArrayOrString{
						Type:      pipeline.ParamTypeString,
						StringVal: buildpacksCache,
					},
				},
				pipeline.Param{
					Name: "PLATFORM_DIR",
					Value: pipeline.ArrayOrString{
						Type:      pipeline.ParamTypeString,
						StringVal: platformDir,
					},
				},
			},
			Resources: &pipeline.PipelineTaskResources{
				Outputs: []pipeline.PipelineTaskOutputResource{
					pipeline.PipelineTaskOutputResource{
						Name:     image,
						Resource: buildImage,
					},
				},
			},
		}
		if owner.Spec.Source.SourceSubPath != nil {
			param := pipeline.Param{
				Name: sourceSubpath,
				Value: pipeline.ArrayOrString{
					Type:      pipeline.ParamTypeString,
					StringVal: *owner.Spec.Source.SourceSubPath,
				},
			}
			taskBuild.Params = append(taskBuild.Params, param)
		}
		expected.Spec.Tasks = []pipeline.PipelineTask{taskFetchSrc, taskBuild}

		expected.Spec.DeepCopyInto(&p.Spec)
		p.SetOwnerReferences(nil)
		return ctrl.SetControllerReference(owner, p, r.Scheme)
	}
}

func (r *FunctionReconciler) CreateOrUpdatePipeline(owner *v1alpha1.Function) error {
	log := r.Log.WithName("CreateOrUpdatePipeline")
	ctx := context.Background()

	p := pipeline.Pipeline{}
	p.Name = fmt.Sprintf("%s-%s", owner.Name, buildpacksPipeline)
	p.Namespace = owner.Namespace
	if result, err := controllerutil.CreateOrUpdate(ctx, r.Client, &p, r.mutatePipeline(&p, owner)); err != nil {
		log.Error(err, "Failed to CreateOrUpdate Pipeline", "result", result)
		return err
	}

	return nil
}

func (r *FunctionReconciler) mutatePipelineRun(pr *pipeline.PipelineRun, owner *v1alpha1.Function) controllerutil.MutateFn {
	return func() error {
		cms := v1.ConfigMapVolumeSource{
			Items: []v1.KeyToPath{
				v1.KeyToPath{
					Key:  functionTarget,
					Path: "env/" + functionTarget,
				},
				v1.KeyToPath{
					Key:  functionSignatureType,
					Path: "env/" + functionSignatureType,
				},
			},
		}
		cms.Name = fmt.Sprintf("%s-%s", owner.Name, platformEnv)

		expected := pipeline.PipelineRun{
			Spec: pipeline.PipelineRunSpec{
				ServiceAccountName: fmt.Sprintf("%s-%s", owner.Name, buildpacksSa),
				PipelineRef:        &pipeline.PipelineRef{Name: fmt.Sprintf("%s-%s", owner.Name, buildpacksPipeline)},
				Workspaces: []pipeline.WorkspaceBinding{
					pipeline.WorkspaceBinding{
						Name: workspaceShare,
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("%s-%s", owner.Name, buildpackSourcePvc),
						},
					},
				},
				Resources: []pipeline.PipelineResourceBinding{
					pipeline.PipelineResourceBinding{
						Name: buildImage,
						ResourceRef: &pipeline.PipelineResourceRef{
							Name: fmt.Sprintf("%s-%s", owner.Name, buildpacksAppImage),
						},
					},
				},
				PodTemplate: &pipeline.PodTemplate{
					Volumes: []v1.Volume{
						v1.Volume{
							Name: buildpacksCache,
							VolumeSource: v1.VolumeSource{
								EmptyDir: &v1.EmptyDirVolumeSource{},
							},
						},
						v1.Volume{
							Name: platformDir,
							VolumeSource: v1.VolumeSource{
								ConfigMap: &cms,
							},
						},
					},
				},
			},
		}

		expected.Spec.DeepCopyInto(&pr.Spec)
		pr.SetOwnerReferences(nil)
		return ctrl.SetControllerReference(owner, pr, r.Scheme)
	}
}

func (r *FunctionReconciler) CreateOrUpdatePipelineRun(owner *v1alpha1.Function) error {
	pr := pipeline.PipelineRun{}
	pr.Name = fmt.Sprintf("%s-%s", owner.Name, buildpacksPipelineRun)
	pr.Namespace = owner.Namespace

	log := r.Log.WithName("CreateOrUpdatePipelineRun")
	ctx := context.Background()
	if result, err := controllerutil.CreateOrUpdate(ctx, r.Client, &pr, r.mutatePipelineRun(&pr, owner)); err != nil {
		log.Error(err, "Failed to CreateOrUpdate PipelineRun", "result", result)
		return err
	}

	return nil
}

func (r *FunctionReconciler) Cleanup() {

}
