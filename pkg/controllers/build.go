package controllers

import (
	goerrors "errors"
	"fmt"
	"github.com/ghodss/yaml"
	openfunction "github.com/openfunction/pkg/apis/v1alpha1"
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
	buildSa               = "build-service-account"
	buildFuncImage        = "build-func-image"
	builderImage          = "BUILDER_IMAGE"
	buildCache            = "build-cache"
	buildPipeline         = "build-pipeline"
	BuildPipelineRun      = "build-pipelinerun"
	buildImage            = "build-image"
	buildpackSourcePvc    = "build-source-pvc"
	platformEnv           = "platform-env"
	image                 = "image"
	registryUrlKey        = "tekton.dev/docker-0"
	registryUrl           = "https://index.docker.io/v1/"
	sourceSubpath         = "SOURCE_SUBPATH"
	subDirectory          = "subdirectory"
	buildTask             = "build"
	gitCloneTask          = "git-clone"
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
		buildTask:    tmplBuild,
		gitCloneTask: tmplGitClone,
	}
)

func UnmarshalTask(task string) (*pipeline.Task, error) {
	var t pipeline.Task
	return &t, yaml.Unmarshal([]byte(task), &t)
}

func (r *BuilderReconciler) mutateTask(task *pipeline.Task, builder *openfunction.Builder, name string) controllerutil.MutateFn {
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
		return ctrl.SetControllerReference(builder, task, r.Scheme)
	}
}

func (r *BuilderReconciler) CreateOrUpdateTask(builder *openfunction.Builder, name string) error {
	log := r.Log.WithName("CreateOrUpdateTask")

	task := pipeline.Task{}
	task.Name = fmt.Sprintf("%s-%s", builder.Name, name)
	task.Namespace = builder.Namespace
	if result, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, &task, r.mutateTask(&task, builder, name)); err != nil {
		log.Error(err, "Failed to CreateOrUpdate Task", "result", result)
		return err
	}
	return nil
}

func (r *BuilderReconciler) mutateConfigMap(cm *v1.ConfigMap, builder *openfunction.Builder) controllerutil.MutateFn {
	return func() error {
		expected := v1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      cm.Name,
				Namespace: builder.Namespace,
			},
			Data: map[string]string{
				functionSignatureType: builder.Spec.FuncType,
				functionTarget:        builder.Spec.FuncName,
			},
		}

		if len(cm.Data) == 0 {
			cm.Data = map[string]string{
				functionSignatureType: expected.Data[functionSignatureType],
				functionTarget:        expected.Data[functionTarget],
			}
			expected.DeepCopyInto(cm)
			cm.SetOwnerReferences(nil)
			return ctrl.SetControllerReference(builder, cm, r.Scheme)
		}
		return nil
	}
}

func (r *BuilderReconciler) CreateOrUpdateConfigMap(builder *openfunction.Builder) error {
	log := r.Log.WithName("CreateOrUpdateConfigMap")

	cm := v1.ConfigMap{}
	cm.Name = fmt.Sprintf("%s-%s", builder.Name, platformEnv)
	cm.Namespace = builder.Namespace
	if result, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, &cm, r.mutateConfigMap(&cm, builder)); err != nil {
		log.Error(err, "Failed to CreateOrUpdate ConfigMap", "result", result)
		return err
	}
	return nil
}

func (r *BuilderReconciler) mutatePVC(pvc *v1.PersistentVolumeClaim, builder *openfunction.Builder) controllerutil.MutateFn {
	return func() error {
		expected := v1.PersistentVolumeClaim{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "PersistentVolumeClaim",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvc.Name,
				Namespace: builder.Namespace,
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
		return ctrl.SetControllerReference(builder, pvc, r.Scheme)
	}
}

func (r *BuilderReconciler) CreateOrUpdateBuildpackPVCs(builder *openfunction.Builder) error {
	log := r.Log.WithName("CreateBuildpackPVCs")

	pvcs := []string{fmt.Sprintf("%s-%s", builder.Name, buildpackSourcePvc)}
	for _, v := range pvcs {
		pvc := v1.PersistentVolumeClaim{}
		pvc.Name = v
		pvc.Namespace = builder.Namespace
		if result, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, &pvc, r.mutatePVC(&pvc, builder)); err != nil {
			log.Error(err, "Failed to CreateOrUpdate PersistentVolumeClaim", "result", result)
			return err
		}
	}
	return nil
}

func (r *BuilderReconciler) mutateRegistryAuth(sa *v1.ServiceAccount, builder *openfunction.Builder) controllerutil.MutateFn {
	return func() error {
		s := v1.Secret{}
		if err := r.Client.Get(r.ctx, types.NamespacedName{Namespace: builder.Namespace, Name: builder.Spec.Registry.Account.Name}, &s); err != nil {
			return err
		}
		var url string
		if builder.Spec.Registry.Url == nil {
			url = registryUrl
		} else {
			url = *builder.Spec.Registry.Url
		}
		s.Annotations[registryUrlKey] = url
		s.Type = "kubernetes.io/basic-auth"

		if err := r.Client.Update(r.ctx, &s); err != nil {
			return err
		}

		expected := v1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", builder.Name, buildSa),
				Namespace: builder.Namespace,
			},
			Secrets: []v1.ObjectReference{
				v1.ObjectReference{
					APIVersion: "v1",
					Kind:       "Secret",
					Name:       s.Name,
					Namespace:  builder.Namespace,
				},
			},
		}

		expected.DeepCopyInto(sa)
		sa.SetOwnerReferences(nil)
		return ctrl.SetControllerReference(builder, sa, r.Scheme)
	}
}

func (r *BuilderReconciler) CreateOrUpdateRegistryAuth(builder *openfunction.Builder) error {
	log := r.Log.WithName("CreateOrUpdateRegistryAuth")
	sa := v1.ServiceAccount{}
	sa.Name = fmt.Sprintf("%s-%s", builder.Name, buildSa)
	sa.Namespace = builder.Namespace
	if result, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, &sa, r.mutateRegistryAuth(&sa, builder)); err != nil {
		log.Error(err, "Failed to CreateOrUpdate ServiceAccount", "result", result)
		return err
	}
	return nil
}

func (r *BuilderReconciler) mutatePipelineResource(res *pipelineres.PipelineResource, builder *openfunction.Builder) controllerutil.MutateFn {
	return func() error {
		expected := pipelineres.PipelineResource{
			Spec: pipelineres.PipelineResourceSpec{
				Type: pipelineres.PipelineResourceTypeImage,
				Params: []pipelineres.ResourceParam{
					pipelineres.ResourceParam{
						Name:  url,
						Value: builder.Spec.Image,
					},
				},
			},
		}

		expected.Spec.DeepCopyInto(&res.Spec)
		res.SetOwnerReferences(nil)
		return ctrl.SetControllerReference(builder, res, r.Scheme)
	}
}

func (r *BuilderReconciler) CreateOrUpdatePipelineResource(builder *openfunction.Builder) error {
	log := r.Log.WithName("CreatePipelineResource")

	res := pipelineres.PipelineResource{}
	res.Name = fmt.Sprintf("%s-%s", builder.Name, buildFuncImage)
	res.Namespace = builder.Namespace
	if result, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, &res, r.mutatePipelineResource(&res, builder)); err != nil {
		log.Error(err, "Failed to CreateOrUpdate PipelineResource", "result", result)
		return err
	}
	return nil
}

func (r *BuilderReconciler) mutatePipeline(p *pipeline.Pipeline, builder *openfunction.Builder) controllerutil.MutateFn {
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

		taskFetchSrcName := fmt.Sprintf("%s-%s", builder.Name, gitCloneTask)
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
						StringVal: builder.Spec.Source.Url,
					},
				},
			},
		}
		if builder.Spec.Source.DeleteExisting != nil {
			param := pipeline.Param{
				Name: subDirectory,
				Value: pipeline.ArrayOrString{
					Type:      pipeline.ParamTypeString,
					StringVal: *builder.Spec.Source.DeleteExisting,
				},
			}
			taskFetchSrc.Params = append(taskFetchSrc.Params, param)
		}

		buildTaskName := fmt.Sprintf("%s-%s", builder.Name, buildTask)
		buildTask := pipeline.PipelineTask{
			Name:    buildTaskName,
			TaskRef: &pipeline.TaskRef{Name: buildTaskName},
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
						StringVal: builder.Spec.Builder,
					},
				},
				pipeline.Param{
					Name: "CACHE",
					Value: pipeline.ArrayOrString{
						Type:      pipeline.ParamTypeString,
						StringVal: buildCache,
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
		if builder.Spec.Source.SourceSubPath != nil {
			param := pipeline.Param{
				Name: sourceSubpath,
				Value: pipeline.ArrayOrString{
					Type:      pipeline.ParamTypeString,
					StringVal: *builder.Spec.Source.SourceSubPath,
				},
			}
			buildTask.Params = append(buildTask.Params, param)
		}
		expected.Spec.Tasks = []pipeline.PipelineTask{taskFetchSrc, buildTask}

		expected.Spec.DeepCopyInto(&p.Spec)
		p.SetOwnerReferences(nil)
		return ctrl.SetControllerReference(builder, p, r.Scheme)
	}
}

func (r *BuilderReconciler) CreateOrUpdatePipeline(builder *openfunction.Builder) error {
	log := r.Log.WithName("CreateOrUpdatePipeline")

	p := pipeline.Pipeline{}
	p.Name = fmt.Sprintf("%s-%s", builder.Name, buildPipeline)
	p.Namespace = builder.Namespace
	if result, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, &p, r.mutatePipeline(&p, builder)); err != nil {
		log.Error(err, "Failed to CreateOrUpdate Pipeline", "result", result)
		return err
	}

	return nil
}

func (r *BuilderReconciler) mutatePipelineRun(pr *pipeline.PipelineRun, builder *openfunction.Builder) controllerutil.MutateFn {
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
		cms.Name = fmt.Sprintf("%s-%s", builder.Name, platformEnv)

		expected := pipeline.PipelineRun{
			Spec: pipeline.PipelineRunSpec{
				ServiceAccountName: fmt.Sprintf("%s-%s", builder.Name, buildSa),
				PipelineRef:        &pipeline.PipelineRef{Name: fmt.Sprintf("%s-%s", builder.Name, buildPipeline)},
				Workspaces: []pipeline.WorkspaceBinding{
					pipeline.WorkspaceBinding{
						Name: workspaceShare,
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("%s-%s", builder.Name, buildpackSourcePvc),
						},
					},
				},
				Resources: []pipeline.PipelineResourceBinding{
					pipeline.PipelineResourceBinding{
						Name: buildImage,
						ResourceRef: &pipeline.PipelineResourceRef{
							Name: fmt.Sprintf("%s-%s", builder.Name, buildFuncImage),
						},
					},
				},
				PodTemplate: &pipeline.PodTemplate{
					Volumes: []v1.Volume{
						v1.Volume{
							Name: buildCache,
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
		return ctrl.SetControllerReference(builder, pr, r.Scheme)
	}
}

func (r *BuilderReconciler) CreateOrUpdatePipelineRun(builder *openfunction.Builder) error {
	log := r.Log.WithName("CreateOrUpdatePipelineRun")

	pr := pipeline.PipelineRun{}
	pr.Name = fmt.Sprintf("%s-%s", builder.Name, BuildPipelineRun)
	pr.Namespace = builder.Namespace

	if result, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, &pr, r.mutatePipelineRun(&pr, builder)); err != nil {
		log.Error(err, "Failed to CreateOrUpdate PipelineRun", "result", result)
		return err
	}

	return nil
}
