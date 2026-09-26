#!/usr/bin/env bash
# 캠페인 교훈 — 캠페인이 스스로 배운다.
#
# 한 캠페인은 같은 일을 서른 저장소에서 되풀이한다. 그런데 한 저장소에서 리뷰가 거절한 이유,
# PR 처리기가 심사에서 걸어낸 결함, 머지 뒤 회귀한 원인은 그 저장소의 원장에만 남았고, 다음
# 저장소는 같은 자리에서 같은 실수를 했다 (guides 캠페인에서 "가이드의 API 메서드를 소스에서
# 확인하지 않음" 이 열두 번 걸렸다).
#
# 이 스크립트는 캠페인마다 그 증거를 모아 Claude 에게 **일반 규칙**(어느 저장소에서든 확인할
# 것)으로 정제하게 하고, 그 결과를
#   state/campaign-lessons/<id>.json  (규칙·근거·입력 해시)
#   state/campaign-lessons/<id>.md    (프롬프트에 그대로 붙는 본문)
# 에 둔다. 러너는 캠페인 회차 프롬프트(CAMPAIGN_NOTE)와 PR 처리기 심사(HOLD_NOTE)에 이 본문을
# 자동으로 붙인다. 증거가 바뀌지 않았으면(입력 해시 같음) 모델을 부르지 않는다.
#
#   bin/campaign-lessons.sh             활성 캠페인 전부
#   bin/campaign-lessons.sh --dry-run   무엇을 넣을지만 보여 준다 (모델 호출 없음)
#   bin/campaign-lessons.sh <id>        한 캠페인만
# 헬스체크(bin/health.sh, 30분)가 부른다. 증거가 그대로면 비용 0.
set -uo pipefail
export HOME="${HOME:-/home/hkjang}"
export PATH="$HOME/.local/bin:$HOME/.nvm/versions/node/v22.23.1/bin:/usr/local/bin:/usr/bin:/bin"
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="${AIDEV_STATE:-$REPO_DIR/state}"; DATA="${AIDEV_DATA:-$REPO_DIR/docs/data}"
OUT_DIR="$STATE/campaign-lessons"; mkdir -p "$OUT_DIR"
MODEL="${MODEL:-claude-opus-5}"; BUDGET="${AIDEV_CL_BUDGET:-1.0}"
DRY=0; ONLY=""
for a in "$@"; do case "$a" in --dry-run) DRY=1;; *) ONLY=$a;; esac; done
RUNS="$DATA/runs.jsonl"; LESSONS="$STATE/lessons.jsonl"; SHEP="$STATE/shepherd.jsonl"
[ -s "$RUNS" ] || exit 0

# 캠페인 PR → 프로젝트 맵 (PR 하나가 어느 캠페인 것인지). 처리기 기록·교훈은 PR 로만 이어진다.
prmap=$(jq -sc '[ .[] | select((.campaign//"")!="" and ((.pr//"")|length)>0) | {pr, campaign, project} ] | unique_by(.pr) | map({(.pr): {c: .campaign, p: .project}}) | add // {}' "$RUNS" 2>/dev/null)

collect(){ # $1=캠페인 id → 증거 줄 (stdout)
  local id=$1 rid p
  # 1) 독립 리뷰가 거절한 이유 (state/runs/<run>/review.json)
  while IFS=$'\t' read -r p rid; do
    [ -n "$rid" ] && [ -f "$STATE/runs/$rid/review.json" ] || continue
    jq -r --arg p "$p" '.reasons[]? | "[\($p) · 리뷰 거절] " + (.|tostring|.[0:400])' "$STATE/runs/$rid/review.json" 2>/dev/null
  done < <(jq -r --arg id "$id" 'select(.campaign==$id and .stages.review.state=="rejected") | "\(.project)\t\(.run_id)"' "$RUNS" 2>/dev/null)
  # 2) PR 처리기가 심사에서 걸어낸 것 · 고치지 못한 것
  [ -s "$SHEP" ] && jq -r --arg id "$id" --argjson m "$prmap" '
      select((.action=="review-reject" or .action=="fix-failed" or .action=="needs-human") and ($m[.pr]?.c // "")==$id)
      | "[\(.project) · 처리기 \(.action)] " + ((.detail//"")|gsub("\n";" ")|.[0:400])' "$SHEP" 2>/dev/null
  # 3) 머지 뒤 회귀·사람 반려 등 교훈 (PR 로 캠페인과 잇는다)
  [ -s "$LESSONS" ] && jq -r --arg id "$id" --argjson m "$prmap" '
      select(((.pr//"")|length)>0 and ($m[.pr]?.c // "")==$id)
      | "[\(.project) · \(.kind)] " + ((.detail//"")|gsub("\n";" ")|.[0:400])' "$LESSONS" 2>/dev/null
}

distill(){ # $1=id $2=goal $3=증거 파일 → JSON(rules) stdout, 실패면 빈 출력
  local id=$1 goal=$2 ev=$3 out err text
  local prompt="당신은 여러 저장소에 같은 기능을 넣는 개선 캠페인의 회고를 맡았습니다. 아래는 캠페인 \"$id\" 를 여러 저장소에서 진행하며 리뷰·심사가 거절했거나 머지 뒤 문제가 된 것들의 원문입니다. 다음 저장소가 **시작하기 전에 읽고 같은 자리에서 걸리지 않게** 할 일반 규칙으로 정제하세요.

## 캠페인 목표 (앞부분)
$(head -c 900 <<<"$goal")

## 증거 (저장소 · 출처 · 내용)
$(cat "$ev")

## 규칙을 쓰는 법
- 최대 8개. 두 번 이상 나온 것, 고치기 어려운 것부터.
- 저장소 이름·경로·함수명이 아니라 **어느 저장소에서든 확인할 수 있는 말**로 쓰세요 (\"X 를 하기 전에 Y 를 소스에서 확인한다\" 처럼).
- 각 규칙에 무엇을 하라는 것인지(rule), 왜인지 한 문장(why), 근거가 된 저장소(projects), 몇 번 나왔는지(count)를 적으세요.
- 증거가 서로 다른 얘기라 묶을 수 없으면 억지로 일반화하지 말고 그대로 한 줄씩 두세요.
- 파일을 읽거나 명령을 실행하지 마세요. 이 원문만으로 판단합니다.

마지막 줄에 다음 JSON 한 줄만 쓰세요(설명·코드펜스 없이):
{\"rules\":[{\"rule\":\"...\",\"why\":\"...\",\"projects\":[\"...\"],\"count\":1}]}"
  out=$(mktemp); err=$(mktemp)
  timeout -k 30 420 claude -p "$prompt" --model "$MODEL" --settings '{"attribution":{"commit":"","pr":""}}' \
    --permission-mode plan --allowedTools "" --max-budget-usd "$BUDGET" --output-format json <"/dev/null" >"$out" 2>"$err"   # stdin 을 닫는다: 열어 두면 while 루프의 입력(캠페인 목록)을 삼켜 첫 캠페인만 돌았다
  text=$(jq -r '.result // ""' "$out" 2>/dev/null)
  # 사용량 한도 응답을 결과로 쓰지 않는다 — 짧은 안내문 한 줄이 보고서·교훈·취향으로
  # 저장되면 그 파일을 읽는 모든 회차가 그것을 규칙으로 읽는다 (2026-09-21 이사회가 그랬다).
  if grep -qiE "usage limit|limit reached|weekly limit|5-hour limit|rate.?limit|quota|credit balance|resets? at|resets [A-Z][a-z]{2} " <<<"$text" \
     && [ "$(printf '%s' "$text" | wc -c)" -lt 400 ]; then
    echo "사용량 한도 응답 — 결과를 쓰지 않고 다음 기회로 넘긴다"
    rm -f "$out"; continue
  fi
  CL_COST=$(jq -r '.total_cost_usd // 0' "$out" 2>/dev/null); CL_COST=${CL_COST:-0}
  rm -f "$out" "$err"
  python3 - "$text" <<'PY'
import json, re, sys
t = sys.argv[1]
m = None
# 마지막 줄에 온전한 JSON 이 오는 것이 정상이지만, 앞뒤에 말이 붙어도 "rules" 를 가진 가장 앞의 객체를 찾는다
for start in [i for i, ch in enumerate(t) if ch == "{"]:
    for end in range(len(t), start, -1):
        if t[end-1] != "}":
            continue
        try:
            d = json.loads(t[start:end])
        except Exception:
            continue
        if isinstance(d, dict) and isinstance(d.get("rules"), list):
            m = d
            break
    if m:
        break
if not m:
    sys.exit(1)
rules = []
for r in m["rules"][:8]:
    if not isinstance(r, dict) or not str(r.get("rule", "")).strip():
        continue
    rules.append({"rule": str(r.get("rule", ""))[:300], "why": str(r.get("why", ""))[:300],
                  "projects": [str(p) for p in (r.get("projects") or [])][:12], "count": int(r.get("count") or 1)})
print(json.dumps({"rules": rules}, ensure_ascii=False))
PY
}

write_md(){ # $1=id $2=json 파일
  local id=$1 j=$2 md="$OUT_DIR/$id.md"
  if [ "$(jq -r '.rules|length' "$j")" -eq 0 ]; then rm -f "$md"; return 0; fi
  {
    echo "## 이 캠페인에서 다른 저장소가 이미 걸린 것 — 시작 전에 확인하고, 끝내기 전에 다시 확인하세요"
    echo "(캠페인 \"$id\" 의 리뷰 거절·심사 거절·회귀 $(jq -r '.items' "$j")건에서 정제한 규칙. 러너가 자동으로 붙였습니다.)"
    jq -r '.rules[] | "- **\(.rule)** — \(.why) (근거: \(.projects|join(", ")), \(.count)회)"' "$j"
  } > "$md"
}

total_cost=0; changed=0
while IFS=$'\t' read -r id goal64; do
  [ -n "$id" ] || continue
  [ -z "$ONLY" ] || [ "$ONLY" = "$id" ] || continue
  ev=$(mktemp); collect "$id" | awk 'NF' | sort -u > "$ev"
  n=$(wc -l < "$ev" | tr -d ' ')
  hash=$(sha256sum "$ev" | cut -c1-16); j="$OUT_DIR/$id.json"
  if [ "$DRY" -eq 1 ]; then
    echo "=== $id — 증거 $n건 (hash $hash, 저장된 $(jq -r '.input_hash // "없음"' "$j" 2>/dev/null))"; head -6 "$ev"; [ "$n" -gt 6 ] && echo "  … (+$((n-6)))"; echo
    rm -f "$ev"; continue
  fi
  if [ "$n" -eq 0 ]; then
    [ -f "$j" ] || jq -cn --arg id "$id" --arg g "$(date -Iseconds)" --arg h "$hash" '{campaign:$id,generated:$g,input_hash:$h,items:0,source:"none",rules:[]}' > "$j"
    rm -f "$ev" "$OUT_DIR/$id.md"; continue
  fi
  if [ "$(jq -r '.input_hash // ""' "$j" 2>/dev/null)" = "$hash" ]; then rm -f "$ev"; continue; fi
  goal=$(base64 -d <<<"$goal64" 2>/dev/null)
  CL_COST=0; rules=$(distill "$id" "$goal" "$ev"); src=claude
  if [ -z "$rules" ]; then
    # 모델이 답을 못 냈으면 원문을 그대로 둔다 — 규칙으로 다듬지 못했어도 다음 저장소가 읽을 값은 있다
    rules=$(jq -Rs '{rules: (split("\n") | map(select(length>0)) | .[0:8] | map({rule: (.[0:300]), why: "원문 그대로 (정제 실패)", projects: [], count: 1}))}' "$ev"); src=raw
  fi
  jq -cn --arg id "$id" --arg g "$(date -Iseconds)" --arg h "$hash" --argjson n "$n" --arg s "$src" --argjson c "${CL_COST:-0}" --argjson r "$rules" \
    '{campaign:$id,generated:$g,input_hash:$h,items:$n,source:$s,cost_usd:$c,rules:$r.rules}' > "$j"
  write_md "$id" "$j"
  jq -cn --arg ts "$(date -Iseconds)" --arg d "$(date +%F)" --arg id "$id" --argjson c "${CL_COST:-0}" \
    '{ts:$ts,date:$d,project:"(campaign)",phase:"campaign-lessons",run_id:"",campaign:$id,subtype:"success",cost_usd:$c}' >> "$DATA/usage.jsonl"
  total_cost=$(awk -v a="$total_cost" -v b="${CL_COST:-0}" 'BEGIN{print a+b}'); changed=$((changed+1))
  echo "$id: 증거 $n건 → 규칙 $(jq -r '.rules|length' "$j")개 ($src, \$$CL_COST)"
  rm -f "$ev"
done < <(jq -r '.campaigns[]? | select(.done!=true and (.id|startswith("example-")|not)) | "\(.id)\t\(.goal|@base64)"' "$STATE/campaigns.json" 2>/dev/null)
[ "$DRY" -eq 1 ] || echo "campaign-lessons: 갱신 $changed개, \$$total_cost"
if [ "$changed" -gt 0 ]; then
  "$HERE/tg.sh" "📚 캠페인 교훈 갱신 — $changed개 캠페인 (\$$total_cost). 다음 회차부터 프롬프트와 PR 처리기 심사에 붙습니다.
https://hkjang.github.io/aidev/#campaigns" >/dev/null 2>&1 || true
fi
exit 0
