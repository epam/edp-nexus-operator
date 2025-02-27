package webhook

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var _ = Describe("webhooks registration", func() {
	When("ValidatingWebhookConfiguration exists", func() {
		BeforeEach(func() {
			By("creating ValidatingWebhookConfiguration")
			webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: ctrl.ObjectMeta{
					Name: getValidationWebHookName("default"),
				},
			}
			Expect(k8sClient.Create(ctx, webhook)).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: ctrl.ObjectMeta{
					Name: getValidationWebHookName("default"),
				},
			}
			err := ctrlclient.IgnoreNotFound(k8sClient.Delete(ctx, webhook))
			Expect(err).ToNot(HaveOccurred())
		})
		It("should register validation webhooks", func() {
			By("creating manager")
			k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
				Scheme: k8sClient.Scheme(),
				Metrics: metricsserver.Options{
					BindAddress: "0",
				},
			})
			Expect(err).ToNot(HaveOccurred())

			By("registering validation webhooks")
			err = RegisterValidationWebHook(ctx, k8sManager, "default")
			Expect(err).ToNot(HaveOccurred())
		})
		When("scheme doesn't contain nexus types", func() {
			It("should return error", func() {
				By("creating manager")
				k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
					Scheme: scheme.Scheme,
					Metrics: metricsserver.Options{
						BindAddress: "0",
					},
				})
				Expect(err).ToNot(HaveOccurred())

				By("registering validation webhooks")
				err = RegisterValidationWebHook(ctx, k8sManager, "default")
				Expect(err).To(HaveOccurred())
			})
		})
		When("SETUP_SELF_SIGNED_CERTIFICATES is false", func() {
			BeforeEach(func() {
				err := os.Setenv("SETUP_SELF_SIGNED_CERTIFICATES", "false")
				Expect(err).ToNot(HaveOccurred())
				DeferCleanup(os.Unsetenv, "SETUP_SELF_SIGNED_CERTIFICATES")
			})
			It("should register validation webhooks", func() {
				By("creating manager")
				k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
					Scheme: k8sClient.Scheme(),
					Metrics: metricsserver.Options{
						BindAddress: "0",
					},
				})
				Expect(err).ToNot(HaveOccurred())

				By("registering validation webhooks")
				err = RegisterValidationWebHook(ctx, k8sManager, "default")
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
	When("MutatingWebhookConfiguration doesn't exist", func() {
		It("should return error", func() {
			By("creating manager")
			k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
				Scheme: k8sClient.Scheme(),
				Metrics: metricsserver.Options{
					BindAddress: "0",
				},
			})
			Expect(err).ToNot(HaveOccurred())

			By("registering mutation webhooks")
			err = RegisterValidationWebHook(ctx, k8sManager, "default")
			Expect(err).To(HaveOccurred())
		})
	})
})
