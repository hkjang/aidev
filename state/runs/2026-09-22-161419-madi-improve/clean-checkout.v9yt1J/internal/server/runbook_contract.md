# Isolated runbook execution

No madi host shell, os/exec, Docker socket, ad-hoc SSH, arbitrary script upload, or user-provided `sh -c` is part of this module. External runners are required. Runtime remains one Go service with four bootstrap environment values.

## Runners

Administrator-only workspace-scoped runner configuration, encrypted bearer credentials and verified HTTPS CA. Runner kinds: AWX API v2 job templates (Ansible) and Kubernetes batch/v1 Jobs in an administrator-selected isolated namespace. Disabled by default. Each runner has a fixed administrator action catalogue and a revision. Each action has a name, permitted executor roles/teams, typed parameters, timeout, and either a fixed AWX job template or exact image digest, fixed executable argv and parameter bindings. User parameters never become shell source.

Kubernetes validates the target namespace's restricted Pod Security Admission and a default-deny NetworkPolicy covering runner pods, plus administrator ResourceQuota limits. Per-job requests/limits, deadline, no retries, non-root, read-only root filesystem, no service account token, drop ALL capabilities, RuntimeDefault seccomp and no host namespace/volumes are mandatory. Offline operator must preload the permitted images in the runner cluster. AWX remains responsible for its inventory, credentials and execution-node isolation; madi's catalogue constrains template IDs and declared survey values.

## Document, immutable plan, request and execution

Document metadata contains purpose/prerequisites/validation/rollback, owner (document owner) and last successful test time. Configured steps reference administrator action IDs and validated parameter objects. Validation and rollback use separately configured action sequences; neither permits arbitrary commands. Each phase is separately requested and explicitly confirmed.

Preparation snapshots the exact document version/body hash, runbook metadata version, phase, action definitions, runner revisions and concrete parameter values. Generic resource kind `runbook` targets this immutable execution plan. If service approval is enabled, its explicit workspace/space runbook policy applies; publication approval never authorizes execution. With service approval disabled, no approval UI or request is introduced, but current execution ACL and explicit `EXECUTE <plan-id>` confirmation remain mandatory.

Execute revalidates the exact plan and current document/runner/action permissions. When approval is required, ConsumeApprovalTx and EnqueueJob are committed in the same PostgreSQL transaction. Work is launched only after commit. Generic API tokens/plugins cannot perform the final human execution confirmation.

## External effects and recovery

Every step has a durable launch ledger. Before network launch it transitions queued→launching under row lock. Kubernetes job name is deterministic from execution/step IDs; recovery checks the exact labels/spec before reattaching. AWX launch has no assumed idempotency guarantee: if a process dies or response is lost while launching, it becomes unknown and is never automatically relaunched. Operators reconcile the external system before making a new explicitly approved plan.

Persist external job identity, status, bounded log cursor/events. Follow status and logs without holding database transactions. Recheck active user/document/runner ACL during polling; on revocation request external cancellation and stop exposing logs. Cancellation and timeout are explicit durable states. A cancelled worker must not silently leave an untracked active remote job. Failed/uncertain cancellation is visible as unknown. Backups retain execution history but restore never resumes old pending/running jobs or makes consumed approval reusable.

## Protocol sources

- [AWX API reference](https://docs.ansible.com/projects/awx/en/latest/rest_api/api_ref.html): job template launch, job status/stdout and cancellation.
- [Kubernetes Jobs](https://kubernetes.io/docs/concepts/workloads/controllers/job/): durable batch workloads, deadlines and completion.
- [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/): restricted execution controls.
- [NetworkPolicy](https://kubernetes.io/docs/concepts/services-networking/network-policies/): namespace network isolation (requires an enforcing CNI).
