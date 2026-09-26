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
  # 사실 자체가 새 소식이다. 다만 되풀이될수록 간격을 늘린다: umm 의 stuck-dirty 는
  # 7일 연속 같은 답을 사서 같은 알림을 보냈고, 아무것도 달라지지 않았다 (2026-09-18~24).
  # 횟수 기록이 없으면 지난 진단 기록에서 센다 — 이 파일이 생기기 전의 반복도 반복이다
  [ -f "$stamp.n" ] || jq -r --arg k "$kind" --arg s "$subject" 'select(.kind==$k and .subject==$s)|.ts' "$DIAGNOSES" 2>/dev/null | wc -l > "$stamp.n"
  reps=$(cat "$stamp.n" 2>/dev/null || echo 0); reps=${reps:-0}
  mult=$(( reps < 1 ? 1 : (reps > 7 ? 7 : reps) ))
  if [ -f "$stamp" ] && [ "$(( $(date +%s) - $(stat -c %Y "$stamp" 2>/dev/null || echo 0) ))" -lt "$(( QUIET_SECONDS * mult ))" ]; then
    continue
  fi
  # 세 번 물어도 그대로면 더 사지 않는다 — 진단이 아니라 사람의 결정이 없는 것이다.
  if [ "$reps" -ge 3 ] && [ "$DRY" -eq 0 ]; then
    : > "$stamp"; echo "$(( reps + 1 ))" > "$stamp.n"
    case "$kind" in
      stuck-dirty) how="git -C ../$subject status --porcelain 으로 확인 → 커밋하거나, 생성물이면 state/$subject.policy.json 의 ignore_dirty 에 경로를 넣으세요";;
      *) how="state/diagnoses.jsonl 에서 [$kind] $subject 의 지난 진단을 보세요";;
    esac
    "$HERE/tg.sh" "🙋 [$kind] $subject — 같은 상태가 ${reps}번째입니다. 진단은 그만 사고 사람 결정을 기다립니다.
$summary

할 일: $how" >/dev/null 2>&1 &
    echo "[$kind] $subject → ${reps}회 반복, 질문 대신 에스컬레이션"
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
  # 사용량 한도 응답을 결과로 쓰지 않는다 — 짧은 안내문 한 줄이 보고서·교훈·취향으로
  # 저장되면 그 파일을 읽는 모든 회차가 그것을 규칙으로 읽는다 (2026-09-21 이사회가 그랬다).
  if grep -qiE "usage limit|limit reached|weekly limit|5-hour limit|rate.?limit|quota|credit balance|resets? at|resets [A-Z][a-z]{2} " <<<"$text" \
     && [ "$(printf '%s' "$text" | wc -c)" -lt 400 ]; then
    echo "사용량 한도 응답 — 결과를 쓰지 않고 다음 기회로 넘긴다"
    rm -f "$out" "$err"; continue
  fi
  cost=$(jq -r '.total_cost_usd // 0' "$out" 2>/dev/null)
  verdict=$(grep -oE '\{"cause".*\}' <<<"$text" | tail -1)
  : > "$stamp"; echo "$(( reps + 1 ))" > "$stamp.n"

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
