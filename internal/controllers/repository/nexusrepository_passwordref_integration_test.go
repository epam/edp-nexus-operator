package repository

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

var (
	goAuthPath    = []string{"spec", "go", "proxy", "httpClient", "authentication"}
	mavenAuthPath = []string{"spec", "maven", "proxy", "httpClient", "authentication"}
)

func secretRefSelector(name, key string) *common.SecretKeySelector {
	s := &common.SecretKeySelector{Key: key}
	s.Name = name

	return s
}

func goProxyWithAuth(name string, auth *nexusApi.HTTPClientAuthentication) *nexusApi.NexusRepository {
	return &nexusApi.NexusRepository{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: nexusApi.NexusRepositorySpec{
			Go: &nexusApi.GoSpec{Proxy: &nexusApi.GoProxyRepository{
				ProxySpec: nexusApi.ProxySpec{
					Name:   name,
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
					NegativeCache: nexusApi.NegativeCache{Enabled: true, TTL: 1440},
					HTTPClient:    nexusApi.HTTPClient{AutoBlock: true, Authentication: auth},
				},
			}},
			NexusRef: common.NexusRef{Name: nexusCRName},
		},
	}
}

func mavenProxyWithAuth(name string, auth *nexusApi.HTTPClientAuthenticationWithPreemptive) *nexusApi.NexusRepository {
	return &nexusApi.NexusRepository{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: nexusApi.NexusRepositorySpec{
			Maven: &nexusApi.MavenSpec{Proxy: &nexusApi.MavenProxyRepository{
				Name:   name,
				Online: true,
				Storage: nexusApi.Storage{
					BlobStoreName:               "default",
					StrictContentTypeValidation: true,
				},
				Proxy: nexusApi.Proxy{
					ContentMaxAge:  1440,
					MetadataMaxAge: 1440,
					RemoteURL:      "https://maven-proxy-url",
				},
				NegativeCache: nexusApi.NegativeCache{Enabled: true, TTL: 1440},
				HTTPClient: nexusApi.HTTPClientWithPreemptiveAuth{
					AutoBlock:      true,
					Authentication: auth,
				},
				Maven: nexusApi.Maven{
					VersionPolicy:      "RELEASE",
					LayoutPolicy:       "STRICT",
					ContentDisposition: "INLINE",
				},
			}},
			NexusRef: common.NexusRef{Name: nexusCRName},
		},
	}
}

// withExplicitEmpty returns the repository as an unstructured object with fields set to an empty string.
// The typed client drops empty strings at marshal; the API server receives them only this way.
func withExplicitEmpty(repository *nexusApi.NexusRepository, authPath []string, fields ...string) *unstructured.Unstructured {
	content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(repository)
	Expect(err).NotTo(HaveOccurred())

	obj := &unstructured.Unstructured{Object: content}
	obj.SetGroupVersionKind(nexusApi.GroupVersion.WithKind("NexusRepository"))

	for _, field := range fields {
		Expect(unstructured.SetNestedField(obj.Object, "", append(append([]string{}, authPath...), field)...)).To(Succeed())
	}

	return obj
}

func repositoryStatus(g Gomega, name string) nexusApi.NexusRepositoryStatus {
	current := &nexusApi.NexusRepository{}
	g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, current)).To(Succeed())

	return current.Status
}

func deleteRepositoryOnCleanup(name string) {
	DeferCleanup(func() {
		current := &nexusApi.NexusRepository{}

		err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, current)
		if k8sErrors.IsNotFound(err) {
			return
		}

		Expect(err).NotTo(HaveOccurred())
		Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, current))).To(Succeed())
		Eventually(func() bool {
			return k8sErrors.IsNotFound(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, current))
		}, timeout, interval).Should(BeTrue())
	})
}

var _ = Describe("NexusRepository httpClient authentication", func() {
	Context("CRD validation", func() {
		DescribeTable("rejects",
			func(repository *nexusApi.NexusRepository, message string) {
				err := k8sClient.Create(ctx, repository)
				if err == nil {
					deleteRepositoryOnCleanup(repository.Name)
				}

				Expect(err).To(HaveOccurred())
				Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
				Expect(err.Error()).To(ContainSubstring(message))
			},
			Entry("password and passwordRef together",
				goProxyWithAuth("secretref-cel-both", &nexusApi.HTTPClientAuthentication{
					Type: "username", Username: "alice", Password: "pw", PasswordRef: secretRefSelector("secretref-cred", "password"),
				}),
				"password and passwordRef are mutually exclusive"),
			Entry("username without a password source",
				goProxyWithAuth("secretref-cel-none", &nexusApi.HTTPClientAuthentication{Type: "username", Username: "alice"}),
				"one of password or passwordRef must be set"),
			Entry("passwordRef with an empty key",
				goProxyWithAuth("secretref-cel-key", &nexusApi.HTTPClientAuthentication{
					Type: "username", Username: "alice", PasswordRef: secretRefSelector("secretref-cred", ""),
				}),
				"passwordRef name and key must not be empty"),
			Entry("passwordRef with an empty name",
				goProxyWithAuth("secretref-cel-name", &nexusApi.HTTPClientAuthentication{
					Type: "username", Username: "alice", PasswordRef: secretRefSelector("", "password"),
				}),
				"passwordRef name and key must not be empty"),
			Entry("passwordRef without username",
				goProxyWithAuth("secretref-cel-nouser", &nexusApi.HTTPClientAuthentication{
					Type: "username", PasswordRef: secretRefSelector("secretref-cred", "password"),
				}),
				"username"),
			Entry("maven username without a password source",
				mavenProxyWithAuth("secretref-cel-maven-user", &nexusApi.HTTPClientAuthenticationWithPreemptive{
					Type: "username", Username: "alice",
				}),
				"username requires password or passwordRef, and a password requires username"),
			Entry("maven passwordRef without username",
				mavenProxyWithAuth("secretref-cel-maven-ref", &nexusApi.HTTPClientAuthenticationWithPreemptive{
					Type: "username", PasswordRef: secretRefSelector("secretref-cred", "password"),
				}),
				"username requires password or passwordRef, and a password requires username"),
			Entry("maven password and passwordRef together",
				mavenProxyWithAuth("secretref-cel-maven-both", &nexusApi.HTTPClientAuthenticationWithPreemptive{
					Type: "username", Username: "alice", Password: "pw", PasswordRef: secretRefSelector("secretref-cred", "password"),
				}),
				"password and passwordRef are mutually exclusive"),
		)

		It("treats an empty password as absent", func() {
			const name = "secretref-cel-empty-password"

			repository := withExplicitEmpty(
				goProxyWithAuth(name, &nexusApi.HTTPClientAuthentication{Type: "username", Username: "alice"}),
				goAuthPath, "password",
			)

			err := k8sClient.Create(ctx, repository)
			if err == nil {
				deleteRepositoryOnCleanup(name)
			}

			Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
			Expect(err.Error()).To(ContainSubstring("one of password or passwordRef must be set"))
		})

		It("accepts an empty password next to passwordRef", func() {
			const name = "secretref-cel-empty-password-ref"

			repository := withExplicitEmpty(
				goProxyWithAuth(name, &nexusApi.HTTPClientAuthentication{
					Type: "username", Username: "alice", PasswordRef: secretRefSelector("secretref-cred", "password"),
				}),
				goAuthPath, "password",
			)
			Expect(k8sClient.Create(ctx, repository)).To(Succeed())
			deleteRepositoryOnCleanup(name)
		})

		It("treats an empty maven username as absent", func() {
			const name = "secretref-cel-empty-maven-user"

			repository := withExplicitEmpty(
				mavenProxyWithAuth(name, &nexusApi.HTTPClientAuthenticationWithPreemptive{
					Type: "username", PasswordRef: secretRefSelector("secretref-cred", "password"),
				}),
				mavenAuthPath, "username",
			)

			err := k8sClient.Create(ctx, repository)
			if err == nil {
				deleteRepositoryOnCleanup(name)
			}

			Expect(k8sErrors.IsInvalid(err)).To(BeTrue(), "got: %v", err)
			Expect(err.Error()).To(ContainSubstring("username requires password or passwordRef, and a password requires username"))
		})

		It("accepts a maven proxy with preemptive authentication only", func() {
			const name = "secretref-maven-preemptive"

			repository := mavenProxyWithAuth(name, &nexusApi.HTTPClientAuthenticationWithPreemptive{
				Type: "username", Preemptive: ptr.To(true),
			})
			Expect(k8sClient.Create(ctx, repository)).To(Succeed())
			deleteRepositoryOnCleanup(name)
		})
	})

	Context("passwordRef", func() {
		It("creates the repository from a Secret and keeps the password out of the resource", func() {
			const (
				name       = "secretref-go-proxy"
				secretName = "secretref-cred"
			)

			Expect(k8sClient.Create(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: namespace},
				Data:       map[string][]byte{"password": []byte("pw-1")},
			})).To(Succeed())

			repository := goProxyWithAuth(name, &nexusApi.HTTPClientAuthentication{
				Type: "username", Username: "alice", PasswordRef: secretRefSelector(secretName, "password"),
			})
			Expect(k8sClient.Create(ctx, repository)).To(Succeed())
			deleteRepositoryOnCleanup(name)

			Eventually(func(g Gomega) {
				status := repositoryStatus(g, name)
				g.Expect(status.Value).To(Equal(common.StatusCreated))
				g.Expect(status.Error).To(BeEmpty())
			}, timeout, interval).Should(Succeed())

			stored := &nexusApi.NexusRepository{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, stored)).To(Succeed())
			Expect(stored.Spec.Go.Proxy.HTTPClient.Authentication.Password).To(BeEmpty()) //nolint:staticcheck // covers the deprecated password field
			Expect(stored.Status.Error).NotTo(ContainSubstring("pw-1"))
		})

		It("reconciles when the referenced Secret appears after the resource", func() {
			const (
				name       = "secretref-late-secret"
				secretName = "secretref-late-cred"
			)

			repository := goProxyWithAuth(name, &nexusApi.HTTPClientAuthentication{
				Type: "username", Username: "alice", PasswordRef: secretRefSelector(secretName, "password"),
			})
			Expect(k8sClient.Create(ctx, repository)).To(Succeed())
			deleteRepositoryOnCleanup(name)

			Eventually(func(g Gomega) {
				status := repositoryStatus(g, name)
				g.Expect(status.Value).To(Equal(common.StatusError))
				g.Expect(status.Error).To(ContainSubstring(secretName))
			}, timeout, interval).Should(Succeed())

			Expect(k8sClient.Create(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: namespace},
				Data:       map[string][]byte{"password": []byte("pw-1")},
			})).To(Succeed())

			// The error requeue is 30 s, longer than the timeout: only the Secret watch can make this pass.
			Eventually(func(g Gomega) {
				status := repositoryStatus(g, name)
				g.Expect(status.Value).To(Equal(common.StatusCreated))
				g.Expect(status.Error).To(BeEmpty())
			}, timeout, interval).Should(Succeed())
		})

		It("reports a Secret value with trailing whitespace without the value", func() {
			const (
				name       = "secretref-newline"
				secretName = "secretref-newline-cred"
			)

			Expect(k8sClient.Create(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: namespace},
				Data:       map[string][]byte{"password": []byte("pw-newline\n")},
			})).To(Succeed())

			Expect(k8sClient.Create(ctx, goProxyWithAuth(name, &nexusApi.HTTPClientAuthentication{
				Type: "username", Username: "alice", PasswordRef: secretRefSelector(secretName, "password"),
			}))).To(Succeed())
			deleteRepositoryOnCleanup(name)

			Eventually(func(g Gomega) {
				status := repositoryStatus(g, name)
				g.Expect(status.Value).To(Equal(common.StatusError))
				g.Expect(status.Error).To(ContainSubstring("has leading or trailing whitespace"))
				g.Expect(status.Error).NotTo(ContainSubstring("pw-newline"))
			}, timeout, interval).Should(Succeed())
		})
	})
})
