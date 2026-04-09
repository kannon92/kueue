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

package jobframework

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta2"
)

// LocalQueueDefaults holds fields that a LocalQueue can default onto a Workload.
// This is an internal type, not part of the Kueue API. It provides a centralized
// place to collect all LocalQueue-level defaults, making it straightforward to
// extend when new defaulting fields are added to LocalQueueSpec (e.g. priority,
// TAS configuration).
type LocalQueueDefaults struct {
	MaximumExecutionTimeSeconds *int32
}

// GetLocalQueueDefaults looks up the LocalQueue by name and namespace, and
// extracts defaulting fields into a LocalQueueDefaults struct. Returns nil if
// the queue name is empty or the LocalQueue cannot be found.
func GetLocalQueueDefaults(ctx context.Context, c client.Client, queueName kueue.LocalQueueName, namespace string) *LocalQueueDefaults {
	log := ctrl.LoggerFrom(ctx)

	if queueName == "" {
		return nil
	}

	var lq kueue.LocalQueue
	if err := c.Get(ctx, types.NamespacedName{Name: string(queueName), Namespace: namespace}, &lq); err != nil {
		log.V(3).Info("Could not look up LocalQueue for defaults", "queue", queueName, "error", err)
		return nil
	}

	defaults := &LocalQueueDefaults{}
	if lq.Spec.WorkloadDefaults != nil {
		defaults.MaximumExecutionTimeSeconds = lq.Spec.WorkloadDefaults.MaximumExecutionTimeSeconds
	}
	return defaults
}

// ApplyLocalQueueDefaults applies LocalQueue defaults to the workload.
// Fields already set on the workload (e.g. from job labels) are not overridden.
func ApplyLocalQueueDefaults(defaults *LocalQueueDefaults, wl *kueue.Workload) {
	if defaults == nil {
		return
	}

	if wl.Spec.MaximumExecutionTimeSeconds == nil && defaults.MaximumExecutionTimeSeconds != nil {
		wl.Spec.MaximumExecutionTimeSeconds = defaults.MaximumExecutionTimeSeconds
	}
}
