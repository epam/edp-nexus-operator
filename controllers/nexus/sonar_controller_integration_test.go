package nexus

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

var _ = Describe("Nexus controller", func() {
	const (
		nexusName = "test-nexus"
		namespace = "default"

		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	It("Should create Nexus object with secret auth", func() {
		By("By creating a secret")
		secret := &v1.Secret{
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
		By("By creating a new Nexus object")
		newNexus := &nexusApi.Nexus{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nexusName,
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
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusName, Namespace: namespace}, createdNexus)
			if err != nil {
				return false
			}

			return createdNexus.Status.Connected && createdNexus.Status.Error == ""

		}, timeout, interval).Should(BeTrue())
	})
})
