package e2e

import (
	"os"
	"testing"

	componentsv1alpha1 "github.com/dapr/dapr/pkg/apis/components/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1alpha1 "github.com/openfunction/apis/core/v1alpha1"
	corev1alpha2 "github.com/openfunction/apis/core/v1alpha2"
	openfunctionevent "github.com/openfunction/apis/events/v1alpha1"
	shipwrightv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	knserving "knative.dev/serving/pkg/client/clientset/versioned/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	scheme     *runtime.Scheme
	cl         client.Client
	clientSet  *kubernetes.Clientset
	restConfig *rest.Config

	tag       string
	namespace string
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func(done Done) {

	scheme = runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = knserving.AddToScheme(scheme)
	_ = corev1alpha1.AddToScheme(scheme)
	_ = corev1alpha2.AddToScheme(scheme)
	_ = componentsv1alpha1.AddToScheme(scheme)
	_ = kedav1alpha1.AddToScheme(scheme)
	_ = openfunctionevent.AddToScheme(scheme)
	_ = shipwrightv1alpha1.AddToScheme(scheme)

	tag = os.Getenv("TAG")
	if tag == "" {
		tag = "latest"
	}

	namespace = os.Getenv("TEST_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	var err error
	clientSet, restConfig, err = KubeConfig()
	Expect(err).NotTo(HaveOccurred())

	mapper, err := func(c *rest.Config) (meta.RESTMapper, error) {
		return apiutil.NewDynamicRESTMapper(c)
	}(restConfig)
	Expect(err).NotTo(HaveOccurred())

	c, err := client.New(restConfig, client.Options{Scheme: scheme, Mapper: mapper})
	Expect(err).NotTo(HaveOccurred())

	cl, err = client.NewDelegatingClient(client.NewDelegatingClientInput{
		CacheReader:       c,
		Client:            c,
		UncachedObjects:   nil,
		CacheUnstructured: false,
	})
	Expect(err).NotTo(HaveOccurred())

	Expect(createCurlPod()).NotTo(HaveOccurred())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	deleteCurlPod()
})
