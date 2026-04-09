# KEP-7029: LocalQueue Default Maximum Execution Time

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
    - [LocalQueue Spec](#localqueue-spec)
  - [Controller](#controller)
    - [Workload Creation](#workload-creation)
    - [Workload Equivalency Check](#workload-equivalency-check)
    - [Internal LocalQueueDefaults Struct](#internal-localqueuedefaults-struct)
  - [Precedence Rules](#precedence-rules)
  - [Why Workload, Not Job](#why-workload-not-job)
  - [Specific Field vs. Generic Defaulting](#specific-field-vs-generic-defaulting)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit Tests](#unit-tests)
    - [Integration tests](#integration-tests)
      - [controller/jobs/job](#controllerjobsjob)
      - [webhook/core/localqueue](#webhookcorelocalqueue)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Future Extensions](#future-extensions)
  - [Consolidating LocalQueue Defaults](#consolidating-localqueue-defaults)
<!-- /toc -->

## Summary

Add the ability to configure a default maximum execution time on a LocalQueue.
When a job is submitted to a LocalQueue that has a default maximum execution time
configured, the resulting workload will automatically have its
`maximumExecutionTimeSeconds` field set, unless the job already specifies a value
via the `kueue.x-k8s.io/max-exec-time-seconds` label.

This builds on the existing maximum execution time feature introduced in
[KEP-3125](../3125-maximum-execution-time/README.md).

## Motivation

Cluster administrators often want to enforce time limits on jobs submitted to
specific queues. Today, setting the maximum execution time requires each job to
carry the `kueue.x-k8s.io/max-exec-time-seconds` label. This places the burden
on individual users to remember to set the label, and there is no way for an
administrator to enforce a default timeout at the queue level.

This has been requested by the community in multiple discussions:

- https://github.com/kubernetes-sigs/kueue/discussions/6684
- https://github.com/kubernetes-sigs/kueue/issues/6587#issuecomment-3342233971

### Goals

- Allow cluster administrators to configure a default maximum execution time on a
  LocalQueue via a new field in `LocalQueueSpec`.
- Automatically apply this default to workloads created for jobs submitted to the
  LocalQueue, when the job does not specify its own maximum execution time.
- Preserve backward compatibility: jobs that already specify the
  `kueue.x-k8s.io/max-exec-time-seconds` label are unaffected.

### Non-Goals

- Enforcing a maximum cap that overrides a job-level timeout. The LocalQueue value
  is a default, not a ceiling. If the job specifies a longer timeout via the label,
  the job value is used.
- Changing the scheduling behavior based on the maximum execution time.
- Adding the default timeout at the ClusterQueue level. This may be considered in
  a future enhancement.

## Proposal

Add a new optional field `maximumExecutionTimeSeconds` to `LocalQueueSpec`. When
a workload is created for a job submitted to a LocalQueue with this field set,
and the job does not have the `kueue.x-k8s.io/max-exec-time-seconds` label, the
workload's `spec.maximumExecutionTimeSeconds` is populated with the LocalQueue's
value.

The existing enforcement mechanism in the workload controller
(`reconcileMaxExecutionTime`) handles the rest — no changes are needed there.

### User Stories

#### Story 1

As a cluster administrator, I want to set a default maximum execution time of
1 hour on a LocalQueue used for interactive workloads, so that forgotten or
runaway jobs are automatically cleaned up without requiring each user to set
the timeout label on their jobs.

#### Story 2

As a batch user submitting jobs to a LocalQueue with a default timeout of 1 hour,
I want to override the default by setting `kueue.x-k8s.io/max-exec-time-seconds`
to 7200 on my job, so that my long-running training job is allowed more time.

### Notes/Constraints/Caveats

- The default is applied at workload creation time. Existing workloads are not
  retroactively updated in the following scenarios:

  - **LocalQueue's timeout field is modified**: If an administrator changes the
    `maximumExecutionTimeSeconds` value on a LocalQueue, existing workloads keep
    their original timeout. Only newly created workloads pick up the new value.

  - **Job moves to a different LocalQueue**: If a user changes the
    `kueue.x-k8s.io/queue-name` label on a Job to point to a different
    LocalQueue, the workload's `QueueName` is updated in-place, but its
    `MaximumExecutionTimeSeconds` is not recalculated from the new LocalQueue.
    This is intentional: the timeout was a default applied at creation, and
    changing it mid-flight could be disruptive (e.g., a workload admitted with a
    1-hour timeout should not suddenly receive a 15-minute timeout from the new
    queue). This is consistent with existing behavior where the queue change path
    only updates the `QueueName` field on the workload. If the workload is
    recreated (e.g., due to job spec changes that trigger non-equivalency), the
    new LocalQueue's default would apply.

  - **Readmission**: Changing the queue label does not trigger readmission today.
    This is existing behavior, and this KEP does not change it. The timeout
    inherited at creation time persists through queue changes.

- Prebuilt workloads are not affected. They are expected to be fully specified
  externally.

### Risks and Mitigations

- **Risk**: An additional API call to look up the LocalQueue during workload
  creation adds latency to the reconcile loop.
  - **Mitigation**: The lookup is a single GET call that only happens once per
    workload creation. The LocalQueue is in the same namespace as the job and is
    likely cached by the controller-runtime client cache. The performance impact
    is negligible.

## Design Details

### API

#### LocalQueue Spec

Add a new optional `workloadDefaults` field to `LocalQueueSpec` in both
`v1beta2` and `v1beta1`. This struct groups all default values that a LocalQueue
can apply to workloads, making it straightforward to add future defaults (e.g.,
priority class, TAS configuration) without cluttering the top-level spec:

```go
// LocalQueueSpec defines the desired state of LocalQueue
type LocalQueueSpec struct {
    // ...existing fields...

    // workloadDefaults defines default values that are applied to workloads
    // submitted to this LocalQueue when the workload does not already specify them.
    // +optional
    WorkloadDefaults *LocalQueueWorkloadDefaults `json:"workloadDefaults,omitempty"`
}

// LocalQueueWorkloadDefaults defines default values that are applied to
// workloads submitted to this LocalQueue when the workload does not already
// specify them.
type LocalQueueWorkloadDefaults struct {
    // maximumExecutionTimeSeconds if provided, determines the default maximum
    // time, in seconds, for workloads submitted to this LocalQueue.
    // This value is used when the job does not already specify a maximum execution
    // time via the kueue.x-k8s.io/max-exec-time-seconds label.
    //
    // +optional
    // +kubebuilder:validation:Minimum=1
    MaximumExecutionTimeSeconds *int32 `json:"maximumExecutionTimeSeconds,omitempty"`
}
```

Since the struct and field names are identical in both API versions, auto-generated
conversion functions handle the `v1beta1 <-> v1beta2` conversion without manual
code.

### Controller

#### Workload Creation

In the job framework reconciler (`pkg/controller/jobframework/reconciler.go`),
after constructing the workload and before creating it, the reconciler checks if
`wl.Spec.MaximumExecutionTimeSeconds` is nil. If so, it looks up the LocalQueue
and applies its `WorkloadDefaults.MaximumExecutionTimeSeconds` value if set.

This logic is applied in two code paths:

1. **`handleJobWithNoWorkload`** — when a new workload is created for a job.
2. **`updateWorkloadToMatchJob`** — when an existing workload is reconstructed to
   match a modified job.

A new helper method on `JobReconciler` performs a GET on the LocalQueue using the
workload's `QueueName` and namespace, and copies the LocalQueue's
`WorkloadDefaults.MaximumExecutionTimeSeconds` into the workload spec if it is set.

If the LocalQueue lookup fails (e.g., the queue does not exist yet), the error is
logged but workload creation proceeds without the default. This is consistent with
existing behavior where a job can reference a non-existent queue.

#### Workload Equivalency Check

The existing `EquivalentToWorkload` function compares the workload's
`MaximumExecutionTimeSeconds` against the job's label value. When the timeout
originates from the LocalQueue (and the job has no label), this comparison would
incorrectly flag a mismatch.

The fix: only compare `MaximumExecutionTimeSeconds` when the job explicitly has
the `kueue.x-k8s.io/max-exec-time-seconds` label. If the label is absent, the
workload's value may have been inherited from the LocalQueue, and no mismatch
should be reported. The check first looks for the presence of the label on the
job object, and only performs the value comparison if the label exists.

#### Internal LocalQueueDefaults Struct

To prepare for future LocalQueue-level defaults (e.g., priority, TAS
configuration), the implementation uses an internal `LocalQueueDefaults` struct
in `pkg/controller/jobframework/localqueue_defaults.go`. This struct is not part
of the Kueue API — it is a controller-internal type that centralizes the
extraction and application of LocalQueue defaults onto a Workload.

```go
// LocalQueueDefaults holds fields that a LocalQueue can default onto a Workload.
type LocalQueueDefaults struct {
    MaximumExecutionTimeSeconds *int32
}
```

Two helper functions operate on this struct:

- `GetLocalQueueDefaults(ctx, client, queueName, namespace)` — performs a single
  LocalQueue GET and extracts all defaulting fields into the struct.
- `ApplyLocalQueueDefaults(defaults, workload)` — applies the defaults to the
  workload, skipping any field that is already set.

This design means a single LocalQueue lookup populates all defaults, and adding a
new defaulting field in the future requires only:

1. Adding the field to `LocalQueueWorkloadDefaults` (API type) and `LocalQueueDefaults` (internal type).
2. Populating it in `GetLocalQueueDefaults` from `lq.Spec.WorkloadDefaults`.
3. Applying it in `ApplyLocalQueueDefaults`.

No changes to the reconciler call sites are needed. See also
[Future Extensions](#future-extensions) for how this struct enables future
defaulting use cases.

### Precedence Rules

The precedence for `maximumExecutionTimeSeconds` on a workload is:

1. **Job label** (`kueue.x-k8s.io/max-exec-time-seconds`) — highest priority.
   If the job has this label, its value is always used.
2. **LocalQueue default** (`spec.workloadDefaults.maximumExecutionTimeSeconds`) — used only if the
   job does not have the label.
3. **None** — if neither the job nor the LocalQueue specifies a value, no maximum
   execution time is set on the workload.

### Why Workload, Not Job

The default is applied to the Workload object, not the user-created Job. This is
a deliberate design decision:

- **No surprise mutations**: Mutating the Job to inject a
  `kueue.x-k8s.io/max-exec-time-seconds` label the user never set would be
  surprising and could interfere with user tooling that reads Job labels.
- **Universal coverage**: Not all job types go through Kueue's mutating admission
  webhooks. Applying the default at the workload level in the reconciler ensures
  it works for every integrated job type without requiring webhook changes.
- **Consistent layering**: The Workload is Kueue's internal representation of a
  job. Applying Kueue-internal defaults there is consistent with how other
  workload fields (e.g., pod set specs, queue name) are populated from the job
  framework reconciler rather than mutated on the original object.

See also the [Alternatives](#alternatives) section for the rejected approaches of
using a mutating webhook on Jobs or Workloads.

### Specific Field vs. Generic Defaulting

This KEP adds a specific `maximumExecutionTimeSeconds` field within
`LocalQueueSpec.workloadDefaults` rather than a generic label/annotation
defaulting mechanism. The `workloadDefaults` struct provides a natural grouping
for future default fields while keeping each field explicitly typed. The reasons:

- **No established pattern**: Today, `LocalQueueSpec` has very few fields
  (`clusterQueue`, `stopPolicy`, `fairSharing`). There is no existing pattern for
  generic defaulting from LocalQueue to workload, so introducing one for a single
  field would be premature.
- **Specific validation and precedence**: `maximumExecutionTimeSeconds` has
  specific precedence rules (job label > LocalQueue default > none) and interacts
  with the workload equivalency check. A generic mechanism would need its own
  design to handle precedence, validation, and equivalency for arbitrary
  labels/annotations — adding complexity without a clear second use case.
- **Simpler to validate**: A typed field with `+kubebuilder:validation:Minimum=1`
  is simpler to validate, document, and reason about than a generic string map.
- **Future extensibility**: If more LocalQueue-level defaults emerge (e.g., from
  [discussion #7129](https://github.com/kubernetes-sigs/kueue/discussions/7129)),
  the pattern established here — specific field with clear precedence rules — can
  inform the design of a generic mechanism. Refactoring from specific to generic
  is straightforward once multiple use cases exist.

See also the [Alternatives](#alternatives) section for the generic defaulting
alternative.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

No regressions in the existing tests.

#### Unit Tests

As needed to provide coverage for the new code.

- `pkg/controller/jobframework`: coverage for `applyLocalQueueMaxExecutionTime`
  and the updated `EquivalentToWorkload` logic.
- `pkg/controller/jobs/job`: test cases covering LocalQueue default applied,
  job label taking precedence, and equivalency passing with LocalQueue-sourced
  timeout.

#### Integration tests

##### controller/jobs/job

Add "A job gets the LocalQueue default maximum execution time when no label is set"

Add "A job label maximum execution time takes precedence over LocalQueue default"

Add "A job moving to a different LocalQueue keeps the original timeout"

Add "An existing workload is not affected when the LocalQueue's timeout is modified"

##### webhook/core/localqueue

Add validation tests for the new `maximumExecutionTimeSeconds` field (must be >= 1).

### Graduation Criteria

This feature is a small, additive extension to the existing maximum execution time
feature (KEP-3125). It does not require a feature gate since it only changes
behavior when the administrator explicitly sets the new field on a LocalQueue.

## Implementation History

- 2026-04-09: KEP created

## Drawbacks

- Adds an additional API call (LocalQueue GET) during workload creation. However,
  this is a single cached read and the cost is negligible.
- Administrators must configure the default per LocalQueue. A ClusterQueue-level
  default is not included in this KEP but could be added later.

## Alternatives

1. **Mutating webhook on Workload**: Instead of applying the default in the
   reconciler, a mutating admission webhook could intercept Workload creation and
   set the default. This was rejected because Workloads do not currently have a
   custom mutating webhook, and adding one for this single field introduces
   unnecessary architectural complexity.

2. **Mutating webhook on Jobs**: Apply the `kueue.x-k8s.io/max-exec-time-seconds`
   label to the Job itself during admission. This was rejected because it modifies
   the user's Job object, which may be unexpected, and it does not work for job
   types that are not managed by Kueue's webhooks.

3. **ClusterQueue-level default**: Instead of (or in addition to) the LocalQueue,
   set the default at the ClusterQueue level. While useful, this has different
   semantics (all queues backed by the ClusterQueue would inherit it) and can be
   added as a follow-up enhancement. The LocalQueue level provides more granular
   control and is what was requested in the community discussions.

4. **Generic label/annotation defaulting on LocalQueue**: Instead of a specific
   `maximumExecutionTimeSeconds` field, add a generic mechanism for LocalQueues to
   default arbitrary labels or annotations onto workloads. This was considered but
   rejected for now because it introduces significant design complexity (precedence
   rules, validation, equivalency interactions) without a clear second use case.
   If additional LocalQueue-level defaults are needed in the future, the pattern
   established by this KEP can inform a more generic design. See
   [Specific Field vs. Generic Defaulting](#specific-field-vs-generic-defaulting)
   for the full rationale.

## Future Extensions

### Consolidating LocalQueue Defaults

The `LocalQueueWorkloadDefaults` API type and the internal `LocalQueueDefaults`
struct introduced in this KEP are designed to be the single place where
LocalQueue-to-Workload defaults are collected and applied. As new defaulting
fields are needed, they should be added to `LocalQueueWorkloadDefaults` (the API
type) and wired through `LocalQueueDefaults` (the internal struct) rather than
adding separate lookup functions per field.

Potential future fields that could be added to `LocalQueueWorkloadDefaults`:

- **Priority**: A default `WorkloadPriorityClass` on the LocalQueue, applied when
  the job does not specify one via label or pod spec. This would follow the same
  precedence pattern: job-level > LocalQueue default > none.

- **TAS configuration**: Default topology-aware scheduling annotations (e.g.,
  `podset-required-topology`, `podset-preferred-topology`) that the LocalQueue
  can inject into workload pod sets when the job does not specify them.

- **Resource adjustments**: Default resource limits or overhead applied to
  workloads submitted to a queue.

Each new field would require:

1. A new field on `LocalQueueWorkloadDefaults` (API change, with its own KEP).
2. A corresponding field on `LocalQueueDefaults` (internal struct).
3. Population in `GetLocalQueueDefaults` from `lq.Spec.WorkloadDefaults`.
4. Application in `ApplyLocalQueueDefaults` with appropriate precedence logic.
5. Updates to `EquivalentToWorkload` if the field participates in equivalency
   checks.

This approach keeps the API surface explicit and typed (each default is a
specific field with validation), while the internal struct provides a clean
extension point that avoids scattering LocalQueue lookups across the reconciler.
