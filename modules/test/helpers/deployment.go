/*
Copyright 2022 Red Hat
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helpers

import (
	"encoding/json"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
)

// GetDeployment -
func (tc *TestHelper) GetDeployment(name types.NamespacedName) *appsv1.Deployment {
	deployment := &appsv1.Deployment{}
	gomega.Eventually(func(g gomega.Gomega) {
		g.Expect(tc.K8sClient.Get(tc.Ctx, name, deployment)).Should(gomega.Succeed())
	}, tc.Timeout, tc.Interval).Should(gomega.Succeed())

	return deployment
}

// ListDeployments -
func (tc *TestHelper) ListDeployments(namespace string) *appsv1.DeploymentList {
	deployments := &appsv1.DeploymentList{}
	gomega.Expect(tc.K8sClient.List(tc.Ctx, deployments, client.InNamespace(namespace))).Should(gomega.Succeed())

	return deployments
}

// SimulateDeploymentReplicaReady -
func (tc *TestHelper) SimulateDeploymentReplicaReady(name types.NamespacedName) {
	gomega.Eventually(func(g gomega.Gomega) {
		deployment := tc.GetDeployment(name)

		deployment.Status.Replicas = 1
		deployment.Status.ReadyReplicas = 1
		g.Expect(tc.K8sClient.Status().Update(tc.Ctx, deployment)).To(gomega.Succeed())
	}, tc.Timeout, tc.Interval).Should(gomega.Succeed())

	tc.Logger.Info("Simulated Deployment success", "on", name)
}

// SimulateDeploymentReadyWithPods -
func (tc *TestHelper) SimulateDeploymentReadyWithPods(name types.NamespacedName, networkIPs map[string][]string) {
	depl := tc.GetDeployment(name)
	for i := 0; i < int(*depl.Spec.Replicas); i++ {
		pod := &corev1.Pod{
			ObjectMeta: depl.Spec.Template.ObjectMeta,
			Spec:       depl.Spec.Template.Spec,
		}
		pod.ObjectMeta.Namespace = name.Namespace
		pod.ObjectMeta.GenerateName = name.Name
		// NOTE(gibi): If there is a mount that refers to a volume created via
		// persistent volume claim then that mount won't have a corresponding
		// volume created in EnvTest as we are not simulating the k8s volume
		// claim logic here at the moment. Therefore the Pod create would fail
		// with a missing volume. So to avoid that we remove every mount and
		// volume from the pod we create here.
		pod.Spec.Volumes = []corev1.Volume{}
		pod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{}

		var netStatus []networkv1.NetworkStatus
		for network, IPs := range networkIPs {
			netStatus = append(
				netStatus,
				networkv1.NetworkStatus{
					Name: network,
					IPs:  IPs,
				},
			)
		}
		netStatusAnnotation, err := json.Marshal(netStatus)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		pod.Annotations[networkv1.NetworkStatusAnnot] = string(netStatusAnnotation)

		gomega.Expect(tc.K8sClient.Create(tc.Ctx, pod)).Should(gomega.Succeed())
	}

	gomega.Eventually(func(g gomega.Gomega) {
		depl := tc.GetDeployment(name)
		depl.Status.Replicas = 1
		depl.Status.ReadyReplicas = 1
		g.Expect(tc.K8sClient.Status().Update(tc.Ctx, depl)).To(gomega.Succeed())

	}, tc.Timeout, tc.Interval).Should(gomega.Succeed())

	tc.Logger.Info("Simulated deployment success", "on", name)
}
