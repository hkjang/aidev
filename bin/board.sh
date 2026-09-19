#!/usr/bin/env bash
# 이사회(주간) — 경영진 부서가 회사 상태를 읽고 다음 주의 결정을 제안한다.
#
# aidev 를 회사처럼 운영한다면 누군가는 매주 "돈과 주의를 어디에 둘지" 를 정해야 한다. 지금까지 그것은
# 운영자가 텔레그램에서 즉흥적으로 했다. 이 스크립트는 headcount 의 경영진 부서(executive·finance·pmo·
# people·data-analytics·corporate-strategy) 스킬을 실은 읽기 전용 세션이 대시보드 요약·비용·성적표·캠페인
# 진행·머지 뒤 수정률·비교 실험 표·교훈을 읽고, **ops.sh 동사로 적을 수 있는 제안 목록**을 낸다.
# 적용은 사람이 한다: 코파일럿에 "board apply 1 3" (authority: proposes — 제안은 하되 스스로 적용하지 않는다).
#
# 산출물: state/board/<ISO주>.md (메모), state/board/<ISO주>.json (제안 목록). 헬스체크가 주마다 한 번 부른다.
#   bin/board.sh            이번 주 이사회 (이미 열렸으면 건너뜀)
#   bin/board.sh --force    다시 연다
#   bin/board.sh --reparse  메모는 그대로 두고 제안 목록만 다시 뽑는다 (파서를 고쳤을 때)
set -uo pipefail
export HOME="${HOME:-/home/hkjang}"
export PATH="$HOME/.local/bin:$HOME/.nvm/versions/node/v22.23.1/bin:/usr/local/bin:/usr/bin:/bin"
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"; REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="$REPO_DIR/state"; DATA="$REPO_DIR/docs/data"; ROOT="${ROOT:-/mnt/c/Users/USER/projects}"
HEADCOUNT_DIR="${HEADCOUNT_DIR:-$ROOT/headcount}"
OUTD="$STATE/board"; mkdir -p "$OUTD"; WEEK=$(date +%G-W%V); MD="$OUTD/$WEEK.md"; JS="$OUTD/$WEEK.json"
MODEL="${MODEL:-claude-opus-5}"; BUDGET="${AIDEV_BOARD_BUDGET:-4}"
MODE=${1:-}
case "$MODE" in --force|--reparse) ;; *) [ -f "$JS" ] && { echo "이번 주($WEEK) 이사회는 이미 열렸다: $JS"; exit 0; };; esac
[ -f "$STATE/NO-BOARD" ] && { echo "NO-BOARD — 이사회 중지"; exit 0; }
command -v claude >/dev/null || { echo "claude 없음"; exit 1; }

pargs=(); for d in executive finance pmo people data-analytics corporate-strategy; do [ -f "$HEADCOUNT_DIR/plugins/$d/.claude-plugin/plugin.json" ] && pargs+=(--plugin-dir "$HEADCOUNT_DIR/plugins/$d"); done
last_md=$(ls -1 "$OUTD"/*.md 2>/dev/null | grep -v "$WEEK" | tail -1)
summary=$(jq -c '{today, totals, quality, campaign_progress: [.campaign_progress[]|{id,done,processed,total,pct,spent_usd,budget_usd,until,counts,lessons:(.lessons|length)}], shepherd, agents, alerts:[.alerts[]|{project,why}], stops, health:{ok:.health.ok,problems:.health.problems}}' "$DATA/summary.json" 2>/dev/null)
postmerge=$(jq -c '.summary' "$DATA/postmerge.json" 2>/dev/null)
ab=$(cat "$REPO_DIR"/docs/paper/experiments/ab-*.md 2>/dev/null | head -c 4000)
lessons=$(tail -n 15 "$STATE/lessons.jsonl" 2>/dev/null | jq -r '"- \(.date) [\(.project)] \(.kind): \(.detail|.[0:140])"' 2>/dev/null)
prefs=$(cat "$STATE/operator-preferences.md" 2>/dev/null | head -c 2500)
verbs=$(bash "$HERE/ops.sh" help)
prompt="당신은 자율 개선 회사 aidev 의 주간 이사회입니다. 이 세션에는 경영진 부서(executive·finance·pmo·people·data-analytics·corporate-strategy)의 스킬이 실려 있습니다. 시작하기 전에 Skill 도구로 \`executive:chief-executive\`, \`finance:capital-allocation\`, \`pmo:portfolio-governance\`, \`people:performance-management\` 을 불러 그 방법을 따르세요. 회사는 사람이 소유한 저장소 46개를 러너와 에이전트들이 무인으로 개선·머지·릴리즈합니다. 운영자(CEO 겸 유일한 사람)는 hkjang 입니다. 지금은 $(date '+%Y-%m-%d (%a)'), ISO 주 $WEEK 입니다.

## 이번 주 자료
### 대시보드 요약 (summary.json)
$summary

### 머지 뒤 30일 수정률 (postmerge.json)
${postmerge:-(없음)}

### 비교 실험 표 (arm 별)
${ab:-(아직 없음)}

### 최근 교훈
${lessons:-(없음)}

### 운영자 취향 (지켜야 할 것)
${prefs:-(없음)}

### 지난 이사회 메모
$( [ -n "$last_md" ] && head -c 3000 "$last_md" || echo "(첫 이사회)")

## 회사가 쓸 수 있는 동사 (제안은 반드시 이 동사로만)
$verbs

## 해야 할 일
1. **상황 판단** — 지난주에 무엇이 됐고 무엇이 안 됐나. 돈은 어디에 갔고 무엇을 만들었나(머지당 비용, 캠페인별 처리율, 검토 대기 적체, 머지 뒤 수정률, 사람 필요 건수). 숫자로 말하고 짐작을 사실처럼 쓰지 마세요.
2. **하나의 제약** — 이번 주 회사를 가장 세게 묶는 제약 하나를 이름 붙이세요(예: 검토 대기 적체, 특정 캠페인의 막힘, 예산 소진, 회귀).
3. **결정 제안 3~7개** — 각각 ops.sh 동사 하나로 적을 수 있어야 합니다(campaign budget/until/pause/resume/add/remove, shepherd on|off, stop/resume <범위>, run <프로젝트> \"<명세>\", draft-campaign \"<목표>\"). 각 제안에 왜(근거 숫자), 무엇을 포기하는지(자원은 유한하다), 되돌리는 법을 적으세요. 우선순위 목록에 '아래 선' 이 있어야 합니다 — 하지 않기로 한 것도 적으세요.
4. **역할 성적** — 성적표(agents)에서 조정할 역할이 있으면(정찰 채택률, 비평 승인 뒤 수정률, 처리기 사람 필요율) 제안에 넣되, 정책 파일을 바꾸는 일은 사람 몫이므로 '권고' 로 표시하세요.
5. 파일을 쓰거나 명령을 실행하지 마세요. 읽고 판단만 합니다.

## 출력
한국어 메모를 쓰세요: 제목 '# 이사회 $WEEK', 절은 '## 상황' / '## 이번 주의 제약' / '## 결정 제안' / '## 하지 않는 것' / '## 역할 권고', 40줄 안. **결정 제안은 반드시** \`N. \\\`동사 인자...\\\` — 왜. 포기: ... 되돌림: \\\`동사 인자...\\\`\` 꼴의 번호 목록으로 쓰세요(백틱 안이 그대로 실행되는 명령입니다). 메모 뒤 마지막 줄에 다음 JSON 한 줄을 덧붙이세요(코드펜스 없이):
{\"constraint\":\"한 문장\",\"proposals\":[{\"n\":1,\"cmd\":[\"campaign\",\"budget\",\"<id>\",\"<usd>\"],\"why\":\"근거\",\"gives_up\":\"포기하는 것\",\"revert\":\"되돌리는 법\"}],\"not_doing\":[\"...\"],\"role_advice\":[\"...\"]}"

if [ "$MODE" = --reparse ] && [ -s "$MD" ]; then
  text=$(cat "$MD"); cost=$(jq -r '.cost_usd // 0' "$JS" 2>/dev/null); cost=${cost:-0}
else
  out=$(mktemp)
  ( cd "$REPO_DIR" && timeout -k 30 900 claude -p "$prompt" --model "$MODEL" --settings '{"attribution":{"commit":"","pr":""}}' \
      --permission-mode plan --allowedTools "Read Grep Glob Skill" --max-budget-usd "$BUDGET" --output-format json ${pargs[@]+"${pargs[@]}"} </dev/null >"$out" 2>>"$REPO_DIR/logs/board.err" )
  text=$(jq -r '.result // ""' "$out" 2>/dev/null); cost=$(jq -r '.total_cost_usd // 0' "$out" 2>/dev/null); cost=${cost:-0}; rm -f "$out"
  [ -n "$text" ] || { echo "이사회 세션이 답을 내지 못함"; exit 1; }
fi

# 제안 목록: 마지막 줄의 JSON 이 있으면 그것을, 없으면 메모의 "N. `동사 ...` — 이유" 줄에서 뽑는다 (모델이 JSON 을 빼먹어도 제안은 남는다)
js=$(python3 "$HERE/board-parse.py" <<<"$text") || js=""
[ -n "$js" ] || js='{"constraint":"","proposals":[],"not_doing":[],"role_advice":[]}'
printf '%s\n' "$text" | grep -v '^{"constraint"' > "$MD"
jq -n --arg w "$WEEK" --arg ts "$(date -Iseconds)" --argjson c "$cost" --argjson body "$js" '{week:$w, generated:$ts, cost_usd:$c} + $body' > "$JS"
[ "$MODE" = --reparse ] || jq -cn --arg ts "$(date -Iseconds)" --arg d "$(date +%F)" --argjson c "$cost" '{ts:$ts,date:$d,project:"(board)",phase:"board",run_id:"",campaign:"",subtype:"success",cost_usd:$c}' >> "$DATA/usage.jsonl"
n=$(jq -r '.proposals|length' "$JS")
echo "이사회 $WEEK: 제안 $n건 (\$$cost) → $MD"
"$HERE/tg.sh" "🏛 이사회 $WEEK — 이번 주의 제약: $(jq -r '.constraint' "$JS" | head -c 200)
$(jq -r '.proposals[] | "\(.n). \(.cmd|join(" ")) — \(.why|.[0:110])"' "$JS")
하지 않는 것: $(jq -r '.not_doing|join(" · ")' "$JS" | head -c 300)

적용하려면 답장: board apply 1 3   (전체: board apply all · 메모: board)" >/dev/null 2>&1 || true
( cd "$REPO_DIR" && flock -w 120 9 && git add state/board && git -c user.name=hkjang -c user.email=gagagiga@naver.com commit -qm "board: $WEEK 이사회 — 제안 $n건" && git pull -q --rebase origin main && git push -q origin main ) 9>"$HOME/.auto-improve/sync.lock" >/dev/null 2>&1 || true
exit 0
