#!/usr/bin/env bash
# 자율 개선 에이전트 러너 — https://github.com/hkjang/aidev
#   사용법: bin/run.sh [--dry-run] [--count N] [--project NAME] [--days 30] [--budget 8] [--no-merge] [--no-sync]
#                     [--no-release] [--release-budget 6]
#   - 머지가 되면 릴리즈 에이전트를 한 번 더 돌린다: 그 저장소의 이전 릴리즈 방식(태그·버전 파일·CHANGELOG·워크플로·
#     GitHub Release)을 확인해 같은 방식으로 다음 버전을 만들고, 러너가 커밋·태그를 푸시하고 필요하면 GitHub Release 를 만든다
#   - 최근 N일 내 커밋이 있고, 작업트리가 깨끗하며, origin 원격이 있는 저장소를 후보로 삼는다
#   - 라운드로빈으로 하나(또는 N개)씩 골라 임시 worktree에서 claude -p 를 돌린다
#   - 커밋이 생기면 브랜치를 푸시하고 gh 로 PR을 연 뒤 바로 base 브랜치에 머지한다 (--no-merge 로 끔)
#   - 머지 후 로컬 체크아웃을 fast-forward 로 따라가게 한다
#   - 회차가 끝나면 원장(state/)과 러너 로그(logs/*.log)를 이 저장소에 커밋·푸시한다 (--no-sync 로 끔)
set -euo pipefail

ROOT="${ROOT:-/mnt/c/Users/USER/projects}"
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="$REPO_DIR/state"; LOGS="$REPO_DIR/logs"
WT_BASE="${WT_BASE:-$HOME/.cache/auto-improve-wt}"
DAYS=30; COUNT=1; BUDGET=8; DRY=0; ONLY=""; MERGE=1; SYNC=1; RELEASE=1; RBUDGET=6
MODEL="${MODEL:-claude-opus-5}"
# aidev 자신은 후보에서 뺀다 — 에이전트가 자기 러너를 고치게 두지 않는다
# Naviq 는 사용자 요청으로 제외 (2026-09-02)
EXCLUDE_RE='^(aidev|Naviq|sqlpad|_tmp.*|visitflow-node-modules.*|새 폴더)$'

while [ $# -gt 0 ]; do case "$1" in
  --dry-run) DRY=1;; --count) COUNT=$2; shift;; --project) ONLY=$2; shift;;
  --days) DAYS=$2; shift;; --budget) BUDGET=$2; shift;; --no-merge) MERGE=0;; --no-sync) SYNC=0;;
  --no-release) RELEASE=0;; --release-budget) RBUDGET=$2; shift;;
  --release-only) RELEASE_ONLY=$2; shift;;
  *) echo "unknown arg $1"; exit 2;; esac; shift; done

mkdir -p "$STATE" "$LOGS" "$WT_BASE"
RUN_DATE=$(date +%Y-%m-%d)
LOG="$LOGS/$RUN_DATE.log"
log(){ echo "[$(date +%H:%M:%S)] $*" | tee -a "$LOG"; }

# 원장·로그를 aidev 저장소에 남긴다. 실패해도 회차는 계속한다.
sync_repo(){
  [ "$SYNC" -eq 1 ] || return 0
  ( cd "$REPO_DIR" && git add -A state logs >/dev/null 2>&1 \
    && { git diff --cached --quiet || git commit -qm "$1"; } \
    && git push -q origin HEAD ) >>"$LOG" 2>&1 && log "aidev synced: $1" || log "aidev sync FAILED: $1"
}

# 머지된 뒤 그 저장소의 관례대로 릴리즈한다. 에이전트는 detached worktree 에서 커밋·태그만 만들고,
# 푸시와 GitHub Release 는 여기서 한다. 사용: release_project <프로젝트> <base> <변경 요약>
release_project(){
  local n=$1 base=$2 summary=$3
  local rwt="$WT_BASE/$n-release" rfile="$STATE/$n.release.json" envfile="$STATE/$n.env"
  rm -f "$rfile"
  # 로컬엔 옛 태그만 있는 저장소가 많다 — 원격 태그를 먼저 받아야 "이전 릴리즈"가 맞는다
  # --force: 로컬에 같은 이름의 다른 태그가 있으면 fetch 가 실패한다(Clustara v0.9.5). 원격이 기준이다.
  git -C "$repo" fetch -q --force --tags origin "$base" >>"$LOG" 2>&1 || log "$n: tag fetch had errors (continuing)"
  git -C "$repo" worktree remove --force "$rwt" 2>/dev/null || true
  git -C "$repo" worktree add --detach "$rwt" "origin/$base" >>"$LOG" 2>&1
  local rprompt
  rprompt=$(RELEASE_FILE="$rfile" CHANGE_SUMMARY="$summary" \
            envsubst '$RELEASE_FILE $CHANGE_SUMMARY' < "$REPO_DIR/release-prompt.md")
  ( cd "$rwt" && { [ -f "$envfile" ] && set -a && . "$envfile" && set +a; } ; claude -p "$rprompt" \
      --model "$MODEL" \
      --permission-mode acceptEdits \
      --allowedTools "Bash,Read,Edit,Write,Glob,Grep" \
      --add-dir "$STATE" \
      --max-budget-usd "$RBUDGET" \
      --output-format text ) > "$LOGS/$RUN_DATE-$n-release.txt" 2>&1 || log "$n: release agent exited non-zero"

  local status ahead tag title notes ghrel tagged
  status=$(jq -r '.status // "missing"' "$rfile" 2>/dev/null || echo missing)
  ahead=$(git -C "$rwt" rev-list --count "origin/$base..HEAD")
  tag=$(jq -r '.tag // ""' "$rfile" 2>/dev/null || true)
  tagged=0; [ -n "$tag" ] && git -C "$rwt" rev-parse -q --verify "refs/tags/$tag" >/dev/null 2>&1 && tagged=1
  # 태그만 찍는 관례(버전 커밋 없음)도 릴리즈다 — 커밋이 없어도 새 태그가 있으면 내보낸다
  if [ "$status" = released ] && { [ "$ahead" -gt 0 ] || [ "$tagged" -eq 1 ]; }; then
    title=$(jq -r '.title // ""' "$rfile")
    notes=$(jq -r '.notes_file // ""' "$rfile"); ghrel=$(jq -r '.github_release // false' "$rfile")
    if { [ "$ahead" -eq 0 ] || git -C "$rwt" push origin "HEAD:$base" >>"$LOG" 2>&1; } \
       && { [ "$tagged" -eq 0 ] || git -C "$rwt" push origin "refs/tags/$tag" >>"$LOG" 2>&1; }; then
      log "$n: released ${tag:-(no tag)}"; result="$result, released ${tag:-$(jq -r .version "$rfile")}"
      if [ "$ghrel" = true ] && [ -n "$tag" ]; then
        local -a nargs=(--generate-notes)
        [ -n "$notes" ] && [ -f "$notes" ] && nargs=(--notes-file "$notes")
        (cd "$rwt" && gh release create "$tag" --title "${title:-$tag}" "${nargs[@]}" >>"$LOG" 2>&1) \
          && log "$n: GitHub Release $tag created" \
          || log "$n: GitHub Release create FAILED (CI 가 이미 만들었을 수 있음)"
      fi
      git -C "$repo" pull --ff-only origin "$base" >>"$LOG" 2>&1 || true
      printf -- '- 릴리즈: %s (%s)\n' "${tag:-$(jq -r .version "$rfile")}" "$RUN_DATE" >> "$ledger"
    else
      log "$n: release push FAILED — 로컬 커밋·태그는 버려짐"; result="$result, release push failed"
    fi
  else
    log "$n: release $status ($(jq -r '.reason // ""' "$rfile" 2>/dev/null))"; result="$result, release $status"
  fi
  git -C "$repo" worktree remove --force "$rwt" >>"$LOG" 2>&1 || true
}

# ---- 릴리즈만 (이미 머지된 프로젝트를 관례대로 릴리즈) ---------------------------
if [ -n "${RELEASE_ONLY:-}" ]; then
  n=$RELEASE_ONLY; repo="$ROOT/$n"; ledger="$STATE/$n.md"; result="release-only"
  base=$(git -C "$repo" symbolic-ref --short HEAD)
  log "=== $n release-only (base=$base)"
  release_project "$n" "$base" "$(tail -n 8 "$ledger" 2>/dev/null)"
  sync_repo "run($RUN_DATE): $n — $result"
  log "done"; exit 0
fi

# ---- 후보 선정 ---------------------------------------------------------------
candidates=()
since=$(date -d "-$DAYS days" +%s)
for d in "$ROOT"/*/; do
  n=$(basename "$d"); [ -d "$d/.git" ] || continue
  [[ "$n" =~ $EXCLUDE_RE ]] && continue
  [ -n "$ONLY" ] && [ "$n" != "$ONLY" ] && continue
  last=$(git -C "$d" log -1 --format=%ct 2>/dev/null || echo 0)
  [ "$last" -ge "$since" ] || continue
  git -C "$d" remote get-url origin >/dev/null 2>&1 || continue
  [ -z "$(git -C "$d" status --porcelain 2>/dev/null)" ] || { log "skip $n: dirty working tree"; continue; }
  candidates+=("$n")
done
[ ${#candidates[@]} -gt 0 ] || { log "no candidates"; exit 0; }
log "candidates(${#candidates[@]}): ${candidates[*]}"

# ---- 라운드로빈 --------------------------------------------------------------
CURSOR="$STATE/.cursor"; idx=$(cat "$CURSOR" 2>/dev/null || echo 0)
picked=()
for ((i=0;i<COUNT && i<${#candidates[@]};i++)); do
  picked+=("${candidates[$(( (idx+i) % ${#candidates[@]} ))]}"); done
echo $(( (idx+COUNT) % ${#candidates[@]} )) > "$CURSOR"
log "picked: ${picked[*]}"
[ $DRY -eq 1 ] && exit 0

# ---- 프로젝트별 실행 ----------------------------------------------------------
for n in "${picked[@]}"; do
  repo="$ROOT/$n"; ledger="$STATE/$n.md"; slug="auto/$RUN_DATE-$(date +%H%M)"
  wt="$WT_BASE/$n"
  base=$(git -C "$repo" symbolic-ref --short HEAD)
  log "=== $n (base=$base, branch=$slug)"
  git -C "$repo" worktree remove --force "$wt" 2>/dev/null || true
  git -C "$repo" worktree add -b "$slug" "$wt" "$base" >>"$LOG" 2>&1

  prompt=$(LEDGER_FILE="$ledger" RUN_DATE="$RUN_DATE" \
           LEDGER_CONTENT="$(cat "$ledger" 2>/dev/null || echo '(없음)')" \
           envsubst '$LEDGER_FILE $RUN_DATE $LEDGER_CONTENT' < "$REPO_DIR/prompt.md")

  # 프로젝트별 환경(예: WEEKLY_TEST_POSTGRES_DSN)은 state/<프로젝트>.env 에 두면 에이전트에 전달된다 (.env 는 git 제외)
  envfile="$STATE/$n.env"
  ( cd "$wt" && { [ -f "$envfile" ] && set -a && . "$envfile" && set +a; } ; claude -p "$prompt" \
      --model "$MODEL" \
      --permission-mode acceptEdits \
      --allowedTools "Bash,Read,Edit,Write,Glob,Grep,WebFetch,WebSearch" \
      --add-dir "$STATE" \
      --max-budget-usd "$BUDGET" \
      --output-format text ) > "$LOGS/$RUN_DATE-$n.txt" 2>&1 || log "$n: claude exited non-zero"

  ahead=$(git -C "$wt" rev-list --count "$base..$slug")
  # 에이전트가 예산 소진 등으로 원장을 못 남겼으면 커밋 제목으로 대신 기록한다
  if [ "$ahead" -gt 0 ] && ! grep -q "^## $RUN_DATE" "$ledger" 2>/dev/null; then
    { printf '## %s\n- 선택: %s\n- 결과: 성공(원장 미기록, 러너가 대체 기록)\n- 요약: 커밋 %s개\n' \
        "$RUN_DATE" "$(git -C "$wt" log -1 --format=%s)" "$ahead"; } >> "$ledger"
  fi
  result="no change"
  if [ "$ahead" -gt 0 ]; then
    git -C "$wt" push -u origin "$slug" >>"$LOG" 2>&1
    body=$(printf '자율 개선 에이전트가 생성한 PR입니다.\n\n%s\n\n🤖 auto-improve %s · https://github.com/hkjang/aidev' \
           "$(tail -n 12 "$ledger" 2>/dev/null)" "$RUN_DATE")
    url=$(cd "$wt" && gh pr create --base "$base" --head "$slug" \
           --title "auto-improve: $(git -C "$wt" log -1 --format=%s)" --body "$body" 2>>"$LOG" || true)
    log "$n: $ahead commit(s) → PR $url"
    result="PR $url"
    if [ "$MERGE" -eq 1 ] && [ -n "$url" ]; then
      # gh 는 머지 뒤 로컬 브랜치를 지우고 base 로 옮기려 하므로, worktree 를 먼저 걷어내고 main 체크아웃에서 돌린다
      git -C "$repo" worktree remove --force "$wt" >>"$LOG" 2>&1 || true
      if (cd "$repo" && gh pr merge "$url" --merge --delete-branch >>"$LOG" 2>&1); then
        log "$n: merged into $base"; result="merged $url"
        git -C "$repo" pull --ff-only origin "$base" >>"$LOG" 2>&1 && log "$n: local $base fast-forwarded" \
          || log "$n: local $base not updated (pull --ff-only failed)"
        [ "$RELEASE" -eq 1 ] && release_project "$n" "$base" "$(tail -n 8 "$ledger" 2>/dev/null)"
      else
        log "$n: merge FAILED — PR left open for review"; result="merge failed $url"
      fi
    fi
  else
    log "$n: no commits; deleting branch"
    git -C "$repo" branch -D "$slug" >>"$LOG" 2>&1 || true
  fi
  git -C "$repo" worktree remove --force "$wt" >>"$LOG" 2>&1 || true
  sync_repo "run($RUN_DATE): $n — $result"
done
log "done"
