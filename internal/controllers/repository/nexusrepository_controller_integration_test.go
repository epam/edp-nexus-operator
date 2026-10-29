package repository

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

var _ = Describe("NexusRepository controller", func() {
	nexusRepositoryCRName := "nexus-repository"
	It("Should create NexusRepository object", func() {
		By("By creating a new NexusRepository object")
		newNexusRepository := goProxyRepository(nexusRepositoryCRName, namespace, nil)
		newNexusRepository.Spec.Go.Proxy.Name = "go-proxy"
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
	It("Should re-reconcile NexusRepository when the referenced Secret changes", func() {
		const (
			repoName   = "go-proxy-secret-auth"
			secretName = "proxy-auth"
		)

		key := types.NamespacedName{Name: repoName, Namespace: namespace}
		secretKey := types.NamespacedName{Name: secretName, Namespace: namespace}

		eventuallyCreated := func() {
			Eventually(func(g Gomega) {
				got := &nexusApi.NexusRepository{}
				g.Expect(k8sClient.Get(ctx, key, got)).Should(Succeed())
				g.Expect(got.Status.Value).Should(Equal(common.StatusCreated))
				g.Expect(got.Status.Error).Should(BeEmpty())
			}, timeout, interval).Should(Succeed())
		}

		By("Creating the credentials Secret and NexusRepository with Secret references")
		authSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: namespace},
			Data: map[string][]byte{
				"username": []byte("proxy-user"),
				"password": []byte("proxy-password"),
			},
		}
		Expect(k8sClient.Create(ctx, authSecret)).Should(Succeed())

		repo := goProxyRepository(repoName, namespace, &nexusApi.HTTPClientAuthentication{
			Type:                 "username",
			UsernameSecretKeyRef: selector(secretName, "username"),
			PasswordSecretKeyRef: selector(secretName, "password"),
		})
		Expect(k8sClient.Create(ctx, repo)).Should(Succeed())
		eventuallyCreated()

		By("Removing the password key from the Secret")
		Expect(k8sClient.Get(ctx, secretKey, authSecret)).Should(Succeed())
		delete(authSecret.Data, "password")
		Expect(k8sClient.Update(ctx, authSecret)).Should(Succeed())
		Eventually(func(g Gomega) {
			got := &nexusApi.NexusRepository{}
			g.Expect(k8sClient.Get(ctx, key, got)).Should(Succeed())
			g.Expect(got.Status.Value).Should(Equal(common.StatusError))
			g.Expect(got.Status.Error).Should(ContainSubstring(secretName))
		}, timeout, interval).Should(Succeed())

		By("Restoring the password key")
		Expect(k8sClient.Get(ctx, secretKey, authSecret)).Should(Succeed())
		authSecret.Data["password"] = []byte("proxy-password")
		Expect(k8sClient.Update(ctx, authSecret)).Should(Succeed())
		eventuallyCreated()

		By("Cleaning up")
		Expect(k8sClient.Delete(ctx, repo)).Should(Succeed())
	})
	It("Should validate HTTP client authentication of a proxy repository", func() {
		const (
			wantPasswordErr = "exactly one of password or passwordSecretKeyRef must be set"
			wantUsernameErr = "exactly one of username or usernameSecretKeyRef must be set"
			wantRefErr      = "name and key of the secret must not be empty"
		)

		tests := []struct {
			name    string
			auth    *nexusApi.HTTPClientAuthentication
			wantErr string
		}{
			{name: "no authentication"},
			{
				name: "username reference with empty secret name",
				auth: &nexusApi.HTTPClientAuthentication{
					Type: "username", UsernameSecretKeyRef: selector("", "user"), Password: "p",
				},
				wantErr: wantRefErr,
			},
			{
				name: "password reference with empty key",
				auth: &nexusApi.HTTPClientAuthentication{
					Type: "username", Username: "u", PasswordSecretKeyRef: selector("proxy-auth", ""),
				},
				wantErr: wantRefErr,
			},
			{
				name: "inline credentials",
				auth: &nexusApi.HTTPClientAuthentication{Type: "username", Username: "u", Password: "p"},
			},
			{
				name: "secret references",
				auth: &nexusApi.HTTPClientAuthentication{
					Type: "username", UsernameSecretKeyRef: selector("proxy-auth", "user"), PasswordSecretKeyRef: selector("proxy-auth", "pass"),
				},
			},
			{
				name: "inline username with secret password",
				auth: &nexusApi.HTTPClientAuthentication{
					Type: "ntlm", Username: "u", PasswordSecretKeyRef: selector("proxy-auth", "pass"),
				},
			},
			{
				name: "password and password reference",
				auth: &nexusApi.HTTPClientAuthentication{
					Type: "username", Username: "u", Password: "p", PasswordSecretKeyRef: selector("proxy-auth", "pass"),
				},
				wantErr: wantPasswordErr,
			},
			{
				name: "username and username reference",
				auth: &nexusApi.HTTPClientAuthentication{
					Type: "username", Username: "u", UsernameSecretKeyRef: selector("proxy-auth", "user"), Password: "p",
				},
				wantErr: wantUsernameErr,
			},
			{
				name:    "no password source",
				auth:    &nexusApi.HTTPClientAuthentication{Type: "username", UsernameSecretKeyRef: selector("proxy-auth", "user")},
				wantErr: wantPasswordErr,
			},
			{
				name:    "no username source",
				auth:    &nexusApi.HTTPClientAuthentication{Type: "username", PasswordSecretKeyRef: selector("proxy-auth", "pass")},
				wantErr: wantUsernameErr,
			},
		}

		for _, tt := range tests {
			By(tt.name)

			// A dry run applies the schema and CEL rules without persisting the object.
			err := k8sClient.Create(ctx, goProxyRepository("go-proxy-validation", namespace, tt.auth), client.DryRunAll)

			if tt.wantErr == "" {
				Expect(err).ShouldNot(HaveOccurred())

				continue
			}

			Expect(err).Should(MatchError(ContainSubstring(tt.wantErr)))
		}
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
