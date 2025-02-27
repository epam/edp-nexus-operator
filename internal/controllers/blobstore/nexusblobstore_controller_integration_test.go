package blobstore

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

var _ = Describe("NexusBlobStore controller", func() {
	nexusBlobStoreCRName := "nexus-blobstore"
	It("Should create NexusBlobStore object", func() {
		By("By creating a new NexusBlobStore object")
		newNexusBlobStore := &nexusApi.NexusBlobStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nexusBlobStoreCRName,
				Namespace: namespace,
			},
			Spec: nexusApi.NexusBlobStoreSpec{
				Name: "test-blobstore",
				SoftQuota: &nexusApi.SoftQuota{
					Type:  nexusApi.SoftQuotaSpaceUsedQuota,
					Limit: 100,
				},
				File: &nexusApi.File{
					Path: "test-blobstore",
				},
				NexusRef: common.NexusRef{
					Name: nexusCRName,
				},
			},
		}
		Expect(k8sClient.Create(ctx, newNexusBlobStore)).Should(Succeed())
		Eventually(func() bool {
			createdNexusBlobStore := &nexusApi.NexusBlobStore{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusBlobStoreCRName, Namespace: namespace}, createdNexusBlobStore)
			if err != nil {
				return false
			}

			return createdNexusBlobStore.Status.Value == common.StatusCreated && createdNexusBlobStore.Status.Error == ""
		}, timeout, interval).Should(BeTrue())
	})
	It("Should update NexusBlobStore object", func() {
		By("Getting NexusBlobStore object")
		createdNexusBlobStore := &nexusApi.NexusBlobStore{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: nexusBlobStoreCRName, Namespace: namespace}, createdNexusBlobStore)).
			Should(Succeed())

		By("Updating NexusBlobStore object")
		createdNexusBlobStore.Spec.SoftQuota.Limit = 500
		createdNexusBlobStore.Spec.File.Path = "test-blobstore-updated"

		Expect(k8sClient.Update(ctx, createdNexusBlobStore)).Should(Succeed())
		Consistently(func() bool {
			createdNexusBlobStore := &nexusApi.NexusBlobStore{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusBlobStoreCRName, Namespace: namespace}, createdNexusBlobStore)
			if err != nil {
				return false
			}

			return createdNexusBlobStore.Status.Value == common.StatusCreated && createdNexusBlobStore.Status.Error == ""
		}, timeout, interval).Should(BeTrue())
	})
	It("Should delete NexusBlobStore object", func() {
		By("Getting NexusBlobStore object")
		createdNexusBlobStore := &nexusApi.NexusBlobStore{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: nexusBlobStoreCRName, Namespace: namespace}, createdNexusBlobStore)).
			Should(Succeed())

		By("Deleting NexusBlobStore object")
		Expect(k8sClient.Delete(ctx, createdNexusBlobStore)).Should(Succeed())
		Eventually(func() bool {
			createdNexusBlobStore := &nexusApi.NexusBlobStore{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusBlobStoreCRName, Namespace: namespace}, createdNexusBlobStore)
			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})
	It("should fail to connect to S3 without credentials", func() {
		By("By creating a new NexusBlobStore object")
		newNexusBlobStore := &nexusApi.NexusBlobStore{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nexus-blobstore-s3-without-credentials",
				Namespace: namespace,
			},
			Spec: nexusApi.NexusBlobStoreSpec{
				Name: "nexus-blobstore-s3-without-credentials",
				SoftQuota: &nexusApi.SoftQuota{
					Type:  nexusApi.SoftQuotaSpaceUsedQuota,
					Limit: 100,
				},
				S3: &nexusApi.S3{
					Bucket: nexusApi.S3Bucket{
						Name: "test-bucket",
					},
				},
				NexusRef: common.NexusRef{
					Name: nexusCRName,
				},
			},
		}
		Expect(k8sClient.Create(ctx, newNexusBlobStore)).Should(Succeed())
		Eventually(func(g Gomega) {
			createdNexusBlobStore := &nexusApi.NexusBlobStore{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: "nexus-blobstore-s3-without-credentials", Namespace: namespace}, createdNexusBlobStore)
			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(createdNexusBlobStore.Status.Value).Should(Equal(common.StatusError))
			g.Expect(createdNexusBlobStore.Status.Error).ShouldNot(BeEmpty())
		}).WithPolling(time.Second).WithTimeout(timeout).Should(Succeed())
	})
})
