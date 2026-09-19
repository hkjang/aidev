#!/usr/bin/env bash
# 실험 E1 — 교차 모델 비평 (cross-model critic).
#
# 동기: LLM 심판은 자기(또는 자기와 비슷한 분포의) 출력을 선호한다는 보고가 있다
# (Zheng et al. 2023; Wataoka et al. 2024). aidev 의 비평가와 구현자는 같은 모델 계열(Claude)이다.
# 그래서 Claude 비평가가 이미 판정한 PR 을 **다른 계열(OpenAI Codex)** 이 같은 프롬프트로 다시
# 심사하게 해 일치도를 잰다. 불일치가 크면 교차 모델 비평을 기본으로 켤 근거가 되고,
# 작으면 자기 선호 편향이 이 설정에서는 크지 않다는 근거가 된다.
#
# 설계: 최근 Claude 비평이 승인·머지한 PR N건과 거절한 PR N건. Codex 는 read-only 샌드박스에서
# 같은 review-prompt.md 를 받고(파일을 쓸 수 없으므로 마지막 메시지에 JSON 만), 판정·위험도·사유를 낸다.
# 결과: docs/paper/experiments/cross-critic.jsonl (PR 당 한 줄) + cross-critic.summary.json
#   N=6 bin/exp-cross-critic.sh
set -uo pipefail
export HOME="${HOME:-/home/hkjang}"
export PATH="$HOME/.local/bin:$HOME/.nvm/versions/node/v22.23.1/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"; REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="$REPO_DIR/state"; DATA="$REPO_DIR/docs/data"; ROOT="${ROOT:-/mnt/c/Users/USER/projects}"
OUTD="$REPO_DIR/docs/paper/experiments"; mkdir -p "$OUTD"; LOG="$OUTD/cross-critic.jsonl"; touch "$LOG"
N=${N:-6}; SINCE=${SINCE:-2026-09-12}; MODEL_CODEX=${MODEL_CODEX:-}
command -v codex >/dev/null || { echo "codex 없음"; exit 1; }

pick(){ # $1=approved|rejected → project\tpr\tbase\thead\trun_id\trisk
  if [ "$1" = approved ]; then
    jq -r --arg s "$SINCE" 'select(.stages.review.state=="approved" and (.outcome=="release-ready" or .outcome=="merged") and .date>=$s and (.pr|length)>0 and (.head_sha|length)>0) | "\(.project)\t\(.pr)\t\(.base_sha)\t\(.head_sha)\t\(.run_id)"' "$DATA/runs.jsonl" | tail -n "$N"
  else
    jq -r --arg s "$SINCE" 'select(.stages.review.state=="rejected" and .date>=$s and (.pr|length)>0 and (.head_sha|length)>0) | "\(.project)\t\(.pr)\t\(.base_sha)\t\(.head_sha)\t\(.run_id)"' "$DATA/runs.jsonl" | tail -n "$N"
  fi
}

run_one(){ # $1=group $2..=fields
  local group=$1 project=$2 pr=$3 base=$4 head=$5 rid=$6 num repo wt prompt last ev out claude_v claude_r codex_json t0
  num=${pr##*/}; repo="$ROOT/$project"
  grep -q "\"pr\":\"$pr\"" "$LOG" && { echo "skip $project #$num (이미 함)"; return; }
  [ -d "$repo/.git" ] || { echo "skip $project (clone 없음)"; return; }
  claude_v=$(jq -r '.verdict // ""' "$STATE/runs/$rid/review.json" 2>/dev/null); claude_r=$(jq -r '.risk // ""' "$STATE/runs/$rid/review.json" 2>/dev/null)
  [ -n "$claude_v" ] || { echo "skip $project #$num (Claude review.json 없음)"; return; }
  git -C "$repo" fetch -q origin "refs/pull/$num/head:refs/exp/pr-$num" 2>/dev/null || { echo "skip $project #$num (fetch 실패)"; return; }
  git -C "$repo" cat-file -e "$head" 2>/dev/null || { echo "skip $project #$num (head $head 없음)"; return; }
  git -C "$repo" cat-file -e "$base" 2>/dev/null || git -C "$repo" fetch -q origin "$base" 2>/dev/null || true
  wt="$HOME/.cache/auto-improve-wt/$project-exp"
  git -C "$repo" worktree remove --force "$wt" >/dev/null 2>&1; rm -rf "$wt"; git -C "$repo" worktree prune >/dev/null 2>&1
  git -C "$repo" worktree add --detach "$wt" "$head" >/dev/null 2>&1 || { echo "skip $project #$num (worktree 실패)"; return; }
  prompt=$(BASE="$base" REVIEW_FILE="(파일을 쓸 수 없습니다 — 아래 지시대로 마지막 메시지에 출력)" PROFILE="$(cat "$STATE/$project.profile.md" 2>/dev/null || echo '(없음)')" \
           JOURNAL="(없음 — 이 실험에서는 노트를 주지 않습니다)" JOURNAL_FILE="(쓰지 마세요)" ARBITER_HISTORY="(없음)" \
           envsubst '$BASE $REVIEW_FILE $PROFILE $JOURNAL $JOURNAL_FILE $ARBITER_HISTORY' < "$REPO_DIR/review-prompt.md")
  prompt="$prompt

## 이 세션의 출력 방식 (실험)
이 세션은 읽기 전용 샌드박스라 파일을 쓸 수 없습니다. 노트 파일도 쓰지 마세요. 대신 **마지막 메시지의 마지막 줄**에 위 JSON 한 개를 그대로 출력하세요(코드펜스 없이)."
  last=$(mktemp); ev=$(mktemp); t0=$(date +%s)
  ( cd "$wt" && timeout -k 30 900 codex exec -s read-only --skip-git-repo-check --ephemeral --json ${MODEL_CODEX:+-m "$MODEL_CODEX"} -o "$last" "$prompt" </dev/null >"$ev" 2>"$ev.err" )
  codex_json=$(python3 - "$last" <<'PY'
import json, sys
t = open(sys.argv[1], encoding="utf-8", errors="replace").read() if len(sys.argv) > 1 else ""
m = None
for start in [i for i, ch in enumerate(t) if ch == "{"]:
    for end in range(len(t), start, -1):
        if t[end-1] != "}": continue
        try: d = json.loads(t[start:end])
        except Exception: continue
        if isinstance(d, dict) and d.get("verdict") in ("approve", "reject"): m = d; break
    if m: break
print(json.dumps(m, ensure_ascii=False) if m else "null")
PY
)
  local it ot
  it=$(jq -rs 'map(select(.type=="turn.completed")|.usage.input_tokens)|add // 0' "$ev" 2>/dev/null); ot=$(jq -rs 'map(select(.type=="turn.completed")|.usage.output_tokens)|add // 0' "$ev" 2>/dev/null)
  jq -cn --arg ts "$(date -Iseconds)" --arg g "$group" --arg p "$project" --arg pr "$pr" --arg base "$base" --arg head "$head" --arg rid "$rid" \
     --arg cv "$claude_v" --arg cr "$claude_r" --argjson codex "${codex_json:-null}" --argjson secs "$(( $(date +%s) - t0 ))" --argjson it "${it:-0}" --argjson ot "${ot:-0}" \
     '{ts:$ts,group:$g,project:$p,pr:$pr,base:$base,head:$head,run_id:$rid,claude:{verdict:$cv,risk:$cr},codex:$codex,seconds:$secs,codex_tokens:{in:$it,out:$ot}}' >> "$LOG"
  echo "$group $project #$num — Claude $claude_v/$claude_r · Codex $(jq -r '.verdict // "?"' <<<"${codex_json:-null}")/$(jq -r '.risk // "?"' <<<"${codex_json:-null}") ($(( $(date +%s) - t0 ))s)"
  rm -f "$last" "$ev" "$ev.err"; git -C "$repo" worktree remove --force "$wt" >/dev/null 2>&1
}

while IFS=$'\t' read -r p pr b h rid; do [ -n "$p" ] && run_one approved "$p" "$pr" "$b" "$h" "$rid"; done < <(pick approved)
while IFS=$'\t' read -r p pr b h rid; do [ -n "$p" ] && run_one rejected "$p" "$pr" "$b" "$h" "$rid"; done < <(pick rejected)

# 요약: 일치율, 그룹별 Codex 판정 분포
jq -s '
  def cnt(f): map(select(f)) | length;
  {n: length, judged: cnt(.codex != null),
   agree: cnt(.codex != null and .codex.verdict == .claude.verdict),
   approved_group: {n: cnt(.group=="approved"), codex_approve: cnt(.group=="approved" and .codex.verdict=="approve"), codex_reject: cnt(.group=="approved" and .codex.verdict=="reject")},
   rejected_group: {n: cnt(.group=="rejected"), codex_approve: cnt(.group=="rejected" and .codex.verdict=="approve"), codex_reject: cnt(.group=="rejected" and .codex.verdict=="reject")},
   median_seconds: (map(.seconds) | sort | .[length/2|floor])}' "$LOG" > "$OUTD/cross-critic.summary.json"
cat "$OUTD/cross-critic.summary.json"
