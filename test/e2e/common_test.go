package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	openfunction "github.com/openfunction/apis/core/v1alpha2"
	shipwrightv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KubeConfig returns all required clients to speak with
// the k8s API
func KubeConfig() (*kubernetes.Clientset, *rest.Config, error) {
	location := os.Getenv("KUBECONFIG")
	if location == "" {
		location = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", location)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return clientset, config, nil
}

func createCurlPod() error {

	bs, err := ioutil.ReadFile("data/pod.yaml")
	if err != nil {
		return err
	}

	pod := &corev1.Pod{}
	err = yaml.Unmarshal(bs, pod)
	if err != nil {
		return err
	}

	err = cl.Create(context.Background(), pod)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func deleteCurlPod() {

	bs, err := ioutil.ReadFile("data/pod.yaml")
	if err != nil {
		return
	}

	pod := &corev1.Pod{}
	err = yaml.Unmarshal(bs, pod)
	if err != nil {
		return
	}

	var period int64 = 0
	err = cl.Delete(context.Background(), pod, &client.DeleteOptions{
		GracePeriodSeconds: &period,
	})
	if err != nil {
		return
	}
}

func logf(format string, args ...interface{}) {
	currentTime := time.Now().UTC().Format(time.RFC3339)

	fmt.Printf(
		fmt.Sprintf("%s %s\n", currentTime, format),
		args...,
	)
}

func createFunction(file string) (*openfunction.Function, error) {

	bs, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	fn := &openfunction.Function{}
	if err := yaml.Unmarshal(bs, fn); err != nil {
		return nil, err
	}

	fn.Namespace = namespace
	fn.Spec.Image = fn.Spec.Image[0:strings.Index(fn.Spec.Image, ":")]
	fn.Spec.Image = fmt.Sprintf("%s:%s", fn.Spec.Image, tag)

	return fn, err
}

func checkFunction(fn *openfunction.Function) (bool, error) {

	function := &openfunction.Function{}
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(fn), function); err != nil {
		logf("[CheckFunction] get function %s/%s error, %s", fn.Name, fn.Namespace, err.Error())
		return false, nil
	}

	if function.Status.Build == nil {
		logf("[CheckFunction] function %s/%s is still building", fn.Name, fn.Namespace)
		return false, nil
	}

	if function.Status.Build.State == "" || function.Status.Build.State == openfunction.Building {
		logf("[CheckFunction] function %s/%s is still building", fn.Name, fn.Namespace)
		return false, nil
	} else if function.Status.Build.State == openfunction.Succeeded ||
		function.Status.Build.State == openfunction.Skipped {
		logf("[CheckFunction] function %s/%s build %s", fn.Name, fn.Namespace, function.Status.Build.State)

		if function.Status.Serving == nil {
			logf("[CheckFunction] function %s/%s serving is not running", fn.Name, fn.Namespace)
			return false, nil
		}

		if function.Status.Serving.State == openfunction.Running ||
			function.Status.Serving.State == openfunction.Skipped {
			logf("[CheckFunction] function %s/%s serving is %s", fn.Name, fn.Namespace, function.Status.Serving.State)
			return true, nil
		} else {
			logf("[CheckFunction] function %s/%s serving is not running", fn.Name, fn.Namespace)
			return false, nil
		}
	} else {
		logf("[CheckFunction] function %s/%s build %s", fn.Name, fn.Namespace, function.Status.Build.State)
		return false, fmt.Errorf("function build failed")
	}
}

func accessFunction(fn *openfunction.Function) error {

	function := &openfunction.Function{}
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(fn), function); err != nil {
		return err
	}

	time.Sleep(time.Second * 10)
	req := clientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name("curl").
		Namespace("default").
		SubResource("exec").
		Param("container", "curl")

	req.VersionedParams(
		&corev1.PodExecOptions{
			Command: []string{
				"sh",
				"-c",
				fmt.Sprintf("curl %s", function.Status.URL),
			},
			Stdin:  false,
			Stdout: true,
			Stderr: true,
			TTY:    false,
		},
		runtime.NewParameterCodec(scheme),
	)

	var stdout, stderr bytes.Buffer
	exec, err := remotecommand.NewSPDYExecutor(restConfig, "GET", req.URL())
	if err != nil {
		return err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return err
	}

	resp := stdout.String()
	fmt.Println(stdout.String())
	if resp != "Hello, World!\n" {
		return fmt.Errorf("access function failed")
	}

	return nil
}

// printTestFailureDebugInfo will output the status of Function, Builder, Serving
func printTestFailureDebugInfo(fn *openfunction.Function) {

	function := &openfunction.Function{}
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(fn), function); err != nil {
		logf("[PrintTestFailureDebugInfo] get Function %s/%s error, %s", fn.Name, fn.Namespace, err.Error())
		return
	}

	function.ManagedFields = nil
	logf("Print failed Function")
	printObject(function)

	if function.Status.Build != nil {
		builder := &openfunction.Builder{
			ObjectMeta: metav1.ObjectMeta{
				Name:      function.Status.Build.ResourceRef,
				Namespace: fn.Namespace,
			},
		}
		if err := cl.Get(context.Background(), client.ObjectKeyFromObject(builder), builder); err != nil {
			logf("[PrintTestFailureDebugInfo] get Builder %s/%s error, %s", builder.Name, builder.Namespace, err.Error())
			return
		}

		builder.ManagedFields = nil
		logf("Print failed Builder")
		printObject(builder)

		if builder.Status.ResourceRef != nil {
			build := &shipwrightv1alpha1.Build{
				ObjectMeta: metav1.ObjectMeta{
					Name:      builder.Status.ResourceRef["shipwright.io/build"],
					Namespace: builder.Namespace,
				},
			}

			if err := cl.Get(context.Background(), client.ObjectKeyFromObject(build), build); err != nil {
				logf("[PrintTestFailureDebugInfo] get Build %s/%s error, %s", build.Name, build.Namespace, err.Error())
				return
			}

			build.ManagedFields = nil
			logf("Print failed Build")
			printObject(build)

			buildrun := &shipwrightv1alpha1.BuildRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      builder.Status.ResourceRef["shipwright.io/buildRun"],
					Namespace: builder.Namespace,
				},
			}

			if err := cl.Get(context.Background(), client.ObjectKeyFromObject(buildrun), buildrun); err != nil {
				logf("[PrintTestFailureDebugInfo] get BuildRun %s/%s error, %s", buildrun.Name, buildrun.Namespace, err.Error())
				return
			}

			buildrun.ManagedFields = nil
			logf("Print failed BuildRun")
			printObject(buildrun)
		}

	}

	if function.Status.Serving != nil {
		serving := &openfunction.Serving{
			ObjectMeta: metav1.ObjectMeta{
				Name:      function.Status.Serving.ResourceRef,
				Namespace: fn.Namespace,
			},
		}
		if err := cl.Get(context.Background(), client.ObjectKeyFromObject(serving), serving); err != nil {
			logf("[PrintTestFailureDebugInfo] get Serving %s/%s error, %s", serving.Name, serving.Namespace, err.Error())
			return
		}

		serving.ManagedFields = nil
		logf("Print failed Serving")
		printObject(serving)
	}
}

func printObject(obj interface{}) {

	bs, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		logf("[PrintObject] print object error, %s", err.Error())
		return
	}

	fmt.Println(string(bs))
}
