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
DAYS=30; COUNT=1; BUDGET=8; DRY=0; ONLY=""; MERGE=1; SYNC=1; RELEASE=1; RBUDGET=10; REVIEW=1; RVBUDGET=4
MODEL="${MODEL:-claude-opus-5}"
# 모든 커밋(에이전트·러너)은 hkjang 명의로 — 저장소별 git 설정과 무관하게 강제한다
export GIT_AUTHOR_NAME=hkjang GIT_AUTHOR_EMAIL=gagagiga@naver.com
export GIT_COMMITTER_NAME=hkjang GIT_COMMITTER_EMAIL=gagagiga@naver.com
# 에이전트 세션이 커밋/PR 에 Claude 공동 작성자 트레일러를 붙이지 않게 한다
CLAUDE_SETTINGS='{"attribution":{"commit":"","pr":""}}'
# aidev 자신은 후보에서 뺀다 — 에이전트가 자기 러너를 고치게 두지 않는다
# Naviq 는 사용자 요청으로 제외 (2026-09-02)
EXCLUDE_RE='^(aidev|Naviq|sqlpad|_tmp.*|visitflow-node-modules.*|새 폴더)$'
# 일일 상한·휴면 규칙 (state/caps.env)
MAX_DAILY_COST=80; MAX_DAILY_ROUNDS=60; MAX_DAILY_RELEASES=30; DORMANT_AFTER=3; DORMANT_DAYS=7
[ -f "$REPO_DIR/state/caps.env" ] && . "$REPO_DIR/state/caps.env"

while [ $# -gt 0 ]; do case "$1" in
  --dry-run) DRY=1;; --count) COUNT=$2; shift;; --project) ONLY=$2; shift;;
  --days) DAYS=$2; shift;; --budget) BUDGET=$2; shift;; --no-merge) MERGE=0;; --no-sync) SYNC=0;;
  --no-release) RELEASE=0;; --release-budget) RBUDGET=$2; shift;;
  --release-only) RELEASE_ONLY=$2; shift;;
  --assets-only) ASSETS_ONLY=$2; shift;; --no-review) REVIEW=0;;
  *) echo "unknown arg $1"; exit 2;; esac; shift; done

mkdir -p "$STATE" "$LOGS" "$WT_BASE"
RUN_DATE=$(date +%Y-%m-%d)
LOG="$LOGS/$RUN_DATE.log"
log(){ echo "[$(date +%H:%M:%S)] $*" | tee -a "$LOG"; }

# claude -p 의 JSON 결과에서 비용·토큰·시간을 docs/data/usage.jsonl 에 남기고, 사람이 읽을 답변은 .txt 에 붙인다
record_usage(){ # $1=프로젝트 $2=단계(improve/release/assets) $3=json $4=txt
  local n=$1 phase=$2 j=$3 t=$4
  mkdir -p "$REPO_DIR/docs/data"
  if jq -e '.type=="result"' "$j" >/dev/null 2>&1; then
    jq -r '.result // ""' "$j" >> "$t"
    jq -c --arg ts "$(date -Iseconds)" --arg d "$RUN_DATE" --arg p "$n" --arg ph "$phase" \
      '{ts:$ts,date:$d,project:$p,phase:$ph,subtype:(.subtype//""),duration_ms:(.duration_ms//0),num_turns:(.num_turns//0),
        cost_usd:(.total_cost_usd//0),input_tokens:(.usage.input_tokens//0),output_tokens:(.usage.output_tokens//0),
        cache_read:(.usage.cache_read_input_tokens//0),cache_create:(.usage.cache_creation_input_tokens//0)}' "$j" \
      >> "$REPO_DIR/docs/data/usage.jsonl"
    log "$n: $phase — $(jq -r '"\(.num_turns//0) turns, \((.duration_ms//0)/60000|floor)m, $\(.total_cost_usd//0|.*100|round/100), \(.subtype//"")"' "$j")"
  else
    cat "$j" >> "$t" 2>/dev/null  # JSON 이 아니면(예: 예산 초과 메시지) 그대로 남긴다
  fi
}

# 회차 기록 한 줄을 docs/data/runs.jsonl 에 남기고 GitHub Pages 일일 보고를 다시 만든다
record_run(){ # $1=프로젝트 $2=결과  (RUN_META 가 있으면 변경 요약을 함께 남긴다)
  mkdir -p "$REPO_DIR/docs/data"
  jq -cn --arg ts "$(date -Iseconds)" --arg d "$RUN_DATE" --arg p "$1" --arg r "$2" --argjson m "${RUN_META:-{\}}" \
     '{ts:$ts,date:$d,project:$p,result:$r} + $m' >> "$REPO_DIR/docs/data/runs.jsonl"
  RUN_META="{}"
  "$HERE/digest.sh" >>"$LOG" 2>&1 || true
  "$HERE/regress.sh" >>"$LOG" 2>&1 || true
  python3 "$HERE/report.py" >>"$LOG" 2>&1 || log "report.py FAILED"
  "$HERE/notify.sh" >>"$LOG" 2>&1 || true
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
  local n=$1 sha=$2 i total pending failed last_total=""
  for i in $(seq 1 40); do
    read -r total pending failed < <(cd "$repo" && gh api "repos/{owner}/{repo}/commits/$sha/check-runs" \
      --jq '[.check_runs|length, ([.check_runs[]|select(.status!="completed")]|length), ([.check_runs[]|select(.conclusion=="failure")]|length)] | @tsv' 2>/dev/null || echo "0 0 0")
    # check-run 은 잡이 시작될 때 하나씩 생긴다 — 빠른 검사만 먼저 끝난 순간을 "완료"로 오판하지 않도록
    # 두 번 연속(30초 간격) 같은 개수로 모두 완료여야 통과로 본다 (moina v0.1.21 이 그렇게 새어 나갔다)
    if [ "${total:-0}" -gt 0 ] && [ "${pending:-0}" -eq 0 ] && [ "${total}" = "${last_total:-}" ]; then
      CHECKS_FAILED=${failed:-0}
      [ "${failed:-0}" -eq 0 ] && log "$n: CI on ${sha:0:7} passed ($total checks)" || log "$n: CI on ${sha:0:7} has $failed failed check(s)"
      return 0
    fi
    last_total=$total
    [ "$i" -eq 1 ] && log "$n: waiting for CI on ${sha:0:7} (checks: ${total:-0}, pending: ${pending:-0})"
    [ "${total:-0}" -eq 0 ] && [ "$i" -ge 10 ] && { CHECKS_FAILED=0; log "$n: no CI checks on ${sha:0:7} after 5 min — continuing"; return 0; }
    sleep 30
  done
  CHECKS_FAILED=0; log "$n: CI on ${sha:0:7} still pending after 20 min — continuing"; return 0
}

# 릴리즈 워크플로 실패 복구: 한 번 재실행하고, 그래도 실패하면 실패 단계·로그 요지를 수정 큐에 넣는다
retry_release_workflow(){
  local n=$1 tag=$2 rid i st conc step excerpt
  rid=$(cd "$repo" && gh run list --limit 40 --json databaseId,headBranch,conclusion \
        --jq "[.[] | select(.headBranch==\"$tag\" and .conclusion==\"failure\")] | .[0].databaseId" 2>/dev/null)
  [ -n "$rid" ] && [ "$rid" != null ] || return 0
  (cd "$repo" && gh run rerun "$rid" --failed >>"$LOG" 2>&1) || { log "$n: rerun of $rid not possible"; return 0; }
  log "$n: release workflow $rid re-run for $tag — waiting"
  for i in $(seq 1 40); do
    sleep 30
    read -r st conc < <(cd "$repo" && gh run view "$rid" --json status,conclusion --jq '"\(.status) \(.conclusion//"")"' 2>/dev/null || echo "unknown")
    [ "$st" = completed ] && break
  done
  if [ "${conc:-}" = success ]; then
    log "$n: release workflow recovered on rerun ($tag)"; result="${result/ (release workflow FAILED)/ (recovered on rerun)}"
    return 0
  fi
  step=$(cd "$repo" && gh run view "$rid" --json jobs --jq '[.jobs[] | .steps[] | select(.conclusion=="failure") | .name] | join(", ")' 2>/dev/null)
  excerpt=$(cd "$repo" && gh run view "$rid" --log-failed 2>/dev/null | sed 's/^[^\t]*\t[^\t]*\t//' | grep -i -E "error|fail|expected|mismatch" | grep -v -i "deprecat" | head -6 | cut -c1-200 | tr '\n' ' ' | sed 's/\t/ /g')
  local note="릴리즈 워크플로 실패 2회 — 태그 $tag, 실패 단계: ${step:-?}. 실행: $(cd "$repo" && gh run view "$rid" --json url --jq .url 2>/dev/null). 로그 요지: ${excerpt:-없음}"
  mkdir -p "$STATE"; touch "$STATE/fix-queue.tsv"
  local msha; msha=$(cd "$repo" && gh pr list --state merged --limit 1 --json mergeCommit --jq '.[0].mergeCommit.oid // ""' 2>/dev/null)
  grep -q -P "^$n\t" "$STATE/fix-queue.tsv" || printf '%s\t%s\t%s\n' "$n" "$note" "$msha" >> "$STATE/fix-queue.tsv"
  jq -cn --arg ts "$(date -Iseconds)" --arg d "$RUN_DATE" --arg p "$n" --arg pr "$tag" --arg detail "릴리즈 워크플로가 2회 실패($step). 릴리즈 관련 검증은 머지 전에 로컬에서 재현할 것." \
     '{ts:$ts,date:$d,project:$p,kind:"release-workflow-failed",pr:$pr,detail:$detail}' >> "$STATE/lessons.jsonl"
  log "$n: queued for fix round ($step)"; result="$result, queued for fix"
}

# 리뷰 게이트 — 구현과 다른 세션이 diff 만 읽고 머지 반대 사유를 찾는다. reject 면 1 을 돌려주고 PR 에 사유를 단다.
review_pr(){ # $1=프로젝트 $2=base $3=PR url
  local n=$1 base=$2 url=$3 rvfile="$STATE/$n.review.json" rprompt verdict
  rm -f "$rvfile"
  rprompt=$(BASE="$base" REVIEW_FILE="$rvfile" envsubst '$BASE $REVIEW_FILE' < "$REPO_DIR/review-prompt.md")
  ( cd "$wt" && claude -p "$rprompt" --model "$MODEL" --settings "$CLAUDE_SETTINGS" --permission-mode acceptEdits \
      --allowedTools "Bash,Read,Glob,Grep,Write" --add-dir "$STATE" --max-budget-usd "$RVBUDGET" \
      --output-format json ) > "$LOGS/$RUN_DATE-$n-review.json" 2>"$LOGS/$RUN_DATE-$n-review.txt" || log "$n: review agent exited non-zero"
  record_usage "$n" review "$LOGS/$RUN_DATE-$n-review.json" "$LOGS/$RUN_DATE-$n-review.txt"
  verdict=$(jq -r '.verdict // "approve"' "$rvfile" 2>/dev/null || echo approve)
  if [ "$verdict" = reject ]; then
    local reasons; reasons=$(jq -r '.reasons[]? | "- " + .' "$rvfile" 2>/dev/null)
    log "$n: REVIEW REJECTED — $(jq -r '.reasons|join(" / ")' "$rvfile" 2>/dev/null | cut -c1-200)"
    (cd "$repo" && gh pr comment "$url" --body "🧐 리뷰 에이전트가 머지를 보류했습니다 — 사람이 판단해 주세요.

$reasons

(위험도: $(jq -r '.risk // "?"' "$rvfile"), 자율 개선 러너 자동 코멘트)" >>"$LOG" 2>&1) || true
    result="review rejected, PR open $url"
    return 1
  fi
  log "$n: review approved (risk $(jq -r '.risk // "?"' "$rvfile" 2>/dev/null))"
  return 0
}

# 보호 파일 — default.guard + <프로젝트>.guard 의 정규식에 걸리는 파일을 건드렸으면 자동 머지하지 않는다
guarded_files(){ # $1=프로젝트 $2=base ; 걸린 파일 목록을 출력
  local pat; pat=$(cat "$STATE/default.guard" "$STATE/$1.guard" 2>/dev/null | grep -v '^#' | grep -v '^[[:space:]]*$')
  [ -n "$pat" ] || return 0
  git -C "$wt" diff --name-only "$2..HEAD" 2>/dev/null | grep -E -f <(printf '%s\n' "$pat") || true
}

# 자동 롤백 — 수정 회차까지 실패하면 원래 머지 커밋을 되돌리는 PR 을 연다 (머지는 사람)
rollback_project(){ # $1=프로젝트 $2=머지 커밋
  local n=$1 sha=$2 rwt="$WT_BASE/$n-revert" br="revert/$RUN_DATE-$(date +%H%M)" url
  [ -n "$sha" ] || { log "$n: rollback skipped (no merge sha)"; return 0; }
  git -C "$repo" fetch -q origin "$base" >>"$LOG" 2>&1 || true
  git -C "$repo" worktree remove --force "$rwt" 2>/dev/null || true
  git -C "$repo" worktree add -b "$br" "$rwt" "origin/$base" >>"$LOG" 2>&1 || return 0
  if (cd "$rwt" && { git revert --no-edit -m 1 "$sha" || git revert --no-edit "$sha"; } >>"$LOG" 2>&1); then
    git -C "$rwt" push -u origin "$br" >>"$LOG" 2>&1
    url=$(cd "$rwt" && gh pr create --base "$base" --head "$br" --title "revert: 자율 개선 변경 되돌리기 (${sha:0:7})" \
          --body "릴리즈 워크플로가 반복 실패했고 수정 회차도 실패해 ${sha:0:7} 을 되돌리는 PR 입니다. 사람이 검토 후 머지해 주세요.

🤖 aidev 자동 롤백 · https://hkjang.github.io/aidev/projects/$n/" 2>>"$LOG" || true)
    log "$n: rollback PR $url (사람 머지)"; result="$result, rollback PR $url"
    jq -cn --arg ts "$(date -Iseconds)" --arg d "$RUN_DATE" --arg p "$n" --arg pr "$url" --arg detail "머지 ${sha:0:7} 이 릴리즈를 반복해서 깨뜨려 되돌림 PR 을 열었다. 같은 접근은 피할 것." \
       '{ts:$ts,date:$d,project:$p,kind:"rolled-back",pr:$pr,detail:$detail}' >> "$STATE/lessons.jsonl"
  else
    log "$n: rollback revert FAILED (conflict) — 사람 개입 필요"; result="$result, rollback failed"
  fi
  git -C "$repo" worktree remove --force "$rwt" >>"$LOG" 2>&1 || true
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
        case "$wf" in
          *failure*) result="$result, ASSETS MISSING (release workflow FAILED)"; retry_release_workflow "$n" "$tag";;
          *) result="$result, ASSETS MISSING";;
        esac
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
      --output-format json ) > "$LOGS/$RUN_DATE-$n-release.json" 2>"$LOGS/$RUN_DATE-$n-release.txt" || log "$n: release agent exited non-zero"
  record_usage "$n" "$mode" "$LOGS/$RUN_DATE-$n-release.json" "$LOGS/$RUN_DATE-$n-release.txt"

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

# ---- 일일 상한 ---------------------------------------------------------------
if [ -z "$ONLY" ] && [ $DRY -eq 0 ]; then
  today_cost=$(jq -s --arg d "$RUN_DATE" '[.[]|select(.date==$d)|.cost_usd//0]|add // 0' "$REPO_DIR/docs/data/usage.jsonl" 2>/dev/null || echo 0)
  today_rounds=$(jq -s --arg d "$RUN_DATE" '[.[]|select(.date==$d)]|length' "$REPO_DIR/docs/data/runs.jsonl" 2>/dev/null || echo 0)
  today_rel=$(jq -s --arg d "$RUN_DATE" '[.[]|select(.date==$d)|select(.result|test("released v?[0-9]"))]|length' "$REPO_DIR/docs/data/runs.jsonl" 2>/dev/null || echo 0)
  cap=""
  awk -v c="$today_cost" -v m="$MAX_DAILY_COST" 'BEGIN{exit !(c>=m)}' && cap="비용 \$$today_cost ≥ \$$MAX_DAILY_COST"
  [ "${today_rounds:-0}" -ge "$MAX_DAILY_ROUNDS" ] && cap="회차 $today_rounds ≥ $MAX_DAILY_ROUNDS"
  [ "${today_rel:-0}" -ge "$MAX_DAILY_RELEASES" ] && cap="릴리즈 $today_rel ≥ $MAX_DAILY_RELEASES"
  if [ -n "$cap" ]; then
    log "daily cap reached: $cap — 오늘은 더 돌지 않는다"
    if [ ! -f "$STATE/.cap-$RUN_DATE" ]; then
      touch "$STATE/.cap-$RUN_DATE"
      ps1=$(wslpath -w "$HERE/toast.ps1" 2>/dev/null); [ -n "$ps1" ] && powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$ps1" -Title "aidev 일일 상한 도달" -Message "$cap" >/dev/null 2>&1
      jq -cn --arg ts "$(date -Iseconds)" --arg d "$RUN_DATE" --arg r "daily cap reached: $cap" '{ts:$ts,date:$d,project:"(runner)",result:$r}' >> "$REPO_DIR/docs/data/runs.jsonl"
      python3 "$HERE/report.py" >>"$LOG" 2>&1 || true; sync_repo "run($RUN_DATE): daily cap — $cap"
    fi
    exit 0
  fi
fi
# 수동 트리거(이슈 라벨 run) 수집
[ -z "$ONLY" ] && "$HERE/inbox.sh" >>"$LOG" 2>&1 || true

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
  # 휴면: 최근 N회가 모두 "변경 없음"이고 마지막이 DORMANT_DAYS 안이면 건너뛴다 (수동/수정 큐는 예외)
  if [ -z "$ONLY" ] && [ -s "$REPO_DIR/docs/data/runs.jsonl" ]; then
    read -r streak lastd < <(jq -rs --arg p "$n" --argjson k "$DORMANT_AFTER" '[.[]|select(.project==$p)] | (.[-$k:] ) as $l | [( ($l|length)==$k and all($l[]; .result|test("no change")) ), ($l[-1].date // "")] | @tsv' "$REPO_DIR/docs/data/runs.jsonl" 2>/dev/null || echo "false ")
    if [ "$streak" = true ] && [ -n "$lastd" ] && [ $(( ($(date +%s) - $(date -d "$lastd" +%s)) / 86400 )) -lt "$DORMANT_DAYS" ] \
       && ! grep -q -P "^$n\t" "$STATE/fix-queue.tsv" "$STATE/run-queue.tsv" 2>/dev/null; then
      log "skip $n: dormant (변경 없음 ${DORMANT_AFTER}연속, $lastd)"; continue
    fi
  fi
  candidates+=("$n")
done
[ ${#candidates[@]} -gt 0 ] || { log "no candidates"; exit 0; }
log "candidates(${#candidates[@]}): ${candidates[*]}"

# ---- 라운드로빈 --------------------------------------------------------------
CURSOR="$STATE/.cursor"; idx=$(cat "$CURSOR" 2>/dev/null || echo 0)
picked=()
for ((i=0;i<COUNT && i<${#candidates[@]};i++)); do
  picked+=("${candidates[$(( (idx+i) % ${#candidates[@]} ))]}"); done
FIX_PROJECT=""; FIX_NOTE_TEXT=""
FIXQ="$STATE/fix-queue.tsv"
if [ -z "$ONLY" ] && [ -s "$FIXQ" ]; then
  # 릴리즈 워크플로가 두 번 실패한 프로젝트는 라운드로빈보다 먼저, 원인 수정 과제로 배정한다
  while IFS=$'\t' read -r fp fnote fsha; do
    [ -n "$fp" ] || continue
    if printf '%s\n' "${candidates[@]}" | grep -qx "$fp"; then
      picked=("$fp"); FIX_PROJECT="$fp"; FIX_NOTE_TEXT="$fnote"; FIX_SHA="${fsha:-}"; log "fix-queue: picked $fp"; break
    fi
  done < "$FIXQ"
fi
RUN_ISSUE=""; RUNQ="$STATE/run-queue.tsv"
if [ -z "$FIX_PROJECT" ] && [ -z "$ONLY" ] && [ -s "$RUNQ" ]; then
  while IFS=$'\t' read -r rp rnote rnum; do
    [ -n "$rp" ] || continue
    if printf '%s\n' "${candidates[@]}" | grep -qx "$rp"; then
      picked=("$rp"); RUN_PROJECT="$rp"; RUN_ISSUE="$rnum"; log "run-queue: picked $rp ($rnote)"; break
    else
      log "run-queue: $rp 은 후보가 아님(더러운 트리/원격 없음/30일 무활동) — 이슈에 안내"; \
      (cd "$REPO_DIR" && gh issue comment "$rnum" --body "\`$rp\` 은 지금 후보가 아닙니다(미커밋 변경이 있거나, 원격이 없거나, 30일간 커밋이 없음). 정리 후 다시 라벨을 달아 주세요." >/dev/null 2>&1; gh issue edit "$rnum" --remove-label run >/dev/null 2>&1) || true
      grep -v -P "^$rp\t" "$RUNQ" > "$RUNQ.tmp"; mv "$RUNQ.tmp" "$RUNQ"
    fi
  done < "$RUNQ"
fi
[ -n "$FIX_PROJECT" ] || [ -n "${RUN_PROJECT:-}" ] || echo $(( (idx+COUNT) % ${#candidates[@]} )) > "$CURSOR"
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

  fix_note=""
  if [ "$n" = "${FIX_PROJECT:-}" ]; then
    fix_note="## 우선 과제 (자동 배정) — 이번 회차는 새 아이디어 대신 아래 실패를 고치세요
$(printf '%b' "$FIX_NOTE_TEXT")
릴리즈 워크플로가 같은 이유로 두 번 실패했습니다. 워크플로 파일(.github/workflows)과 실패한 단계의 스크립트·테스트를 읽고 원인을 고치세요. 워크플로 자체를 느슨하게 만들어 통과시키는 것은 금지입니다(검증을 지우거나 continue-on-error 를 넣지 마세요). 고친 뒤 같은 검증을 로컬에서 재현해 통과를 확인하고, 원장에 '수정 과제' 로 기록하세요."
  fi
  lessons=$(jq -r --arg p "$n" 'select(.project==$p) | "- \(.date) [\(.kind)] \(.detail)"' "$STATE/lessons.jsonl" 2>/dev/null | tail -n 8)
  ideas_file="$STATE/$n.ideas.json"
  ideas=$(jq -r '.[]? | select(.status=="pending") | "- [\(.value)/\(.risk)/\(.size)] \(.title) — \(.note // "")"' "$ideas_file" 2>/dev/null | head -n 12)
  prompt=$(LEDGER_FILE="$ledger" RUN_DATE="$RUN_DATE" FIX_NOTE="$fix_note" LESSONS="${lessons:-(없음)}" \
           IDEAS_FILE="$ideas_file" IDEAS_CONTENT="${ideas:-(없음)}" \
           LEDGER_CONTENT="$(cat "$ledger" 2>/dev/null || echo '(없음)')" \
           envsubst '$LEDGER_FILE $RUN_DATE $LEDGER_CONTENT $FIX_NOTE $LESSONS $IDEAS_FILE $IDEAS_CONTENT' < "$REPO_DIR/prompt.md")

  # 프로젝트별 환경(예: WEEKLY_TEST_POSTGRES_DSN)은 state/<프로젝트>.env 에 두면 에이전트에 전달된다 (.env 는 git 제외)
  envfile="$STATE/$n.env"
  ( cd "$wt" && { [ -f "$envfile" ] && set -a && . "$envfile" && set +a; } ; claude -p "$prompt" \
      --model "$MODEL" \
      --settings "$CLAUDE_SETTINGS" \
      --permission-mode acceptEdits \
      --allowedTools "Bash,Read,Edit,Write,Glob,Grep,WebFetch,WebSearch" \
      --add-dir "$STATE" \
      --max-budget-usd "$BUDGET" \
      --output-format json ) > "$LOGS/$RUN_DATE-$n.json" 2>"$LOGS/$RUN_DATE-$n.txt" || log "$n: claude exited non-zero"
  record_usage "$n" improve "$LOGS/$RUN_DATE-$n.json" "$LOGS/$RUN_DATE-$n.txt"

  ahead=$(git -C "$wt" rev-list --count "$base..$slug")
  # 에이전트가 예산 소진 등으로 원장을 못 남겼으면 커밋 제목으로 대신 기록한다
  if [ "$ahead" -gt 0 ] && ! grep -q "^## $RUN_DATE" "$ledger" 2>/dev/null; then
    { printf '## %s\n- 선택: %s\n- 결과: 성공(원장 미기록, 러너가 대체 기록)\n- 요약: 커밋 %s개\n' \
        "$RUN_DATE" "$(git -C "$wt" log -1 --format=%s)" "$ahead"; } >> "$ledger"
  fi
  result="no change"; RUN_META="{}"
  if [ "$ahead" -gt 0 ]; then
    # 대시보드 미리보기용: 파일 수·증감·테스트 파일 수·커밋 제목
    RUN_META=$(git -C "$wt" diff --numstat "$base..HEAD" | awk 'BEGIN{f=0;a=0;d=0;t=0} {f++; a+=$1; d+=$2; if ($3 ~ /(^|\/)(test|tests|spec|__tests__)\/|_test\.|\.test\.|\.spec\.|Test\.java|test_.*\.py/) t++} END{printf "{\"files\":%d,\"additions\":%d,\"deletions\":%d,\"tests\":%d}", f,a,d,t}')
    RUN_META=$(jq -c --arg t "$(git -C "$wt" log -1 --format=%s)" '. + {title:$t}' <<<"$RUN_META" 2>/dev/null || echo "{}")
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
      guarded=$(guarded_files "$n" "$base")
      if [ -n "$guarded" ]; then
        ci_ok=0; log "$n: GUARDED files touched — not merging: $(tr '\n' ' ' <<<"$guarded")"; result="guarded files, PR open $url"
        (cd "$repo" && gh pr comment "$url" --body "🔒 보호 파일을 건드려 자동 머지하지 않습니다. 사람이 검토해 주세요.

$(sed 's/^/- /' <<<"$guarded")

(state/default.guard 규칙, 자율 개선 러너 자동 코멘트)" >>"$LOG" 2>&1) || true
      elif [ "$REVIEW" -eq 1 ] && ! review_pr "$n" "$base" "$url"; then
        ci_ok=0
      else
        wait_for_checks "$n" "$(git -C "$wt" rev-parse HEAD)"
        if [ "${CHECKS_FAILED:-0}" -gt 0 ]; then
          ci_ok=0; log "$n: CI FAILED on PR — not merging, PR left open"; result="CI failed, PR open $url"
        fi
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
  if [ "$n" = "${RUN_PROJECT:-}" ]; then
    grep -v -P "^$n\t" "$RUNQ" > "$RUNQ.tmp" 2>/dev/null; mv "$RUNQ.tmp" "$RUNQ"; result="manual: $result"
    [ -n "$RUN_ISSUE" ] && (cd "$REPO_DIR" && gh issue comment "$RUN_ISSUE" --body "✅ 실행 완료 — $result

https://hkjang.github.io/aidev/projects/$n/" >/dev/null 2>&1; gh issue close "$RUN_ISSUE" >/dev/null 2>&1) || true
  fi
  if [ "$n" = "${FIX_PROJECT:-}" ]; then
    grep -v -P "^$n\t" "$FIXQ" > "$FIXQ.tmp" 2>/dev/null; mv "$FIXQ.tmp" "$FIXQ"; result="fix-round: $result"
    case "$result" in *merged*) ;; *) log "$n: fix round did not merge — rolling back"; rollback_project "$n" "${FIX_SHA:-}";; esac
  fi
  record_run "$n" "$result"
  sync_repo "run($RUN_DATE): $n — $result"
done
log "done"
