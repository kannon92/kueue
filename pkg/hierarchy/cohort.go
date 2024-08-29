/*
Copyright 2024 The Kubernetes Authors.

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

package hierarchy

import "k8s.io/apimachinery/pkg/util/sets"

type Cohort[CQ, C nodeBase] struct {
	childCqs sets.Set[CQ]
}

func (c *Cohort[CQ, C]) Parent() C {
	var zero C
	return zero
}

func (c *Cohort[CQ, C]) HasParent() bool {
	return false
}

func (c *Cohort[CQ, C]) ChildCQs() []CQ {
	return c.childCqs.UnsortedList()
}

func NewCohort[CQ, C nodeBase]() Cohort[CQ, C] {
	return Cohort[CQ, C]{childCqs: sets.New[CQ]()}
}

// implement cohortNode interface

func (c *Cohort[CQ, C]) insertClusterQueue(cq CQ) {
	c.childCqs.Insert(cq)
}

func (c *Cohort[CQ, C]) deleteClusterQueue(cq CQ) {
	c.childCqs.Delete(cq)
}

func (c *Cohort[CQ, C]) childCount() int {
	return c.childCqs.Len()
}
