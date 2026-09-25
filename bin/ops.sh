#!/usr/bin/env bash
# 운영 동사 모음 — 텔레그램 코파일럿(bin/copilot.sh)이 자연어를 이 동사로 바꿔 실행한다. 사람이 셸에서 그대로 써도 된다.
#
# 동사 하나는 한 가지 일만 하고, 결과를 사람이 읽을 한국어 몇 줄로 돌려준다. 코파일럿 세션에는
# 이 스크립트만 실행 권한이 있으므로, 여기 없는 일은 코파일럿도 못 한다 — 그것이 안전 경계다.
#
#   읽기:  status · project <이름> · why <이름> · inbox · cost [일수] · lessons <이름|캠페인id> · campaign list|show <id> · drafts · stops
#   실행:  approve <PR url|이름> · reject <PR url|이름> [사유] · run <이름> [요청 명세] · stop <all|merge|release|이름> [사유] · resume <범위>
#          campaign add|remove <id> <이름> · campaign pause|resume <id> · campaign budget <id> <usd> · campaign until <id> <YYYY-MM-DD>
#          shepherd on|off · release <이름> · draft-campaign "<목표 문장>" · activate-campaign <slug> · discard-draft <slug>
#          board (이번 주 이사회 제안 보기) · board apply <번호...|all> (제안 적용) · board open (지금 열기)
set -uo pipefail

# gh pr edit --add-label 은 gh 2.45 에서 projectCards GraphQL 때문에 항상 실패한다 (2026-09-24). REST 로 붙인다.
ops_pr_label(){ # $1=PR URL $2=라벨
  local slug num; slug=$(sed -E 's#^https?://[^/]+/([^/]+/[^/]+)/pull/[0-9]+.*$#\1#' <<<"$1"); num=${1##*/}
  [ -n "$slug" ] && [ -n "$num" ] || return 1
  gh api -X POST "repos/$slug/issues/$num/labels" -f "labels[]=$2" >/dev/null 2>&1
}
export HOME="${HOME:-/home/hkjang}"
export PATH="$HOME/.local/bin:$HOME/.nvm/versions/node/v22.23.1/bin:/usr/local/bin:/usr/bin:/bin"
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="$REPO_DIR/state"; DATA="$REPO_DIR/docs/data"; SUMMARY="$DATA/summary.json"; RUNS="$DATA/runs.jsonl"
ROOT="${ROOT:-/mnt/c/Users/USER/projects}"
verb=${1:-help}; shift || true

sync_state(){ # 상태 변경을 aidev 원격에 올린다 (러너 sync 와 같은 잠금)
  ( cd "$REPO_DIR" && flock -w 120 9 && rm -rf .git/rebase-merge .git/rebase-apply 2>/dev/null
    git add -A state drafts ./*-STANDARD.md 2>/dev/null; git diff --cached --quiet || git -c user.name=hkjang -c user.email=gagagiga@naver.com commit -qm "ops: $*"
    git pull -q --rebase origin main && git push -q origin main ) 9>"$HOME/.auto-improve/sync.lock" >/dev/null 2>&1 \
    && echo "(aidev 에 반영됨)" || echo "(로컬에는 반영됨 — 원격 푸시는 다음 회차가 함)"
}
is_project(){ [ -d "$ROOT/$1/.git" ] && [ -f "$STATE/$1.md" ]; }
resolve_pr(){ # $1=PR url 또는 프로젝트 → 그 프로젝트의 열린(검토 대기·CI 실패) 러너 PR
  case "$1" in https://github.com/*/pull/*) echo "$1"; return 0;; esac
  jq -rs --arg p "$1" --arg s "$(date -d '-14 days' +%F)" '
    [ .[] | select(.project==$p and .date>=$s and ((.pr//"")|length)>0) ] | group_by(.pr) | map(.[-1])
    | map(select(.outcome=="review-pending" or .outcome=="verify-failed")) | sort_by(.ts) | last | .pr // empty' "$RUNS" 2>/dev/null
}
repo_of(){ sed -E 's#https://github.com/([^/]+/[^/]+)/pull/.*#\1#' <<<"$1"; }
last_run_dir(){ jq -r --arg p "$1" 'select(.project==$p) | .run_id' "$RUNS" 2>/dev/null | tail -1; }
need(){ [ -n "${1:-}" ] || { echo "사용법: bin/ops.sh $2"; exit 2; }; }

case "$verb" in
help)
  sed -n '/^#   읽기:/,/^#          shepherd/p' "$0" | sed 's/^#  *//'
  ;;
status)
  [ -f "$SUMMARY" ] || { echo "summary.json 이 아직 없음"; exit 1; }
  jq -r '"오늘 \(.today.date): 회차 \(.today.total) · 릴리즈 \(.today.released) · 머지 \(.today.merged) · 변경없음 \(.today.nochange) · 실패 \(.today.failed) · 비용 $\(.today.cost_usd) (마지막 갱신 \(.generated[0:16]))
누적 회차 \(.totals.runs) · 릴리즈 \(.totals.released) · 비용 $\(.totals.cost_usd)
경고 \(.alerts|length)건 · 작업함 \(.inbox|length)건 · 수정 과제 \(.fix_queue|length)건 · 중지 \(if (.stops|length)>0 then ([.stops[].scope]|join(",")) else "없음" end)
PR 처리기 \(if .shepherd.enabled then "켜짐" else "꺼짐" end): 마지막 \(.shepherd.last_pass[0:16] // "없음") · 오늘 승인 \(.shepherd.today.approve//0) · 수정 \(.shepherd.today["fix-pushed"]//0) · 사람 필요 \(.shepherd.pending_human)
건강: \(if .health.ok then "정상" else (.health.problems|join("; ")) end)"' "$SUMMARY"
  jq -r '.campaign_progress[] | select(.done|not) | "캠페인 \(.id): \(.processed)/\(.total) (\(.pct)%) — 완료 \(.counts.done//0) · PR대기 \(.counts.pr//0) · 막힘 \(.counts.stuck//0) · 미착수 \(.counts.todo//0) · $\(.spent_usd)/$\(.budget_usd) · 기한 \(.until) · 교훈 \(.lessons|length)"' "$SUMMARY"
  [ -n "$(jq -r '.alerts[]? | .project' "$SUMMARY")" ] && { echo "경고:"; jq -r '.alerts[] | "- \(.project): \(.why|.[0:120])"' "$SUMMARY" | head -8; }
  ;;
project)
  need "${1:-}" "project <이름>"; p=$1; is_project "$p" || { echo "러너 대상 프로젝트가 아님: $p (state/$p.md 없음)"; exit 1; }
  jq -r --arg p "$p" '.projects[] | select(.name==$p) | "\(.name) — 건강 \(.grade) (\(.grade_why))
마지막 회차 \(.last_ts[0:16]) [\(.last_status)]: \(.last_result)
최근 릴리즈 \(.release.tag // "없음") \(.release.status // "") · 회차 \(.runs) · 릴리즈 \(.released) · 누적 $\(.cost)
아이디어 대기 \(.ideas_pending)/\(.ideas_total) · 교훈 \(.lessons)"' "$SUMMARY"
  echo "자율화 단계: $(jq -r --arg p "$p" '.projects_autonomy[$p] // "release"' "$SUMMARY")$( [ -f "$STATE/STOP-$p" ] && echo " · ⛔ 중지됨: $(cat "$STATE/STOP-$p")")"
  echo "최근 회차:"; jq -r --arg p "$p" 'select(.project==$p) | "- \(.ts[0:16]) [\(.outcome)]\(if (.campaign//"")!="" then " (" + .campaign + ")" else "" end) \(.result|.[0:110])"' "$RUNS" | tail -5
  pr=$(resolve_pr "$p"); [ -n "$pr" ] && echo "열린 러너 PR: $pr" || echo "열린 러너 PR: 없음"
  jq -r --arg p "$p" '.inbox[] | select(.project==$p) | "작업함: \(.why|.[0:120]) · 처리기: \(.shepherd // "—")"' "$SUMMARY" | head -3
  grep -P "^$p\t" "$STATE/fix-queue.tsv" 2>/dev/null | cut -f2 | sed 's/^/수정 과제: /'
  jq -r --arg p "$p" '.[]? | select(.status=="pending") | "- [\(.value)/\(.risk)/\(.size)] \(.title)"' "$STATE/$p.ideas.json" 2>/dev/null | head -4 | sed '1i 대기 아이디어(상위):'
  ;;
why)
  need "${1:-}" "why <이름>"; p=$1; rid=$(last_run_dir "$p"); [ -n "$rid" ] || { echo "회차 기록 없음: $p"; exit 1; }
  d="$STATE/runs/$rid"
  jq -r --arg r "$rid" 'select(.run_id==$r) | "마지막 회차 \(.run_id) — [\(.outcome)] \(.result)"' "$RUNS" | tail -1
  [ -f "$d/stages.json" ] && { echo "단계:"; jq -r 'to_entries[] | "- \(.key): \(.value.state) — \(.value.reason|.[0:160])"' "$d/stages.json"; }
  [ -f "$d/verify.gate.json" ] && echo "검증: $(jq -r '.reason' "$d/verify.gate.json")"
  [ -f "$d/verify.txt" ] && { echo "검증 출력(끝부분):"; grep -v '^\s*$' "$d/verify.txt" | tail -12 | cut -c1-200; }
  [ -f "$d/review.json" ] && { echo "리뷰: $(jq -r '.verdict' "$d/review.json") (risk $(jq -r '.risk' "$d/review.json"))"; jq -r '.reasons[]? | "- " + (.|.[0:300])' "$d/review.json"; }
  [ -f "$d/journal.md" ] && { echo "회차 노트(역할들이 서로 남긴 것):"; grep -v '^- \[러너' "$d/journal.md" | tail -n 30 | cut -c1-220; }
  [ -f "$d/agent-improve.txt" ] && { echo "에이전트 마지막 말:"; tail -c 700 "$d/agent-improve.txt" | tr '\n' ' ' | cut -c1-700; echo; }
  pr=$(resolve_pr "$p"); [ -n "$pr" ] && { echo "PR 처리기 기록($pr):"; jq -r --arg pr "$pr" 'select(.pr==$pr) | "- \(.ts[0:16]) \(.cause) → \(.action): \(.detail|.[0:200])"' "$STATE/shepherd.jsonl" 2>/dev/null | tail -4; }
  jq -r --arg p "$p" 'select(.project==$p) | "교훈 \(.date) [\(.kind)] \(.detail|.[0:160])"' "$STATE/lessons.jsonl" 2>/dev/null | tail -3
  ;;
inbox)
  n=$(jq '.inbox|length' "$SUMMARY"); echo "작업함 $n건 (사람 필요가 위)"
  jq -r '.inbox | sort_by(.shepherd_action != "needs-human") | .[] | "- [\(.kind)] \(.project): \(.title|.[0:60]) \(.url)\n    왜: \(.why|.[0:140])\n    처리기: \(.shepherd // "아직 안 봄")"' "$SUMMARY" | head -60
  ;;
cost)
  days=${1:-7}
  jq -rs --arg s "$(date -d "-$days days" +%F)" '[ .[] | select(.date>=$s) ] | group_by(.date) | .[] | "\(.[0].date): $\(map(.cost_usd//0)|add|.*100|round/100) (\(length)세션)"' "$DATA/usage.jsonl"
  jq -rs --arg s "$(date -d "-$days days" +%F)" '[ .[] | select(.date>=$s) ] | "합계 $\(map(.cost_usd//0)|add|.*100|round/100) / \($s) 이후 · 프로젝트별 상위: " + ([group_by(.project)[] | {p: .[0].project, c: (map(.cost_usd//0)|add)}] | sort_by(-.c) | .[0:5] | map("\(.p) $\(.c|.*10|round/10)") | join(", "))' "$DATA/usage.jsonl"
  ;;
lessons)
  need "${1:-}" "lessons <이름|캠페인id>"; x=$1
  if [ -f "$STATE/campaign-lessons/$x.md" ]; then cat "$STATE/campaign-lessons/$x.md"
  else jq -r --arg p "$x" 'select(.project==$p) | "- \(.date) [\(.kind)] \(.detail|.[0:220])"' "$STATE/lessons.jsonl" 2>/dev/null | tail -10; fi
  ;;
stops) bash "$HERE/stop.sh" status ;;
approve|reject)
  need "${1:-}" "$verb <PR url|이름> [사유]"; pr=$(resolve_pr "$1"); [ -n "$pr" ] || { echo "열린 러너 PR 을 찾지 못함: $1"; exit 1; }
  repo=$(repo_of "$pr"); why=${2:-}
  if [ "$verb" = approve ]; then
    gh label create aidev-approved -R "$repo" --color 0E8A16 --description "러너가 CI 확인 후 승인 커밋에만 머지" >/dev/null 2>&1
    ops_pr_label "$pr" aidev-approved && echo "승인 라벨 달음: $pr — 다음 회차(10분 안) 승인 스윕이 CI 확인 뒤 머지·릴리즈" || { echo "라벨 실패: $pr"; exit 1; }
    gh api -X DELETE "repos/$repo/issues/${pr##*/}/labels/aidev-rejected" >/dev/null 2>&1 || true
  else
    gh label create aidev-rejected -R "$repo" --color B60205 --description "사람이 반려 — 러너가 닫는다" >/dev/null 2>&1
    ops_pr_label "$pr" aidev-rejected && echo "반려 라벨 달음: $pr — 다음 회차에 닫고 교훈으로 기록" || { echo "라벨 실패: $pr"; exit 1; }
    [ -n "$why" ] && gh pr comment "$pr" --body "반려 사유(코파일럿): $why" >/dev/null 2>&1
    gh api -X DELETE "repos/$repo/issues/${pr##*/}/labels/aidev-approved" >/dev/null 2>&1 || true
  fi
  ;;
run)
  need "${1:-}" "run <이름> [요청 명세]"; p=$1; spec=${2:-}; is_project "$p" || { echo "러너 대상 프로젝트가 아님: $p"; exit 1; }
  q="$STATE/run-queue.tsv"; touch "$q"
  grep -q -P "^$p\t" "$q" && { echo "$p 는 이미 수동 큐에 있음"; exit 0; }
  # 빈 칸을 연속 탭으로 두면 안 된다 — IFS 에 탭이 들어가면 read 가 연속된 탭을 하나로 합쳐
  # 뒤 열이 앞으로 밀린다. 그래서 코파일럿으로 넣은 요청의 명세가 회차에 전달되지 않았다
  # (2026-09-26 확인). 열은 프로젝트·메모·이슈번호·긴급도·명세 다섯이고 전부 값을 넣는다.
  printf '%s\t코파일럿 요청: %s\t0\tnormal\t%s\n' "$p" "$(cut -c1-80 <<<"${spec:-(명세 없음)}" | tr '\t' ' ')" "$(tr '\t\n' '  ' <<<"$spec")" >> "$q"
  echo "수동 큐에 넣음: $p${spec:+ — \"$(cut -c1-80 <<<"$spec")\"} (다음 회차, 10분 안에 우선 실행)"; sync_state "run $p"
  ;;
stop)   need "${1:-}" "stop <all|merge|release|이름> [사유]"; bash "$HERE/stop.sh" "$1" on "${2:-코파일럿에서 중지}" ;;
resume) need "${1:-}" "resume <all|merge|release|이름>"; bash "$HERE/stop.sh" "$1" off ;;
shepherd)
  case "${1:-}" in
    off) touch "$STATE/NO-SHEPHERD"; echo "PR 처리기 꺼짐 (state/NO-SHEPHERD)"; sync_state "shepherd off";;
    on)  rm -f "$STATE/NO-SHEPHERD"; echo "PR 처리기 켜짐"; sync_state "shepherd on";;
    *)   [ -f "$STATE/NO-SHEPHERD" ] && echo "PR 처리기: 꺼짐" || echo "PR 처리기: 켜짐"; jq -r '.shepherd | "마지막 \(.last_pass[0:16] // "없음") · 오늘 \(.today)"' "$SUMMARY";;
  esac ;;
campaign)
  cj="$STATE/campaigns.json"; sub=${1:-list}; id=${2:-}; arg=${3:-}
  case "$sub" in
    list) jq -r '.campaigns[] | select(.id|startswith("example-")|not) | "\(.id): \(if .done then "완료" elif .paused then "일시정지" else "진행" end) · 대상 \(.projects|length) · $\(.budget_usd) · 기한 \(.until)"' "$cj" ;;
    show) need "$id" "campaign show <id>"; jq -r --arg id "$id" '.campaigns[] | select(.id==$id) | "\(.id) — \(if .done then "완료" elif .paused then "일시정지" else "진행" end), $\(.budget_usd), 기한 \(.until)\n대상: \(.projects|join(", "))\n목표: \(.goal|.[0:500])"' "$cj"
          jq -r --arg id "$id" '.campaign_progress[] | select(.id==$id) | "진행: \(.processed)/\(.total) (\(.pct)%) · " + (.projects|to_entries|map("\(.key)=\(.value)")|join(" "))' "$SUMMARY" ;;
    add|remove) need "$arg" "campaign $sub <id> <이름>"; jq -e --arg id "$id" '.campaigns[]|select(.id==$id)' "$cj" >/dev/null || { echo "캠페인 없음: $id"; exit 1; }
          if [ "$sub" = add ]; then is_project "$arg" || { echo "러너 대상 프로젝트가 아님: $arg"; exit 1; }; jq --arg id "$id" --arg p "$arg" '(.campaigns[]|select(.id==$id)).projects |= (. + [$p] | unique)' "$cj" > "$cj.tmp"
          else jq --arg id "$id" --arg p "$arg" '(.campaigns[]|select(.id==$id)).projects |= map(select(. != $p))' "$cj" > "$cj.tmp"; fi
          mv "$cj.tmp" "$cj"; echo "$id 대상 $( [ "$sub" = add ] && echo 추가 || echo 제외): $arg (지금 $(jq -r --arg id "$id" '.campaigns[]|select(.id==$id)|.projects|length' "$cj")개)"; sync_state "campaign $sub $id $arg" ;;
    pause|resume) need "$id" "campaign $sub <id>"; jq --arg id "$id" --argjson v "$( [ "$sub" = pause ] && echo true || echo false )" '(.campaigns[]|select(.id==$id)).paused = $v' "$cj" > "$cj.tmp" && mv "$cj.tmp" "$cj"
          echo "$id $( [ "$sub" = pause ] && echo "일시 정지 — 새 회차를 배정하지 않음 (이미 열린 PR 은 처리기가 계속 봄)" || echo "재개")"; sync_state "campaign $sub $id" ;;
    budget) need "$arg" "campaign budget <id> <usd>"; jq --arg id "$id" --argjson b "$arg" '(.campaigns[]|select(.id==$id)).budget_usd = $b' "$cj" > "$cj.tmp" && mv "$cj.tmp" "$cj"; echo "$id 예산 → \$$arg"; sync_state "campaign budget $id $arg" ;;
    until)  need "$arg" "campaign until <id> <YYYY-MM-DD>"; date -d "$arg" >/dev/null 2>&1 || { echo "날짜 형식: YYYY-MM-DD"; exit 2; }; jq --arg id "$id" --arg u "$arg" '(.campaigns[]|select(.id==$id)).until = $u' "$cj" > "$cj.tmp" && mv "$cj.tmp" "$cj"; echo "$id 기한 → $arg"; sync_state "campaign until $id $arg" ;;
    *) echo "campaign list|show|add|remove|pause|resume|budget|until"; exit 2;;
  esac ;;
release)
  need "${1:-}" "release <이름>"; p=$1; is_project "$p" || { echo "러너 대상 프로젝트가 아님: $p"; exit 1; }
  ( setsid nohup bash "$HERE/daily.sh" --release-only "$p" >/dev/null 2>&1 & ) ; echo "릴리즈 회차를 시작함: $p (러너가 다른 회차로 바쁘면 이번엔 건너뜀 — 끝나면 텔레그램 알림)"
  ;;
draft-campaign)
  goal="$*"; need "$goal" "draft-campaign \"<목표 문장>\""
  slug="$(date +%Y%m)-$(printf '%s' "$goal" | md5sum | cut -c1-6)"
  ( setsid nohup bash "$HERE/campaign-draft.sh" "$goal" "$slug" >>"$REPO_DIR/logs/campaign-draft.log" 2>&1 & )
  echo "캠페인 초안 작업을 시작함 (slug $slug, 10~20분). 저장소들을 훑어 참조 구현·대상·표준 문서·예산을 잡고, 끝나면 텔레그램으로 요약을 보냄. 마음에 들면 'activate-campaign $slug'."
  ;;
drafts)
  for d in "$REPO_DIR"/drafts/*/; do [ -d "$d" ] || continue; s=$(basename "$d"); f="$d/campaign.json"
    if [ -f "$f" ]; then echo "- $s: $(jq -r '"\(.id) · 대상 \(.projects|length) · $\(.budget_usd) · 기한 \(.until)"' "$f") $( [ -f "$d/.activated" ] && echo "(적용됨)")"; else echo "- $s: (작성 중)"; fi; done
  [ -z "$(ls -d "$REPO_DIR"/drafts/*/ 2>/dev/null)" ] && echo "초안 없음"
  ;;
activate-campaign)
  need "${1:-}" "activate-campaign <slug>"; d="$REPO_DIR/drafts/$1"; f="$d/campaign.json"; [ -f "$f" ] || { echo "초안 없음: $1"; exit 1; }
  jq -e '.id and (.projects|length>0) and .until and .goal' "$f" >/dev/null || { echo "초안 campaign.json 이 불완전함"; exit 1; }
  id=$(jq -r .id "$f"); jq -e --arg id "$id" '.campaigns[]|select(.id==$id)' "$STATE/campaigns.json" >/dev/null && { echo "같은 id 의 캠페인이 이미 있음: $id"; exit 1; }
  std=$(jq -r '.standard_file // empty' "$f"); [ -n "$std" ] && [ -f "$d/STANDARD.md" ] && cp "$d/STANDARD.md" "$REPO_DIR/$std"
  jq --slurpfile c "$f" '.campaigns += [ $c[0] | {id, goal, projects, budget_usd, improve_budget_usd: (.improve_budget_usd // 14), until, expected_guard: (.expected_guard // []), done: false} ]' "$STATE/campaigns.json" > "$STATE/campaigns.json.tmp" && mv "$STATE/campaigns.json.tmp" "$STATE/campaigns.json"
  date -Iseconds > "$d/.activated"
  echo "캠페인 시작: $id — 대상 $(jq -r '.projects|length' "$f")개, \$$(jq -r .budget_usd "$f"), 기한 $(jq -r .until "$f")${std:+, 표준 $std}. 다음 회차부터 배정됨."; sync_state "campaign activate $id"
  ;;
discard-draft) need "${1:-}" "discard-draft <slug>"; rm -rf "$REPO_DIR/drafts/$1" && echo "초안 삭제: $1"; sync_state "discard draft $1" ;;
board)
  # 이사회(bin/board.sh) 제안을 보고 적용한다. 제안은 ops.sh 동사라 여기서 그대로 실행한다.
  bd="$STATE/board"; latest=$(ls -1 "$bd"/*.json 2>/dev/null | tail -1)
  case "${1:-}" in
    "") [ -n "$latest" ] || { echo "이사회 기록 없음 (bin/board.sh 로 연다)"; exit 0; }
        echo "이사회 $(jq -r .week "$latest") — 제약: $(jq -r .constraint "$latest")"
        jq -r '.proposals[] | "\(.n). [\(if .applied then "적용됨 " + .applied[0:10] else "대기" end)] \(.cmd|join(" ")) — \(.why|.[0:140])\n    포기: \(.gives_up|.[0:100]) · 되돌리기: \(.revert|.[0:100])"' "$latest"
        echo "하지 않는 것: $(jq -r '.not_doing|join(" · ")' "$latest")"; echo "역할 권고: $(jq -r '.role_advice|join(" · ")' "$latest")"; echo "메모: ${latest%.json}.md";;
    open) bash "$HERE/board.sh" --force ;;
    apply) shift; [ -n "$latest" ] || { echo "이사회 기록 없음"; exit 1; }
        sel="$*"; [ -n "$sel" ] || { echo "board apply <번호...|all>"; exit 2; }
        for n_ in $( [ "$sel" = all ] && jq -r '.proposals[]|.n' "$latest" || echo "$sel" ); do
          mapfile -t cmd < <(jq -r --argjson n "$n_" '.proposals[]|select(.n==$n)|.cmd[]' "$latest")
          [ ${#cmd[@]} -gt 0 ] || { echo "$n_: 없는 제안"; continue; }
          echo "▶ $n_: ${cmd[*]}"; bash "$0" "${cmd[@]}" 2>&1 | sed 's/^/   /'
          jq --argjson n "$n_" --arg ts "$(date -Iseconds)" '(.proposals[]|select(.n==$n)).applied=$ts' "$latest" > "$latest.tmp" && mv "$latest.tmp" "$latest"
        done; sync_state "board apply $sel" ;;
    *) echo "board [open|apply <번호...|all>]"; exit 2;;
  esac ;;
*) echo "모르는 동사: $verb"; bash "$0" help; exit 2;;
esac
