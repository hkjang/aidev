#!/usr/bin/env bash
# 텔레그램 코파일럿 — 알림 봇에 답장하면 그것이 곧 운영 명령이다.
#
# 지금까지 봇은 말만 했다. "weekly 릴리즈 왜 실패했어?", "moina PR 승인해", "mcp-oauth 캠페인에서
# sqlon 빼", "오늘 뭐 했어?", "AgentHub 멈춰", "모든 서비스에 다크모드 넣는 캠페인 초안 잡아 줘" 를
# 폰에서 치면, 코파일럿 세션이 aidev 상태를 읽고 bin/ops.sh 의 동사로 바꿔 실행한 뒤 결과를 답한다.
#
# 안전 경계:
#   - 알림 설정의 그 대화(AIDEV_TELEGRAM_CHAT)에서 온 메시지만 받는다. 다른 대화는 기록만 남기고 무시.
#   - 세션이 실행할 수 있는 것은 bin/ops.sh 뿐이다(Read/Grep/Glob 은 읽기). 여기 없는 일은 못 한다.
#   - 바꾸는 동사(승인·반려·중지·캠페인 편집·릴리즈·캠페인 시작)는 메시지가 분명히 시켰을 때만.
#     애매하면 한 줄로 제안하고 묻는다. "ㅇㅇ/응/해" 같은 동의는 직전 제안에 대한 지시로 본다.
#   - 대화 기록은 저장소 밖(~/.auto-improve/copilot-history.jsonl)에 둔다.
#
# 실행: Windows 작업 스케줄러 AutoImproveCopilot 이 10분마다 부르고, 한 번 불리면 ~9.5분 동안
# 텔레그램 long-poll 을 반복한다(flock 으로 겹침 방지) → 사실상 상시 대기, 응답 지연 수 초 + 세션 시간.
#   끄기: state/NO-COPILOT
set -uo pipefail
export HOME=/home/hkjang
export PATH="$HOME/.local/bin:$HOME/.nvm/versions/node/v22.23.1/bin:$HOME/miniconda3/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"
export LANG=C.UTF-8
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"; export AIDEV_BIN="$HERE"
REPO_DIR="$(cd "$HERE/.." && pwd)"; STATE="$REPO_DIR/state"; DATA="$REPO_DIR/docs/data"; LOGS="$REPO_DIR/logs"
mkdir -p "$LOGS" "$HOME/.auto-improve"
exec 3>&1; exec >>"$LOGS/copilot.log" 2>&1
[ -f "$STATE/NO-COPILOT" ] && exit 0
[ -f "$HOME/.auto-improve/notify.env" ] && { set -a; . "$HOME/.auto-improve/notify.env"; set +a; }
: "${AIDEV_TELEGRAM_TOKEN:=}"; : "${AIDEV_TELEGRAM_CHAT:=}"
[ -n "$AIDEV_TELEGRAM_TOKEN" ] && [ -n "$AIDEV_TELEGRAM_CHAT" ] || { echo "$(date '+%F %T') notify.env 에 토큰/대화 id 없음"; exit 0; }
exec 9>"$HOME/.auto-improve/copilot.lock"; flock -n 9 || exit 0
API="https://api.telegram.org/bot$AIDEV_TELEGRAM_TOKEN"
OFF="$HOME/.auto-improve/tg.offset"; HIST="$HOME/.auto-improve/copilot-history.jsonl"; touch "$HIST"
MODEL="${MODEL:-claude-opus-5}"; BUDGET="${COPILOT_BUDGET:-1.5}"; LOOP=${COPILOT_LOOP_SECONDS:-560}
log(){ echo "[$(date '+%F %T')] $*"; }

reply(){ AIDEV_TG_QUIET_SECONDS=0 "$HERE/tg.sh" "$1" >/dev/null 2>&1 || true; }
hist_add(){ jq -cn --arg ts "$(date -Iseconds)" --arg r "$1" --arg t "$2" '{ts:$ts,role:$r,text:$t}' >> "$HIST"; }

handle(){ # $1=사용자 메시지
  local text=$1 history verbs prompt out answer cost
  hist_add user "$text"
  curl -s -m 5 -o /dev/null "$API/sendChatAction" --data-urlencode "chat_id=$AIDEV_TELEGRAM_CHAT" --data-urlencode "action=typing" || true
  history=$(tail -n 16 "$HIST" | head -n 15 | jq -r '"\(.ts[5:16]) \(.role): \(.text|.[0:500])"')
  verbs=$(bash "$HERE/ops.sh" help)
  prompt="당신은 aidev(자율 개선 러너)의 운영 코파일럿입니다. 운영자 hkjang 이 텔레그램으로 보낸 메시지에 답하고, 필요하면 실행합니다. 지금은 $(date '+%Y-%m-%d %H:%M') 입니다.

## 할 수 있는 것 — bin/ops.sh 의 동사뿐입니다 (\`bash bin/ops.sh <동사> ...\`)
$verbs
- 인자에 공백이 있으면 따옴표로 감쌉니다. 프로젝트 이름은 대소문자를 지킵니다 (예: AgentHub, weekly, ReSSO).
- 더 알아야 하면 읽으세요: 원장 state/<이름>.md, 회차 기록 docs/data/runs.jsonl, 요약 docs/data/summary.json, 캠페인 state/campaigns.json, 표준 *-STANDARD.md, 처리기 기록 state/shepherd.jsonl. 프로젝트 소스는 /mnt/c/Users/USER/projects/<이름>.

## 규칙
- 읽는 동사(status, project, why, inbox, cost, lessons, campaign list|show, drafts, stops, shepherd)는 자유롭게 실행해 사실을 확인한 뒤 답합니다. 짐작을 사실처럼 쓰지 마세요.
- 바꾸는 동사(approve, reject, run, stop, resume, campaign add|remove|pause|resume|budget|until, shepherd on|off, release, draft-campaign, activate-campaign, discard-draft)는 **이 메시지가 분명히 시켰을 때만** 실행합니다. 애매하면 무엇을 하려는지 한 줄로 제안하고 물어보세요. 직전 대화에서 당신이 제안한 것에 \"ㅇㅇ/응/해/고/오케이\" 처럼 동의하면 그것을 실행합니다.
- 실행했으면 무엇을 했고 결과가 무엇인지 그대로 전합니다. 실패했으면 실패했다고 합니다.
- 파일을 고치거나 다른 명령을 쓰지 마세요. bin/ops.sh 밖의 일을 시키면 할 수 없다고 말하고, 대신 가능한 것을 제안하세요.
- 답은 텔레그램 한 통입니다. 핵심을 먼저, 짧게(대개 3~8줄), 마크다운 기호(#, **, \`) 없이 평문으로. 링크는 그대로 씁니다. 한국어로.

## 최근 대화 (오래된 것부터)
${history:-(없음)}

## 지금 온 메시지
$text"
  out=$(cd "$REPO_DIR" && timeout -k 20 420 claude -p "$prompt" --model "$MODEL" --settings '{"attribution":{"commit":"","pr":""}}' \
        --permission-mode acceptEdits --allowedTools "Bash(bash bin/ops.sh:*) Bash(bin/ops.sh:*) Read Grep Glob" \
        --max-budget-usd "$BUDGET" --output-format json 2>>"$LOGS/copilot.err")
  answer=$(jq -r '.result // ""' <<<"$out" 2>/dev/null)
  cost=$(jq -r '.total_cost_usd // 0' <<<"$out" 2>/dev/null); cost=${cost:-0}
  [ -n "$answer" ] || answer="답을 만들지 못했습니다 (세션 오류 또는 한도). 잠시 뒤 다시 보내 주세요. 로그: logs/copilot.log"
  hist_add assistant "$answer"
  reply "$answer"
  jq -cn --arg ts "$(date -Iseconds)" --arg d "$(date +%F)" --argjson c "$cost" --argjson n "$(jq -r '.num_turns // 0' <<<"$out" 2>/dev/null || echo 0)" \
    '{ts:$ts,date:$d,project:"(copilot)",phase:"copilot",run_id:"",campaign:"",subtype:"success",cost_usd:$c,num_turns:$n}' >> "$DATA/usage.jsonl"
  log "답함 (\$$cost): $(cut -c1-120 <<<"$text" | tr '\n' ' ') → $(cut -c1-120 <<<"$answer" | tr '\n' ' ')"
}

# 시험: bin/copilot.sh --ask "메시지" — 텔레그램 없이 같은 세션을 돌려 답을 화면에 찍는다 (기록에는 남는다)
if [ "${1:-}" = --ask ]; then
  reply(){ printf '%s\n' "$1" >&3; }
  handle "${2:?메시지}"; exit 0
fi

deadline=$(( $(date +%s) + LOOP ))
while [ "$(date +%s)" -lt "$deadline" ]; do
  [ -f "$STATE/NO-COPILOT" ] && exit 0
  off=$(cat "$OFF" 2>/dev/null || echo 0)
  upd=$(curl -s -m 60 "$API/getUpdates?timeout=45&offset=$off&allowed_updates=%5B%22message%22%5D") || { sleep 5; continue; }
  jq -e '.ok == true' <<<"$upd" >/dev/null 2>&1 || { log "getUpdates 실패: $(cut -c1-200 <<<"$upd")"; sleep 10; continue; }
  n=$(jq '.result|length' <<<"$upd"); [ "${n:-0}" -gt 0 ] || continue
  echo "$(( $(jq '.result[-1].update_id' <<<"$upd") + 1 ))" > "$OFF"     # 처리 전에 밀어 둔다 — 세션이 죽어도 같은 메시지를 되풀이하지 않는다
  while IFS= read -r m; do
    chat=$(jq -r '.message.chat.id' <<<"$m"); text=$(jq -r '.message.text // ""' <<<"$m")
    [ "$chat" = "$AIDEV_TELEGRAM_CHAT" ] || { log "허용되지 않은 대화 $chat 의 메시지 무시"; continue; }
    [ -n "$text" ] || continue
    log "받음: $(cut -c1-160 <<<"$text" | tr '\n' ' ')"
    handle "$text"
  done < <(jq -c '.result[] | select(.message.text != null)' <<<"$upd")
done
exit 0
