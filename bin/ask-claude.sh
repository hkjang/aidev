#!/usr/bin/env bash
# 러너가 스스로 설명하지 못하는 것을 클로드에게 물어본다.
#
# 오늘 하루에만 같은 실패가 34번 되풀이됐고, 대상이 0개인 캠페인이 "전부 완료" 로
# 닫혔고, 통과할 수 없는 검사를 12번 다시 확인했다. 전부 로그에 있었지만 읽는 사람이
# 없었다. bin/triage.py 가 그런 자리를 찾아내면, 여기서 증거를 모아 진단을 청한다.
#
# 답은 **읽을 것**이지 실행할 것이 아니다. 이 스크립트는 머지도 릴리즈도 하지 않고,
# 저장소를 고치지도 않는다 — 진단을 적어 두고 알리는 데까지다. 무엇을 할지는 사람이나
# 수정 큐가 정한다. 스스로 이상하다고 느낀 것을 스스로 고치게 두면, 잘못 짚었을 때
# 그것을 바로잡을 자리가 남지 않는다.
#
#   bin/ask-claude.sh            찾은 것마다 한 번씩 묻는다
#   bin/ask-claude.sh --dry-run  무엇을 물을지만 보여 준다
set -uo pipefail
export HOME="${HOME:-/home/hkjang}"
export PATH="$HOME/.local/bin:$HOME/.nvm/versions/node/v22.23.1/bin:/usr/local/bin:/usr/bin:/bin:/mnt/c/WINDOWS/system32"
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="$REPO_DIR/state"; DATA="$REPO_DIR/docs/data"; LOGS="$REPO_DIR/logs"
DIAGNOSES="$STATE/diagnoses.jsonl"
SEEN="$HOME/.auto-improve/.asked"
MODEL="${MODEL:-claude-opus-5}"
BUDGET="${AIDEV_ASK_BUDGET:-1.5}"     # 진단 하나에 쓸 돈. 고치는 일이 아니라 읽는 일이다.
QUIET_SECONDS=${AIDEV_ASK_QUIET:-86400}
MAX_PER_RUN=${AIDEV_ASK_MAX:-2}
DRY=0; [ "${1:-}" = "--dry-run" ] && DRY=1

mkdir -p "$SEEN" "$(dirname "$DIAGNOSES")"
command -v claude >/dev/null 2>&1 || { echo "claude 를 찾지 못했습니다"; exit 0; }

# 전체 간격. 헬스체크가 30분마다 부르는데 그때마다 물으면 하루에 마흔 번이 넘는다.
# 이상은 대개 몇 시간을 두고 굳어지므로, 그보다 자주 물어봐야 새로 알 것이 없다.
COOLDOWN=${AIDEV_ASK_COOLDOWN:-7200}
last="$HOME/.auto-improve/.asked-last"
if [ "$DRY" -eq 0 ] && [ -f "$last" ] \
   && [ "$(( $(date +%s) - $(stat -c %Y "$last" 2>/dev/null || echo 0) ))" -lt "$COOLDOWN" ]; then
  exit 0
fi

asked=0
while IFS= read -r item; do
  [ -n "$item" ] || continue
  [ "$asked" -ge "$MAX_PER_RUN" ] && break
  kind=$(jq -r '.kind' <<<"$item"); subject=$(jq -r '.subject' <<<"$item")
  summary=$(jq -r '.summary' <<<"$item"); question=$(jq -r '.question' <<<"$item")
  key=$(printf '%s|%s' "$kind" "$subject" | md5sum | cut -c1-16); stamp="$SEEN/$key"
  # 같은 것을 매번 다시 묻지 않는다. 하루가 지나도 그대로면 다시 묻는다 — 안 풀렸다는
  # 사실 자체가 새 소식이다.
  if [ -f "$stamp" ] && [ "$(( $(date +%s) - $(stat -c %Y "$stamp" 2>/dev/null || echo 0) ))" -lt "$QUIET_SECONDS" ]; then
    continue
  fi

  evidence=$(jq -r '.evidence[]' <<<"$item")
  prompt="당신은 자율 개선 러너(aidev)를 지키는 사람입니다. 러너가 스스로 설명하지 못하는 것을 하나 가져왔습니다.

## 무엇이 이상한가
$summary

## 증거
$evidence

## 묻는 것
$question

## 답하는 방법
- 고치지 마세요. 이 자리에서는 **읽고 판단하는 것만** 합니다. 파일을 바꾸거나 명령을 실행하지 마세요.
- 필요하면 저장소를 읽어 확인하세요: 러너는 /mnt/c/Users/USER/projects/aidev 에 있고, 대상 저장소는 그 옆 폴더들입니다.
- 짐작을 사실처럼 쓰지 마세요. 확인한 것과 추측한 것을 구분해 주세요.
- 마지막 줄에 다음 JSON 한 줄만 덧붙이세요(설명 없이):
  {\"cause\":\"한 문장\",\"where\":\"runner|project|external|unknown\",\"action\":\"한 문장으로 다음에 할 일\",\"confidence\":\"high|medium|low\"}"

  if [ "$DRY" -eq 1 ]; then
    echo "=== [$kind] $subject"; echo "$summary"; echo "--- 증거"; echo "$evidence" | head -4; echo
    asked=$((asked+1)); continue
  fi

  out=$(mktemp); err=$(mktemp)
  timeout -k 30 600 claude -p "$prompt" --model "$MODEL" \
    --settings '{"attribution":{"commit":"","pr":""}}' --permission-mode plan \
    --allowedTools "Read Grep Glob Bash(git log:*) Bash(git show:*) Bash(git diff:*) Bash(gh pr view:*) Bash(gh run view:*)" \
    --max-budget-usd "$BUDGET" --output-format json >"$out" 2>"$err"
  text=$(jq -r '.result // ""' "$out" 2>/dev/null)
  cost=$(jq -r '.total_cost_usd // 0' "$out" 2>/dev/null)
  verdict=$(grep -oE '\{"cause".*\}' <<<"$text" | tail -1)
  : > "$stamp"

  jq -cn --arg ts "$(date -Iseconds)" --arg kind "$kind" --arg subject "$subject" \
     --arg summary "$summary" --arg text "$text" --argjson verdict "${verdict:-null}" \
     --argjson cost "${cost:-0}" \
     '{ts:$ts,kind:$kind,subject:$subject,summary:$summary,verdict:$verdict,cost_usd:$cost,answer:$text}' \
     >> "$DIAGNOSES"

  cause=$(jq -r '.cause // "?"' <<<"${verdict:-\{\}}" 2>/dev/null)
  where=$(jq -r '.where // "?"' <<<"${verdict:-\{\}}" 2>/dev/null)
  action=$(jq -r '.action // "?"' <<<"${verdict:-\{\}}" 2>/dev/null)
  conf=$(jq -r '.confidence // "?"' <<<"${verdict:-\{\}}" 2>/dev/null)
  echo "[$kind] $subject → $where / $conf · \$$cost"
  "$HERE/tg.sh" "🔍 러너가 이상한 것을 하나 물어봤습니다 — $subject

$summary

원인($conf, $where): $cause
다음에 할 일: $action

전문: state/diagnoses.jsonl" >/dev/null 2>&1 &
  rm -f "$out" "$err"
  asked=$((asked+1))
done < <(python3 "$HERE/triage.py" --json 2>/dev/null)

[ "$asked" -gt 0 ] && [ "$DRY" -eq 0 ] && : > "$last"
[ "$asked" -eq 0 ] && echo "물어볼 것 없음"
exit 0
