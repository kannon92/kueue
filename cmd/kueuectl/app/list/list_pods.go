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

package list

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kubectlget "k8s.io/kubectl/pkg/cmd/get"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/kueue/client-go/clientset/versioned/scheme"
	"sigs.k8s.io/kueue/cmd/kueuectl/app/util"
	kueuejob "sigs.k8s.io/kueue/pkg/controller/jobs/job"
	kueuejobset "sigs.k8s.io/kueue/pkg/controller/jobs/jobset"
	kueuemxjob "sigs.k8s.io/kueue/pkg/controller/jobs/kubeflow/jobs/mxjob"
	kueuepaddlejob "sigs.k8s.io/kueue/pkg/controller/jobs/kubeflow/jobs/paddlejob"
	kueuepytorchjob "sigs.k8s.io/kueue/pkg/controller/jobs/kubeflow/jobs/pytorchjob"
	kueuetfjob "sigs.k8s.io/kueue/pkg/controller/jobs/kubeflow/jobs/tfjob"
	kueuexgboostjob "sigs.k8s.io/kueue/pkg/controller/jobs/kubeflow/jobs/xgboostjob"
	kueuempijob "sigs.k8s.io/kueue/pkg/controller/jobs/mpijob"
	"sigs.k8s.io/kueue/pkg/controller/jobs/pod"
	kueueraycluster "sigs.k8s.io/kueue/pkg/controller/jobs/raycluster"
	kueuerayjob "sigs.k8s.io/kueue/pkg/controller/jobs/rayjob"
)

const (
	podLong = `Lists all pods that matches the given criteria: Should be part of the specified Job kind,
belonging to the specified namespace, matching
the label selector or the field selector.`
	podExample = `  # List Pods
kueuectl list pods --for job/job-name`
)

var jobsWithPodLabelSelector = []JobWithPodLabelSelector{
	&kueuejob.Job{},
	&kueuejobset.JobSet{},
	&kueuemxjob.JobControl{},
	&kueuepaddlejob.JobControl{},
	&kueuetfjob.JobControl{},
	&kueuepytorchjob.JobControl{},
	&kueuexgboostjob.JobControl{},
	&kueuempijob.MPIJob{},
	&pod.Pod{},
	&kueueraycluster.RayCluster{},
	&kueuerayjob.RayJob{},
}

type PodOptions struct {
	PrintFlags *genericclioptions.PrintFlags

	Limit                  int64
	AllNamespaces          bool
	ServerPrint            bool
	Namespace              string
	LabelSelector          string
	FieldSelector          string
	UserSpecifiedForObject string
	ForName                string
	ForGVK                 schema.GroupVersionKind
	ForObject              *unstructured.Unstructured
	PodLabelSelector       string

	Clientset k8s.Interface

	genericiooptions.IOStreams
}

type JobWithPodLabelSelector interface {
	Object() client.Object
	GVK() schema.GroupVersionKind
	PodLabelSelector() string
}

func NewPodOptions(streams genericiooptions.IOStreams) *PodOptions {
	return &PodOptions{
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(scheme.Scheme),
		IOStreams:  streams,
	}
}

func NewPodCmd(clientGetter util.ClientGetter, streams genericiooptions.IOStreams) *cobra.Command {
	o := NewPodOptions(streams)

	cmd := &cobra.Command{
		Use:                   "pods --for TYPE[.API-GROUP]/NAME",
		DisableFlagsInUseLine: true,
		Aliases:               []string{"po"},
		Short:                 "List Pods belong to a Job Kind",
		Long:                  podLong,
		Example:               podExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			err := o.Complete(cmd, clientGetter)
			if err != nil {
				return err
			}
			if o.ForObject == nil {
				return nil
			}
			if len(o.PodLabelSelector) == 0 {
				return nil
			}
			return o.Run(cmd.Context(), clientGetter)
		},
	}

	o.PrintFlags.AddFlags(cmd)

	addAllNamespacesFlagVar(cmd, &o.AllNamespaces)
	addFieldSelectorFlagVar(cmd, &o.FieldSelector)
	addLabelSelectorFlagVar(cmd, &o.LabelSelector)
	addForObjectFlagVar(cmd, &o.UserSpecifiedForObject)

	_ = cmd.MarkFlagRequired("for")

	return cmd
}

// Complete takes the command arguments and infers any remaining options.
func (o *PodOptions) Complete(cmd *cobra.Command, clientGetter util.ClientGetter) error {
	var err error

	o.Limit, err = listRequestLimit()
	if err != nil {
		return err
	}

	outputOption := ptr.Deref(o.PrintFlags.OutputFormat, "")
	if outputOption == "" || strings.Contains(outputOption, "wide") {
		o.ServerPrint = true
	}

	o.Namespace, _, err = clientGetter.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	o.Clientset, err = clientGetter.K8sClientSet()
	if err != nil {
		return err
	}

	mapper, err := clientGetter.ToRESTMapper()
	if err != nil {
		return err
	}
	var found bool
	o.ForGVK, o.ForName, found, err = decodeResourceTypeName(mapper, o.UserSpecifiedForObject)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("invalid value '%s' used in --for flag; value must be in the format TYPE[.API-GROUP]/NAME", o.UserSpecifiedForObject)
	}

	infos, err := o.getForObjectInfos(clientGetter)
	if err != nil {
		return err
	}

	if len(infos) == 0 {
		o.printNoResourcesFound()
		return nil
	}

	o.ForObject, err = o.getForObject(infos)
	if err != nil {
		return err
	}

	o.PodLabelSelector, err = o.getPodLabelSelector()
	if err != nil {
		return err
	}

	if len(o.PodLabelSelector) == 0 {
		o.printNoResourcesFound()
		return nil
	}

	return nil
}

// getForObjectInfos builds and executes a dynamic client query for a resource specified in --for
func (o *PodOptions) getForObjectInfos(clientGetter util.ClientGetter) ([]*resource.Info, error) {
	r := clientGetter.NewResourceBuilder().
		Unstructured().
		NamespaceParam(o.Namespace).
		DefaultNamespace().
		AllNamespaces(o.AllNamespaces).
		FieldSelectorParam(fmt.Sprintf("metadata.name=%s", o.ForName)).
		ResourceTypeOrNameArgs(true, o.ForGVK.Kind).
		ContinueOnError().
		Latest().
		Flatten().
		Do()

	if r == nil {
		return nil, fmt.Errorf("Error building client for: %s/%s", o.ForGVK.Kind, o.ForName)
	}

	if err := r.Err(); err != nil {
		return nil, err
	}

	infos, err := r.Infos()
	if err != nil {
		return nil, err
	}

	return infos, nil
}

func (o *PodOptions) getForObject(infos []*resource.Info) (*unstructured.Unstructured, error) {
	job, ok := infos[0].Object.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("Invalid object %+v. Unexpected type %T", job, infos[0].Object)
	}

	return job, nil
}

func (o *PodOptions) getJobController() JobWithPodLabelSelector {
	for _, jobController := range jobsWithPodLabelSelector {
		if jobController.GVK() == o.ForGVK {
			return jobController
		}
	}
	return nil
}

// getPodLabelSelector returns the podLabels used as a standard selector for jobs
func (o *PodOptions) getPodLabelSelector() (string, error) {
	jobController := o.getJobController()
	if jobController == nil {
		return "", fmt.Errorf("unsupported kind: %s", o.ForObject.GetKind())
	}

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(o.ForObject.UnstructuredContent(), jobController.Object())
	if err != nil {
		return "", fmt.Errorf("failed to convert unstructured object: %w", err)
	}

	return jobController.PodLabelSelector(), nil
}

type trackingWriterWrapper struct {
	Delegate io.Writer
	Written  int
}

func (t *trackingWriterWrapper) Write(p []byte) (n int, err error) {
	t.Written += len(p)
	return t.Delegate.Write(p)
}

// Run prints the pods for a specific Job
func (o *PodOptions) Run(ctx context.Context, clientGetter util.ClientGetter) error {
	trackingWriter := &trackingWriterWrapper{Delegate: o.Out}
	tabWriter := printers.GetNewTabWriter(trackingWriter)

	infos, err := o.getPodsInfos(clientGetter)
	if err != nil {
		return err
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return err
	}

	for _, pod := range infos {
		if err = printer.PrintObj(pod.Object, tabWriter); err != nil {
			return err
		}
	}

	if err = tabWriter.Flush(); err != nil {
		return err
	}

	if trackingWriter.Written == 0 {
		o.printNoResourcesFound()
	}

	return nil
}

func (o *PodOptions) ToPrinter() (printers.ResourcePrinterFunc, error) {
	if o.ServerPrint {
		tablePrinter := printers.NewTablePrinter(printers.PrintOptions{
			NoHeaders:     false,
			WithNamespace: o.AllNamespaces,
			WithKind:      false,
			Wide:          ptr.Deref(o.PrintFlags.OutputFormat, "") == "wide",
			ShowLabels:    false,
			ColumnLabels:  nil,
		})

		printer := &kubectlget.TablePrinter{Delegate: tablePrinter}

		return printer.PrintObj, nil
	}

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return nil, err
	}

	return printer.PrintObj, nil
}

// getPodsInfos gets the pods raw infos directly from the API server
func (o *PodOptions) getPodsInfos(clientGetter util.ClientGetter) ([]*resource.Info, error) {
	namespace := o.Namespace
	if o.AllNamespaces {
		namespace = ""
	}

	podLabelSelector := o.PodLabelSelector
	if len(o.LabelSelector) != 0 {
		podLabelSelector = "," + o.PodLabelSelector
	}

	r := clientGetter.NewResourceBuilder().Unstructured().
		NamespaceParam(namespace).DefaultNamespace().AllNamespaces(o.AllNamespaces).
		FieldSelectorParam(o.FieldSelector).
		LabelSelectorParam(o.LabelSelector+podLabelSelector).
		ResourceTypeOrNameArgs(true, "pods").
		ContinueOnError().
		RequestChunksOf(o.Limit).
		Latest().
		Flatten().
		TransformRequests(o.transformRequests).
		Do()

	if err := r.Err(); err != nil {
		return nil, err
	}

	infos, err := r.Infos()
	if err != nil {
		return nil, err
	}

	return infos, nil
}

func (o *PodOptions) transformRequests(req *rest.Request) {
	if !o.ServerPrint {
		return
	}
	req.SetHeader("Accept", strings.Join([]string{
		fmt.Sprintf("application/json;as=Table;v=%s;g=%s", metav1.SchemeGroupVersion.Version, metav1.GroupName),
		"application/json",
	}, ","))
}

// printNoResourcesFound handles output when there is no object found in any namespaces
func (o *PodOptions) printNoResourcesFound() {
	if !o.AllNamespaces {
		fmt.Fprintf(o.ErrOut, "No resources found in %s namespace.\n", o.Namespace)
	} else {
		fmt.Fprintln(o.ErrOut, "No resources found.")
	}
}
