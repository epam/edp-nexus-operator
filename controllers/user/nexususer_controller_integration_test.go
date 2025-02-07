package user

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/datadrivers/go-nexus-client/nexus3"
	nexus3client "github.com/datadrivers/go-nexus-client/nexus3/pkg/client"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

var _ = Describe("NexusUser controller", func() {
	nexusUserCRName := "nexus-user"
	nexusUserSecretName := "nexus-user-secret"

	It("Should create NexusUser object", func() {
		By("By user secret")
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nexusUserSecretName,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"password": []byte("password"),
			},
		}
		Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
		By("By creating a new NexusUser object")
		newNexusUser := &nexusApi.NexusUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nexusUserCRName,
				Namespace: namespace,
			},
			Spec: nexusApi.NexusUserSpec{
				ID:        "test-user",
				FirstName: "user-first-name",
				LastName:  "user-last-name",
				Email:     "user-email@gmail.com",
				Secret:    "$nexus-user-secret:password",
				Status:    nexusApi.UserStatusActive,
				Roles:     []string{"nx-admin"},
				NexusRef: common.NexusRef{
					Name: nexusCRName,
				},
			},
		}
		Expect(k8sClient.Create(ctx, newNexusUser)).Should(Succeed())
		Eventually(func() bool {
			createdNexusUser := &nexusApi.NexusUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusUserCRName, Namespace: namespace}, createdNexusUser)
			if err != nil {
				return false
			}

			return createdNexusUser.Status.Value == common.StatusCreated && createdNexusUser.Status.Error == ""
		}, timeout, interval).Should(BeTrue())
	})
	It("Should update NexusUser object", func() {
		By("Getting NexusUser object")
		createdNexusUser := &nexusApi.NexusUser{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: nexusUserCRName, Namespace: namespace}, createdNexusUser)).
			Should(Succeed())

		By("Updating NexusUser object")
		createdNexusUser.Spec.FirstName = "updated first name"

		Expect(k8sClient.Update(ctx, createdNexusUser)).Should(Succeed())
		Consistently(func() bool {
			updatedNexusUser := &nexusApi.NexusUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusUserCRName, Namespace: namespace}, updatedNexusUser)
			if err != nil {
				return false
			}

			return updatedNexusUser.Status.Value == common.StatusCreated && updatedNexusUser.Status.Error == ""
		}, timeout, interval).Should(BeTrue())
	})
	It("Should update user password", func() {
		By("Getting NexusUser secret")
		secret := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: nexusUserSecretName, Namespace: namespace}, secret)).Should(Succeed())

		By("Updating NexusUser secret")
		secret.Data["password"] = []byte("updated-password")
		Expect(k8sClient.Update(ctx, secret)).Should(Succeed())

		Eventually(func(g Gomega) {
			cl := nexus3.NewClient(nexus3client.Config{
				URL:      nexusUrl,
				Username: "test-user",
				Password: "updated-password",
			})

			_, err := cl.Security.User.Get("test-user")

			g.Expect(err).ShouldNot(HaveOccurred(), "User should be able to login with updated password")
		}).WithTimeout(time.Second * 10).WithPolling(time.Second).Should(Succeed())
	})
	It("Should delete NexusUser object", func() {
		By("Getting NexusUser object")
		createdNexusUser := &nexusApi.NexusUser{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: nexusUserCRName, Namespace: namespace}, createdNexusUser)).
			Should(Succeed())

		By("Deleting NexusUser object")
		Expect(k8sClient.Delete(ctx, createdNexusUser)).Should(Succeed())
		By("Waiting for NexusUser to be deleted")
		time.Sleep(time.Second)
		Eventually(func() bool {
			createdNexusUser := &nexusApi.NexusUser{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusUserCRName, Namespace: namespace}, createdNexusUser)
			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})
})
