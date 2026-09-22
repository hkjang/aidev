package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

func runbookRunnerLabels() map[string]any {
	return map[string]any{"app.kubernetes.io/part-of": "madi-runbook"}
}
func runbookSelectorMatches(selector map[string]any) bool {
	labels := runbookRunnerLabels()
	for key, value := range runbookObject(selector, "matchLabels") {
		if !reflect.DeepEqual(labels[key], value) {
			return false
		}
	}
	for _, raw := range runbookArray(selector, "matchExpressions") {
		expr, ok := raw.(map[string]any)
		if !ok {
			return true
		}
		key := str(expr, "key")
		value, present := labels[key]
		values := listStrings(expr["values"])
		switch str(expr, "operator") {
		case "In":
			if !present || !slices.Contains(values, fmt.Sprint(value)) {
				return false
			}
		case "NotIn":
			if present && slices.Contains(values, fmt.Sprint(value)) {
				return false
			}
		case "Exists":
			if !present {
				return false
			}
		case "DoesNotExist":
			if present {
				return false
			}
		default:
			return true
		}
	}
	return true
}
func runbookQuantity(value string, cpu bool) float64 {
	multiplier := 1.0
	for _, suffix := range []struct {
		name   string
		factor float64
	}{{"Ki", 1024}, {"Mi", 1024 * 1024}, {"Gi", 1024 * 1024 * 1024}, {"Ti", 1024 * 1024 * 1024 * 1024}, {"m", 0.001}} {
		if strings.HasSuffix(value, suffix.name) {
			value = strings.TrimSuffix(value, suffix.name)
			multiplier = suffix.factor
			break
		}
	}
	n, e := strconv.ParseFloat(value, 64)
	if e != nil || n <= 0 || math.IsInf(n, 0) || math.IsNaN(n) {
		return 0
	}
	if cpu {
		return n * multiplier * 1000
	}
	return n * multiplier
}
func (r *runbookRemote) kubernetesPreflight(ctx context.Context, action runbookAction) (map[string]any, error) {
	namespace, e := r.json(ctx, "GET", r.namespacePath(), nil)
	if e != nil {
		return nil, e
	}
	meta := runbookObject(namespace, "metadata")
	labels := runbookObject(meta, "labels")
	if str(labels, "pod-security.kubernetes.io/enforce") != "restricted" || str(labels, "pod-security.kubernetes.io/enforce-version") != "latest" {
		return nil, approvalProblem(400, "전용 namespace에 restricted/latest Pod Security Admission을 적용하세요")
	}
	policies, e := r.json(ctx, "GET", "/apis/networking.k8s.io/v1/namespaces/"+url.PathEscape(str(r.runner.Config, "namespace"))+"/networkpolicies?limit=101", nil)
	if e != nil {
		return nil, e
	}
	if len(runbookArray(policies, "items")) > 100 || str(runbookObject(policies, "metadata"), "continue") != "" {
		return nil, approvalProblem(400, "실행 namespace의 네트워크 정책은 100개 이하로 제한하세요")
	}
	ingress, egress := false, false
	policySnapshot := []any{}
	for _, raw := range runbookArray(policies, "items") {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("네트워크 정책 형식이 잘못되었습니다")
		}
		spec := runbookObject(item, "spec")
		if len(runbookArray(spec, "ingress")) > 0 || len(runbookArray(spec, "egress")) > 0 {
			return nil, approvalProblem(400, "전용 실행 namespace의 네트워크 정책은 ingress/egress 모두 deny-all이어야 합니다")
		}
		if !runbookSelectorMatches(runbookObject(spec, "podSelector")) {
			continue
		}
		types := listStrings(spec["policyTypes"])
		ingress = ingress || slices.Contains(types, "Ingress")
		egress = egress || slices.Contains(types, "Egress")
		m := runbookObject(item, "metadata")
		policySnapshot = append(policySnapshot, map[string]any{"name": m["name"], "uid": m["uid"], "spec": spec})
	}
	if !ingress || !egress {
		return nil, approvalProblem(400, "실행 Pod에 적용되는 양방향 deny-all NetworkPolicy가 필요합니다. CNI의 정책 적용도 운영자가 확인해야 합니다")
	}
	quotas, e := r.json(ctx, "GET", r.namespacePath()+"/resourcequotas?limit=101", nil)
	if e != nil {
		return nil, e
	}
	if len(runbookArray(quotas, "items")) > 100 || str(runbookObject(quotas, "metadata"), "continue") != "" {
		return nil, approvalProblem(400, "실행 namespace의 ResourceQuota는 100개 이하로 제한하세요")
	}
	quotaSnapshot := []any{}
	bounded := false
	for _, raw := range runbookArray(quotas, "items") {
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("ResourceQuota 형식이 잘못되었습니다")
		}
		spec := runbookObject(item, "spec")
		if len(runbookArray(spec, "scopes")) > 0 || len(runbookObject(spec, "scopeSelector")) > 0 {
			continue
		}
		hard := runbookObject(spec, "hard")
		applied := runbookObject(runbookObject(item, "status"), "hard")
		if !reflect.DeepEqual(hard, applied) {
			continue
		}
		valid := runbookQuantity(str(hard, "pods"), false) >= 1 && runbookQuantity(str(hard, "count/jobs.batch"), false) >= 1
		for _, key := range []string{"requests.cpu", "limits.cpu"} {
			valid = valid && runbookQuantity(str(hard, key), true) >= float64(action.CPUMilli)
		}
		for _, key := range []string{"requests.memory", "limits.memory"} {
			valid = valid && runbookQuantity(str(hard, key), false) >= float64(action.MemoryMi)*1024*1024
		}
		if valid {
			bounded = true
			m := runbookObject(item, "metadata")
			quotaSnapshot = append(quotaSnapshot, map[string]any{"name": m["name"], "uid": m["uid"], "hard": hard})
		}
	}
	if !bounded {
		return nil, approvalProblem(400, "Pod·Job 개수와 CPU·메모리 requests/limits를 모두 제한하는 적용 가능한 ResourceQuota가 필요합니다")
	}
	out := map[string]any{"namespace": str(r.runner.Config, "namespace"), "namespace_uid": meta["uid"], "pod_security": "restricted/latest", "network_policies": policySnapshot, "quotas": quotaSnapshot}
	if name := str(r.runner.Config, "runtime_class"); name != "" {
		runtime, e := r.json(ctx, "GET", "/apis/node.k8s.io/v1/runtimeclasses/"+url.PathEscape(name), nil)
		if e != nil {
			return nil, e
		}
		out["runtime_class"] = map[string]any{"name": name, "uid": runbookObject(runtime, "metadata")["uid"], "handler": runtime["handler"]}
	}
	return out, nil
}
func (r *runbookRemote) kubernetesJob(executionID string, index int, step runbookPlanStep) map[string]any {
	labels := runbookRunnerLabels()
	labels["madi.io/execution"] = executionID
	labels["madi.io/step"] = strconv.Itoa(index)
	security := map[string]any{"runAsNonRoot": true, "runAsUser": 65532, "runAsGroup": 65532, "seccompProfile": map[string]any{"type": "RuntimeDefault"}}
	resources := map[string]any{"cpu": fmt.Sprintf("%dm", step.Action.CPUMilli), "memory": fmt.Sprintf("%dMi", step.Action.MemoryMi)}
	container := map[string]any{"name": "runner", "image": step.Action.Image, "imagePullPolicy": "IfNotPresent", "command": step.Argv[:1], "args": step.Argv[1:], "securityContext": map[string]any{"allowPrivilegeEscalation": false, "readOnlyRootFilesystem": true, "privileged": false, "capabilities": map[string]any{"drop": []string{"ALL"}}}, "resources": map[string]any{"requests": resources, "limits": resources}, "volumeMounts": []any{map[string]any{"name": "tmp", "mountPath": "/tmp"}}}
	pod := map[string]any{"restartPolicy": "Never", "automountServiceAccountToken": false, "enableServiceLinks": false, "hostNetwork": false, "hostPID": false, "hostIPC": false, "os": map[string]any{"name": "linux"}, "nodeSelector": map[string]any{"kubernetes.io/os": "linux"}, "securityContext": security, "containers": []any{container}, "volumes": []any{map[string]any{"name": "tmp", "emptyDir": map[string]any{"medium": "Memory", "sizeLimit": "16Mi"}}}}
	if name := str(r.runner.Config, "runtime_class"); name != "" {
		pod["runtimeClassName"] = name
	}
	return map[string]any{"apiVersion": "batch/v1", "kind": "Job", "metadata": map[string]any{"name": runbookJobName(executionID, index), "labels": labels, "annotations": map[string]any{"madi.io/plan-hash": digest(string(jsonValue(step)))}}, "spec": map[string]any{"parallelism": 1, "completions": 1, "backoffLimit": 0, "activeDeadlineSeconds": step.Action.Timeout, "ttlSecondsAfterFinished": 86400, "template": map[string]any{"metadata": map[string]any{"labels": labels}, "spec": pod}}}
}
func runbookJSONSubset(actual, expected any) bool {
	switch want := expected.(type) {
	case map[string]any:
		got, ok := actual.(map[string]any)
		if !ok {
			return false
		}
		for k, v := range want {
			if !runbookJSONSubset(got[k], v) {
				return false
			}
		}
		return true
	case []any:
		got, ok := actual.([]any)
		if !ok || len(got) != len(want) {
			return false
		}
		for i := range want {
			if !runbookJSONSubset(got[i], want[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(actual, expected)
	}
}
func (r *runbookRemote) verifyKubernetesJob(actual, expected map[string]any) bool {
	// Normalize the locally built int/string slices to JSON wire values.
	var want map[string]any
	_ = json.Unmarshal(jsonValue(expected), &want)
	// The API omits false-valued non-pointer fields and canonicalizes resource
	// quantities (1000m → 1, 1024Mi → 1Gi). Compare their semantics, not spelling.
	pod := runbookObject(runbookObject(runbookObject(actual, "spec"), "template"), "spec")
	for _, key := range []string{"hostNetwork", "hostPID", "hostIPC"} {
		if _, ok := pod[key]; !ok {
			pod[key] = false
		}
	}
	for _, root := range []map[string]any{actual, want} {
		p := runbookObject(runbookObject(runbookObject(root, "spec"), "template"), "spec")
		for _, raw := range runbookArray(p, "containers") {
			c, _ := raw.(map[string]any)
			if _, ok := c["args"]; !ok {
				c["args"] = []any{}
			}
			for _, side := range []string{"requests", "limits"} {
				q := runbookObject(runbookObject(c, "resources"), side)
				if len(q) > 0 {
					q["cpu"] = runbookQuantity(str(q, "cpu"), true)
					q["memory"] = runbookQuantity(str(q, "memory"), false)
				}
			}
		}
	}
	if !runbookJSONSubset(actual, want) {
		return false
	}
	if len(runbookArray(pod, "containers")) != 1 || len(runbookArray(pod, "initContainers")) > 0 || len(runbookArray(pod, "ephemeralContainers")) > 0 || len(runbookArray(pod, "volumes")) != 1 {
		return false
	}
	container, _ := runbookArray(pod, "containers")[0].(map[string]any)
	if len(runbookArray(container, "env")) > 0 || len(runbookArray(container, "envFrom")) > 0 || len(runbookArray(container, "volumeMounts")) != 1 {
		return false
	}
	if len(runbookArray(runbookObject(runbookObject(container, "securityContext"), "capabilities"), "add")) > 0 {
		return false
	}
	return true
}
func (r *runbookRemote) kubernetesLaunch(ctx context.Context, executionID string, index int, step runbookPlanStep) (string, error) {
	job := r.kubernetesJob(executionID, index, step)
	base := "/apis/batch/v1/namespaces/" + url.PathEscape(str(r.runner.Config, "namespace")) + "/jobs"
	result, e := r.json(ctx, "POST", base, job)
	if e != nil {
		var remote *runbookRemoteError
		if !errors.As(e, &remote) || remote.Status != 409 {
			return "", e
		}
		result, e = r.json(ctx, "GET", base+"/"+runbookJobName(executionID, index), nil)
		if e != nil {
			return "", e
		}
	}
	uid := str(runbookObject(result, "metadata"), "uid")
	if !validID(uid) {
		return "", fmt.Errorf("Kubernetes Job UID를 확인할 수 없습니다")
	}
	if str(runbookObject(result, "metadata"), "name") != runbookJobName(executionID, index) {
		return "", fmt.Errorf("Kubernetes Job 이름이 요청과 다릅니다")
	}
	if !r.verifyKubernetesJob(result, job) {
		return runbookJobName(executionID, index) + "@" + uid, fmt.Errorf("Kubernetes Job이 승인된 실행 사양과 일치하지 않아 중지합니다")
	}
	return runbookJobName(executionID, index) + "@" + uid, nil
}
func runbookKubernetesID(externalID string) (string, string, error) {
	name, uid, ok := strings.Cut(externalID, "@")
	if !ok || !runbookNamespace.MatchString(name) || !validID(uid) {
		return "", "", fmt.Errorf("Kubernetes 실행 ID를 확인하세요")
	}
	return name, uid, nil
}
func (r *runbookRemote) kubernetesPoll(ctx context.Context, externalID string, step runbookPlanStep, output bool) (runbookRemoteState, error) {
	var state runbookRemoteState
	name, uid, e := runbookKubernetesID(externalID)
	if e != nil {
		return state, e
	}
	job, e := r.json(ctx, "GET", "/apis/batch/v1/namespaces/"+url.PathEscape(str(r.runner.Config, "namespace"))+"/jobs/"+name, nil)
	if e != nil {
		return state, e
	}
	if str(runbookObject(job, "metadata"), "uid") != uid {
		return state, fmt.Errorf("Kubernetes Job이 다른 리소스로 교체되었습니다")
	}
	state.Status = "running"
	status := runbookObject(job, "status")
	if number(status, "succeeded", 0) > 0 {
		state.Status = "succeeded"
	}
	if number(status, "failed", 0) > 0 {
		state.Status = "failed"
	}
	for _, raw := range runbookArray(status, "conditions") {
		c, _ := raw.(map[string]any)
		if str(c, "status") == "True" && oneOf(str(c, "type"), "Failed", "FailureTarget") {
			state.Status = "failed"
		}
	}
	if !output {
		return state, nil
	}
	expected := r.kubernetesJob(newID(), 0, step)
	delete(expected, "metadata")
	delete(runbookObject(runbookObject(expected, "spec"), "template"), "metadata")
	if !r.verifyKubernetesJob(job, expected) {
		return state, &runbookSafetyError{"실행 중인 Kubernetes Job의 격리 사양이 변경되었습니다"}
	}
	pods, e := r.json(ctx, "GET", r.namespacePath()+"/pods?labelSelector="+url.QueryEscape("job-name="+name)+"&limit=10", nil)
	if e != nil {
		return state, e
	}
	for _, raw := range runbookArray(pods, "items") {
		pod, _ := raw.(map[string]any)
		meta := runbookObject(pod, "metadata")
		owned := false
		for _, ref := range runbookArray(meta, "ownerReferences") {
			owner, _ := ref.(map[string]any)
			owned = owned || (str(owner, "uid") == uid && str(owner, "kind") == "Job")
		}
		if !owned {
			continue
		}
		podExpected := map[string]any{"spec": map[string]any{"template": map[string]any{"spec": runbookObject(runbookObject(runbookObject(expected, "spec"), "template"), "spec")}}}
		podActual := map[string]any{"spec": map[string]any{"template": map[string]any{"spec": runbookObject(pod, "spec")}}}
		if !r.verifyKubernetesJob(podActual, podExpected) {
			return state, &runbookSafetyError{"실제 Kubernetes Pod의 격리 사양이 승인된 계획과 다릅니다"}
		}
		podName := str(meta, "name")
		if !runbookNamespace.MatchString(podName) {
			return state, fmt.Errorf("Kubernetes Pod 이름을 확인하세요")
		}
		logs, e := r.request(ctx, "GET", r.namespacePath()+"/pods/"+podName+"/log?container=runner&timestamps=false&limitBytes=2000000", nil)
		if e != nil {
			var remote *runbookRemoteError
			if errors.As(e, &remote) && remote.Status == 400 && state.Status == "running" {
				return state, nil
			}
			return state, e
		}
		state.Output = strings.ToValidUTF8(string(logs), "�")
		state.Truncated = len(logs) >= 2000000
		return state, nil
	}
	return state, nil
}
func (r *runbookRemote) kubernetesCancel(ctx context.Context, externalID string) error {
	name, uid, e := runbookKubernetesID(externalID)
	if e != nil {
		return e
	}
	_, e = r.json(ctx, "DELETE", "/apis/batch/v1/namespaces/"+url.PathEscape(str(r.runner.Config, "namespace"))+"/jobs/"+name, map[string]any{"apiVersion": "v1", "kind": "DeleteOptions", "propagationPolicy": "Foreground", "preconditions": map[string]any{"uid": uid}})
	var remote *runbookRemoteError
	if errors.As(e, &remote) && remote.Status == 404 {
		return nil
	}
	return e
}
