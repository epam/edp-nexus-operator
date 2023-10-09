package role

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

var _ = Describe("NexusRole controller", func() {
	nexusRoleCRName := "nexus-role"
	It("Should create NexusRole object", func() {
		By("By creating a new NexusRole object")
		newNexusRole := &nexusApi.NexusRole{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nexusRoleCRName,
				Namespace: namespace,
			},
			Spec: nexusApi.NexusRoleSpec{
				ID:          "test-role",
				Name:        "test role",
				Description: "test role",
				Privileges:  []string{"nx-blobstores-all"},
				NexusRef: common.NexusRef{
					Name: nexusCRName,
				},
			},
		}
		Expect(k8sClient.Create(ctx, newNexusRole)).Should(Succeed())
		Eventually(func() bool {
			createdNexusRole := &nexusApi.NexusRole{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusRoleCRName, Namespace: namespace}, createdNexusRole)
			if err != nil {
				return false
			}

			return createdNexusRole.Status.Value == common.StatusCreated && createdNexusRole.Status.Error == ""
		}, timeout, interval).Should(BeTrue())
	})
	It("Should update NexusRole object", func() {
		By("Getting NexusRole object")
		createdNexusRole := &nexusApi.NexusRole{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: nexusRoleCRName, Namespace: namespace}, createdNexusRole)).
			Should(Succeed())

		By("Updating NexusRole object")
		createdNexusRole.Spec.Description = "updated description"
		createdNexusRole.Spec.Privileges = append(createdNexusRole.Spec.Privileges, "nx-users-all")

		Expect(k8sClient.Update(ctx, createdNexusRole)).Should(Succeed())
		Consistently(func() bool {
			createdNexusRole := &nexusApi.NexusRole{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusRoleCRName, Namespace: namespace}, createdNexusRole)
			if err != nil {
				return false
			}

			return createdNexusRole.Status.Value == common.StatusCreated && createdNexusRole.Status.Error == ""
		}, timeout, interval).Should(BeTrue())
	})
	It("Should delete NexusRole object", func() {
		By("Getting NexusRole object")
		createdNexusRole := &nexusApi.NexusRole{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: nexusRoleCRName, Namespace: namespace}, createdNexusRole)).
			Should(Succeed())

		By("Deleting NexusRole object")
		Expect(k8sClient.Delete(ctx, createdNexusRole)).Should(Succeed())
		Eventually(func() bool {
			createdNexusRole := &nexusApi.NexusRole{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusRoleCRName, Namespace: namespace}, createdNexusRole)
			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})
})
