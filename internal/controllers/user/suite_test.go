package user

import (
	"context"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
	"github.com/epam/edp-nexus-operator/internal/controllers/nexus"
	nexusclient "github.com/epam/edp-nexus-operator/pkg/client/nexus"
	"github.com/epam/edp-nexus-operator/pkg/testutils"
)

const (
	timeout     = time.Second * 10
	interval    = time.Millisecond * 250
	namespace   = "test-nexus-user"
	nexusCRName = "test-nexus"
)

var (
	cfg           *rest.Config
	k8sClient     client.Client
	testEnv       *envtest.Environment
	ctx           context.Context
	cancel        context.CancelFunc
	nexusUrl      string
	nexusUser     string
	nexusPassword string
)

func TestNexusUser(t *testing.T) {
	RegisterFailHandler(Fail)

	if os.Getenv("TEST_NEXUS_URL") == "" {
		t.Skip("TEST_NEXUS_URL is not set")
	}

	RunSpecs(t, "Nexus Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.Background())
	nexusUrl = os.Getenv("TEST_NEXUS_URL")
	nexusUser = os.Getenv("TEST_NEXUS_USER")
	nexusPassword = os.Getenv("TEST_NEXUS_PASSWORD")

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     testutils.GetCRDDirectoryPaths(),
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: testutils.GetFirstFoundEnvTestBinaryDir(),
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := runtime.NewScheme()
	Expect(nexusApi.AddToScheme(scheme)).NotTo(HaveOccurred())
	Expect(corev1.AddToScheme(scheme)).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
	})
	Expect(err).ToNot(HaveOccurred())

	err = nexus.NewNexusReconciler(
		k8sManager.GetClient(),
		k8sManager.GetScheme(),
		nexusclient.NewApiClientProvider(k8sManager.GetClient()),
	).
		SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = NewNexusUserReconciler(
		k8sManager.GetClient(),
		k8sManager.GetScheme(),
		nexusclient.NewApiClientProvider(k8sManager.GetClient()),
	).
		SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	By("By creating namespace")
	Expect(k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})).Should(Succeed())

	By("By creating a secret")
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nexus-auth-secret",
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"user":     []byte(nexusUser),
			"password": []byte(nexusPassword),
		},
	}
	Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
	By("By creating Nexus object")
	newNexus := &nexusApi.Nexus{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nexusCRName,
			Namespace: namespace,
		},
		Spec: nexusApi.NexusSpec{
			Url:    nexusUrl,
			Secret: secret.Name,
		},
	}
	Expect(k8sClient.Create(ctx, newNexus)).Should(Succeed())
	Eventually(func() bool {
		createdNexus := &nexusApi.Nexus{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusCRName, Namespace: namespace}, createdNexus)
		if err != nil {
			return false
		}

		return createdNexus.Status.Connected && createdNexus.Status.Error == ""

	}, timeout, interval).Should(BeTrue())

})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
