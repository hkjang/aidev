#!/usr/bin/env bash
# aidev 자율 개선 러너 v2 — 에이전트는 변경과 결과를 "제안"하고, 러너가 검증하며, 게시(푸시·머지·릴리즈)는 러너만 한다.
#   사용법: bin/run.sh [--dry-run] [--count N] [--project NAME] [--days 30] [--budget USD] [--release-budget USD]
#                     [--no-merge] [--no-release] [--no-review] [--no-sync] [--release-only NAME] [--assets-only NAME[:TAG]]
#   흐름: 상한 → 수동/수정 큐 → 후보(휴면 제외) → 기준 커밋 고정 → 격리된 에이전트 → 러너 검증 → 비밀정보 검사 → PR
#         → 보호 파일 → 독립 리뷰 → 기준 브랜치 이동 시 재검증 → CI(gate) → 커밋 일치 머지 → 릴리즈(gate) → 자산 검증 → 회귀 감시·보고·알림
#   원칙: 확인하지 못한 것은 성공이 아니다 (판정은 bin/gate.py, 회귀 테스트 tests/test_gate.py).
set -uo pipefail  # -e 는 쓰지 않는다: 명령 치환 속 파이프 실패가 회차를 조용히 죽였다(ideas.json 없음 등). 실패는 각 단계에서 명시적으로 다룬다

ROOT="${ROOT:-/mnt/c/Users/USER/projects}"
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="$REPO_DIR/state"; LOGS="$REPO_DIR/logs"; RUNS="$STATE/runs"
WT_BASE="${WT_BASE:-$HOME/.cache/auto-improve-wt}"
DAYS=30; COUNT=1; BUDGET=""; RBUDGET=""; DRY=0; ONLY=""; MERGE=1; SYNC=1; RELEASE=1; REVIEW=1
MODEL="${MODEL:-claude-opus-5}"
REAL_HOME="$HOME"; CLAUDE_CFG="${CLAUDE_CONFIG_DIR:-$REAL_HOME/.claude}"
export GIT_AUTHOR_NAME=hkjang GIT_AUTHOR_EMAIL=gagagiga@naver.com GIT_COMMITTER_NAME=hkjang GIT_COMMITTER_EMAIL=gagagiga@naver.com
CLAUDE_SETTINGS='{"attribution":{"commit":"","pr":""}}'
EXCLUDE_RE='^(aidev|Naviq|sqlpad|_tmp.*|visitflow-node-modules.*|새 폴더)$'
MAX_DAILY_COST=80; MAX_DAILY_ROUNDS=60; MAX_DAILY_RELEASES=30; DORMANT_AFTER=3; DORMANT_DAYS=7
[ -f "$STATE/caps.env" ] && . "$STATE/caps.env"

while [ $# -gt 0 ]; do case "$1" in
  --dry-run) DRY=1;; --count) COUNT=$2; shift;; --project) ONLY=$2; shift;; --days) DAYS=$2; shift;;
  --budget) BUDGET=$2; shift;; --release-budget) RBUDGET=$2; shift;;
  --no-merge) MERGE=0;; --no-sync) SYNC=0;; --no-release) RELEASE=0;; --no-review) REVIEW=0;;
  --release-only) RELEASE_ONLY=$2; shift;; --assets-only) ASSETS_ONLY=$2; shift;;
  *) echo "unknown arg $1"; exit 2;; esac; shift; done

mkdir -p "$STATE" "$LOGS" "$WT_BASE" "$RUNS" "$REPO_DIR/docs/data"
RUN_DATE=$(date +%Y-%m-%d); LOG="$LOGS/$RUN_DATE.log"
log(){ echo "[$(date +%H:%M:%S)] $*" | tee -a "$LOG"; }
GATE="python3 $HERE/gate.py"

# ---------------------------------------------------------------- 정책 · 실행 단위
# 정책: default.policy.json 위에 <프로젝트>.policy.json 을 덮는다. 에이전트는 이 파일들을 보지 못한다.
policy(){ # $1=프로젝트 $2=jq 경로 (예: .base_branch)
  jq -r "$2 // empty" <(jq -s '.[0] * (.[1] // {})' "$STATE/default.policy.json" <([ -f "$STATE/$1.policy.json" ] && cat "$STATE/$1.policy.json" || echo '{}')) 2>/dev/null
}
new_run(){ # $1=프로젝트 $2=종류 → RUN_ID, OUT 설정
  RUN_ID="$RUN_DATE-$(date +%H%M%S)-$1-$2"; OUT="$RUNS/$RUN_ID"; mkdir -p "$OUT/assets" "$OUT/home"
  printf '{"run_id":"%s","project":"%s","kind":"%s","started":"%s"}\n' "$RUN_ID" "$1" "$2" "$(date -Iseconds)" > "$OUT/run.json"
}
stage(){ # $1=단계 $2=상태 $3=사유 — $OUT/stages.json 에 누적
  local f="$OUT/stages.json"; [ -f "$f" ] || echo '{}' > "$f"
  jq --arg k "$1" --arg s "$2" --arg r "$3" --arg t "$(date -Iseconds)" '.[$k]={state:$s,reason:$r,at:$t}' "$f" > "$f.tmp" && mv "$f.tmp" "$f"
  log "$n: [$1] $2 — $3"
}

# ---------------------------------------------------------------- 격리된 에이전트 실행
# 에이전트 세션: 임시 HOME(→ gh 미인증, git 자격증명 없음, 홈의 비밀 파일 없음), 토큰 환경변수 제거, 결과는 $OUT 에만.
# Claude 자체 설정은 CLAUDE_CONFIG_DIR 로 넘긴다. 빌드 캐시는 실제 경로(비밀 아님)로 연결해 속도를 유지한다.
run_agent(){ # $1=단계 $2=프롬프트 $3=작업 디렉터리 $4=예산 $5=허용 도구
  local phase=$1 prompt=$2 wd=$3 budget=$4 tools=$5 envfile="$STATE/$n.env"
  # 임시 홈에는 Claude 의 계정 설정(~/.claude.json)만 복사한다 — 자격증명은 CLAUDE_CONFIG_DIR/.credentials.json 에서 읽는다
  mkdir -p "$OUT/home"; [ -f "$REAL_HOME/.claude.json" ] && cp "$REAL_HOME/.claude.json" "$OUT/home/.claude.json"
  ( cd "$wd" && env -i \
      HOME="$OUT/home" USER="$USER" LANG=C.UTF-8 TERM=dumb PATH="$PATH" \
      CLAUDE_CONFIG_DIR="$CLAUDE_CFG" \
      GIT_AUTHOR_NAME="$GIT_AUTHOR_NAME" GIT_AUTHOR_EMAIL="$GIT_AUTHOR_EMAIL" GIT_COMMITTER_NAME="$GIT_COMMITTER_NAME" GIT_COMMITTER_EMAIL="$GIT_COMMITTER_EMAIL" \
      GOPATH="$REAL_HOME/go" GOMODCACHE="$REAL_HOME/go/pkg/mod" GOCACHE="$REAL_HOME/.cache/go-build" \
      npm_config_cache="$REAL_HOME/.npm" NVM_DIR="$REAL_HOME/.nvm" PIP_CACHE_DIR="$REAL_HOME/.cache/pip" \
      DOCKER_HOST="${DOCKER_HOST:-}" AIDEV_OUT="$OUT" \
      bash -c '[ -f "$0" ] && { set -a; . "$0"; set +a; }; exec "$@"' "$envfile" \
      claude -p "$prompt" --model "$MODEL" --settings "$CLAUDE_SETTINGS" --permission-mode acceptEdits \
        --allowedTools "$tools" --add-dir "$OUT" --max-budget-usd "$budget" --output-format json \
  ) > "$OUT/agent-$phase.json" 2>"$OUT/agent-$phase.txt" || log "$n: $phase agent exited non-zero"
  record_usage "$n" "$phase" "$OUT/agent-$phase.json" "$OUT/agent-$phase.txt"
}
record_usage(){ # $1=프로젝트 $2=단계 $3=json $4=txt
  local n=$1 phase=$2 j=$3 t=$4
  if jq -e '.type=="result"' "$j" >/dev/null 2>&1; then
    jq -r '.result // ""' "$j" >> "$t"
    jq -c --arg ts "$(date -Iseconds)" --arg d "$RUN_DATE" --arg p "$n" --arg ph "$phase" --arg rid "${RUN_ID:-}" \
      '{ts:$ts,date:$d,project:$p,phase:$ph,run_id:$rid,subtype:(.subtype//""),duration_ms:(.duration_ms//0),num_turns:(.num_turns//0),
        cost_usd:(.total_cost_usd//null),input_tokens:(.usage.input_tokens//0),output_tokens:(.usage.output_tokens//0),
        cache_read:(.usage.cache_read_input_tokens//0),cache_create:(.usage.cache_creation_input_tokens//0)}' "$j" >> "$REPO_DIR/docs/data/usage.jsonl"
    log "$n: $phase — $(jq -r '"\(.num_turns//0) turns, \((.duration_ms//0)/60000|floor)m, $\(.total_cost_usd//0|.*100|round/100), \(.subtype//"")"' "$j")"
  else
    cat "$j" >> "$t" 2>/dev/null || true
    jq -cn --arg ts "$(date -Iseconds)" --arg d "$RUN_DATE" --arg p "$n" --arg ph "$phase" --arg rid "${RUN_ID:-}" \
      '{ts:$ts,date:$d,project:$p,phase:$ph,run_id:$rid,subtype:"unknown",cost_usd:null}' >> "$REPO_DIR/docs/data/usage.jsonl"
    log "$n: $phase — 결과 JSON 없음 (비용 미확인)"
  fi
}
# 잔여 예산: 오늘 쓴 비용 + 이 단계 예산이 상한을 넘으면 시작하지 않는다
budget_ok(){ # $1=이번 단계 예산
  local spent; spent=$(jq -s --arg d "$RUN_DATE" '[.[]|select(.date==$d)|.cost_usd//0]|add // 0' "$REPO_DIR/docs/data/usage.jsonl" 2>/dev/null || echo 0)
  awk -v s="$spent" -v b="${1:-0}" -v m="$MAX_DAILY_COST" 'BEGIN{exit !(s+b<=m)}'
}

# ---------------------------------------------------------------- 러너 직접 검증
# 정책 verify 가 있으면 그것을, 없으면 저장소 종류로 자동 감지한 명령을 러너가 직접 실행한다. 에이전트의 "통과했다"는 말은 믿지 않는다.
run_verify(){ # $1=작업 디렉터리 $2=결과 파일
  local wd=$1 outf=$2 src=policy tmo; local -a cmds=()
  mapfile -t cmds < <(policy "$n" '.verify[]?')
  tmo=$(policy "$n" '.verify_timeout_seconds'); tmo=${tmo:-1800}
  if [ ${#cmds[@]} -eq 0 ]; then
    src=auto
    [ -f "$wd/go.mod" ] && cmds+=("go build ./..." "go vet ./..." "go test ./...")
    if [ -f "$wd/package.json" ] && jq -e '.scripts.test' "$wd/package.json" >/dev/null 2>&1; then cmds+=("[ -d node_modules ] || npm ci --no-audit --no-fund" "npm test --silent"); fi
    if [ -f "$wd/package.json" ] && jq -e '.scripts.build' "$wd/package.json" >/dev/null 2>&1; then cmds+=("[ -d node_modules ] || npm ci --no-audit --no-fund" "npm run build --silent"); fi
    if [ -f "$wd/pyproject.toml" ] || [ -f "$wd/pytest.ini" ] || ls "$wd"/tests/*.py >/dev/null 2>&1; then cmds+=("python3 -m pytest -q -x"); fi
    [ -f "$wd/Cargo.toml" ] && cmds+=("cargo test --quiet")
    if [ ${#cmds[@]} -eq 0 ] && [ -f "$wd/Makefile" ] && grep -qE '^test:' "$wd/Makefile"; then cmds+=("make test"); fi
  fi
  [ ${#cmds[@]} -gt 0 ] || src=none
  echo "{\"source\":\"$src\",\"commands\":[]}" > "$outf"
  local c rc t0 t1
  for c in "${cmds[@]}"; do
    t0=$(date +%s)
    ( [ -f "$STATE/$n.env" ] && { set -a; . "$STATE/$n.env"; set +a; }; cd "$wd" && timeout "$tmo" bash -o pipefail -c "$c" ) >> "$OUT/verify.txt" 2>&1 && rc=0 || rc=$?
    t1=$(date +%s)
    jq --arg c "$c" --argjson rc "$rc" --argjson s "$((t1-t0))" '.commands += [{cmd:$c,exit:$rc,seconds:$s}]' "$outf" > "$outf.tmp" && mv "$outf.tmp" "$outf"
    log "$n: verify \`$c\` → exit $rc ($((t1-t0))s)"
    [ "$rc" -eq 0 ] || break
  done
  $GATE verify "$outf" > "$OUT/verify.gate.json" 2>/dev/null && return 0 || return 1
}

# ---------------------------------------------------------------- CI · 보호 파일 · 리뷰 · 비밀정보
# CI: check-runs 를 페이지 전체로 받아 gate 에 넘긴다. 두 번 연속 통과해야 한다(잡이 늦게 등록되는 경우). 확인 불가는 차단.
ci_gate(){ # $1=sha → CI_STATE, CI_REASON 설정; 0=통과
  local sha=$1 i passes=0 req allow f="$OUT/ci-${1:0:12}.json" g
  req=$(policy "$n" '.required_checks | join(",")'); allow=$(policy "$n" '.allow_merge_without_ci')
  for i in $(seq 1 40); do
    (cd "$repo" && gh api --paginate "repos/{owner}/{repo}/commits/$sha/check-runs" 2>/dev/null | jq -s '.') > "$f" 2>/dev/null || echo '{"message":"gh api failed"}' > "$f"
    g=$($GATE ci "$f" --sha "$sha" --required "$req" $( [ "$allow" = true ] && echo --allow-no-ci ) 2>/dev/null || true)
    CI_STATE=$(jq -r .state <<<"$g" 2>/dev/null || echo api-error); CI_REASON=$(jq -r .reason <<<"$g" 2>/dev/null || echo "gate 실행 실패")
    case "$CI_STATE" in
      success|no-ci-allowed) passes=$((passes+1)); [ $passes -ge 2 ] && return 0;;
      pending|no-ci|api-error) passes=0; { [ "$CI_STATE" = no-ci ] && [ $i -ge 10 ]; } && return 1; { [ "$CI_STATE" = api-error ] && [ $i -ge 6 ]; } && return 1;;
      *) return 1;;
    esac
    [ $i -eq 1 ] && log "$n: waiting for CI on ${sha:0:7} ($CI_STATE)"
    sleep 30
  done
  CI_STATE=timeout; CI_REASON="20분 안에 CI 완료를 확인하지 못함"; return 1
}
guarded_files(){ # $1=base
  local pat; pat=$(cat "$STATE/default.guard" "$STATE/$n.guard" 2>/dev/null | grep -v '^#' | grep -v '^[[:space:]]*$')
  [ -n "$pat" ] || return 0
  git -C "$wt" diff --name-only "$1..HEAD" 2>/dev/null | grep -E -f <(printf '%s\n' "$pat") || true
}
review_gate(){ # $1=base $2=PR url → 0=승인
  local rprompt g rb; rb=$(policy "$n" '.budget_usd.review'); rb=${rb:-4}
  budget_ok "$rb" || { stage review hold "예산 부족으로 리뷰를 돌리지 못함"; return 1; }
  rprompt=$(BASE="$1" REVIEW_FILE="$OUT/review.json" envsubst '$BASE $REVIEW_FILE' < "$REPO_DIR/review-prompt.md")
  run_agent review "$rprompt" "$wt" "$rb" "Bash,Read,Glob,Grep,Write"
  g=$($GATE review "$OUT/review.json" 2>/dev/null || true)
  if jq -e .ok <<<"$g" >/dev/null 2>&1; then stage review approved "$(jq -r .reason <<<"$g")"; return 0; fi
  stage review "$(jq -r '.state // "invalid"' <<<"$g" 2>/dev/null || echo invalid)" "$(jq -r '.reason // "리뷰 결과 없음"' <<<"$g" 2>/dev/null || echo '리뷰 결과 없음')"
  (cd "$repo" && gh pr comment "$2" --body "🧐 리뷰 게이트 보류 — $(jq -r '.reason // "리뷰 결과 없음"' <<<"$g" 2>/dev/null)

$(jq -r '.reasons[]? | "- " + .' "$OUT/review.json" 2>/dev/null)
(run $RUN_ID, 자율 개선 러너)" >>"$LOG" 2>&1) || true
  return 1
}
secrets_gate(){ # $1=설명 $2=파일(- 는 stdin)
  local g; g=$($GATE secrets "$2" 2>/dev/null || true)
  jq -e .ok <<<"$g" >/dev/null 2>&1 && return 0
  log "$n: SECRETS in $1 — $(jq -r .reason <<<"$g" 2>/dev/null)"; return 1
}

# ---------------------------------------------------------------- 기록 · 동기화
record_run(){ # $1=프로젝트 $2=결과 문장 $3=outcome
  local pr; pr=$(grep -o 'https://github.com/[^ ,]*/pull/[0-9]*' <<<"$2" | head -1 || true)
  local st='{}'; [ -f "${OUT:-/nonexistent}/stages.json" ] && st=$(cat "$OUT/stages.json")
  jq -cn --arg ts "$(date -Iseconds)" --arg d "$RUN_DATE" --arg p "$1" --arg r "$2" --arg o "$3" --arg rid "${RUN_ID:-}" \
     --arg b "${BASE_SHA:-}" --arg h "${HEAD_SHA:-}" --arg pr "$pr" --argjson m "${RUN_META:-{\}}" --argjson st "$st" \
     '{ts:$ts,date:$d,project:$p,result:$r,outcome:$o,run_id:$rid,base_sha:$b,head_sha:$h,pr:$pr,stages:$st} + $m' >> "$REPO_DIR/docs/data/runs.jsonl"
  RUN_META="{}"
  "$HERE/digest.sh" >>"$LOG" 2>&1 || true
}
# 에이전트가 $OUT 에 남긴 원장 항목·아이디어를 러너가 검사한 뒤 영구 기록에 반영한다 (비밀·내부 정보는 비공개 파일로)
merge_outputs(){
  local e="$OUT/ledger-entry.md"
  if [ -s "$e" ]; then
    if $GATE secrets "$e" --internal >/dev/null 2>&1; then cat "$e" >> "$STATE/$n.md"; echo >> "$STATE/$n.md"
    else mkdir -p "$STATE/private"; cat "$e" >> "$STATE/private/$n.md"; printf '## %s\n- (원장 항목에 비밀/내부 정보 의심 문자열이 있어 비공개 기록으로 옮김 — run %s)\n\n' "$RUN_DATE" "$RUN_ID" >> "$STATE/$n.md"; log "$n: ledger entry moved to private"; fi
  fi
  if [ -s "$OUT/ideas.json" ] && $GATE ideas "$OUT/ideas.json" >/dev/null 2>&1; then cp "$OUT/ideas.json" "$STATE/$n.ideas.json"
  elif [ -s "$OUT/ideas.json" ]; then log "$n: ideas.json 스키마 불합격 — 무시"; fi
  rm -rf "$OUT/home"  # 임시 홈은 남기지 않는다
}
redact_log(){ sed -E -i 's/gh[pousr]_[A-Za-z0-9]{20,}/ghX_[redacted]/g; s/github_pat_[A-Za-z0-9_]{20,}/github_pat_[redacted]/g; s#([a-z][a-z0-9+.-]*://[^/[:space:]:@]+):[^/[:space:]:@]+@#\1:[redacted]@#g; s/AKIA[0-9A-Z]{16}/AKIA[redacted]/g' "$1" 2>/dev/null || true; }
sync_repo(){
  [ "$SYNC" -eq 1 ] || return 0
  "$HERE/regress.sh" >>"$LOG" 2>&1 || true
  redact_log "$LOG"
  python3 "$HERE/report.py" >>"$LOG" 2>&1 || log "report.py FAILED"
  "$HERE/notify.sh" >>"$LOG" 2>&1 || true
  ( cd "$REPO_DIR" && git add -A state logs docs >/dev/null 2>&1 \
    && { git diff --cached --quiet || git commit -qm "$1"; } \
    && { git pull -q --rebase --autostash origin main >/dev/null 2>&1 || true; } && git push -q origin HEAD ) >>"$LOG" 2>&1 \
    && log "aidev synced: $1" || log "aidev sync FAILED: $1"
}

# ---------------------------------------------------------------- 릴리즈
release_context(){ # GitHub 에서 에이전트가 볼 정보를 미리 뽑는다 (에이전트는 gh 를 못 쓴다)
  { echo "### 최근 GitHub Release"; (cd "$repo" && gh release list --limit 5 2>/dev/null) || echo "(없음)"
    for t in $(cd "$repo" && gh release list --limit 3 --json tagName --jq '.[].tagName' 2>/dev/null); do
      echo; echo "### Release $t"; (cd "$repo" && gh release view "$t" --json name,assets,body --jq '"제목: \(.name)\n자산: \([.assets[].name]|join(", "))\n본문(앞부분): \(.body|.[0:600])"') 2>/dev/null
    done
    echo; echo "### 워크플로 파일"; ls "$repo/.github/workflows" 2>/dev/null || echo "(없음)"; } > "$OUT/release-context.md" 2>/dev/null
}
retry_release_workflow(){ # $1=태그 → 한 번 재실행; 또 실패면 수정 큐 + 교훈
  local tag=$1 rid i st conc step excerpt msha
  rid=$(cd "$repo" && gh run list --limit 40 --json databaseId,headBranch,conclusion --jq "[.[] | select(.headBranch==\"$tag\" and .conclusion==\"failure\")] | .[0].databaseId" 2>/dev/null)
  [ -n "$rid" ] && [ "$rid" != null ] || return 0
  (cd "$repo" && gh run rerun "$rid" --failed >>"$LOG" 2>&1) || { log "$n: rerun of $rid not possible"; return 0; }
  log "$n: release workflow $rid re-run for $tag — waiting"
  for i in $(seq 1 40); do sleep 30; read -r st conc < <(cd "$repo" && gh run view "$rid" --json status,conclusion --jq '"\(.status) \(.conclusion//"")"' 2>/dev/null || echo "unknown"); [ "$st" = completed ] && break; done
  if [ "${conc:-}" = success ]; then stage workflow recovered "재실행으로 성공 ($tag)"; return 0; fi
  step=$(cd "$repo" && gh run view "$rid" --json jobs --jq '[.jobs[] | .steps[] | select(.conclusion=="failure") | .name] | join(", ")' 2>/dev/null)
  excerpt=$(cd "$repo" && gh run view "$rid" --log-failed 2>/dev/null | sed 's/^[^\t]*\t[^\t]*\t//' | grep -i -E "error|fail|expected|mismatch" | grep -v -i deprecat | head -6 | cut -c1-200 | tr '\n' ' ' | sed 's/\t/ /g')
  msha=$(cd "$repo" && gh pr list --state merged --limit 1 --json mergeCommit --jq '.[0].mergeCommit.oid // ""' 2>/dev/null)
  touch "$STATE/fix-queue.tsv"
  grep -q -P "^$n\t" "$STATE/fix-queue.tsv" || printf '%s\t%s\t%s\n' "$n" "릴리즈 워크플로 실패 2회 — 태그 $tag, 실패 단계: ${step:-?}. 로그 요지: ${excerpt:-없음}" "$msha" >> "$STATE/fix-queue.tsv"
  jq -cn --arg ts "$(date -Iseconds)" --arg d "$RUN_DATE" --arg p "$n" --arg pr "$tag" --arg detail "릴리즈 워크플로가 2회 실패($step). 릴리즈 관련 검증은 머지 전에 로컬에서 재현할 것." '{ts:$ts,date:$d,project:$p,kind:"release-workflow-failed",pr:$pr,detail:$detail}' >> "$STATE/lessons.jsonl"
  stage workflow failed-twice "$step"; result="$result, queued for fix"
}
# Release 존재 보장 + 자산 업로드(불변: 같은 이름이 있으면 체크섬 비교, 다르면 충돌) + 자산 검증
publish_release(){ # $1=태그 $2=제목 $3=노트 $4=ghrel $5=자산 목록 파일(줄마다 경로)
  local tag=$1 title=$2 notes=$3 ghrel=$4 i cur_n prev prev_n new_n wf
  [ -n "$tag" ] || return 0
  local -a assets=(); [ -s "$5" ] && mapfile -t assets < "$5"
  if ! (cd "$repo" && gh release view "$tag" >/dev/null 2>&1); then
    if [ "$ghrel" = true ]; then
      local -a nargs=(--generate-notes); [ -n "$notes" ] && [ -f "$notes" ] && nargs=(--notes-file "$notes")
      (cd "$repo" && gh release create "$tag" --title "${title:-$tag}" "${nargs[@]}" >>"$LOG" 2>&1) && stage gh-release created "GitHub Release $tag" || stage gh-release create-failed "gh release create 실패"
    elif [ ${#assets[@]} -gt 0 ]; then
      for i in $(seq 1 30); do (cd "$repo" && gh release view "$tag" >/dev/null 2>&1) && break; sleep 30; done
      (cd "$repo" && gh release view "$tag" >/dev/null 2>&1) || { log "$n: 워크플로가 15분 안에 Release 를 만들지 않음 — 직접 만든다"; (cd "$repo" && gh release create "$tag" --title "${title:-$tag}" --generate-notes >>"$LOG" 2>&1) || true; }
    fi
  fi
  if [ ${#assets[@]} -gt 0 ]; then
    local a name existing sum_new sum_old conflict=0
    for a in "${assets[@]}"; do
      name=$(basename "$a"); sum_new=$(sha256sum "$a" | cut -d' ' -f1)
      existing=$(cd "$repo" && gh release view "$tag" --json assets --jq ".assets[] | select(.name==\"$name\") | .url" 2>/dev/null | head -1)
      if [ -n "$existing" ]; then
        sum_old=$(curl -sL -m 600 "$existing" | sha256sum | cut -d' ' -f1)
        if [ "$sum_old" = "$sum_new" ]; then log "$n: asset $name 동일(이미 게시됨)"; continue; fi
        log "$n: ASSET CONFLICT $name — 게시된 파일과 체크섬이 다름, 덮어쓰지 않음"; conflict=1; continue
      fi
      (cd "$repo" && gh release upload "$tag" "$a" >>"$LOG" 2>&1) && log "$n: uploaded $name ($sum_new)" || { log "$n: upload FAILED $name"; conflict=1; }
    done
    [ $conflict -eq 0 ] && stage assets uploaded "${#assets[@]}개" || { stage assets conflict "충돌/실패 있음 — 사람 확인"; result="$result, asset conflict"; }
  fi
  prev=$(cd "$repo" && gh release list --limit 10 --json tagName --jq "[.[].tagName] | map(select(. != \"$tag\")) | .[0] // empty" 2>/dev/null)
  prev_n=0; [ -n "$prev" ] && prev_n=$(cd "$repo" && gh release view "$prev" --json assets --jq '.assets | length' 2>/dev/null || echo 0)
  if [ "${prev_n:-0}" -gt 0 ]; then
    for i in $(seq 1 30); do new_n=$(cd "$repo" && gh release view "$tag" --json assets --jq '.assets | length' 2>/dev/null || echo 0); [ "${new_n:-0}" -gt 0 ] && break; sleep 30; done
    if [ "${new_n:-0}" -gt 0 ]; then stage assets verified "$tag 자산 $new_n개 (이전 $prev: $prev_n)"; OUTCOME=release-ready
    else
      wf=$(cd "$repo" && gh run list --limit 40 --json headBranch,conclusion,status,name --jq "[.[] | select(.headBranch==\"$tag\")] | .[0] | \"\(.name): \(.status)/\(.conclusion)\"" 2>/dev/null)
      stage assets missing "이전 $prev 엔 $prev_n개, $tag 엔 0개 — 워크플로: ${wf:-없음}"; OUTCOME=releasing
      case "$wf" in *failure*) result="$result, ASSETS MISSING (release workflow FAILED)"; retry_release_workflow "$tag";; *) result="$result, ASSETS MISSING";; esac
    fi
  else
    OUTCOME=release-ready; stage assets n/a "이전 릴리즈에도 자산 없음"
  fi
  cur_n=$(cd "$repo" && gh release view "$tag" --json assets --jq '.assets|length' 2>/dev/null || echo 0)
  jq --argjson n "${cur_n:-0}" --arg prev "${prev:-}" --argjson pn "${prev_n:-0}" --arg rid "$RUN_ID" '.assets_count=$n | .prev_tag=$prev | .prev_assets_count=$pn | .run_id=$rid' "$STATE/$n.release.json" > "$STATE/$n.release.json.tmp" 2>/dev/null && mv "$STATE/$n.release.json.tmp" "$STATE/$n.release.json" || true
}
release_project(){ # $1=base $2=변경 요약 [$3=assets] — 에이전트는 커밋·태그·자산만 만들고, 게시는 여기서
  local base=$1 summary=$2 mode=${3:-release} rwt="$WT_BASE/$n-release" rfile="$OUT/release.json" rprompt ref="origin/$1" mode_note="" latest="" g budget
  budget=$(policy "$n" ".budget_usd.$( [ "$mode" = assets ] && echo assets || echo release)"); [ -n "$RBUDGET" ] && budget=$RBUDGET; budget=${budget:-10}
  budget_ok "$budget" || { stage release hold "예산 부족"; result="$result, release hold (budget)"; return 0; }
  git -C "$repo" fetch -q --force --tags origin "$base" >>"$LOG" 2>&1 || log "$n: tag fetch had errors (continuing)"
  if [ "$mode" = assets ]; then
    latest=${ASSETS_TAG:-$(cd "$repo" && gh release list --limit 1 --json tagName --jq '.[0].tagName' 2>/dev/null)}
    [ -n "$latest" ] || { stage assets skipped "GitHub Release 없음"; return 0; }
    ref="refs/tags/$latest"
    mode_note="## 이번 세션은 자산만 만든다
이미 태그 \`$latest\` 와 GitHub Release 가 나가 있지만 이전 릴리즈에 있던 자산이 빠졌다. **버전을 올리거나 커밋·태그를 만들지 말고**, 체크아웃된 \`$latest\` 로 이전 릴리즈와 같은 자산을 같은 방법·같은 이름 규칙으로 \`$OUT/assets/\` 에 만들어 \`assets\` 에 적기만 하라. JSON 의 \`status\` 는 \`released\`, \`tag\` 는 \`$latest\`, \`github_release\` 는 \`false\`."
  fi
  git -C "$repo" worktree remove --force "$rwt" 2>/dev/null || true
  git -C "$repo" worktree add --detach "$rwt" "$ref" >>"$LOG" 2>&1
  git -C "$rwt" remote set-url --push origin DISABLED >/dev/null 2>&1 || true
  release_context
  rprompt=$(RELEASE_FILE="$rfile" OUT_DIR="$OUT" CHANGE_SUMMARY="$summary" MODE_NOTE="$mode_note" RELEASE_CONTEXT="$(cat "$OUT/release-context.md")" \
            envsubst '$RELEASE_FILE $OUT_DIR $CHANGE_SUMMARY $MODE_NOTE $RELEASE_CONTEXT' < "$REPO_DIR/release-prompt.md")
  run_agent "$( [ "$mode" = assets ] && echo assets || echo release)" "$rprompt" "$rwt" "$budget" "Bash,Read,Edit,Write,Glob,Grep"
  g=$($GATE release "$rfile" --out-dir "$OUT" 2>/dev/null || true)
  cp "$rfile" "$STATE/$n.release.json" 2>/dev/null || echo '{"status":"missing"}' > "$STATE/$n.release.json"
  local status tag title notes ghrel ahead tagged
  status=$(jq -r '.state // "missing"' <<<"$g" 2>/dev/null || echo missing)
  jq -r '.assets_ok[]?' <<<"$g" > "$OUT/assets.list" 2>/dev/null || true
  if ! jq -e .ok <<<"$g" >/dev/null 2>&1; then
    stage release "$status" "$(jq -r '.reason // "결과 없음"' <<<"$g" 2>/dev/null || echo '결과 없음')"; result="$result, release $status"
    git -C "$repo" worktree remove --force "$rwt" >>"$LOG" 2>&1 || true; return 0
  fi
  tag=$(jq -r '.tag // ""' "$rfile"); title=$(jq -r '.title // ""' "$rfile"); notes=$(jq -r '.notes_file // ""' "$rfile"); ghrel=$(jq -r '.github_release // false' "$rfile")
  if [ "$mode" = assets ]; then
    result="$result, assets for $latest"; OUTCOME=releasing; publish_release "$latest" "$title" "$notes" false "$OUT/assets.list"
    git -C "$repo" worktree remove --force "$rwt" >>"$LOG" 2>&1 || true; return 0
  fi
  ahead=$(git -C "$rwt" rev-list --count "origin/$base..HEAD")
  tagged=0; [ -n "$tag" ] && git -C "$rwt" rev-parse -q --verify "refs/tags/$tag" >/dev/null 2>&1 && tagged=1
  if [ "$ahead" -eq 0 ] && [ "$tagged" -eq 0 ]; then stage release nothing "커밋도 태그도 없음"; git -C "$repo" worktree remove --force "$rwt" >>"$LOG" 2>&1 || true; return 0; fi
  # 릴리즈 커밋 비밀정보 검사 → 푸시 → 그 커밋의 CI 성공 확인 → 태그 푸시 (CI 실패한 커밋에는 태그를 밀지 않는다)
  if [ "$ahead" -gt 0 ] && ! git -C "$rwt" diff "origin/$base..HEAD" | secrets_gate "release diff" -; then stage release blocked "릴리즈 커밋에 비밀정보 의심"; result="$result, release blocked (secrets)"; git -C "$repo" worktree remove --force "$rwt" >>"$LOG" 2>&1 || true; return 0; fi
  git -C "$rwt" remote set-url --push origin "$(git -C "$repo" remote get-url origin)" >/dev/null 2>&1
  if [ "$ahead" -gt 0 ]; then
    git -C "$rwt" push origin "HEAD:$base" >>"$LOG" 2>&1 || { stage release push-failed "릴리즈 커밋 푸시 실패"; result="$result, release push failed"; git -C "$repo" worktree remove --force "$rwt" >>"$LOG" 2>&1 || true; return 0; }
    if [ "$tagged" -eq 1 ] && ! ci_gate "$(git -C "$rwt" rev-parse HEAD)"; then
      stage release ci-blocked "릴리즈 커밋 CI: $CI_STATE — $CI_REASON (태그 보류)"; result="$result, release tag held ($CI_STATE)"; OUTCOME=releasing
      git -C "$repo" pull --ff-only origin "$base" >>"$LOG" 2>&1 || true; git -C "$repo" worktree remove --force "$rwt" >>"$LOG" 2>&1 || true; return 0
    fi
  fi
  if [ "$tagged" -eq 1 ]; then
    git -C "$rwt" push origin "refs/tags/$tag" >>"$LOG" 2>&1 || { stage release tag-push-failed "태그 푸시 실패"; result="$result, tag push failed"; git -C "$repo" worktree remove --force "$rwt" >>"$LOG" 2>&1 || true; return 0; }
  fi
  stage release published "${tag:-$(jq -r .version "$rfile")}"; result="$result, released ${tag:-$(jq -r .version "$rfile")}"; OUTCOME=releasing
  git -C "$repo" pull --ff-only origin "$base" >>"$LOG" 2>&1 || true
  printf -- '- 릴리즈: %s (%s, run %s)\n' "${tag:-$(jq -r .version "$rfile")}" "$RUN_DATE" "$RUN_ID" >> "$STATE/$n.md"
  publish_release "$tag" "$title" "$notes" "$ghrel" "$OUT/assets.list"
  git -C "$repo" worktree remove --force "$rwt" >>"$LOG" 2>&1 || true
}
rollback_project(){ # $1=머지 커밋
  local sha=$1 rwt="$WT_BASE/$n-revert" br="revert/$RUN_DATE-$(date +%H%M)" url
  [ -n "$sha" ] || { log "$n: rollback skipped (no merge sha)"; return 0; }
  git -C "$repo" fetch -q origin "$base" >>"$LOG" 2>&1 || true
  git -C "$repo" worktree remove --force "$rwt" 2>/dev/null || true
  git -C "$repo" worktree add -b "$br" "$rwt" "origin/$base" >>"$LOG" 2>&1 || return 0
  if (cd "$rwt" && { git revert --no-edit -m 1 "$sha" || git revert --no-edit "$sha"; } >>"$LOG" 2>&1); then
    git -C "$rwt" push -u origin "$br" >>"$LOG" 2>&1
    url=$(cd "$rwt" && gh pr create --base "$base" --head "$br" --title "revert: 자율 개선 변경 되돌리기 (${sha:0:7})" --body "릴리즈 워크플로가 반복 실패했고 수정 회차도 실패해 ${sha:0:7} 을 되돌리는 PR 입니다. 사람이 검토 후 머지해 주세요.

🤖 aidev 자동 롤백 · run $RUN_ID · https://hkjang.github.io/aidev/projects/$n/" 2>>"$LOG" || true)
    stage rollback pr-opened "$url"; result="$result, rollback PR $url"
    jq -cn --arg ts "$(date -Iseconds)" --arg d "$RUN_DATE" --arg p "$n" --arg pr "$url" --arg detail "머지 ${sha:0:7} 이 릴리즈를 반복해서 깨뜨려 되돌림 PR 을 열었다. 같은 접근은 피할 것." '{ts:$ts,date:$d,project:$p,kind:"rolled-back",pr:$pr,detail:$detail}' >> "$STATE/lessons.jsonl"
  else stage rollback conflict "revert 충돌 — 사람 개입"; result="$result, rollback failed"; fi
  git -C "$repo" worktree remove --force "$rwt" >>"$LOG" 2>&1 || true
}

# ================================================================ 단독 모드
if [ -n "${ASSETS_ONLY:-}" ] || [ -n "${RELEASE_ONLY:-}" ]; then
  if [ -n "${ASSETS_ONLY:-}" ]; then n=${ASSETS_ONLY%%:*}; ASSETS_TAG=""; [[ "$ASSETS_ONLY" == *:* ]] && ASSETS_TAG=${ASSETS_ONLY#*:}; kind=assets; else n=$RELEASE_ONLY; kind=release; fi
  repo="$ROOT/$n"; base=$(policy "$n" '.base_branch'); base=${base:-main}; result="$kind-only"; OUTCOME=merged; RUN_META="{}"; BASE_SHA=""; HEAD_SHA=""
  new_run "$n" "$kind"; log "=== $n $kind-only (base=$base, run $RUN_ID)"
  if [ "$kind" = assets ]; then release_project "$base" "(자산 보충)" assets; else release_project "$base" "$(tail -n 8 "$STATE/$n.md" 2>/dev/null)"; fi
  rm -rf "$OUT/home"; record_run "$n" "$result" "${OUTCOME:-error}"; sync_repo "run($RUN_DATE): $n — $result"; log "done"; exit 0
fi

# ================================================================ 일일 상한 · 큐 · 후보
if [ -z "$ONLY" ] && [ $DRY -eq 0 ]; then
  today_cost=$(jq -s --arg d "$RUN_DATE" '[.[]|select(.date==$d)|.cost_usd//0]|add // 0' "$REPO_DIR/docs/data/usage.jsonl" 2>/dev/null || echo 0)
  today_rounds=$(jq -s --arg d "$RUN_DATE" '[.[]|select(.date==$d)]|length' "$REPO_DIR/docs/data/runs.jsonl" 2>/dev/null || echo 0)
  today_rel=$(jq -s --arg d "$RUN_DATE" '[.[]|select(.date==$d)|select(.result|test("released v?[0-9]"))]|length' "$REPO_DIR/docs/data/runs.jsonl" 2>/dev/null || echo 0)
  cap=""; awk -v c="$today_cost" -v m="$MAX_DAILY_COST" 'BEGIN{exit !(c>=m)}' && cap="비용 \$$today_cost ≥ \$$MAX_DAILY_COST"
  [ "${today_rounds:-0}" -ge "$MAX_DAILY_ROUNDS" ] && cap="회차 $today_rounds ≥ $MAX_DAILY_ROUNDS"
  [ "${today_rel:-0}" -ge "$MAX_DAILY_RELEASES" ] && cap="릴리즈 $today_rel ≥ $MAX_DAILY_RELEASES"
  if [ -n "$cap" ]; then
    log "daily cap reached: $cap"
    if [ ! -f "$STATE/.cap-$RUN_DATE" ]; then
      touch "$STATE/.cap-$RUN_DATE"; ps1=$(wslpath -w "$HERE/toast.ps1" 2>/dev/null); [ -n "$ps1" ] && powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$ps1" -Title "aidev 일일 상한 도달" -Message "$cap" >/dev/null 2>&1
      n="(runner)"; RUN_META="{}"; BASE_SHA=""; HEAD_SHA=""; record_run "(runner)" "daily cap reached: $cap" "error"; sync_repo "run($RUN_DATE): daily cap — $cap"
    fi
    exit 0
  fi
fi
[ -z "$ONLY" ] && "$HERE/inbox.sh" >>"$LOG" 2>&1 || true

candidates=(); since=$(date -d "-$DAYS days" +%s); touch "$STATE/fix-queue.tsv" "$STATE/run-queue.tsv"
for d in "$ROOT"/*/; do
  n=$(basename "$d"); [ -d "$d/.git" ] || continue
  [[ "$n" =~ $EXCLUDE_RE ]] && continue
  [ -n "$ONLY" ] && [ "$n" != "$ONLY" ] && continue
  last=$(git -C "$d" log -1 --format=%ct 2>/dev/null || echo 0); [ "$last" -ge "$since" ] || continue
  git -C "$d" remote get-url origin >/dev/null 2>&1 || continue
  [ -z "$(git -C "$d" status --porcelain 2>/dev/null)" ] || { log "skip $n: dirty working tree"; continue; }
  if [ -z "$ONLY" ] && [ -s "$REPO_DIR/docs/data/runs.jsonl" ]; then
    read -r streak lastd < <(jq -rs --arg p "$n" --argjson k "$DORMANT_AFTER" '[.[]|select(.project==$p)] | (.[-$k:]) as $l | [(($l|length)==$k and all($l[]; .result|test("no change"))), ($l[-1].date // "")] | @tsv' "$REPO_DIR/docs/data/runs.jsonl" 2>/dev/null || echo "false ")
    if [ "$streak" = true ] && [ -n "$lastd" ] && [ $(( ($(date +%s) - $(date -d "$lastd" +%s)) / 86400 )) -lt "$DORMANT_DAYS" ] && ! grep -q -P "^$n\t" "$STATE/fix-queue.tsv" "$STATE/run-queue.tsv" 2>/dev/null; then
      log "skip $n: dormant (변경 없음 ${DORMANT_AFTER}연속, $lastd)"; continue
    fi
  fi
  candidates+=("$n")
done
[ ${#candidates[@]} -gt 0 ] || { log "no candidates"; exit 0; }
log "candidates(${#candidates[@]}): ${candidates[*]}"

CURSOR="$STATE/.cursor"; idx=$(cat "$CURSOR" 2>/dev/null || echo 0); picked=()
for ((i=0;i<COUNT && i<${#candidates[@]};i++)); do picked+=("${candidates[$(( (idx+i) % ${#candidates[@]} ))]}"); done
FIX_PROJECT=""; FIX_NOTE_TEXT=""; FIX_SHA=""; FIXQ="$STATE/fix-queue.tsv"
if [ -z "$ONLY" ] && [ -s "$FIXQ" ]; then
  while IFS=$'\t' read -r fp fnote fsha; do [ -n "$fp" ] || continue
    if printf '%s\n' "${candidates[@]}" | grep -qx "$fp"; then picked=("$fp"); FIX_PROJECT="$fp"; FIX_NOTE_TEXT="$fnote"; FIX_SHA="${fsha:-}"; log "fix-queue: picked $fp"; break; fi
  done < "$FIXQ"
fi
RUN_PROJECT=""; RUN_ISSUE=""; RUNQ="$STATE/run-queue.tsv"
if [ -z "$FIX_PROJECT" ] && [ -z "$ONLY" ] && [ -s "$RUNQ" ]; then
  while IFS=$'\t' read -r rp rnote rnum; do [ -n "$rp" ] || continue
    if printf '%s\n' "${candidates[@]}" | grep -qx "$rp"; then picked=("$rp"); RUN_PROJECT="$rp"; RUN_ISSUE="$rnum"; log "run-queue: picked $rp ($rnote)"; break
    else (cd "$REPO_DIR" && gh issue comment "$rnum" --body "\`$rp\` 은 지금 후보가 아닙니다(미커밋 변경/원격 없음/30일 무활동). 정리 후 다시 라벨을 달아 주세요." >/dev/null 2>&1; gh issue edit "$rnum" --remove-label run >/dev/null 2>&1) || true; grep -v -P "^$rp\t" "$RUNQ" > "$RUNQ.tmp"; mv "$RUNQ.tmp" "$RUNQ"; fi
  done < "$RUNQ"
fi
[ -n "$FIX_PROJECT" ] || [ -n "$RUN_PROJECT" ] || echo $(( (idx+COUNT) % ${#candidates[@]} )) > "$CURSOR"
log "picked: ${picked[*]}"
[ $DRY -eq 1 ] && exit 0

# ================================================================ 프로젝트별 회차
for n in "${picked[@]}"; do
  repo="$ROOT/$n"; ledger="$STATE/$n.md"; wt="$WT_BASE/$n"; result="no change"; OUTCOME=no-change; RUN_META="{}"; HEAD_SHA=""; BASE_SHA=""; url=""
  base=$(policy "$n" '.base_branch'); base=${base:-$(git -C "$repo" symbolic-ref --short HEAD)}
  new_run "$n" improve
  ibudget=$(policy "$n" '.budget_usd.improve'); [ -n "$BUDGET" ] && ibudget=$BUDGET; ibudget=${ibudget:-8}
  log "=== $n (base=$base, run $RUN_ID)"
  budget_ok "$ibudget" || { stage improve hold "예산 부족"; record_run "$n" "hold: budget" "error"; continue; }
  # 기준 커밋 고정: 원격의 base 에서 시작하고 SHA 를 기록한다
  git -C "$repo" fetch -q origin "$base" >>"$LOG" 2>&1 || { stage improve error "fetch 실패"; record_run "$n" "error: fetch" "error"; continue; }
  BASE_SHA=$(git -C "$repo" rev-parse "origin/$base"); slug="auto/$RUN_DATE-$(date +%H%M)"
  git -C "$repo" worktree remove --force "$wt" 2>/dev/null || true
  git -C "$repo" worktree add -b "$slug" "$wt" "$BASE_SHA" >>"$LOG" 2>&1
  git -C "$wt" remote set-url --push origin DISABLED >/dev/null 2>&1 || true
  stage base pinned "$base@${BASE_SHA:0:7}"

  fix_note=""; [ "$n" = "$FIX_PROJECT" ] && fix_note="## 우선 과제 (자동 배정) — 이번 회차는 새 아이디어 대신 아래 실패를 고치세요
$(printf '%b' "$FIX_NOTE_TEXT")
릴리즈 워크플로가 같은 이유로 두 번 실패했습니다. 워크플로 파일과 실패한 단계의 스크립트·테스트를 읽고 원인을 고치세요. 워크플로 자체를 느슨하게 만들어 통과시키는 것은 금지입니다. 고친 뒤 같은 검증을 로컬에서 재현해 통과를 확인하고, 원장에 '수정 과제' 로 기록하세요."
  lessons=$(jq -r --arg p "$n" 'select(.project==$p) | "- \(.date) [\(.kind)] \(.detail)"' "$STATE/lessons.jsonl" 2>/dev/null | tail -n 8)
  ideas=$(jq -r '.[]? | select(.status=="pending") | "- [\(.value)/\(.risk)/\(.size)] \(.title) — \(.note // "")"' "$STATE/$n.ideas.json" 2>/dev/null | head -n 12)
  prompt=$(LEDGER_FILE="$OUT/ledger-entry.md" IDEAS_FILE="$OUT/ideas.json" OUT_DIR="$OUT" RUN_DATE="$RUN_DATE" FIX_NOTE="$fix_note" LESSONS="${lessons:-(없음)}" IDEAS_CONTENT="${ideas:-(없음)}" \
           LEDGER_CONTENT="$(tail -n 60 "$ledger" 2>/dev/null || echo '(없음)')" \
           envsubst '$LEDGER_FILE $IDEAS_FILE $OUT_DIR $RUN_DATE $LEDGER_CONTENT $FIX_NOTE $LESSONS $IDEAS_CONTENT' < "$REPO_DIR/prompt.md")
  run_agent improve "$prompt" "$wt" "$ibudget" "Bash,Read,Edit,Write,Glob,Grep,WebFetch,WebSearch"
  merge_outputs
  jq -e '.type=="result"' "$OUT/agent-improve.json" >/dev/null 2>&1 || { OUTCOME=error; result="error: agent produced no result ($(head -c 120 "$OUT/agent-improve.txt" 2>/dev/null | tr '\n' ' '))"; stage improve error "$result"; }

  ahead=$(git -C "$wt" rev-list --count "$BASE_SHA..HEAD")
  if [ "$ahead" -gt 0 ]; then
    HEAD_SHA=$(git -C "$wt" rev-parse HEAD)
    RUN_META=$(git -C "$wt" diff --numstat "$BASE_SHA..HEAD" | awk 'BEGIN{f=0;a=0;d=0;t=0} {f++; a+=$1; d+=$2; if ($3 ~ /(^|\/)(test|tests|spec|__tests__)\/|_test\.|\.test\.|\.spec\.|Test\.java|test_.*\.py/) t++} END{printf "{\"files\":%d,\"additions\":%d,\"deletions\":%d,\"tests\":%d}", f,a,d,t}')
    RUN_META=$(jq -c --arg t "$(git -C "$wt" log -1 --format=%s)" '. + {title:$t}' <<<"$RUN_META" 2>/dev/null || echo "{}")
    # 1) 러너 직접 검증  2) 비밀정보 검사  — 둘 다 통과해야 PR 을 연다
    if ! run_verify "$wt" "$OUT/verify.json"; then
      stage verify failed "$(jq -r .reason "$OUT/verify.gate.json" 2>/dev/null || echo '검증 실패')"; result="verify failed: $(jq -r .reason "$OUT/verify.gate.json" 2>/dev/null | cut -c1-120)"; OUTCOME=verify-failed
    elif ! git -C "$wt" diff "$BASE_SHA..HEAD" | secrets_gate "diff" -; then
      stage verify failed "변경에 비밀정보 의심 문자열"; result="verify failed: secrets in diff"; OUTCOME=verify-failed
    else
      stage verify passed "$(jq -r .reason "$OUT/verify.gate.json")"
      git -C "$wt" remote set-url --push origin "$(git -C "$repo" remote get-url origin)" >/dev/null 2>&1
      git -C "$wt" push -u origin "$slug" >>"$LOG" 2>&1
      git -C "$wt" remote set-url --push origin DISABLED >/dev/null 2>&1 || true
      body=$(printf '자율 개선 에이전트가 생성한 PR입니다. (run %s, base %s)\n\n%s\n\n🤖 auto-improve %s · https://hkjang.github.io/aidev/projects/%s/' "$RUN_ID" "${BASE_SHA:0:7}" "$(tail -n 12 "$OUT/ledger-entry.md" 2>/dev/null)" "$RUN_DATE" "$n")
      url=$(cd "$repo" && gh pr create --base "$base" --head "$slug" --title "auto-improve: $(git -C "$wt" log -1 --format=%s)" --body "$body" 2>>"$LOG" || true)
      stage pr created "$url"; result="PR $url"; OUTCOME=review-pending
      merge_ok=0; [ "$MERGE" -eq 1 ] && [ "$(policy "$n" '.auto_merge')" = true ] && [ -n "$url" ] && merge_ok=1
      if [ "$merge_ok" -eq 1 ]; then
        guarded=$(guarded_files "$BASE_SHA")
        if [ -n "$guarded" ]; then
          stage guard held "$(tr '\n' ' ' <<<"$guarded")"; result="guarded files, PR open $url"
          (cd "$repo" && gh pr comment "$url" --body "🔒 보호 파일을 건드려 자동 머지하지 않습니다. 사람이 검토해 주세요.

$(sed 's/^/- /' <<<"$guarded")

(run $RUN_ID)" >>"$LOG" 2>&1) || true; merge_ok=0
        elif [ "$REVIEW" -eq 1 ] && ! review_gate "$base" "$url"; then result="review held, PR open $url"; merge_ok=0; fi
      fi
      if [ "$merge_ok" -eq 1 ]; then
        # 기준 브랜치가 그새 움직였으면 리베이스 후 재검증한다 — 검증하지 않은 조합을 머지하지 않는다
        git -C "$repo" fetch -q origin "$base" >>"$LOG" 2>&1 || true
        if [ "$(git -C "$repo" rev-parse "origin/$base")" != "$BASE_SHA" ]; then
          log "$n: base moved (${BASE_SHA:0:7} → $(git -C "$repo" rev-parse --short "origin/$base")) — rebase & re-verify"
          if git -C "$wt" rebase "origin/$base" >>"$LOG" 2>&1 && run_verify "$wt" "$OUT/verify-rebased.json"; then
            BASE_SHA=$(git -C "$repo" rev-parse "origin/$base"); HEAD_SHA=$(git -C "$wt" rev-parse HEAD)
            git -C "$wt" remote set-url --push origin "$(git -C "$repo" remote get-url origin)" >/dev/null 2>&1
            git -C "$wt" push --force-with-lease origin "$slug" >>"$LOG" 2>&1; git -C "$wt" remote set-url --push origin DISABLED >/dev/null 2>&1 || true
            stage base rebased "${BASE_SHA:0:7}, 재검증 통과"
          else git -C "$wt" rebase --abort >/dev/null 2>&1 || true; stage base conflict "리베이스 충돌 또는 재검증 실패 — 보류"; result="base moved, PR open $url"; merge_ok=0; fi
        fi
      fi
      if [ "$merge_ok" -eq 1 ]; then
        if ci_gate "$HEAD_SHA"; then
          stage ci passed "$CI_REASON"
          git -C "$repo" worktree remove --force "$wt" >>"$LOG" 2>&1 || true
          if (cd "$repo" && gh pr merge "$url" --merge --delete-branch --match-head-commit "$HEAD_SHA" >>"$LOG" 2>&1); then
            stage merge done "$HEAD_SHA"; result="merged $url"; OUTCOME=merged
            git -C "$repo" pull --ff-only origin "$base" >>"$LOG" 2>&1 || true
            [ "$RELEASE" -eq 1 ] && [ "$(policy "$n" '.release')" = true ] && release_project "$base" "$(tail -n 8 "$OUT/ledger-entry.md" 2>/dev/null)"
          else stage merge failed "gh pr merge 실패 (커밋 불일치 또는 충돌)"; result="merge failed $url"; OUTCOME=review-pending; fi
        else stage ci "$CI_STATE" "$CI_REASON"; result="CI ${CI_STATE}, PR open $url"; OUTCOME=$( [ "$CI_STATE" = failed ] && echo verify-failed || echo review-pending ); fi
      fi
    fi
  else
    [ "$OUTCOME" = error ] || stage improve no-change "커밋 없음"
    git -C "$repo" branch -D "$slug" >>"$LOG" 2>&1 || true
  fi
  git -C "$repo" worktree remove --force "$wt" >>"$LOG" 2>&1 || true
  rm -rf "$OUT/home"
  if [ "$n" = "$RUN_PROJECT" ]; then
    grep -v -P "^$n\t" "$RUNQ" > "$RUNQ.tmp" 2>/dev/null; mv "$RUNQ.tmp" "$RUNQ"; result="manual: $result"
    [ -n "$RUN_ISSUE" ] && (cd "$REPO_DIR" && gh issue comment "$RUN_ISSUE" --body "✅ 실행 완료 — $result

https://hkjang.github.io/aidev/projects/$n/" >/dev/null 2>&1; gh issue close "$RUN_ISSUE" >/dev/null 2>&1) || true
  fi
  if [ "$n" = "$FIX_PROJECT" ]; then
    grep -v -P "^$n\t" "$FIXQ" > "$FIXQ.tmp" 2>/dev/null; mv "$FIXQ.tmp" "$FIXQ"; result="fix-round: $result"
    case "$OUTCOME" in merged|releasing|release-ready) ;; *) log "$n: fix round did not merge — rolling back"; rollback_project "$FIX_SHA";; esac
  fi
  record_run "$n" "$result" "$OUTCOME"
  sync_repo "run($RUN_DATE): $n — $result"
done
log "done"
