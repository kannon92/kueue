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
// Code generated by lister-gen. DO NOT EDIT.

package v1beta1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1beta1 "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

// LocalQueueLister helps list LocalQueues.
// All objects returned here must be treated as read-only.
type LocalQueueLister interface {
	// List lists all LocalQueues in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1beta1.LocalQueue, err error)
	// LocalQueues returns an object that can list and get LocalQueues.
	LocalQueues(namespace string) LocalQueueNamespaceLister
	LocalQueueListerExpansion
}

// localQueueLister implements the LocalQueueLister interface.
type localQueueLister struct {
	indexer cache.Indexer
}

// NewLocalQueueLister returns a new LocalQueueLister.
func NewLocalQueueLister(indexer cache.Indexer) LocalQueueLister {
	return &localQueueLister{indexer: indexer}
}

// List lists all LocalQueues in the indexer.
func (s *localQueueLister) List(selector labels.Selector) (ret []*v1beta1.LocalQueue, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1beta1.LocalQueue))
	})
	return ret, err
}

// LocalQueues returns an object that can list and get LocalQueues.
func (s *localQueueLister) LocalQueues(namespace string) LocalQueueNamespaceLister {
	return localQueueNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// LocalQueueNamespaceLister helps list and get LocalQueues.
// All objects returned here must be treated as read-only.
type LocalQueueNamespaceLister interface {
	// List lists all LocalQueues in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1beta1.LocalQueue, err error)
	// Get retrieves the LocalQueue from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1beta1.LocalQueue, error)
	LocalQueueNamespaceListerExpansion
}

// localQueueNamespaceLister implements the LocalQueueNamespaceLister
// interface.
type localQueueNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all LocalQueues in the indexer for a given namespace.
func (s localQueueNamespaceLister) List(selector labels.Selector) (ret []*v1beta1.LocalQueue, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1beta1.LocalQueue))
	})
	return ret, err
}

// Get retrieves the LocalQueue from the indexer for a given namespace and name.
func (s localQueueNamespaceLister) Get(name string) (*v1beta1.LocalQueue, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1beta1.Resource("localqueue"), name)
	}
	return obj.(*v1beta1.LocalQueue), nil
}
