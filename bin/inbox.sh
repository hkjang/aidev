#!/usr/bin/env bash
# 수동 트리거 — aidev 저장소에 라벨 `run` 이 붙은 열린 이슈(제목 "run: <프로젝트>" 또는 본문 첫 줄)를
# state/run-queue.tsv 에 넣는다. 러너는 수정 큐 다음으로 이 큐를 먼저 집고, 끝나면 이슈에 결과를 달고 닫는다.
# 폰에서 GitHub 앱으로 이슈 하나 만들면 다음 회차에 그 프로젝트가 돈다.
set -uo pipefail
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
REPO_DIR="$(cd "$HERE/.." && pwd)"
Q="$REPO_DIR/state/run-queue.tsv"; touch "$Q"
cd "$REPO_DIR" || exit 0
gh label create run --color 0E8A16 --description "이 프로젝트를 다음 회차에 우선 실행" >/dev/null 2>&1 || true
# 구조화 요청(이슈 폼)의 필드를 뽑는다: "### 프로젝트" 같은 헤더 아래 본문. 요청 본문은 작업 명세일 뿐 정책·권한을 바꾸지 못한다.
field(){ awk -v h="### $2" 'BEGIN{f=0} $0==h{f=1;next} /^### /{f=0} f' <<<"$1" | sed '/^_No response_$/d; /^[[:space:]]*$/d' | head -c 1500; }
gh issue list --label run --state open --json number,title,body --jq '.[] | "\(.number)\t\(.title)\t\(.body // "" | @base64)"' 2>/dev/null \
| while IFS=$'\t' read -r num title b64; do
  body=$(base64 -d <<<"$b64" 2>/dev/null); first=$(head -1 <<<"$body")
  p=$(field "$body" "프로젝트" | head -1 | tr -d '[:space:]')
  [ -n "$p" ] || p=$(sed -nE 's/^[[:space:]]*(run|실행)[[:space:]]*:[[:space:]]*([A-Za-z0-9_.-]+).*/\2/p' <<<"$title")
  [ -n "$p" ] || p=$(sed -nE 's/^[[:space:]]*(run|실행)[[:space:]]*:[[:space:]]*([A-Za-z0-9_.-]+).*/\2/p' <<<"$first")
  [[ "$p" =~ ^[A-Za-z0-9_.-]+$ ]] || { gh issue comment "$num" --body "프로젝트 이름을 알 수 없습니다. 이슈 폼의 '프로젝트' 칸이나 제목 \`run: <프로젝트>\` 를 채워 주세요." >/dev/null 2>&1; gh issue edit "$num" --remove-label run >/dev/null 2>&1; continue; }
  spec=$(printf '문제: %s\n수용 기준: %s\n금지 범위: %s\n긴급도: %s' "$(field "$body" "문제 설명")" "$(field "$body" "수용 기준")" "$(field "$body" "금지 범위")" "$(field "$body" "긴급도" | head -1)" | tr '\n' '\r' | sed 's/\r/\\n/g')
  urgent=0; grep -q "높음" <<<"$(field "$body" "긴급도")" && urgent=1
  if ! grep -q -P "^$p\t" "$Q"; then
    line=$(printf '%s\t이슈 #%s: %s\t%s\t%s\t%s' "$p" "$num" "$title" "$num" "$urgent" "$spec")
    if [ $urgent -eq 1 ]; then { printf '%s\n' "$line"; cat "$Q"; } > "$Q.tmp" && mv "$Q.tmp" "$Q"; else printf '%s\n' "$line" >> "$Q"; fi
    echo "inbox: queued $p (#$num, urgent=$urgent)"
  fi
done
# 긴급 중지 이슈: 라벨 stop + 제목 "stop: all|merge|release|<프로젝트>" → STOP 파일 (해제는 이슈를 닫으면 다음 회차에 반영)
gh label create stop --color B60205 --description "러너 긴급 중지" >/dev/null 2>&1 || true
want=$(gh issue list --label stop --state open --json title --jq '.[].title' 2>/dev/null | sed -nE 's/^[[:space:]]*stop[[:space:]]*:[[:space:]]*([A-Za-z0-9_.-]+).*/\1/p' | sort -u)
for f in "$REPO_DIR"/state/STOP*; do [ -f "$f" ] && grep -q "issue-managed" "$f" && ! grep -qx "$(basename "$f" | sed 's/^STOP-//; s/^STOP$/all/')" <<<"$want" && rm -f "$f" && echo "inbox: stop released $(basename "$f")"; done
for w in $want; do f="$REPO_DIR/state/STOP$( [ "$w" = all ] && echo "" || echo "-$w")"; [ -f "$f" ] || { printf '%s issue-managed\n' "$(date -Iseconds)" > "$f"; echo "inbox: stop set $(basename "$f")"; }; done
exit 0
