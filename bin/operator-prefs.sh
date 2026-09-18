#!/usr/bin/env bash
# 운영자 취향 학습 — hkjang 이 반복해서 말한 것을 규칙으로 만들어 모든 회차와 심사에 붙인다.
#
# 운영자는 PR 을 반려하며 이유를 남기고, 코파일럿에 지시하고, 러너를 멈추며 사유를 적는다. 그 말들은
# 흩어져 있고 다음 회차의 에이전트는 모른다 — 그래서 같은 이유로 또 반려된다("attribution 넣지 마",
# "가이드에 없는 기능 지어내지 마"). 이 스크립트는 그 증거를 모아 Claude 에게 최대 10개의 규칙으로
# 정제하게 하고 state/operator-preferences.md 에 둔다. 러너는 개선 프롬프트($OPERATOR_PREFS)와
# PR 처리기 심사(HOLD_NOTE)에 붙인다. 증거가 바뀌었을 때만 모델을 부른다.
#
#   bin/operator-prefs.sh [--dry-run]    헬스체크(30분)가 부른다.
set -uo pipefail
export HOME="${HOME:-/home/hkjang}"
export PATH="$HOME/.local/bin:$HOME/.nvm/versions/node/v22.23.1/bin:/usr/local/bin:/usr/bin:/bin"
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
REPO_DIR="$(cd "$HERE/.." && pwd)"; STATE="${AIDEV_STATE:-$REPO_DIR/state}"; DATA="${AIDEV_DATA:-$REPO_DIR/docs/data}"
OUT_JSON="$STATE/operator-preferences.json"; OUT_MD="$STATE/operator-preferences.md"
HIST="$HOME/.auto-improve/copilot-history.jsonl"
MODEL="${MODEL:-claude-opus-5}"; BUDGET="${AIDEV_PREFS_BUDGET:-1.0}"
DRY=0; [ "${1:-}" = --dry-run ] && DRY=1

ev=$(mktemp)
{
  # 1) 사람이 반려한 PR 의 사유·교훈
  jq -r 'select(.kind=="rejected-by-human" or .kind=="review-rejected" or .kind=="demoted" or .kind=="rolled-back") | "[\(.project) · \(.kind)] " + ((.detail//"")|gsub("\n";" ")|.[0:300])' "$STATE/lessons.jsonl" 2>/dev/null
  # 2) 긴급 중지 사유
  for f in "$STATE"/STOP*; do [ -f "$f" ] && echo "[중지 $(basename "$f")] $(head -c 200 "$f")"; done
  # 3) 코파일럿에 한 지시 (사용자 발화만, 짧은 인사·질문은 제외)
  [ -s "$HIST" ] && jq -r 'select(.role=="user") | .text' "$HIST" 2>/dev/null | awk 'length($0) > 8' | tail -n 60 | sed 's/^/[코파일럿 지시] /'
  # 4) 정책 파일에 사람이 적은 메모
  for f in "$STATE"/*.policy.json; do jq -r --arg p "$(basename "$f" .policy.json)" 'select(.note // .memo // "" != "") | "[정책 \($p)] \(.note // .memo)"' "$f" 2>/dev/null; done
} | awk 'NF' | sort -u > "$ev"
n=$(wc -l < "$ev" | tr -d ' '); hash=$(sha256sum "$ev" | cut -c1-16)
if [ "$DRY" -eq 1 ]; then echo "증거 $n건 (hash $hash, 저장된 $(jq -r '.input_hash // "없음"' "$OUT_JSON" 2>/dev/null))"; cat "$ev"; rm -f "$ev"; exit 0; fi
[ "$n" -gt 0 ] || { rm -f "$ev"; exit 0; }
[ "$(jq -r '.input_hash // ""' "$OUT_JSON" 2>/dev/null)" = "$hash" ] && { rm -f "$ev"; exit 0; }

prompt="당신은 자율 개선 러너(aidev)를 운영하는 사람(hkjang)의 취향을 정리합니다. 아래는 그 사람이 PR 을 반려하며 남긴 이유, 러너를 멈추며 적은 사유, 코파일럿에 한 지시들의 원문입니다. 앞으로 코드를 고치는 에이전트와 PR 을 심사하는 에이전트가 **시작 전에 읽고 그대로 따를** 규칙으로 정제하세요.

## 원문
$(cat "$ev")

## 규칙을 쓰는 법
- 최대 10개. 두 번 이상 반복된 것, 어기면 반려로 이어지는 것부터.
- 한 줄짜리 지시문으로: 무엇을 하라/하지 말라 + 왜(한 문장). 특정 프로젝트 이름은 그 프로젝트에만 해당할 때만 적습니다.
- 일회성 명령(\"weekly 멈춰\" 같은 상황 지시)은 취향이 아니므로 빼세요. 되풀이되는 판단 기준만 남깁니다.
- 원문에 없는 것을 지어내지 마세요. 확실하지 않으면 뺍니다.

마지막 줄에 다음 JSON 한 줄만 쓰세요(설명·코드펜스 없이):
{\"rules\":[{\"rule\":\"...\",\"why\":\"...\",\"count\":1}]}"
out=$(mktemp)
timeout -k 30 300 claude -p "$prompt" --model "$MODEL" --settings '{"attribution":{"commit":"","pr":""}}' \
  --permission-mode plan --allowedTools "" --max-budget-usd "$BUDGET" --output-format json </dev/null >"$out" 2>/dev/null
text=$(jq -r '.result // ""' "$out" 2>/dev/null); cost=$(jq -r '.total_cost_usd // 0' "$out" 2>/dev/null); cost=${cost:-0}; rm -f "$out"
rules=$(python3 - "$text" <<'PY'
import json, sys
t = sys.argv[1]; m = None
for start in [i for i, ch in enumerate(t) if ch == "{"]:
    for end in range(len(t), start, -1):
        if t[end-1] != "}": continue
        try: d = json.loads(t[start:end])
        except Exception: continue
        if isinstance(d, dict) and isinstance(d.get("rules"), list): m = d; break
    if m: break
if not m: sys.exit(1)
rules = [{"rule": str(r.get("rule", ""))[:300], "why": str(r.get("why", ""))[:300], "count": int(r.get("count") or 1)}
         for r in m["rules"][:10] if isinstance(r, dict) and str(r.get("rule", "")).strip()]
print(json.dumps({"rules": rules}, ensure_ascii=False))
PY
) || rules=""
[ -n "$rules" ] || { echo "정제 실패 — 규칙을 갱신하지 않음"; rm -f "$ev"; exit 0; }
jq -cn --arg g "$(date -Iseconds)" --arg h "$hash" --argjson n "$n" --argjson c "$cost" --argjson r "$rules" '{generated:$g,input_hash:$h,items:$n,cost_usd:$c,rules:$r.rules}' > "$OUT_JSON"
if [ "$(jq -r '.rules|length' "$OUT_JSON")" -gt 0 ]; then
  { echo "## 운영자(hkjang)가 되풀이해 말한 것 — 이대로 한다 (반려·중지 사유·지시 $n건에서 정제, 러너가 자동으로 붙임)"
    jq -r '.rules[] | "- **\(.rule)** — \(.why)"' "$OUT_JSON"; } > "$OUT_MD"
else rm -f "$OUT_MD"; fi
jq -cn --arg ts "$(date -Iseconds)" --arg d "$(date +%F)" --argjson c "$cost" '{ts:$ts,date:$d,project:"(runner)",phase:"operator-prefs",run_id:"",campaign:"",subtype:"success",cost_usd:$c}' >> "$DATA/usage.jsonl"
echo "운영자 취향 갱신: 증거 $n건 → 규칙 $(jq -r '.rules|length' "$OUT_JSON")개 (\$$cost)"
rm -f "$ev"; exit 0
