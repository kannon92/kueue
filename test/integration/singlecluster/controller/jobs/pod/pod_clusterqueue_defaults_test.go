/*
Copyright The Kubernetes Authors.

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

package pod

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta2"
	controllerconstants "sigs.k8s.io/kueue/pkg/controller/constants"
	"sigs.k8s.io/kueue/pkg/controller/jobframework"
	"sigs.k8s.io/kueue/pkg/features"
	utiltestingapi "sigs.k8s.io/kueue/pkg/util/testing/v1beta2"
	testingpod "sigs.k8s.io/kueue/pkg/util/testingjobs/pod"
	"sigs.k8s.io/kueue/test/util/behavioral"
)

var _ = ginkgo.Describe("Pod controller with ClusterQueue default execution time", ginkgo.Label("job:pod", "area:jobs"), ginkgo.Ordered, ginkgo.ContinueOnFailure, func() {
	var (
		ns         *corev1.Namespace
		onDemand   *kueue.ResourceFlavor
		cqWithTime *kueue.ClusterQueue
		cqNoTime   *kueue.ClusterQueue
		lqWithTime *kueue.LocalQueue
		lqNoTime   *kueue.LocalQueue
	)

	ginkgo.BeforeAll(func() {
		features.SetFeatureGateDuringTest(ginkgo.GinkgoTB(), features.ClusterQueueMaxExecutionTime, true)
		fwk.StartManager(ctx, cfg, managerSetup(false, false, nil,
			jobframework.WithManageJobsWithoutQueueName(false),
		))
	})
	ginkgo.AfterAll(func() {
		fwk.StopManager(ctx)
		features.SetFeatureGateDuringTest(ginkgo.GinkgoTB(), features.ClusterQueueMaxExecutionTime, false)
	})

	ginkgo.BeforeEach(func() {
		ns = behavioral.CreateNamespaceFromPrefixWithLog(ctx, k8sClient, "pod-exec-time-")

		onDemand = utiltestingapi.MakeResourceFlavor("on-demand").NodeLabel(instanceKey, "on-demand").Obj()
		behavioral.MustCreate(ctx, k8sClient, onDemand)

		cqWithTime = utiltestingapi.MakeClusterQueue("exec-time-cq").
			ResourceGroup(
				*utiltestingapi.MakeFlavorQuotas("on-demand").Resource(corev1.ResourceCPU, "5").Obj(),
			).
			MaximumExecutionTimeSeconds(120).
			Obj()
		behavioral.MustCreate(ctx, k8sClient, cqWithTime)

		cqNoTime = utiltestingapi.MakeClusterQueue("exec-time-cq-none").
			ResourceGroup(
				*utiltestingapi.MakeFlavorQuotas("on-demand").Resource(corev1.ResourceCPU, "5").Obj(),
			).Obj()
		behavioral.MustCreate(ctx, k8sClient, cqNoTime)

		lqWithTime = utiltestingapi.MakeLocalQueue("lq-with-timeout", ns.Name).
			ClusterQueue(cqWithTime.Name).
			Obj()
		behavioral.MustCreate(ctx, k8sClient, lqWithTime)

		lqNoTime = utiltestingapi.MakeLocalQueue("lq-without-timeout", ns.Name).
			ClusterQueue(cqNoTime.Name).
			Obj()
		behavioral.MustCreate(ctx, k8sClient, lqNoTime)
	})

	ginkgo.AfterEach(func() {
		gomega.Expect(behavioral.DeleteNamespace(ctx, k8sClient, ns)).To(gomega.Succeed())
		behavioral.ExpectObjectToBeDeleted(ctx, k8sClient, cqWithTime, true)
		behavioral.ExpectObjectToBeDeleted(ctx, k8sClient, cqNoTime, true)
		behavioral.ExpectObjectToBeDeleted(ctx, k8sClient, onDemand, true)
	})

	ginkgo.It("should set workload MaximumExecutionTimeSeconds from ClusterQueue default", func() {
		pod := testingpod.MakePod("pod-no-label", ns.Name).
			Queue(lqWithTime.Name).
			GroupNameLabel("group-cq-default").
			GroupTotalCount("1").
			Request(corev1.ResourceCPU, "1").
			Obj()
		behavioral.MustCreate(ctx, k8sClient, pod)

		createdWorkload := &kueue.Workload{}
		wlLookupKey := types.NamespacedName{
			Name:      "group-cq-default",
			Namespace: ns.Name,
		}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, wlLookupKey, createdWorkload)).To(gomega.Succeed())
			g.Expect(createdWorkload.Spec.MaximumExecutionTimeSeconds).NotTo(gomega.BeNil())
			g.Expect(*createdWorkload.Spec.MaximumExecutionTimeSeconds).To(gomega.Equal(int32(120)))
		}, behavioral.Timeout, behavioral.Interval).Should(gomega.Succeed())
	})

	ginkgo.It("should use pod label over ClusterQueue default for MaximumExecutionTimeSeconds", func() {
		pod := testingpod.MakePod("pod-with-label", ns.Name).
			Queue(lqWithTime.Name).
			GroupNameLabel("group-label-override").
			GroupTotalCount("1").
			Label(controllerconstants.MaxExecTimeSecondsLabel, "30").
			Request(corev1.ResourceCPU, "1").
			Obj()
		behavioral.MustCreate(ctx, k8sClient, pod)

		createdWorkload := &kueue.Workload{}
		wlLookupKey := types.NamespacedName{
			Name:      "group-label-override",
			Namespace: ns.Name,
		}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, wlLookupKey, createdWorkload)).To(gomega.Succeed())
			g.Expect(createdWorkload.Spec.MaximumExecutionTimeSeconds).NotTo(gomega.BeNil())
			g.Expect(*createdWorkload.Spec.MaximumExecutionTimeSeconds).To(gomega.Equal(int32(30)))
		}, behavioral.Timeout, behavioral.Interval).Should(gomega.Succeed())
	})

	ginkgo.It("should not set MaximumExecutionTimeSeconds when ClusterQueue has no default", func() {
		pod := testingpod.MakePod("pod-no-timeout", ns.Name).
			Queue(lqNoTime.Name).
			GroupNameLabel("group-no-timeout").
			GroupTotalCount("1").
			Request(corev1.ResourceCPU, "1").
			Obj()
		behavioral.MustCreate(ctx, k8sClient, pod)

		createdWorkload := &kueue.Workload{}
		wlLookupKey := types.NamespacedName{
			Name:      "group-no-timeout",
			Namespace: ns.Name,
		}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, wlLookupKey, createdWorkload)).To(gomega.Succeed())
		}, behavioral.Timeout, behavioral.Interval).Should(gomega.Succeed())
		gomega.Expect(createdWorkload.Spec.MaximumExecutionTimeSeconds).To(gomega.BeNil())
	})
})

var _ = ginkgo.Describe("Pod controller with ClusterQueue default execution time disabled", ginkgo.Label("job:pod", "area:jobs"), ginkgo.Ordered, ginkgo.ContinueOnFailure, func() {
	var (
		ns         *corev1.Namespace
		onDemand   *kueue.ResourceFlavor
		cqWithTime *kueue.ClusterQueue
		lqWithTime *kueue.LocalQueue
	)

	ginkgo.BeforeAll(func() {
		features.SetFeatureGateDuringTest(ginkgo.GinkgoTB(), features.ClusterQueueMaxExecutionTime, false)
		fwk.StartManager(ctx, cfg, managerSetup(false, false, nil,
			jobframework.WithManageJobsWithoutQueueName(false),
		))
	})
	ginkgo.AfterAll(func() {
		fwk.StopManager(ctx)
	})

	ginkgo.BeforeEach(func() {
		ns = behavioral.CreateNamespaceFromPrefixWithLog(ctx, k8sClient, "pod-exec-time-off-")

		onDemand = utiltestingapi.MakeResourceFlavor("on-demand").NodeLabel(instanceKey, "on-demand").Obj()
		behavioral.MustCreate(ctx, k8sClient, onDemand)

		// The webhook rejects workloadDefaults while the gate is disabled, so the
		// ClusterQueue is created with the gate enabled and the gate is disabled after.
		features.SetFeatureGateDuringTest(ginkgo.GinkgoTB(), features.ClusterQueueMaxExecutionTime, true)
		cqWithTime = utiltestingapi.MakeClusterQueue("exec-time-cq-off").
			ResourceGroup(
				*utiltestingapi.MakeFlavorQuotas("on-demand").Resource(corev1.ResourceCPU, "5").Obj(),
			).
			MaximumExecutionTimeSeconds(120).
			Obj()
		behavioral.MustCreate(ctx, k8sClient, cqWithTime)
		features.SetFeatureGateDuringTest(ginkgo.GinkgoTB(), features.ClusterQueueMaxExecutionTime, false)

		lqWithTime = utiltestingapi.MakeLocalQueue("lq-with-timeout", ns.Name).
			ClusterQueue(cqWithTime.Name).
			Obj()
		behavioral.MustCreate(ctx, k8sClient, lqWithTime)
	})

	ginkgo.AfterEach(func() {
		gomega.Expect(behavioral.DeleteNamespace(ctx, k8sClient, ns)).To(gomega.Succeed())
		behavioral.ExpectObjectToBeDeleted(ctx, k8sClient, cqWithTime, true)
		behavioral.ExpectObjectToBeDeleted(ctx, k8sClient, onDemand, true)
	})

	ginkgo.It("should not set MaximumExecutionTimeSeconds when feature gate is disabled", func() {
		pod := testingpod.MakePod("pod-gate-off", ns.Name).
			Queue(lqWithTime.Name).
			GroupNameLabel("group-gate-off").
			GroupTotalCount("1").
			Request(corev1.ResourceCPU, "1").
			Obj()
		behavioral.MustCreate(ctx, k8sClient, pod)

		createdWorkload := &kueue.Workload{}
		wlLookupKey := types.NamespacedName{
			Name:      "group-gate-off",
			Namespace: ns.Name,
		}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, wlLookupKey, createdWorkload)).To(gomega.Succeed())
		}, behavioral.Timeout, behavioral.Interval).Should(gomega.Succeed())
		gomega.Expect(createdWorkload.Spec.MaximumExecutionTimeSeconds).To(gomega.BeNil())
	})
})
