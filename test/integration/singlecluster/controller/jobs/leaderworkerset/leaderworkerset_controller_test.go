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

package leaderworkerset

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta2"
	"sigs.k8s.io/kueue/pkg/controller/constants"
	"sigs.k8s.io/kueue/pkg/controller/jobframework"
	lwscontroller "sigs.k8s.io/kueue/pkg/controller/jobs/leaderworkerset"
	utiltestingapi "sigs.k8s.io/kueue/pkg/util/testing/v1beta2"
	testingleaderworkerset "sigs.k8s.io/kueue/pkg/util/testingjobs/leaderworkerset"
	"sigs.k8s.io/kueue/pkg/workload"
	"sigs.k8s.io/kueue/test/util"
)

const instanceKey = "cloud.provider.com/instance"

var _ = ginkgo.Describe("LeaderWorkerSet controller with LocalQueue default execution time", ginkgo.Ordered, ginkgo.ContinueOnFailure, func() {
	var (
		ns           *corev1.Namespace
		clusterQueue *kueue.ClusterQueue
		localQueue   *kueue.LocalQueue
		onDemand     *kueue.ResourceFlavor
	)

	ginkgo.BeforeAll(func() {
		fwk.StartManager(ctx, cfg, managerAndSchedulerSetup(
			jobframework.WithManageJobsWithoutQueueName(false),
		))
	})
	ginkgo.AfterAll(func() {
		fwk.StopManager(ctx)
	})

	ginkgo.BeforeEach(func() {
		ns = util.CreateNamespaceFromPrefixWithLog(ctx, k8sClient, "lws-exec-time-")

		onDemand = utiltestingapi.MakeResourceFlavor("on-demand").NodeLabel(instanceKey, "on-demand").Obj()
		util.MustCreate(ctx, k8sClient, onDemand)

		clusterQueue = utiltestingapi.MakeClusterQueue("exec-time-cq").
			ResourceGroup(
				*utiltestingapi.MakeFlavorQuotas("on-demand").Resource(corev1.ResourceCPU, "5").Obj(),
			).Obj()
		util.MustCreate(ctx, k8sClient, clusterQueue)
	})

	ginkgo.AfterEach(func() {
		gomega.Expect(util.DeleteNamespace(ctx, k8sClient, ns)).To(gomega.Succeed())
		util.ExpectObjectToBeDeleted(ctx, k8sClient, clusterQueue, true)
		util.ExpectObjectToBeDeleted(ctx, k8sClient, onDemand, true)
	})

	ginkgo.It("should set workload MaximumExecutionTimeSeconds from LocalQueue default", func() {
		localQueue = utiltestingapi.MakeLocalQueue("lq-with-timeout", ns.Name).
			ClusterQueue(clusterQueue.Name).
			MaximumExecutionTimeSeconds(120).
			Obj()
		util.MustCreate(ctx, k8sClient, localQueue)

		lws := testingleaderworkerset.MakeLeaderWorkerSet("lws-no-label", ns.Name).
			Queue(localQueue.Name).
			Request(corev1.ResourceCPU, "1").
			Obj()
		util.MustCreate(ctx, k8sClient, lws)

		// Get the LWS back so we have its UID for workload name lookup.
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(lws), lws)).To(gomega.Succeed())
		}, util.Timeout, util.Interval).Should(gomega.Succeed())

		wlLookupKey := types.NamespacedName{
			Name:      lwscontroller.GetWorkloadName(lwscontroller.GetOwnerUID(lws), lws.Name, "0"),
			Namespace: ns.Name,
		}

		createdWorkload := &kueue.Workload{}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, wlLookupKey, createdWorkload)).To(gomega.Succeed())
			g.Expect(createdWorkload.Spec.MaximumExecutionTimeSeconds).NotTo(gomega.BeNil())
			g.Expect(*createdWorkload.Spec.MaximumExecutionTimeSeconds).To(gomega.Equal(int32(120)))
		}, util.Timeout, util.Interval).Should(gomega.Succeed())
	})

	ginkgo.It("should use job label over LocalQueue default for MaximumExecutionTimeSeconds", func() {
		localQueue = utiltestingapi.MakeLocalQueue("lq-with-timeout-override", ns.Name).
			ClusterQueue(clusterQueue.Name).
			MaximumExecutionTimeSeconds(3600).
			Obj()
		util.MustCreate(ctx, k8sClient, localQueue)

		lws := testingleaderworkerset.MakeLeaderWorkerSet("lws-with-label", ns.Name).
			Queue(localQueue.Name).
			Request(corev1.ResourceCPU, "1").
			Label(constants.MaxExecTimeSecondsLabel, "30").
			Obj()
		util.MustCreate(ctx, k8sClient, lws)

		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(lws), lws)).To(gomega.Succeed())
		}, util.Timeout, util.Interval).Should(gomega.Succeed())

		wlLookupKey := types.NamespacedName{
			Name:      lwscontroller.GetWorkloadName(lwscontroller.GetOwnerUID(lws), lws.Name, "0"),
			Namespace: ns.Name,
		}

		createdWorkload := &kueue.Workload{}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, wlLookupKey, createdWorkload)).To(gomega.Succeed())
			g.Expect(createdWorkload.Spec.MaximumExecutionTimeSeconds).NotTo(gomega.BeNil())
			g.Expect(*createdWorkload.Spec.MaximumExecutionTimeSeconds).To(gomega.Equal(int32(30)))
		}, util.Timeout, util.Interval).Should(gomega.Succeed())
	})

	ginkgo.It("should not set MaximumExecutionTimeSeconds when LocalQueue has no default", func() {
		localQueue = utiltestingapi.MakeLocalQueue("lq-without-timeout", ns.Name).
			ClusterQueue(clusterQueue.Name).
			Obj()
		util.MustCreate(ctx, k8sClient, localQueue)

		lws := testingleaderworkerset.MakeLeaderWorkerSet("lws-no-timeout", ns.Name).
			Queue(localQueue.Name).
			Request(corev1.ResourceCPU, "1").
			Obj()
		util.MustCreate(ctx, k8sClient, lws)

		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(lws), lws)).To(gomega.Succeed())
		}, util.Timeout, util.Interval).Should(gomega.Succeed())

		wlLookupKey := types.NamespacedName{
			Name:      lwscontroller.GetWorkloadName(lwscontroller.GetOwnerUID(lws), lws.Name, "0"),
			Namespace: ns.Name,
		}

		createdWorkload := &kueue.Workload{}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, wlLookupKey, createdWorkload)).To(gomega.Succeed())
		}, util.Timeout, util.Interval).Should(gomega.Succeed())
		gomega.Expect(createdWorkload.Spec.MaximumExecutionTimeSeconds).To(gomega.BeNil())
	})

	ginkgo.It("should admit and run a LeaderWorkerSet with LocalQueue-sourced timeout without equivalency mismatch", func() {
		localQueue = utiltestingapi.MakeLocalQueue("lq-admit-timeout", ns.Name).
			ClusterQueue(clusterQueue.Name).
			MaximumExecutionTimeSeconds(300).
			Obj()
		util.MustCreate(ctx, k8sClient, localQueue)

		lws := testingleaderworkerset.MakeLeaderWorkerSet("lws-admit-timeout", ns.Name).
			Queue(localQueue.Name).
			Request(corev1.ResourceCPU, "1").
			Obj()
		util.MustCreate(ctx, k8sClient, lws)

		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(lws), lws)).To(gomega.Succeed())
		}, util.Timeout, util.Interval).Should(gomega.Succeed())

		wlLookupKey := types.NamespacedName{
			Name:      lwscontroller.GetWorkloadName(lwscontroller.GetOwnerUID(lws), lws.Name, "0"),
			Namespace: ns.Name,
		}

		createdWorkload := &kueue.Workload{}

		ginkgo.By("checking the workload is created with the LocalQueue timeout")
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, wlLookupKey, createdWorkload)).To(gomega.Succeed())
			g.Expect(createdWorkload.Spec.MaximumExecutionTimeSeconds).NotTo(gomega.BeNil())
			g.Expect(*createdWorkload.Spec.MaximumExecutionTimeSeconds).To(gomega.Equal(int32(300)))
		}, util.Timeout, util.Interval).Should(gomega.Succeed())

		ginkgo.By("checking the workload gets admitted by the scheduler")
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, wlLookupKey, createdWorkload)).To(gomega.Succeed())
			g.Expect(workload.IsAdmitted(createdWorkload)).To(gomega.BeTrue())
		}, util.Timeout, util.Interval).Should(gomega.Succeed())

		ginkgo.By("checking the workload remains admitted (no equivalency mismatch)")
		gomega.Consistently(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, wlLookupKey, createdWorkload)).To(gomega.Succeed())
			g.Expect(workload.IsAdmitted(createdWorkload)).To(gomega.BeTrue())
		}, util.ConsistentDuration, util.ShortInterval).Should(gomega.Succeed())
	})
})
