#!/usr/bin/env bash
# 헬스체크 + 자기 복구 — Windows 작업 스케줄러(AutoImproveHealth)가 30분마다 부른다.
#   - 마지막 회차가 3시간 넘게 없으면 경고 (스케줄러 꺼짐·WSL 다운·gh 인증 만료·잠금 고착)
#   - run.sh 가 3시간 넘게 돌고 있으면(claude 가 매달림) 죽여서 잠금을 푼다
#   - gh 인증, 디스크, 잠금 상태를 docs/data/health.json 에 남기고 문제가 있으면 notify.sh 채널로 알린다
set -uo pipefail
export HOME=/home/hkjang
export PATH="$HOME/.local/bin:$HOME/.nvm/versions/node/v22.23.1/bin:$HOME/miniconda3/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
REPO_DIR="$(cd "$HERE/.." && pwd)"
RUNS="$REPO_DIR/docs/data/runs.jsonl"; OUT="$REPO_DIR/docs/data/health.json"; LOCK="$HOME/.auto-improve/run.lock"
STALE_SEC=$((3*3600)); now=$(date +%s); problems=(); actions=()

last_ts=$(tail -n 1 "$RUNS" 2>/dev/null | jq -r '.ts // empty')
last_epoch=$(date -d "${last_ts:-1970-01-01}" +%s 2>/dev/null || echo 0)
since_last=$(( now - last_epoch ))

# 러너가 돌고 있나, 얼마나 오래
run_pid=$(pgrep -of "bash /tmp/aidev-run\.|bin/run\.sh" || true)
run_age=0; [ -n "$run_pid" ] && run_age=$(ps -o etimes= -p "$run_pid" 2>/dev/null | tr -d ' ' || echo 0)
if [ -n "$run_pid" ] && [ "${run_age:-0}" -gt "$STALE_SEC" ]; then
  pkill -TERM -P "$run_pid" 2>/dev/null; sleep 5; pkill -KILL -P "$run_pid" 2>/dev/null; kill -KILL "$run_pid" 2>/dev/null
  pkill -KILL -f "claude -p" 2>/dev/null
  actions+=("run.sh (pid $run_pid) 가 $((run_age/60))분째 돌고 있어 종료하고 잠금을 풀었다")
fi
# 잠금 고착: 잠금 파일을 잡은 프로세스가 없는데 flock -n 이 실패하면(비정상) 파일을 갈아 끼운다
if [ -f "$LOCK" ] && ! flock -n "$LOCK" true 2>/dev/null && [ -z "$run_pid" ]; then
  rm -f "$LOCK"; actions+=("소유 프로세스 없는 잠금 파일을 제거했다")
fi
[ "$since_last" -gt "$STALE_SEC" ] && [ -z "$run_pid" ] && problems+=("마지막 회차가 $((since_last/3600))시간 전 ($last_ts) — 스케줄러/WSL 확인 필요")
gh auth status >/dev/null 2>&1 || problems+=("gh 인증 실패 — gh auth login 필요")
disk=$(df -P "$REPO_DIR" | awk 'NR==2{print $5}' | tr -d '%'); [ "${disk:-0}" -gt 90 ] && problems+=("디스크 ${disk}% 사용")
docker info >/dev/null 2>&1 || problems+=("docker 를 쓸 수 없음 — 자산 빌드 실패 예상")
sched=$(schtasks.exe /Query /TN AutoImprove /FO CSV 2>/dev/null | tail -1 | tr -d '\r' | awk -F'","' '{print $3}' | tr -d '"')

mkdir -p "$(dirname "$OUT")"
jq -cn --arg ts "$(date -Iseconds)" --arg last "$last_ts" --argjson since "$since_last" --arg pid "${run_pid:-}" --argjson age "${run_age:-0}" \
   --arg sched "${sched:-unknown}" --argjson disk "${disk:-0}" --argjson problems "$(printf '%s\n' "${problems[@]:-}" | jq -R . | jq -s 'map(select(length>0))')" \
   --argjson actions "$(printf '%s\n' "${actions[@]:-}" | jq -R . | jq -s 'map(select(length>0))')" \
   '{checked:$ts,last_run:$last,seconds_since_last:$since,runner_pid:$pid,runner_age_seconds:$age,scheduler_status:$sched,disk_percent:$disk,
     ok:(($problems|length)==0),problems:$problems,actions:$actions}' > "$OUT"
echo "health: ok=$(jq -r .ok "$OUT") problems=${#problems[@]} actions=${#actions[@]}"

# 문제나 복구 조치가 있으면 알린다 (같은 문제는 3시간에 한 번만)
if [ ${#problems[@]} -gt 0 ] || [ ${#actions[@]} -gt 0 ]; then
  stamp="$HOME/.auto-improve/.health-notified"; key=$(printf '%s|' "${problems[@]:-}" "${actions[@]:-}" | md5sum | cut -c1-8)
  if [ "$(cat "$stamp" 2>/dev/null)" != "$key" ] || [ $(( now - $(stat -c %Y "$stamp" 2>/dev/null || echo 0) )) -gt 10800 ]; then
    echo "$key" > "$stamp"
    text="🩺 aidev 헬스체크 $(date '+%m-%d %H:%M')
$(printf -- '- %s\n' "${problems[@]:-}" "${actions[@]:-}" | grep -v '^- $')"
    [ -f "$HOME/.auto-improve/notify.env" ] && { set -a; . "$HOME/.auto-improve/notify.env"; set +a; }
    [ -n "${AIDEV_SLACK_WEBHOOK:-}" ] && curl -s -m 15 -X POST -H 'Content-type: application/json' --data "$(jq -cn --arg t "$text" '{text:$t}')" "$AIDEV_SLACK_WEBHOOK" >/dev/null
    if command -v powershell.exe >/dev/null 2>&1; then
      ps1=$(wslpath -w "$HERE/toast.ps1" 2>/dev/null)
      [ -n "$ps1" ] && powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$ps1" -Title "aidev 헬스체크" -Message "${problems[0]:-${actions[0]:-}}" >/dev/null 2>&1
    fi
  fi
fi
# health.json 을 사이트에 반영 (회차가 안 도는 상황이 바로 이 스크립트가 잡는 것이므로 직접 푸시)
( cd "$REPO_DIR" && git add docs/data/health.json >/dev/null 2>&1 && { git diff --cached --quiet || git -c user.name=hkjang -c user.email=gagagiga@naver.com commit -qm "health: $(date '+%F %H:%M') $( [ ${#problems[@]} -eq 0 ] && echo ok || echo "${#problems[@]} problem(s)")"; } \
  && git pull -q --rebase --autostash origin main && git push -q origin main ) >/dev/null 2>&1 || true
exit 0
