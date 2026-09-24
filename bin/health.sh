#!/usr/bin/env bash
# 헬스체크 + 자기 복구 — Windows 작업 스케줄러(AutoImproveHealth)가 30분마다 부른다.
#   - 마지막 회차가 3시간 넘게 없으면 경고 (스케줄러 꺼짐·WSL 다운·gh 인증 만료·잠금 고착)
#   - run.sh 가 3시간 넘게 돌고 있으면(claude 가 매달림) 죽여서 잠금을 푼다
#   - gh 인증, 디스크, 잠금 상태를 docs/data/health.json 에 남기고 문제가 있으면 notify.sh 채널로 알린다
set -uo pipefail
export HOME=/home/hkjang
export PATH="$HOME/.local/bin:$HOME/.nvm/versions/node/v22.23.1/bin:$HOME/miniconda3/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin:/mnt/c/WINDOWS/system32:/mnt/c/WINDOWS/System32/WindowsPowerShell/v1.0"
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
REPO_DIR="$(cd "$HERE/.." && pwd)"
RUNS="$REPO_DIR/docs/data/runs.jsonl"; OUT="$REPO_DIR/docs/data/health.json"; LOCK="$HOME/.auto-improve/run.lock"
STALE_SEC=$((3*3600)); KICK_SEC=$((40*60)); now=$(date +%s); problems=(); actions=()

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
# 자기 기록이 원격에 올라가고 있나 — 회차는 도는데 push 가 막히면 대시보드·논문 데이터가 조용히 과거에 멈춘다.
if [ -n "$(find "$REPO_DIR/.git/index.lock" -mmin +10 2>/dev/null)" ] && ! pgrep -x git >/dev/null 2>&1; then
  rm -f "$REPO_DIR/.git/index.lock" && actions+=("멈춘 .git/index.lock 을 제거했다 — 커밋이 막혀 있었다")
fi
sync_fail=$(cat "$REPO_DIR/state/.sync-fail" 2>/dev/null || echo 0)
[ "${sync_fail:-0}" -ge 2 ] && problems+=("aidev 자기 동기화 ${sync_fail}회 연속 실패 — logs/sync.log 확인")
ahead=$(git -C "$REPO_DIR" rev-list --count origin/main..HEAD 2>/dev/null || echo 0)
[ "${ahead:-0}" -gt 30 ] && problems+=("원격에 못 올린 커밋 ${ahead}개 — push 가 막혀 있다 (100MB 초과 파일/인증 확인, logs/sync.log)")
gh auth status >/dev/null 2>&1 || problems+=("gh 인증 실패 — gh auth login 필요")
disk=$(df -P "$REPO_DIR" | awk 'NR==2{print $5}' | tr -d '%'); [ "${disk:-0}" -gt 90 ] && problems+=("디스크 ${disk}% 사용")
docker info >/dev/null 2>&1 || problems+=("docker 를 쓸 수 없음 — 자산 빌드 실패 예상")
# 자기 복구: 예전 러너가 남긴 pushurl=DISABLED(저장소 공통 설정)를 발견하면 풀어준다 — 사용자 push 를 막아서는 안 된다
for d in "${ROOT:-/mnt/c/Users/USER/projects}"/*/; do
  [ -d "$d/.git" ] || continue
  if [ "$(git -C "$d" config --get remote.origin.pushurl 2>/dev/null)" = DISABLED ]; then
    git -C "$d" config --unset remote.origin.pushurl && actions+=("$(basename "$d"): pushurl=DISABLED 제거")
  fi
done
sched=$(schtasks.exe /Query /TN AutoImprove /FO LIST 2>/dev/null | iconv -f cp949 -t utf-8 2>/dev/null | tr -d '\r' | grep -E "^(상태|Status):" | head -1 | sed -E 's/^[^:]+:[[:space:]]*//')
next_run=$(schtasks.exe /Query /TN AutoImprove /FO LIST 2>/dev/null | iconv -f cp949 -t utf-8 2>/dev/null | tr -d '\r' | grep -E "^(다음 실행 시간|Next Run Time):" | head -1 | sed -E 's/^[^:]+:[[:space:]]*//')

# 자기 복구: 회차가 끊겼는데 러너도 없고 멈춤·상한 사유도 없으면 스케줄러를 직접 깨운다.
# (2026-09-10: PC 절전 후 부팅 시 트리거가 0x800710E0 으로 거부되며 7시간 동안 한 건도 안 돌았다.
#  30분마다 도는 이 스크립트가 다음 점검에서 되살리도록 한다.)
kick_stamp="$HOME/.auto-improve/.kicked"
if [ "$since_last" -gt "$KICK_SEC" ] && [ -z "$run_pid" ] \
   && [ ! -f "$REPO_DIR/state/STOP" ] && [ ! -f "$REPO_DIR/state/.cap-$(date +%F)" ] \
   && [ "$(( now - $(stat -c %Y "$kick_stamp" 2>/dev/null || echo 0) ))" -gt 1800 ]; then
  case "${sched:-}" in
    *실행*|*Running*) ;;                       # 이미 돌고 있으면 건드리지 않는다
    *) if schtasks.exe /Run /TN AutoImprove >/dev/null 2>&1; then
         date +%s > "$kick_stamp"
         actions+=("회차가 $((since_last/60))분째 없어 AutoImprove 를 직접 실행했다")
       else
         problems+=("AutoImprove 재실행 실패 — 작업 스케줄러 확인 필요")
       fi;;
  esac
fi

mkdir -p "$(dirname "$OUT")"
jq -cn --arg ts "$(date -Iseconds)" --arg last "$last_ts" --argjson since "$since_last" --arg pid "${run_pid:-}" --argjson age "${run_age:-0}" \
   --arg sched "${sched:-unknown}" --arg next "${next_run:-}" --argjson disk "${disk:-0}" --argjson problems "$(printf '%s\n' "${problems[@]:-}" | jq -R . | jq -s 'map(select(length>0))')" \
   --argjson actions "$(printf '%s\n' "${actions[@]:-}" | jq -R . | jq -s 'map(select(length>0))')" \
   '{checked:$ts,last_run:$last,seconds_since_last:$since,runner_pid:$pid,runner_age_seconds:$age,scheduler_status:$sched,scheduler_next_run:$next,disk_percent:$disk,
     ok:(($problems|length)==0),problems:$problems,actions:$actions}' > "$OUT"
echo "health: ok=$(jq -r .ok "$OUT") problems=${#problems[@]} actions=${#actions[@]}"

# 문제나 복구 조치가 있으면 알린다 (같은 문제는 3시간에 한 번만)
if [ ${#problems[@]} -gt 0 ] || [ ${#actions[@]} -gt 0 ]; then
  stamp="$HOME/.auto-improve/.health-notified"; key=$(printf '%s|' "${problems[@]:-}" "${actions[@]:-}" | md5sum | cut -c1-8)
  if [ "$(cat "$stamp" 2>/dev/null)" != "$key" ] || [ $(( now - $(stat -c %Y "$stamp" 2>/dev/null || echo 0) )) -gt 10800 ]; then
    echo "$key" > "$stamp"
    text="🩺 aidev 헬스체크 $(date '+%m-%d %H:%M')
$(printf -- '- %s\n' "${problems[@]:-}" "${actions[@]:-}" | grep -v '^- $')"
    "$HERE/tg.sh" "$text" >/dev/null 2>&1 || true
    [ -f "$HOME/.auto-improve/notify.env" ] && { set -a; . "$HOME/.auto-improve/notify.env"; set +a; }
    [ -n "${AIDEV_SLACK_WEBHOOK:-}" ] && curl -s -m 15 -X POST -H 'Content-type: application/json' --data "$(jq -cn --arg t "$text" '{text:$t}')" "$AIDEV_SLACK_WEBHOOK" >/dev/null
    if command -v powershell.exe >/dev/null 2>&1; then
      ps1=$(wslpath -w "$HERE/toast.ps1" 2>/dev/null)
      [ -n "$ps1" ] && powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$ps1" -Title "aidev 헬스체크" -Message "${problems[0]:-${actions[0]:-}}" >/dev/null 2>&1
    fi
  fi
fi
# CI 가 아예 없는 저장소: PR 이 영구히 "CI 검사 없음" 으로 막힌다 (2026-09-24 sqlon 11건·jasql 5건).
# 검사를 건너뛰게 하는 대신 **CI 를 만들게** 한다 — 정찰이 고르도록 아이디어로 넣어 둔다.
# .github/workflows 는 보호 경로라 그 PR 은 사람 승인을 받게 되고, 한 번 승인하면 그 저장소는 영구히 풀린다.
ci_gap=()
for d in "${ROOT:-/mnt/c/Users/USER/projects}"/*/; do
  n=$(basename "$d"); [ -d "$d/.git" ] || continue
  case "$n" in aidev|headcount|Naviq|sqlpad) continue;; esac
  [ "$(ls "$d/.github/workflows" 2>/dev/null | grep -cE '\.ya?ml$')" = 0 ] || continue
  [ "$(jq -r '.allow_merge_without_ci // false' "$REPO_DIR/state/$n.policy.json" 2>/dev/null)" = true ] && continue
  [ "$(jq -r --arg p "$n" '.[$p].open // 0' "$REPO_DIR/state/open-prs.json" 2>/dev/null || echo 0)" -ge 1 ] || continue
  f="$REPO_DIR/state/$n.ideas.json"; [ -s "$f" ] || echo '[]' > "$f"
  title="GitHub Actions CI 워크플로 추가 — 러너가 돌리는 검증 명령을 그대로 CI 로"
  jq --arg t "$title" --arg d "$(date +%F)" \
     'if any(.[]?; .title==$t and (.status=="pending" or .status=="done")) then . else . + [{title:$t,value:5,risk:1,size:"S",status:"pending",updated:$d,note:"이 저장소에는 워크플로 파일이 없어 러너 PR 이 전부 \"CI 검사 없음\" 으로 막힌다. 러너가 이미 돌리는 검증 명령(build·vet·test)을 그대로 .github/workflows/ci.yml 로 옮겨라. 이 파일은 보호 경로라 사람 승인을 받는다."}] end' \
     "$f" > "$f.tmp" 2>/dev/null && mv "$f.tmp" "$f" || rm -f "$f.tmp"
  # 아이디어만 넣으면 그 저장소는 WIP 상한·성과 쿨다운에 걸려 회차가 안 잡힐 수 있다 —
  # 막혔기 때문에 막힌 것을 못 푸는 교착이다. 수동 작업 큐로 한 회차를 직접 배정한다
  # (run-queue 는 상한·쿨다운과 무관하게 잡히고, 회차가 끝나면 줄에서 지워진다).
  q="$REPO_DIR/state/run-queue.tsv"; touch "$q"
  stamp="$HOME/.auto-improve/.ci-queued-$n"
  if ! grep -q -P "^$n\t" "$q" 2>/dev/null \
     && [ $(( now - $(stat -c %Y "$stamp" 2>/dev/null || echo 0) )) -gt 604800 ]; then
    date +%s > "$stamp"
    printf '%s\t%s\t\t\t%s\n' "$n" "CI 공백 메우기(자동 배정)" \
      "이 저장소에는 .github/workflows 파일이 하나도 없어 러너가 연 PR 이 전부 'CI 검사 없음' 으로 막혀 있습니다(현재 $(jq -r --arg p "$n" '.[$p].open // 0' "$REPO_DIR/state/open-prs.json" 2>/dev/null || echo 0)건).\n\n러너가 이미 이 저장소에서 돌리는 검증 명령(빌드·정적분석·테스트)을 그대로 .github/workflows/ci.yml 로 옮기세요.\n- push 와 pull_request 에서 돌 것\n- 러너가 쓰는 것과 같은 언어 버전·의존성 설치 단계\n- 새 테스트를 쓰거나 기존 테스트를 고치지 말 것. 지금 통과하는 것만 CI 로 옮기는 작업입니다\n- 워크플로 파일은 보호 경로라 이 PR 은 사람 승인을 받습니다. 그래도 됩니다" >> "$q"
    echo "$n: CI 워크플로 추가를 수동 작업 큐에 배정 (PR 이 전부 'CI 검사 없음' 으로 막혀 있다)"
  fi
  ci_gap+=("$n")
done
if [ ${#ci_gap[@]} -gt 0 ]; then
  cg_stamp="$HOME/.auto-improve/.ci-gap"
  if [ "$(cat "$cg_stamp" 2>/dev/null)" != "${ci_gap[*]}" ]; then
    printf '%s' "${ci_gap[*]}" > "$cg_stamp"
    "$HERE/tg.sh" "🧪 CI 워크플로가 없어 PR 이 막히는 저장소: ${ci_gap[*]}
정찰 아이디어로 'CI 워크플로 추가' 를 넣었습니다. 그 PR 은 보호 경로라 사람 승인이 필요합니다.
지금 당장 풀려면: state/<이름>.policy.json 에 {\"allow_merge_without_ci\": true}" >/dev/null 2>&1 || true
  fi
fi

# PR 정리: 6시간마다. 처리기가 "더 못 한다"고 판정하고 3일이 지난 PR 은 닫고 일감은 ideas 로 회수한다.
# 그냥 오래되기만 한 PR 은 목록(state/stale-prs.md)에만 남긴다 — 그건 사람이 bin/pr-gc.sh --close 로.
prgc_stamp="$HOME/.auto-improve/.prgc"
if [ $(( now - $(stat -c %Y "$prgc_stamp" 2>/dev/null || echo 0) )) -gt 21600 ]; then
  date +%s > "$prgc_stamp"; "$HERE/pr-gc.sh" >>"$REPO_DIR/logs/pr-gc.log" 2>&1 || true
fi
# 러너가 스스로 설명하지 못하는 것이 있으면 물어본다. 안에 전체 간격(기본 2시간)과
# 같은 것을 다시 묻지 않는 표시가 있어 헬스체크마다 부르는 것이 안전하다.
"$HERE/ask-claude.sh" >>"$REPO_DIR/logs/ask-claude.log" 2>&1 || true
# 캠페인 교훈: 리뷰·심사 거절과 회귀를 캠페인마다 일반 규칙으로 정제해 다음 저장소 회차에 붙인다.
# 증거가 바뀌었을 때만 모델을 부른다 (입력 해시 비교).
"$HERE/campaign-lessons.sh" >>"$REPO_DIR/logs/campaign-lessons.log" 2>&1 || true
# 운영자 취향: 사람의 반려 사유·중지 사유·코파일럿 지시를 규칙으로 정제해 모든 회차·심사에 붙인다.
"$HERE/operator-prefs.sh" >>"$REPO_DIR/logs/operator-prefs.log" 2>&1 || true
# 머지 뒤 수정 필요율: 러너가 머지한 PR 이 건드린 파일이 30일 안에 다시 고쳐졌는지 (6시간마다).
pm="$REPO_DIR/docs/data/postmerge.json"
if [ ! -f "$pm" ] || [ $(( now - $(stat -c %Y "$pm" 2>/dev/null || echo 0) )) -gt 21600 ]; then
  python3 "$HERE/postmerge.py" >>"$REPO_DIR/logs/postmerge.log" 2>&1 || true
  # 이사회: 주마다 한 번, 경영진 부서 세션이 회사 상태를 읽고 다음 주 결정을 제안한다 (적용은 사람: ops.sh board apply)
  [ -f "$REPO_DIR/state/board/$(date +%G-W%V).json" ] || "$HERE/board.sh" >>"$REPO_DIR/logs/board.log" 2>&1 || true
  # 비교 실험(state/experiment.json)이 켜져 있으면 arm 별 지표 표를 같이 갱신한다
  [ "$(jq -r '.enabled // false' "$REPO_DIR/state/experiment.json" 2>/dev/null)" = true ] && python3 "$HERE/exp-analyze.py" >>"$REPO_DIR/logs/exp-analyze.log" 2>&1 || true
fi

# 프로젝트별 가이드를 docs/guides 로 모은다 — 25개 저장소를 열어 보지 않고 한곳에서 본다.
# 내용이 같으면 git 이 새 blob 을 만들지 않으므로, 회차가 문서를 다시 쓸 때만 늘어난다.
python3 "$HERE/collect-guides.py" >/dev/null 2>&1 || true
# 모아 둔 가이드를 AppStore 의 앱마다 첨부한다 — 바뀐 것만, 가이드 문서만.
# 설정(~/.auto-improve/appstore.env)이 없으면 조용히 넘어간다.
python3 "$HERE/publish-guides.py" >>"$REPO_DIR/logs/publish-guides.log" 2>&1 || true

# health.json 을 사이트에 반영 (회차가 안 도는 상황이 바로 이 스크립트가 잡는 것이므로 직접 푸시)
( flock -w 300 9 || exit 1; cd "$REPO_DIR" && rm -rf .git/rebase-merge .git/rebase-apply 2>/dev/null; { pgrep -x git >/dev/null 2>&1 || rm -f .git/index.lock; } 2>/dev/null; git add docs/data/health.json docs/guides >/dev/null 2>&1 && { git diff --cached --quiet || git -c user.name=hkjang -c user.email=gagagiga@naver.com commit -qm "health: $(date '+%F %H:%M') $( [ ${#problems[@]} -eq 0 ] && echo ok || echo "${#problems[@]} problem(s)")"; } \
  && { git pull -q --rebase origin main || { git rebase --abort >/dev/null 2>&1; git add docs/data/health.json; git pull -q --rebase origin main; }; } && git push -q origin main ) 9>"$HOME/.auto-improve/sync.lock" >>"$REPO_DIR/logs/sync.log" 2>&1 || true
exit 0
