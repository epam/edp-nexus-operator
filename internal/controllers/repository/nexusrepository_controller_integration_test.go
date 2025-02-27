package repository

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

var _ = Describe("NexusRepository controller", func() {
	nexusRepositoryCRName := "nexus-repository"
	It("Should create NexusRepository object", func() {
		By("By creating a new NexusRepository object")
		newNexusRepository := &nexusApi.NexusRepository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nexusRepositoryCRName,
				Namespace: namespace,
			},
			Spec: nexusApi.NexusRepositorySpec{
				Go: &nexusApi.GoSpec{
					Proxy: &nexusApi.GoProxyRepository{
						ProxySpec: nexusApi.ProxySpec{
							Name:   "go-proxy",
							Online: true,
							Storage: nexusApi.Storage{
								BlobStoreName:               "default",
								StrictContentTypeValidation: true,
							},
							Proxy: nexusApi.Proxy{
								ContentMaxAge:  1440,
								MetadataMaxAge: 1440,
								RemoteURL:      "https://go-proxy-url",
							},
							NegativeCache: nexusApi.NegativeCache{
								Enabled: true,
								TTL:     1440,
							},
							HTTPClient: nexusApi.HTTPClient{
								AutoBlock: true,
								Blocked:   false,
							},
						},
					},
				},
				NexusRef: common.NexusRef{
					Name: nexusCRName,
				},
			},
		}
		Expect(k8sClient.Create(ctx, newNexusRepository)).Should(Succeed())
		Eventually(func() bool {
			createdNexusRepository := &nexusApi.NexusRepository{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusRepositoryCRName, Namespace: namespace}, createdNexusRepository)
			if err != nil {
				return false
			}

			return createdNexusRepository.Status.Value == common.StatusCreated && createdNexusRepository.Status.Error == ""
		}, timeout, interval).Should(BeTrue())
	})
	It("Should update NexusRepository object", func() {
		By("Getting NexusRepository object")
		createdNexusRepository := &nexusApi.NexusRepository{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: nexusRepositoryCRName, Namespace: namespace}, createdNexusRepository)).
			Should(Succeed())

		By("Updating NexusRepository object")
		createdNexusRepository.Spec.Go.Proxy.Online = false

		Expect(k8sClient.Update(ctx, createdNexusRepository)).Should(Succeed())
		Consistently(func() bool {
			updatedNexusRepository := &nexusApi.NexusRepository{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusRepositoryCRName, Namespace: namespace}, updatedNexusRepository)
			if err != nil {
				return false
			}

			return updatedNexusRepository.Status.Value == common.StatusCreated && updatedNexusRepository.Status.Error == ""
		}, timeout, interval).Should(BeTrue())
	})
	It("Should delete NexusRepository object", func() {
		By("Getting NexusRepository object")
		createdNexusRepository := &nexusApi.NexusRepository{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: nexusRepositoryCRName, Namespace: namespace}, createdNexusRepository)).
			Should(Succeed())

		By("Deleting NexusRepository object")
		Expect(k8sClient.Delete(ctx, createdNexusRepository)).Should(Succeed())
		By("Waiting for NexusRepository to be deleted")
		time.Sleep(time.Second)
		Eventually(func() bool {
			createdNexusRepository := &nexusApi.NexusRepository{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusRepositoryCRName, Namespace: namespace}, createdNexusRepository)
			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})
})
