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

package cache

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/resources"
)

type BudgetResourceGroup struct {
	CoveredResources sets.Set[corev1.ResourceName]
	Flavors          []kueue.ResourceFlavorReference
	// The set of key labels from all flavors.
	// Those keys define the affinity terms of a workload
	// that can be matched against the flavors.
	LabelKeys sets.Set[string]
}

func (rg *BudgetResourceGroup) Clone() BudgetResourceGroup {
	return BudgetResourceGroup{
		CoveredResources: rg.CoveredResources.Clone(),
		Flavors:          rg.Flavors,
		LabelKeys:        rg.LabelKeys.Clone(),
	}
}

type BudgetResourceQuota struct {
	BudgetHours int32
}

func createBudgetResourceQuota(kueueRgs []kueue.BudgetQuotas) map[resources.FlavorResource]BudgetResourceQuota {
	frCount := 0
	for _, rg := range kueueRgs {
		frCount += len(rg.BudgetQuota)
	}
	quotas := make(map[resources.FlavorResource]BudgetResourceQuota, frCount)
	for _, kueueRg := range kueueRgs {
		for _, bq := range kueueRg.BudgetQuota {
			quotas[resources.FlavorResource{Flavor: kueueRg.Name, Resource: bq.Name}] = BudgetResourceQuota{bq.BudgetHours}
		}
	}
	return quotas
}
