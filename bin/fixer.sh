#!/usr/bin/env bash
# 오류 대응 전용 트랙 — 자율 개선 러너가 흘린 '오류로 끝난 회차'를 모아 따로 고친다.
#
# 메인 러너(bin/daily.sh)는 새 개선을 돌리고, 이쪽은 오류 수정만 맡는다. 그동안
# error·verify-failed·infra-error·릴리즈 실패는 릴리즈 워크플로 2회 실패를 빼면
# 아무도 다시 잡지 않았다 (2026-09-15 hunter 릴리즈 실패가 그대로 남았다).
#
#   흐름: 최근 오류 회차 스캔 → fix-queue 에 적재(중복·이미 회복분 제외) → run.sh --fix-only
#
# 락: 메인 러너와 같은 run.lock 을 공유한다. 두 프로세스가 동시에 워크트리를 건드리지
# 않게 하려는 것이다 — 락을 못 잡으면 이번엔 적재만 하고 처리는 다음 기회로 넘긴다
# (적재된 항목은 메인 러너도 fix-queue 에서 집어 고칠 수 있다).
#
# 스케줄: Windows 작업 스케줄러에 daily.sh 와 별도로 건다 (예: 15분마다 wsl.exe 로).
#   끄기: state/NO-FIXER 파일을 만들면 이 스크립트는 아무것도 하지 않는다.
set -uo pipefail
export HOME=/home/hkjang
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.nvm/versions/node/v22.23.1/bin:$HOME/miniconda3/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin:/mnt/c/WINDOWS/system32:/mnt/c/WINDOWS/System32/WindowsPowerShell/v1.0"
export LANG=C.UTF-8
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"; export AIDEV_BIN="$HERE"
REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="$REPO_DIR/state"; DATA="$REPO_DIR/docs/data"; LOGS="$REPO_DIR/logs"
mkdir -p "$LOGS" "$HOME/.auto-improve"
exec >>"$LOGS/fixer.log" 2>&1
echo "===== $(date '+%F %T') fixer start"

[ -f "$STATE/NO-FIXER" ] && { echo "NO-FIXER — 중지 상태, 아무것도 하지 않는다"; exit 0; }

# 러너·원장이 원격에서 바뀌었을 수 있으니 먼저 따라간다
git -C "$REPO_DIR" pull -q --ff-only origin main || echo "warn: aidev pull 실패, 로컬로 진행"

RUNS="$DATA/runs.jsonl"; FIXQ="$STATE/fix-queue.tsv"; touch "$FIXQ"
since=$(date -d '-2 days' +%F)

# ── 1) 적재: 프로젝트별 '마지막 회차'가 오류로 끝났고 아직 fix-queue 에 없으면 넣는다.
#    마지막이 성공/no-change 면 이미 회복된 것이므로 넣지 않는다. sha 는 비운다
#    (머지된 게 없어 되돌릴 대상이 없다 — run.sh 의 롤백은 빈 sha 를 건너뛴다).
enq=0
if [ -s "$RUNS" ]; then
  while IFS=$'\t' read -r proj outcome reason; do
    [ -n "$proj" ] || continue
    grep -q -P "^$proj\t" "$FIXQ" && continue
    [ -f "$STATE/STOP-$proj" ] && continue
    printf '%s\t%s\t\n' "$proj" "오류 대응(자동 적재): 마지막 회차가 '$outcome' 로 끝났습니다. $reason" >> "$FIXQ"
    echo "enqueue $proj ($outcome)"; enq=$((enq+1))
  done < <(jq -rs --arg c "$since" '
      [ .[] | select(.date>=$c and (.project|type=="string") and (.project|startswith("(")|not)) ]
      | group_by(.project) | map(.[-1])
      | .[] | select(.outcome=="error" or .outcome=="verify-failed" or .outcome=="infra-error")
      | [.project, .outcome, ((.result//"")|gsub("[\t\n]";" ")|.[0:200])] | @tsv' "$RUNS" 2>/dev/null)
fi
# 릴리즈 status=failed 도 적재
for rj in "$STATE"/*.release.json; do
  [ -f "$rj" ] || continue
  [ "$(jq -r '.status // ""' "$rj" 2>/dev/null)" = failed ] || continue
  proj=$(basename "$rj" .release.json)
  grep -q -P "^$proj\t" "$FIXQ" && continue
  [ -f "$STATE/STOP-$proj" ] && continue
  tag=$(jq -r '.tag // ""' "$rj" 2>/dev/null); rsn=$(jq -r '.reason // ""' "$rj" 2>/dev/null)
  printf '%s\t%s\t\n' "$proj" "오류 대응(자동 적재): 릴리즈 실패($tag). $rsn" >> "$FIXQ"
  echo "enqueue $proj (release-failed $tag)"; enq=$((enq+1))
done
echo "적재 $enq건, fix-queue 총 $(wc -l < "$FIXQ")건"
[ "$enq" -gt 0 ] && "$HERE/tg.sh" "🧰 오류 대응 트랙 — $enq건을 fix-queue 에 담았습니다 (총 $(wc -l < "$FIXQ")건). 순서대로 고칩니다." >/dev/null 2>&1 &

# ── 2) 처리: run.sh --fix-only. 메인 러너와 같은 락을 공유(동시 실행 방지). 못 잡으면 적재만 하고 끝.
if [ ! -s "$FIXQ" ]; then echo "fix-queue 비어 있음 — 처리 생략"; echo "===== $(date '+%F %T') fixer done"; exit 0; fi
tmp=$(mktemp /tmp/aidev-fix.XXXXXX.sh) && cp "$HERE/run.sh" "$tmp"
setsid flock -n "$HOME/.auto-improve/run.lock" bash -c 'trap "rm -f $0" EXIT; bash "$0" --fix-only --count 1 "$@"' "$tmp" "$@" &
child=$!; wait "$child" || echo "fixer: 락 점유 중이거나 회차 실패 — 다음 기회에 처리"
echo "===== $(date '+%F %T') fixer done"
