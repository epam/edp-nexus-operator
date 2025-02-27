package cleanuppolicy

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

var _ = Describe("NexusCleanupPolicy controller", func() {
	nexusCleanupPolicyCRName := "nexus-policy"
	It("Should create NexusCleanupPolicy object", func() {
		By("By creating a new NexusCleanupPolicy object")
		newNexusCleanupPolicy := &nexusApi.NexusCleanupPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nexusCleanupPolicyCRName,
				Namespace: namespace,
			},
			Spec: nexusApi.NexusCleanupPolicySpec{
				Name:        "test-policy",
				Format:      "go",
				Description: "test policy",
				Criteria: nexusApi.Criteria{
					LastBlobUpdated: 100,
				},
				NexusRef: common.NexusRef{
					Name: nexusCRName,
				},
			},
		}
		Expect(k8sClient.Create(ctx, newNexusCleanupPolicy)).Should(Succeed())
		Eventually(func() bool {
			createdNexusCleanupPolicy := &nexusApi.NexusCleanupPolicy{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusCleanupPolicyCRName, Namespace: namespace}, createdNexusCleanupPolicy)
			if err != nil {
				return false
			}

			return createdNexusCleanupPolicy.Status.Value == common.StatusCreated && createdNexusCleanupPolicy.Status.Error == ""
		}, timeout, interval).Should(BeTrue())
	})
	It("Should update NexusCleanupPolicy object", func() {
		By("Getting NexusCleanupPolicy object")
		createdNexusCleanupPolicy := &nexusApi.NexusCleanupPolicy{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: nexusCleanupPolicyCRName, Namespace: namespace}, createdNexusCleanupPolicy)).
			Should(Succeed())

		By("Updating NexusCleanupPolicy object")
		createdNexusCleanupPolicy.Spec.Description = "updated description"
		createdNexusCleanupPolicy.Spec.Criteria.LastBlobUpdated = 30

		Expect(k8sClient.Update(ctx, createdNexusCleanupPolicy)).Should(Succeed())
		Consistently(func() bool {
			createdNexusCleanupPolicy := &nexusApi.NexusCleanupPolicy{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusCleanupPolicyCRName, Namespace: namespace}, createdNexusCleanupPolicy)
			if err != nil {
				return false
			}

			return createdNexusCleanupPolicy.Status.Value == common.StatusCreated && createdNexusCleanupPolicy.Status.Error == ""
		}, timeout, interval).Should(BeTrue())
	})
	It("Should delete NexusCleanupPolicy object", func() {
		By("Getting NexusCleanupPolicy object")
		createdNexusCleanupPolicy := &nexusApi.NexusCleanupPolicy{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: nexusCleanupPolicyCRName, Namespace: namespace}, createdNexusCleanupPolicy)).
			Should(Succeed())

		By("Deleting NexusCleanupPolicy object")
		Expect(k8sClient.Delete(ctx, createdNexusCleanupPolicy)).Should(Succeed())
		Eventually(func() bool {
			createdNexusCleanupPolicy := &nexusApi.NexusCleanupPolicy{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusCleanupPolicyCRName, Namespace: namespace}, createdNexusCleanupPolicy)
			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})
})
