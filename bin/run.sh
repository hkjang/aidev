#!/usr/bin/env bash
# 자율 개선 에이전트 러너 — https://github.com/hkjang/aidev
#   사용법: bin/run.sh [--dry-run] [--count N] [--project NAME] [--days 30] [--budget 8] [--no-merge] [--no-sync]
#                     [--no-release] [--release-budget 10] [--release-only NAME] [--assets-only NAME]
#   - --assets-only NAME[:TAG]: 이미 나간 최신(또는 지정) 릴리즈에 이전 릴리즈와 같은 자산(도커 이미지 tar.gz 등)을 만들어 올린다
#   - 머지가 되면 릴리즈 에이전트를 한 번 더 돌린다: 그 저장소의 이전 릴리즈 방식(태그·버전 파일·CHANGELOG·워크플로·
#     GitHub Release)을 확인해 같은 방식으로 다음 버전을 만들고, 러너가 커밋·태그를 푸시하고 필요하면 GitHub Release 를 만든다
#   - 최근 N일 내 커밋이 있고, 작업트리가 깨끗하며, origin 원격이 있는 저장소를 후보로 삼는다
#   - 라운드로빈으로 하나(또는 N개)씩 골라 임시 worktree에서 claude -p 를 돌린다
#   - 커밋이 생기면 브랜치를 푸시하고 gh 로 PR을 연 뒤 바로 base 브랜치에 머지한다 (--no-merge 로 끔)
#   - 머지 후 로컬 체크아웃을 fast-forward 로 따라가게 한다
#   - 회차가 끝나면 원장(state/)과 러너 로그(logs/*.log)를 이 저장소에 커밋·푸시한다 (--no-sync 로 끔)
set -euo pipefail

ROOT="${ROOT:-/mnt/c/Users/USER/projects}"
# daily.sh 는 실행 중 파일이 바뀌어도 안전하도록 이 스크립트를 임시 경로에 복사해 돌린다 — 그때는 AIDEV_BIN 으로 위치를 받는다
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="$REPO_DIR/state"; LOGS="$REPO_DIR/logs"
WT_BASE="${WT_BASE:-$HOME/.cache/auto-improve-wt}"
DAYS=30; COUNT=1; BUDGET=8; DRY=0; ONLY=""; MERGE=1; SYNC=1; RELEASE=1; RBUDGET=10
MODEL="${MODEL:-claude-opus-5}"
# 모든 커밋(에이전트·러너)은 hkjang 명의로 — 저장소별 git 설정과 무관하게 강제한다
export GIT_AUTHOR_NAME=hkjang GIT_AUTHOR_EMAIL=gagagiga@naver.com
export GIT_COMMITTER_NAME=hkjang GIT_COMMITTER_EMAIL=gagagiga@naver.com
# 에이전트 세션이 커밋/PR 에 Claude 공동 작성자 트레일러를 붙이지 않게 한다
CLAUDE_SETTINGS='{"attribution":{"commit":"","pr":""}}'
# aidev 자신은 후보에서 뺀다 — 에이전트가 자기 러너를 고치게 두지 않는다
# Naviq 는 사용자 요청으로 제외 (2026-09-02)
EXCLUDE_RE='^(aidev|Naviq|sqlpad|_tmp.*|visitflow-node-modules.*|새 폴더)$'

while [ $# -gt 0 ]; do case "$1" in
  --dry-run) DRY=1;; --count) COUNT=$2; shift;; --project) ONLY=$2; shift;;
  --days) DAYS=$2; shift;; --budget) BUDGET=$2; shift;; --no-merge) MERGE=0;; --no-sync) SYNC=0;;
  --no-release) RELEASE=0;; --release-budget) RBUDGET=$2; shift;;
  --release-only) RELEASE_ONLY=$2; shift;;
  --assets-only) ASSETS_ONLY=$2; shift;;
  *) echo "unknown arg $1"; exit 2;; esac; shift; done

mkdir -p "$STATE" "$LOGS" "$WT_BASE"
RUN_DATE=$(date +%Y-%m-%d)
LOG="$LOGS/$RUN_DATE.log"
log(){ echo "[$(date +%H:%M:%S)] $*" | tee -a "$LOG"; }

# 회차 기록 한 줄을 docs/data/runs.jsonl 에 남기고 GitHub Pages 일일 보고를 다시 만든다
record_run(){ # $1=프로젝트 $2=결과
  mkdir -p "$REPO_DIR/docs/data"
  jq -cn --arg ts "$(date -Iseconds)" --arg d "$RUN_DATE" --arg p "$1" --arg r "$2" \
     '{ts:$ts,date:$d,project:$p,result:$r}' >> "$REPO_DIR/docs/data/runs.jsonl"
  python3 "$HERE/report.py" >>"$LOG" 2>&1 || log "report.py FAILED"
}

# 원장·로그·보고를 aidev 저장소에 남긴다. 실패해도 회차는 계속한다.
sync_repo(){
  [ "$SYNC" -eq 1 ] || return 0
  ( cd "$REPO_DIR" && git add -A state logs docs >/dev/null 2>&1 \
    && { git diff --cached --quiet || git commit -qm "$1"; } \
    && git push -q origin HEAD ) >>"$LOG" 2>&1 && log "aidev synced: $1" || log "aidev sync FAILED: $1"
}

# 머지된 뒤 그 저장소의 관례대로 릴리즈한다. 에이전트는 detached worktree 에서 커밋·태그만 만들고,
# 푸시와 GitHub Release 는 여기서 한다. 사용: release_project <프로젝트> <base> <변경 요약>
# 릴리즈 커밋의 CI 가 끝날 때까지 기다린 뒤 태그를 민다 — moina 처럼 릴리즈 워크플로가
# "정확히 그 커밋의 CI 성공"을 요구하는 저장소는 커밋과 태그를 같이 밀면 매번 실패했다. 실패해도 태그는 민다.
wait_for_checks(){
  local n=$1 sha=$2 i total pending failed
  for i in $(seq 1 40); do
    read -r total pending failed < <(cd "$repo" && gh api "repos/{owner}/{repo}/commits/$sha/check-runs" \
      --jq '[.check_runs|length, ([.check_runs[]|select(.status!="completed")]|length), ([.check_runs[]|select(.conclusion=="failure")]|length)] | @tsv' 2>/dev/null || echo "0 0 0")
    if [ "${total:-0}" -gt 0 ] && [ "${pending:-0}" -eq 0 ]; then
      CHECKS_FAILED=${failed:-0}
      [ "${failed:-0}" -eq 0 ] && log "$n: CI on ${sha:0:7} passed ($total checks)" || log "$n: CI on ${sha:0:7} has $failed failed check(s)"
      return 0
    fi
    [ "$i" -eq 1 ] && log "$n: waiting for CI on ${sha:0:7} (checks: ${total:-0}, pending: ${pending:-0})"
    [ "${total:-0}" -eq 0 ] && [ "$i" -ge 10 ] && { CHECKS_FAILED=0; log "$n: no CI checks on ${sha:0:7} after 5 min — continuing"; return 0; }
    sleep 30
  done
  CHECKS_FAILED=0; log "$n: CI on ${sha:0:7} still pending after 20 min — continuing"; return 0
}

# GitHub Release 를 보장하고 자산을 올린다. 사용: publish_release <프로젝트> <태그> <제목> <노트파일> <ghrel(true/false)> <release.json>
publish_release(){
  local n=$1 tag=$2 title=$3 notes=$4 ghrel=$5 rfile=$6 i
  [ -n "$tag" ] || return 0
  local -a assets=()
  while IFS= read -r a; do [ -n "$a" ] && [ -f "$a" ] && assets+=("$a"); done < <(jq -r '.assets[]? // empty' "$rfile" 2>/dev/null)
  # 릴리즈가 있어야 자산을 올릴 수 있다: 직접 만들거나(ghrel=true), 워크플로가 만들 때까지 기다린다
  if ! (cd "$repo" && gh release view "$tag" >/dev/null 2>&1); then
    if [ "$ghrel" = true ]; then
      local -a nargs=(--generate-notes)
      [ -n "$notes" ] && [ -f "$notes" ] && nargs=(--notes-file "$notes")
      (cd "$repo" && gh release create "$tag" --title "${title:-$tag}" "${nargs[@]}" >>"$LOG" 2>&1) \
        && log "$n: GitHub Release $tag created" || log "$n: GitHub Release create FAILED"
    elif [ ${#assets[@]} -gt 0 ]; then
      for i in $(seq 1 30); do (cd "$repo" && gh release view "$tag" >/dev/null 2>&1) && break; sleep 30; done
      (cd "$repo" && gh release view "$tag" >/dev/null 2>&1) || {
        log "$n: 워크플로가 15분 안에 Release 를 만들지 않음 — 직접 만든다"
        (cd "$repo" && gh release create "$tag" --title "${title:-$tag}" --generate-notes >>"$LOG" 2>&1) || true; }
    fi
  fi
  if [ ${#assets[@]} -gt 0 ]; then
    if (cd "$repo" && gh release upload "$tag" "${assets[@]}" --clobber >>"$LOG" 2>&1); then
      log "$n: uploaded ${#assets[@]} asset(s) to $tag: $(printf '%s ' "${assets[@]##*/}")"
      result="$result +${#assets[@]} assets"
    else
      log "$n: asset upload FAILED for $tag"; result="$result, asset upload failed"
    fi
    rm -f "${assets[@]}" 2>/dev/null || true
  fi
  # 이전 릴리즈엔 자산이 있었는데 이번엔 없으면 경고 — 워크플로가 만드는 경우는 잠시 기다려 본다
  local prev prev_n new_n
  prev=$(cd "$repo" && gh release list --limit 10 --json tagName --jq "[.[].tagName] | map(select(. != \"$tag\")) | .[0] // empty" 2>/dev/null)
  if [ -n "$prev" ]; then
    prev_n=$(cd "$repo" && gh release view "$prev" --json assets --jq '.assets | length' 2>/dev/null || echo 0)
    if [ "${prev_n:-0}" -gt 0 ]; then
      for i in $(seq 1 30); do
        new_n=$(cd "$repo" && gh release view "$tag" --json assets --jq '.assets | length' 2>/dev/null || echo 0)
        [ "${new_n:-0}" -gt 0 ] && break; sleep 30
      done
      if [ "${new_n:-0}" -gt 0 ]; then log "$n: $tag has $new_n asset(s) (prev $prev: $prev_n)"
      else
        # 워크플로가 실패해서 없는 것인지, 아직 도는 중인지 구분해 남긴다
        local wf; wf=$(cd "$repo" && gh run list --limit 40 --json headBranch,conclusion,status,name \
          --jq "[.[] | select(.headBranch==\"$tag\")] | .[0] | \"\\(.name): \\(.status)/\\(.conclusion)\"" 2>/dev/null)
        log "$n: ASSETS MISSING on $tag (prev $prev had $prev_n) — workflow: ${wf:-none}"
        case "$wf" in *failure*) result="$result, ASSETS MISSING (release workflow FAILED)";; *) result="$result, ASSETS MISSING";; esac
      fi
    fi
  fi
  # 대시보드가 쓰도록 자산 수를 release.json 에 남긴다
  local cur_n; cur_n=$(cd "$repo" && gh release view "$tag" --json assets --jq '.assets|length' 2>/dev/null || echo 0)
  jq --argjson n "${cur_n:-0}" --arg prev "${prev:-}" --argjson pn "${prev_n:-0}" \
     '.assets_count=$n | .prev_tag=$prev | .prev_assets_count=$pn' "$rfile" > "$rfile.tmp" 2>/dev/null && mv "$rfile.tmp" "$rfile" || rm -f "$rfile.tmp"
}

# 머지된 뒤 그 저장소의 관례대로 릴리즈한다. 에이전트는 detached worktree 에서 커밋·태그·자산만 만들고,
# 푸시·GitHub Release·자산 업로드는 여기서 한다. 사용: release_project <프로젝트> <base> <변경 요약> [assets]
#   mode=assets: 이미 나간 최신 태그를 체크아웃해 자산만 만들어 올린다 (버전·커밋·태그 변경 없음)
release_project(){
  local n=$1 base=$2 summary=$3 mode=${4:-release}
  local rwt="$WT_BASE/$n-release" rfile="$STATE/$n.release.json" envfile="$STATE/$n.env"
  rm -f "$rfile"
  # --force: 로컬에 같은 이름의 다른 태그가 있으면 fetch 가 실패한다(Clustara v0.9.5). 원격이 기준이다.
  git -C "$repo" fetch -q --force --tags origin "$base" >>"$LOG" 2>&1 || log "$n: tag fetch had errors (continuing)"
  local ref="origin/$base" mode_note="" latest=""
  if [ "$mode" = assets ]; then
    latest=${ASSETS_TAG:-$(cd "$repo" && gh release list --limit 1 --json tagName --jq '.[0].tagName' 2>/dev/null)}
    [ -n "$latest" ] || { log "$n: no GitHub Release to attach assets to"; return 0; }
    ref="refs/tags/$latest"
    mode_note="## 이번 세션은 자산만 만든다
이미 태그 \`$latest\` 와 GitHub Release 가 나가 있지만 이전 릴리즈에 있던 자산이 빠졌다. **버전을 올리거나 커밋·태그를 만들지 말고**, 체크아웃된 \`$latest\` 로 이전 릴리즈와 같은 자산을 같은 방법·같은 이름 규칙으로 만들어 \`assets\` 에 적기만 하라. JSON 의 \`status\` 는 \`released\`, \`tag\` 는 \`$latest\`, \`github_release\` 는 \`false\`."
  fi
  git -C "$repo" worktree remove --force "$rwt" 2>/dev/null || true
  git -C "$repo" worktree add --detach "$rwt" "$ref" >>"$LOG" 2>&1
  local rprompt
  rprompt=$(RELEASE_FILE="$rfile" CHANGE_SUMMARY="$summary" MODE_NOTE="$mode_note" \
            envsubst '$RELEASE_FILE $CHANGE_SUMMARY $MODE_NOTE' < "$REPO_DIR/release-prompt.md")
  ( cd "$rwt" && { [ -f "$envfile" ] && set -a && . "$envfile" && set +a; } ; claude -p "$rprompt" \
      --model "$MODEL" \
      --settings "$CLAUDE_SETTINGS" \
      --permission-mode acceptEdits \
      --allowedTools "Bash,Read,Edit,Write,Glob,Grep" \
      --add-dir "$STATE" \
      --max-budget-usd "$RBUDGET" \
      --output-format text ) > "$LOGS/$RUN_DATE-$n-release.txt" 2>&1 || log "$n: release agent exited non-zero"

  local status ahead tag title notes ghrel tagged
  status=$(jq -r '.status // "missing"' "$rfile" 2>/dev/null || echo missing)
  tag=$(jq -r '.tag // ""' "$rfile" 2>/dev/null || true)
  title=$(jq -r '.title // ""' "$rfile" 2>/dev/null || true)
  notes=$(jq -r '.notes_file // ""' "$rfile" 2>/dev/null || true)
  ghrel=$(jq -r '.github_release // false' "$rfile" 2>/dev/null || echo false)
  if [ "$mode" = assets ]; then
    if [ "$status" = released ]; then
      result="$result, assets for $latest"; publish_release "$n" "$latest" "$title" "$notes" false "$rfile"
    else
      log "$n: assets $status ($(jq -r '.reason // ""' "$rfile" 2>/dev/null))"; result="$result, assets $status"
    fi
    git -C "$repo" worktree remove --force "$rwt" >>"$LOG" 2>&1 || true
    return 0
  fi
  ahead=$(git -C "$rwt" rev-list --count "origin/$base..HEAD")
  tagged=0; [ -n "$tag" ] && git -C "$rwt" rev-parse -q --verify "refs/tags/$tag" >/dev/null 2>&1 && tagged=1
  # 태그만 찍는 관례(버전 커밋 없음)도 릴리즈다 — 커밋이 없어도 새 태그가 있으면 내보낸다
  if [ "$status" = released ] && { [ "$ahead" -gt 0 ] || [ "$tagged" -eq 1 ]; }; then
    if { [ "$ahead" -eq 0 ] || git -C "$rwt" push origin "HEAD:$base" >>"$LOG" 2>&1; } \
       && { [ "$ahead" -eq 0 ] || [ "$tagged" -eq 0 ] || { wait_for_checks "$n" "$(git -C "$rwt" rev-parse HEAD)" && [ "${CHECKS_FAILED:-0}" -eq 0 ]; }; } \
       && { [ "$tagged" -eq 0 ] || git -C "$rwt" push origin "refs/tags/$tag" >>"$LOG" 2>&1; }; then
      log "$n: released ${tag:-(no tag)}"; result="$result, released ${tag:-$(jq -r .version "$rfile")}"
      git -C "$repo" pull --ff-only origin "$base" >>"$LOG" 2>&1 || true
      printf -- '- 릴리즈: %s (%s)\n' "${tag:-$(jq -r .version "$rfile")}" "$RUN_DATE" >> "$ledger"
      publish_release "$n" "$tag" "$title" "$notes" "$ghrel" "$rfile"
    else
      log "$n: release push FAILED — 로컬 커밋·태그는 버려짐"; result="$result, release push failed"
    fi
  else
    log "$n: release $status ($(jq -r '.reason // ""' "$rfile" 2>/dev/null))"; result="$result, release $status"
  fi
  git -C "$repo" worktree remove --force "$rwt" >>"$LOG" 2>&1 || true
}

# ---- 자산만 (이미 나간 최신 릴리즈에 이전 릴리즈와 같은 자산을 만들어 올림) -------------
if [ -n "${ASSETS_ONLY:-}" ]; then
  # NAME 또는 NAME:TAG — 태그를 주면 최신 릴리즈 대신 그 태그에 자산을 붙인다
  n=${ASSETS_ONLY%%:*}; ASSETS_TAG=""; [[ "$ASSETS_ONLY" == *:* ]] && ASSETS_TAG=${ASSETS_ONLY#*:}
  repo="$ROOT/$n"; ledger="$STATE/$n.md"; result="assets-only${ASSETS_TAG:+ $ASSETS_TAG}"
  base=$(git -C "$repo" symbolic-ref --short HEAD)
  log "=== $n assets-only (base=$base)"
  release_project "$n" "$base" "(자산 보충)" assets
  record_run "$n" "$result"
  sync_repo "run($RUN_DATE): $n — $result"
  log "done"; exit 0
fi

# ---- 릴리즈만 (이미 머지된 프로젝트를 관례대로 릴리즈) ---------------------------
if [ -n "${RELEASE_ONLY:-}" ]; then
  n=$RELEASE_ONLY; repo="$ROOT/$n"; ledger="$STATE/$n.md"; result="release-only"
  base=$(git -C "$repo" symbolic-ref --short HEAD)
  log "=== $n release-only (base=$base)"
  release_project "$n" "$base" "$(tail -n 8 "$ledger" 2>/dev/null)"
  record_run "$n" "$result"
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
      --settings "$CLAUDE_SETTINGS" \
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
    # PR 의 CI 가 끝나고 실패가 없을 때만 머지한다 — moina v0.1.18 은 이미지 빌드가 깨진 채 머지·릴리즈됐다
    ci_ok=1
    if [ "$MERGE" -eq 1 ] && [ -n "$url" ]; then
      wait_for_checks "$n" "$(git -C "$wt" rev-parse HEAD)"
      if [ "${CHECKS_FAILED:-0}" -gt 0 ]; then
        ci_ok=0; log "$n: CI FAILED on PR — not merging, PR left open"; result="CI failed, PR open $url"
      fi
    fi
    if [ "$MERGE" -eq 1 ] && [ -n "$url" ] && [ "$ci_ok" -eq 1 ]; then
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
  record_run "$n" "$result"
  sync_repo "run($RUN_DATE): $n — $result"
done
log "done"
