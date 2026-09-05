#!/usr/bin/env bash
# 실행 증거 패키지 — 한 회차의 판단 근거를 evidence.json 하나로 묶는다 (사고 분석용).
#   포함: 실행 ID·기준/변경 커밋·PR·outcome·단계 기록, 정책/프롬프트/러너/모델 버전, 러너 검증 결과, 리뷰 판정,
#         CI gate 결과, 릴리즈 결과·자산 체크섬, 승인 기록, 사용량. 에이전트 원문(agent-*.txt)과 비밀값은 넣지 않는다.
#   사용: bin/evidence.sh <run 디렉터리>   (record_run 이 자동으로 부른다)
set -uo pipefail
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"; REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="${AIDEV_STATE:-$REPO_DIR/state}"; DATA="${AIDEV_DATA:-$REPO_DIR/docs/data}"
d=${1:?run dir}; [ -d "$d" ] || exit 0
rid=$(basename "$d")
j(){ [ -s "$1" ] && jq -c . "$1" 2>/dev/null || echo null; }
sha_list(){ [ -s "$1" ] && jq -Rn '[inputs | split("  ") | {sha256:.[0], name:.[1]}]' "$1" 2>/dev/null || echo '[]'; }
ci=$(ls "$d"/ci-*.json 2>/dev/null | head -1)
ci_gate=null; [ -n "$ci" ] && ci_gate=$(python3 "$HERE/gate.py" ci "$ci" --sha "$(jq -r '.[0].check_runs[0].head_sha // ""' "$ci" 2>/dev/null)" 2>/dev/null || true); [ -n "$ci_gate" ] || ci_gate=null
pr=$(jq -r '.pr.reason // empty' "$d/stages.json" 2>/dev/null)
approval=null; [ -n "$pr" ] && approval=$(jq -c --arg pr "$pr" 'select(.pr==$pr)' "$STATE/approvals.jsonl" 2>/dev/null | tail -1); [ -n "$approval" ] || approval=null
usage=$(jq -c --arg rid "$rid" 'select(.run_id==$rid)' "$DATA/usage.jsonl" 2>/dev/null | jq -s '.' 2>/dev/null || echo '[]')
record=$(jq -c --arg rid "$rid" 'select(.run_id==$rid)' "$DATA/runs.jsonl" 2>/dev/null | tail -1); [ -n "$record" ] || record=null
jq -n --arg rid "$rid" --arg gen "$(date -Iseconds)" \
   --argjson run "$(j "$d/run.json")" --argjson stages "$(j "$d/stages.json")" --argjson record "$record" \
   --argjson verify "$(j "$d/verify.json")" --argjson verify_gate "$(j "$d/verify.gate.json")" --argjson verify_rebased "$(j "$d/verify-rebased.json")" \
   --argjson review "$(j "$d/review.json")" --argjson release "$(j "$d/release.json")" --argjson ci_gate "$ci_gate" \
   --argjson assets "$(sha_list "$d/assets.sha256")" --argjson approval "$approval" --argjson usage "$usage" \
   '{run_id:$rid, generated:$gen, run:$run, record:$record, stages:$stages,
     provenance:{model:$run.model, policy_version:$run.policy_version, prompts_hash:$run.prompts_hash, runner_version:$run.runner_version,
                 base_sha:($record.base_sha // null), head_sha:($record.head_sha // null), pr:($record.pr // null)},
     verification:{runner:$verify, runner_gate:$verify_gate, after_rebase:$verify_rebased, review:$review, ci_gate:$ci_gate},
     release:{result:$release, assets_sha256:$assets}, approval:$approval, usage:$usage}' > "$d/evidence.json" 2>/dev/null \
  && echo "evidence: $d/evidence.json" || echo "evidence: FAILED for $rid"
