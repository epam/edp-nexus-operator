package script

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/epam/edp-nexus-operator/api/common"
	nexusApi "github.com/epam/edp-nexus-operator/api/v1alpha1"
)

var _ = Describe("NexusScript controller", func() {
	nexusScriptCRName := "nexus-script"
	It("Should create NexusScript object", func() {
		By("By creating a new NexusScript object")
		newNexusScript := &nexusApi.NexusScript{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nexusScriptCRName,
				Namespace: namespace,
			},
			Spec: nexusApi.NexusScriptSpec{
				Name:    "test-script",
				Content: "println('test')",
				Payload: "test-payload",
				NexusRef: common.NexusRef{
					Name: nexusCRName,
				},
			},
		}
		Expect(k8sClient.Create(ctx, newNexusScript)).Should(Succeed())
		Eventually(func() bool {
			createdNexusScript := &nexusApi.NexusScript{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusScriptCRName, Namespace: namespace}, createdNexusScript)
			if err != nil {
				return false
			}

			return createdNexusScript.Status.Value == common.StatusCreated &&
				createdNexusScript.Status.Error == "" &&
				createdNexusScript.Status.Executed
		}, timeout, interval).Should(BeTrue())
	})
	It("Should update NexusScript object", func() {
		By("Getting NexusScript object")
		createdNexusScript := &nexusApi.NexusScript{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: nexusScriptCRName, Namespace: namespace}, createdNexusScript)).
			Should(Succeed())

		By("Updating NexusScript object")
		createdNexusScript.Spec.Content = "println('test2')"

		Expect(k8sClient.Update(ctx, createdNexusScript)).Should(Succeed())
		Consistently(func() bool {
			createdNexusScript := &nexusApi.NexusScript{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusScriptCRName, Namespace: namespace}, createdNexusScript)
			if err != nil {
				return false
			}

			return createdNexusScript.Status.Value == common.StatusCreated &&
				createdNexusScript.Status.Error == "" &&
				createdNexusScript.Status.Executed
		}, timeout, interval).Should(BeTrue())
	})
	It("Should delete NexusScript object", func() {
		By("Getting NexusScript object")
		createdNexusScript := &nexusApi.NexusScript{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: nexusScriptCRName, Namespace: namespace}, createdNexusScript)).
			Should(Succeed())

		By("Deleting NexusScript object")
		Expect(k8sClient.Delete(ctx, createdNexusScript)).Should(Succeed())
		Eventually(func() bool {
			createdNexusScript := &nexusApi.NexusScript{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: nexusScriptCRName, Namespace: namespace}, createdNexusScript)
			return k8sErrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})
})
