package controllers

import (
	goerrors "errors"
	"fmt"
	"strconv"

	"github.com/openfunction/pkg/util"

	"github.com/ghodss/yaml"
	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	openfunction "github.com/openfunction/api/v1alpha1"
)

const (
	buildSa          = "build-service-account"
	builderImage     = "BUILDER_IMAGE"
	buildPipeline    = "build-pipeline"
	BuildPipelineRun = "build-pipelinerun"
	buildpackPVC     = "buildpack-pvc"
	envVars          = "ENV_VARS"
	appImage         = "APP_IMAGE"
	registryUrlKey   = "tekton.dev/docker-0"
	registryUrl      = "https://index.docker.io/v1/"
	sourceSubpath    = "SOURCE_SUBPATH"
	subDirectory     = "subdirectory"
	buildTask        = "build"
	gitCloneTask     = "git-clone"
	url              = "url"
	cacheWorkspace   = "cache-ws"
	sourceWorkspace  = "shared-ws"
	output           = "output"
	cache            = "cache"
	source           = "source"
	revision         = "revision"
	refspec          = "refspec"
	submodules       = "submodules"
	depth            = "depth"
	sslVerify        = "sslVerify"
	deleteExisting   = "deleteExisting"
	httpProxy        = "httpProxy"
	httpsProxy       = "httpsProxy"
	noProxy          = "noProxy"
	verbose          = "verbose"
	gitInitImage     = "gitInitImage"
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
			err := goerrors.New("doesn't exist")
			return err
		}

		expected, err := UnmarshalTask(tmpl)
		if err != nil {
			return err
		}

		for i := range expected.Spec.Steps {
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

	if err := r.Delete(r.ctx, &task); util.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to delete builder Task", "name", task.Name, "namespace", task.Namespace)
		return err
	}

	if err := r.mutateTask(&task, builder, name)(); err != nil {
		log.Error(err, "Failed to mutate builder Task", "name", task.Name, "namespace", task.Namespace)
		return err
	}

	if err := r.Create(r.ctx, &task); err != nil {
		log.Error(err, "Failed to create builder Task", "name", task.Name, "namespace", task.Namespace)
		return err
	}

	log.V(1).Info("Create Task", "name", task.Name, "namespace", task.Namespace)
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

	pvcs := []string{fmt.Sprintf("%s-%s", builder.Name, buildpackPVC)}
	for _, v := range pvcs {
		pvc := v1.PersistentVolumeClaim{}
		pvc.Name = v
		pvc.Namespace = builder.Namespace

		if err := r.Delete(r.ctx, &pvc); util.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to delete PersistentVolumeClaim", "name", pvc.Name, "namespace", pvc.Namespace)
			return err
		}

		if err := r.mutatePVC(&pvc, builder)(); err != nil {
			log.Error(err, "Failed to mutate PersistentVolumeClaim", "name", pvc.Name, "namespace", pvc.Namespace)
			return err
		}

		if err := r.Create(r.ctx, &pvc); err != nil {
			log.Error(err, "Failed to create PersistentVolumeClaim", "name", pvc.Name, "namespace", pvc.Namespace)
			return err
		}

		log.V(1).Info("Create PersistentVolumeClaim", "name", pvc.Name, "namespace", pvc.Namespace)
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
				{
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
	result, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, &sa, r.mutateRegistryAuth(&sa, builder))
	if err != nil {
		log.Error(err, "Failed to CreateOrUpdate ServiceAccount", "result", result)
		return err
	}

	log.V(1).Info("CreateOrUpdate ServiceAccount", "name", sa.Name, "namespace", sa.Namespace, "operator", result)
	return nil
}

func (r *BuilderReconciler) mutatePipeline(p *pipeline.Pipeline, builder *openfunction.Builder) controllerutil.MutateFn {
	return func() error {
		expected := pipeline.Pipeline{
			Spec: pipeline.PipelineSpec{
				Workspaces: []pipeline.PipelineWorkspaceDeclaration{
					{
						Name: sourceWorkspace,
					},
					{
						Name: cacheWorkspace,
					},
				},
			},
		}

		taskFetchSrcName := fmt.Sprintf("%s-%s", builder.Name, gitCloneTask)
		taskFetchSrc := pipeline.PipelineTask{
			Name:    taskFetchSrcName,
			TaskRef: &pipeline.TaskRef{Name: taskFetchSrcName},
			Workspaces: []pipeline.WorkspacePipelineTaskBinding{
				{
					Name:      output,
					Workspace: sourceWorkspace,
				},
			},
			Params: []pipeline.Param{
				{
					Name: url,
					Value: pipeline.ArrayOrString{
						Type:      pipeline.ParamTypeString,
						StringVal: builder.Spec.SrcRepo.Url,
					},
				},
			},
		}

		m := make(map[string]string)
		if builder.Spec.SrcRepo.Revision != nil {
			m[revision] = *builder.Spec.SrcRepo.Revision
		}

		if builder.Spec.SrcRepo.Refspec != nil {
			m[refspec] = *builder.Spec.SrcRepo.Refspec
		}

		if builder.Spec.SrcRepo.Submodules != nil {
			m[submodules] = strconv.FormatBool(*builder.Spec.SrcRepo.Submodules)
		}

		if builder.Spec.SrcRepo.Depth != nil {
			m[depth] = fmt.Sprintf("%d", *builder.Spec.SrcRepo.Depth)
		}

		if builder.Spec.SrcRepo.SslVerify != nil {
			m[sslVerify] = strconv.FormatBool(*builder.Spec.SrcRepo.SslVerify)
		}

		if builder.Spec.SrcRepo.SubDirectory != nil {
			m[subDirectory] = *builder.Spec.SrcRepo.SubDirectory
		}

		if builder.Spec.SrcRepo.DeleteExisting != nil {
			m[deleteExisting] = strconv.FormatBool(*builder.Spec.SrcRepo.DeleteExisting)
		}

		if builder.Spec.SrcRepo.HttpProxy != nil {
			m[httpProxy] = *builder.Spec.SrcRepo.HttpProxy
		}

		if builder.Spec.SrcRepo.HttpsProxy != nil {
			m[httpsProxy] = *builder.Spec.SrcRepo.HttpsProxy
		}

		if builder.Spec.SrcRepo.NoProxy != nil {
			m[noProxy] = *builder.Spec.SrcRepo.NoProxy
		}

		if builder.Spec.SrcRepo.Verbose != nil {
			m[verbose] = strconv.FormatBool(*builder.Spec.SrcRepo.Verbose)
		}

		if builder.Spec.SrcRepo.GitInitImage != nil {
			m[gitInitImage] = *builder.Spec.SrcRepo.GitInitImage
		}

		for k, v := range m {
			taskFetchSrc.Params = append(taskFetchSrc.Params,
				pipeline.Param{
					Name: k,
					Value: pipeline.ArrayOrString{
						Type:      pipeline.ParamTypeString,
						StringVal: v,
					},
				},
			)
		}

		var funcEnv []string
		for k, v := range builder.Spec.Params {
			funcEnv = append(funcEnv, fmt.Sprintf("%s=%s", k, v))
		}

		if builder.Spec.Port != nil && *builder.Spec.Port > 0 {
			funcEnv = append(funcEnv, fmt.Sprintf("%s=%d", "PORT", *builder.Spec.Port))
		}

		buildTaskName := fmt.Sprintf("%s-%s", builder.Name, buildTask)
		buildTask := pipeline.PipelineTask{
			Name:    buildTaskName,
			TaskRef: &pipeline.TaskRef{Name: buildTaskName},
			RunAfter: []string{
				taskFetchSrcName,
			},
			Workspaces: []pipeline.WorkspacePipelineTaskBinding{
				{
					Name:      source,
					Workspace: sourceWorkspace,
				},
				{
					Name:      cache,
					Workspace: cacheWorkspace,
				},
			},
			Params: []pipeline.Param{
				{
					Name: appImage,
					Value: pipeline.ArrayOrString{
						Type:      pipeline.ParamTypeString,
						StringVal: builder.Spec.Image,
					},
				},
				{
					Name: builderImage,
					Value: pipeline.ArrayOrString{
						Type:      pipeline.ParamTypeString,
						StringVal: builder.Spec.Builder,
					},
				},
				{
					Name: envVars,
					Value: pipeline.ArrayOrString{
						Type:     pipeline.ParamTypeArray,
						ArrayVal: funcEnv,
					},
				},
			},
		}
		if builder.Spec.SrcRepo.SourceSubPath != nil {
			param := pipeline.Param{
				Name: sourceSubpath,
				Value: pipeline.ArrayOrString{
					Type:      pipeline.ParamTypeString,
					StringVal: *builder.Spec.SrcRepo.SourceSubPath,
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
	result, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, &p, r.mutatePipeline(&p, builder))
	if err != nil {
		log.Error(err, "Failed to CreateOrUpdate Pipeline", "result", result)
		return err
	}

	log.V(1).Info("CreateOrUpdate Pipeline", "name", p.Name, "namespace", p.Namespace, "operator", result)
	return nil
}

func (r *BuilderReconciler) mutatePipelineRun(pr *pipeline.PipelineRun, builder *openfunction.Builder) controllerutil.MutateFn {
	return func() error {
		expected := pipeline.PipelineRun{
			Spec: pipeline.PipelineRunSpec{
				ServiceAccountName: fmt.Sprintf("%s-%s", builder.Name, buildSa),
				PipelineRef:        &pipeline.PipelineRef{Name: fmt.Sprintf("%s-%s", builder.Name, buildPipeline)},
				Workspaces: []pipeline.WorkspaceBinding{
					{
						Name:    sourceWorkspace,
						SubPath: source,
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("%s-%s", builder.Name, buildpackPVC),
						},
					},
					{
						Name:    cacheWorkspace,
						SubPath: cache,
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("%s-%s", builder.Name, buildpackPVC),
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

	result, err := controllerutil.CreateOrUpdate(r.ctx, r.Client, &pr, r.mutatePipelineRun(&pr, builder))
	if err != nil {
		log.Error(err, "Failed to CreateOrUpdate PipelineRun", "result", result)
		return err
	}

	log.V(1).Info("CreateOrUpdate PipelineRun", "name", pr.Name, "namespace", pr.Namespace, "operator", result)
	return nil
}
