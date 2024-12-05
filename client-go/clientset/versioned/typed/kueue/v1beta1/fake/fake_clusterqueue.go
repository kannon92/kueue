/*
Copyright 2022 The Kubernetes Authors.

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
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1beta1 "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	kueuev1beta1 "sigs.k8s.io/kueue/client-go/applyconfiguration/kueue/v1beta1"
)

// FakeClusterQueues implements ClusterQueueInterface
type FakeClusterQueues struct {
	Fake *FakeKueueV1beta1
}

var clusterqueuesResource = v1beta1.SchemeGroupVersion.WithResource("clusterqueues")

var clusterqueuesKind = v1beta1.SchemeGroupVersion.WithKind("ClusterQueue")

// Get takes name of the clusterQueue, and returns the corresponding clusterQueue object, and an error if there is any.
func (c *FakeClusterQueues) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.ClusterQueue, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(clusterqueuesResource, name), &v1beta1.ClusterQueue{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ClusterQueue), err
}

// List takes label and field selectors, and returns the list of ClusterQueues that match those selectors.
func (c *FakeClusterQueues) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.ClusterQueueList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(clusterqueuesResource, clusterqueuesKind, opts), &v1beta1.ClusterQueueList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.ClusterQueueList{ListMeta: obj.(*v1beta1.ClusterQueueList).ListMeta}
	for _, item := range obj.(*v1beta1.ClusterQueueList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusterQueues.
func (c *FakeClusterQueues) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(clusterqueuesResource, opts))
}

// Create takes the representation of a clusterQueue and creates it.  Returns the server's representation of the clusterQueue, and an error, if there is any.
func (c *FakeClusterQueues) Create(ctx context.Context, clusterQueue *v1beta1.ClusterQueue, opts v1.CreateOptions) (result *v1beta1.ClusterQueue, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(clusterqueuesResource, clusterQueue), &v1beta1.ClusterQueue{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ClusterQueue), err
}

// Update takes the representation of a clusterQueue and updates it. Returns the server's representation of the clusterQueue, and an error, if there is any.
func (c *FakeClusterQueues) Update(ctx context.Context, clusterQueue *v1beta1.ClusterQueue, opts v1.UpdateOptions) (result *v1beta1.ClusterQueue, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(clusterqueuesResource, clusterQueue), &v1beta1.ClusterQueue{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ClusterQueue), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeClusterQueues) UpdateStatus(ctx context.Context, clusterQueue *v1beta1.ClusterQueue, opts v1.UpdateOptions) (*v1beta1.ClusterQueue, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(clusterqueuesResource, "status", clusterQueue), &v1beta1.ClusterQueue{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ClusterQueue), err
}

// Delete takes name of the clusterQueue and deletes it. Returns an error if one occurs.
func (c *FakeClusterQueues) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(clusterqueuesResource, name, opts), &v1beta1.ClusterQueue{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeClusterQueues) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(clusterqueuesResource, listOpts)

	_, err := c.Fake.Invokes(action, &v1beta1.ClusterQueueList{})
	return err
}

// Patch applies the patch and returns the patched clusterQueue.
func (c *FakeClusterQueues) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.ClusterQueue, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(clusterqueuesResource, name, pt, data, subresources...), &v1beta1.ClusterQueue{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ClusterQueue), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied clusterQueue.
func (c *FakeClusterQueues) Apply(ctx context.Context, clusterQueue *kueuev1beta1.ClusterQueueApplyConfiguration, opts v1.ApplyOptions) (result *v1beta1.ClusterQueue, err error) {
	if clusterQueue == nil {
		return nil, fmt.Errorf("clusterQueue provided to Apply must not be nil")
	}
	data, err := json.Marshal(clusterQueue)
	if err != nil {
		return nil, err
	}
	name := clusterQueue.Name
	if name == nil {
		return nil, fmt.Errorf("clusterQueue.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(clusterqueuesResource, *name, types.ApplyPatchType, data), &v1beta1.ClusterQueue{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ClusterQueue), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeClusterQueues) ApplyStatus(ctx context.Context, clusterQueue *kueuev1beta1.ClusterQueueApplyConfiguration, opts v1.ApplyOptions) (result *v1beta1.ClusterQueue, err error) {
	if clusterQueue == nil {
		return nil, fmt.Errorf("clusterQueue provided to Apply must not be nil")
	}
	data, err := json.Marshal(clusterQueue)
	if err != nil {
		return nil, err
	}
	name := clusterQueue.Name
	if name == nil {
		return nil, fmt.Errorf("clusterQueue.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(clusterqueuesResource, *name, types.ApplyPatchType, data, "status"), &v1beta1.ClusterQueue{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ClusterQueue), err
}
