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

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	_ JobReconcilerInterface = (*NoopReconciler)(nil)
)

type NoopReconciler struct {
	gvk schema.GroupVersionKind
}

func (r NoopReconciler) Reconcile(context.Context, reconcile.Request) (reconcile.Result, error) {
	return ctrl.Result{}, nil
}

func (r NoopReconciler) SetupWithManager(ctrl.Manager) error {
	ctrl.Log.V(3).Info("Skipped reconciler setup", "gvk", r.gvk)
	return nil
}

func NewNoopReconcilerFactory(gvk schema.GroupVersionKind) ReconcilerFactory {
	return func(client client.Client, record record.EventRecorder, opts ...Option) JobReconcilerInterface {
		return &NoopReconciler{gvk: gvk}
	}
}
