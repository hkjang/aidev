#!/usr/bin/env bash
# 자연어 한 문장 → 캠페인 초안.
#
# 오늘까지 캠페인 하나를 여는 데 사람이 한 일: 서른 저장소를 훑어 그 기능이 이미 있는 곳과 없는 곳을
# 가르고, 잘 된 곳을 참조 구현으로 고르고, 표준 문서를 쓰고, 대상·예산·기한·보호 경로를 정해
# campaigns.json 에 넣었다 (2026-09-17 MCP OAuth 캠페인이 그랬다). 이 스크립트는 그 일을 한 세션이
# 대신 하게 한다. 결과는 초안(drafts/<slug>/)이고, 시작은 사람이 한다 (bin/ops.sh activate-campaign).
#
#   bin/campaign-draft.sh "<목표 문장>" [slug]
# 산출물: drafts/<slug>/STANDARD.md, campaign.json, summary.md — 끝나면 텔레그램으로 요약.
set -uo pipefail
export HOME="${HOME:-/home/hkjang}"
export PATH="$HOME/.local/bin:$HOME/.nvm/versions/node/v22.23.1/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin"
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
REPO_DIR="$(cd "$HERE/.." && pwd)"; STATE="$REPO_DIR/state"; DATA="$REPO_DIR/docs/data"
ROOT="${ROOT:-/mnt/c/Users/USER/projects}"
goal=${1:?목표 문장}; slug=${2:-$(date +%Y%m)-$(printf '%s' "$goal" | md5sum | cut -c1-6)}
D="$REPO_DIR/drafts/$slug"; mkdir -p "$D"
MODEL="${MODEL:-claude-opus-5}"; BUDGET="${AIDEV_DRAFT_BUDGET:-8}"
echo "===== $(date '+%F %T') draft $slug: $goal"
# 러너 대상 프로젝트 = state/<이름>.md 가 있고 폴더가 있는 것
targets=$(for f in "$STATE"/*.md; do n=$(basename "$f" .md); [ -d "$ROOT/$n/.git" ] && echo "$n"; done | sort | tr '\n' ' ')
until=$(date -d '+90 days' +%F)
prompt="당신은 aidev(여러 저장소를 돌며 같은 기능을 넣는 자율 개선 러너)의 캠페인 설계자입니다. 운영자가 한 문장으로 목표를 줬습니다. 저장소들을 실제로 훑어 캠페인 초안을 만드세요. 사람이 지켜보지 않으니 질문하지 말고 스스로 판단해 끝내세요.

## 목표 (운영자의 말)
$goal

## 러너 대상 저장소 (이 안에서만 고른다)
$targets
각 저장소는 $ROOT/<이름> 에 있다. 대부분 Go 백엔드 + React SPA 이고, 일부는 Node/Python/Java 다.

## 해야 할 일
1. **조사** — 대상 저장소를 grep/ls 로 훑어 (a) 이 목표가 이미 구현된 곳, (b) 부분 구현, (c) 없는 곳, (d) 목표가 성립하지 않는 곳(예: 화면이 없는 CLI 에 UI 기능)을 가른다. 코드를 실제로 열어 확인한 것만 적는다.
2. **참조 구현 선정** — (a) 중 가장 깔끔한 곳 하나(또는 둘)를 참조 구현으로 고르고 핵심 파일 경로를 적는다. 하나도 없으면 참조 구현 없이 표준만 쓴다고 명시한다.
3. **표준 문서** — $REPO_DIR/MAIL-STANDARD.md 와 $REPO_DIR/MCP-OAUTH-STANDARD.md 를 읽고 그 형식·문체(한국어, 왜 이렇게 하는지, 참조 구현, 지켜야 할 것, 설정 키 표, 하지 말아야 할 것, 테스트, 검증 체크리스트)로 $D/STANDARD.md 를 쓴다. 설정 키 이름은 참조 구현 것을 그대로 쓴다. 기본값은 꺼짐이어야 하고, 새로 설치한 곳에서 아무것도 달라지지 않아야 한다는 원칙을 지킨다.
4. **캠페인 정의** — $D/campaign.json 을 쓴다:
   {\"id\":\"<slug>-$(date +%Y-%m)\" 형태의 짧은 영문 id, \"standard_file\":\"<대문자-슬러그>-STANDARD.md\", \"goal\":\"<회차 프롬프트에 그대로 들어갈 지시문>\", \"projects\":[...(c)와 (b)만...], \"budget_usd\":<대상 수 × improve_budget_usd × 1.3 을 10 단위로 올림>, \"improve_budget_usd\":<12~18, 일감 크기에 따라>, \"until\":\"$until\", \"expected_guard\":[<이 일이 건드릴 보호 경로 조각들: auth, oidc, session, migrations, mcp, oauth 중 해당하는 것>], \"reference\":{\"project\":\"...\",\"files\":[...]}, \"already_done\":[...(a)...], \"excluded\":{\"<이름>\":\"<이유>\"}}
   goal 지시문은 $STATE/campaigns.json 의 기존 goal 들과 같은 문체로: 첫 문단에 무엇을 왜, 그다음 '표준은 aidev 저장소의 <standard_file> 에 있다. 먼저 읽고 그대로 따른다: $REPO_DIR/<standard_file>', 참조 구현 경로, 지켜야 할 핵심 3~5가지, 성립하지 않는 저장소는 원장에 적고 변경 없이 끝내라는 문장, 관리자 가이드 갱신 문장.
5. **요약** — $D/summary.md 에 6~10줄: 이미 된 곳, 참조 구현, 대상 수와 예산, 뺀 곳과 이유, 위험(보호 경로), 운영자가 확인할 것.

## 절대 규칙
- 파일은 $D/ 아래에만 씁니다. 저장소 코드·aidev 의 다른 파일을 바꾸지 마세요.
- 확인하지 않은 것을 '있다/없다' 고 쓰지 마세요. 못 본 저장소는 '미확인' 으로 적고 대상에서 뺍니다.
- 비밀값·토큰을 어디에도 쓰지 마세요."
out=$(cd "$REPO_DIR" && timeout -k 30 1500 claude -p "$prompt" --model "$MODEL" --settings '{"attribution":{"commit":"","pr":""}}' \
      --permission-mode acceptEdits --allowedTools "Read Grep Glob Write Bash(ls:*) Bash(find:*) Bash(wc:*) Bash(head:*) Bash(cat:*) Bash(git log:*) Bash(git -C:*)" \
      --add-dir "$ROOT" --max-budget-usd "$BUDGET" --output-format json 2>>"$REPO_DIR/logs/campaign-draft.err")
cost=$(jq -r '.total_cost_usd // 0' <<<"$out" 2>/dev/null); cost=${cost:-0}
jq -cn --arg ts "$(date -Iseconds)" --arg d "$(date +%F)" --arg s "$slug" --argjson c "$cost" \
  '{ts:$ts,date:$d,project:"(campaign)",phase:"campaign-draft",run_id:$s,campaign:"",subtype:"success",cost_usd:$c}' >> "$DATA/usage.jsonl"
if jq -e '.id and (.projects|length>0) and .until and .goal' "$D/campaign.json" >/dev/null 2>&1 && [ -s "$D/STANDARD.md" ]; then
  echo "draft $slug 완료 (\$$cost): $(jq -r '"\(.id) · 대상 \(.projects|length) · $\(.budget_usd)"' "$D/campaign.json")"
  "$HERE/tg.sh" "📝 캠페인 초안 준비됨 — \"$(cut -c1-60 <<<"$goal")\"
$(jq -r '"id \(.id) · 대상 \(.projects|length)개 · $\(.budget_usd) · 기한 \(.until)\n참조 구현: \(.reference.project // "없음")\n이미 된 곳: \((.already_done // [])|join(", "))\n뺀 곳: \((.excluded // {})|to_entries|map("\(.key)(\(.value|.[0:30]))")|join(", "))"' "$D/campaign.json")
$(head -c 700 "$D/summary.md" 2>/dev/null)

시작하려면 답장: activate-campaign $slug
버리려면: discard-draft $slug
초안: https://github.com/hkjang/aidev/tree/main/drafts/$slug" >/dev/null 2>&1 || true
  ( cd "$REPO_DIR" && flock -w 120 9 && git add drafts && git -c user.name=hkjang -c user.email=gagagiga@naver.com commit -qm "draft: 캠페인 초안 $slug — $(cut -c1-50 <<<"$goal")" && git pull -q --rebase origin main && git push -q origin main ) 9>"$HOME/.auto-improve/sync.lock" >/dev/null 2>&1 || true
else
  echo "draft $slug 실패 (\$$cost): 산출물이 불완전함"; ls -la "$D" 2>/dev/null
  "$HERE/tg.sh" "📝 캠페인 초안 실패 — \"$(cut -c1-60 <<<"$goal")\": 세션이 완전한 산출물을 내지 못했습니다 (logs/campaign-draft.log)" >/dev/null 2>&1 || true
fi
echo "===== $(date '+%F %T') draft $slug done"
