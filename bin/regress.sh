#!/usr/bin/env bash
# 회귀 감시 — 최근 24시간에 머지된 회차마다 "그 머지가 main 을 깨뜨렸나"를 GitHub 에서 확인해
# state/lessons.jsonl 에 교훈으로 남긴다. 다음 회차 프롬프트($LESSONS)와 대시보드 /lessons/ 가 읽는다.
#   회귀 판정: 머지 커밋의 check-run 에 failure 가 있음 / 24시간 안에 그 PR 을 되돌린 커밋이 있음
# 한 번 판정한 PR 은 state/.regress-seen 에 적어 다시 보지 않는다 (판정은 머지 2시간 뒤부터, CI 가 끝난 뒤).
set -uo pipefail
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
REPO_DIR="$(cd "$HERE/.." && pwd)"
ROOT="${ROOT:-/mnt/c/Users/USER/projects}"
STATE="$REPO_DIR/state"; RUNS="$REPO_DIR/docs/data/runs.jsonl"; LESSONS="$STATE/lessons.jsonl"; SEEN="$STATE/.regress-seen"
touch "$SEEN" "$LESSONS"
[ -f "$RUNS" ] || exit 0
now=$(date +%s)

lesson(){ # $1=project $2=kind $3=pr $4=detail
  jq -cn --arg ts "$(date -Iseconds)" --arg d "$(date +%F)" --arg p "$1" --arg k "$2" --arg pr "$3" --arg detail "$4" \
     '{ts:$ts,date:$d,project:$p,kind:$k,pr:$pr,detail:$detail}' >> "$LESSONS"
  echo "regress: $1 $2 — $4"
}

jq -r 'select(.result|test("merged https://github.com/[^/]+/[^/]+/pull/[0-9]+")) | "\(.ts)\t\(.project)\t\(.result|capture("merged (?<u>https://github.com/[^/]+/[^/]+/pull/[0-9]+)").u)"' "$RUNS" 2>/dev/null \
| while IFS=$'\t' read -r ts p url; do
  t=$(date -d "$ts" +%s 2>/dev/null || echo 0)
  age=$(( now - t ))
  [ "$age" -gt 7200 ] && [ "$age" -lt 172800 ] || continue     # 2시간 ~ 48시간 사이만 본다
  grep -qx "$url" "$SEEN" && continue
  repo=$(sed -E 's#https://github.com/([^/]+/[^/]+)/pull/.*#\1#' <<<"$url"); num=${url##*/}
  read -r sha merged_at < <(gh pr view "$num" -R "$repo" --json mergeCommit,mergedAt --jq '"\(.mergeCommit.oid // "") \(.mergedAt // "")"' 2>/dev/null || echo " ")
  [ -n "$sha" ] || { echo "$url" >> "$SEEN"; continue; }
  # 1) 머지 커밋의 CI
  read -r total pending failed < <(gh api "repos/$repo/commits/$sha/check-runs" \
     --jq '[.check_runs|length, ([.check_runs[]|select(.status!="completed")]|length), ([.check_runs[]|select(.conclusion=="failure")]|length)] | @tsv' 2>/dev/null || echo "0 0 0")
  [ "${pending:-0}" -gt 0 ] && continue                          # 아직 도는 중이면 다음에
  if [ "${failed:-0}" -gt 0 ]; then
    names=$(gh api "repos/$repo/commits/$sha/check-runs" --jq '[.check_runs[]|select(.conclusion=="failure")|.name]|join(", ")' 2>/dev/null)
    title=$(gh pr view "$num" -R "$repo" --json title --jq .title 2>/dev/null)
    lesson "$p" "ci-broken-after-merge" "$url" "PR #$num \"$title\" 머지 뒤 main CI 실패: ${names:-?}. 이런 유형의 변경은 머지 전에 해당 검사를 로컬에서 재현해야 한다."
  fi
  # 2) 되돌림
  rev=$(gh api "repos/$repo/commits?since=$merged_at&per_page=50" --jq "[.[] | select(.commit.message | test(\"(?i)revert\") and test(\"#$num|${sha:0:7}\"))] | .[0].sha // empty" 2>/dev/null)
  if [ -n "$rev" ]; then
    title=$(gh pr view "$num" -R "$repo" --json title --jq .title 2>/dev/null)
    lesson "$p" "reverted" "$url" "PR #$num \"$title\" 가 머지 뒤 되돌려짐(${rev:0:7}). 같은 접근은 다시 시도하지 말 것."
  fi
  echo "$url" >> "$SEEN"
done
exit 0
