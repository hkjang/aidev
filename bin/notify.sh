#!/usr/bin/env bash
# 주의 필요 알림 — docs/data/summary.json 의 alerts 중 처음 보는 것이 있으면
#   1) GitHub Issue(라벨 alert, 열린 것이 있으면 코멘트)  2) Slack 웹훅  3) 이메일(curl SMTP)  4) Windows 토스트
# 로 알린다. 2)·3) 은 ~/.auto-improve/notify.env 가 있어야 한다 (state/notify.env.example 참조).
# 해소된 경고는 seen 목록에서 빠지므로 재발하면 다시 알린다.
set -uo pipefail
HERE="${AIDEV_BIN:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
REPO_DIR="$(cd "$HERE/.." && pwd)"
STATE="$REPO_DIR/state"; SUMMARY="$REPO_DIR/docs/data/summary.json"; SEEN="$STATE/.alerts-seen.json"
SITE="https://hkjang.github.io/aidev"
[ -f "$SUMMARY" ] || exit 0
[ -f "$SEEN" ] || echo '[]' > "$SEEN"
[ -f "$HOME/.auto-improve/notify.env" ] && { set -a; . "$HOME/.auto-improve/notify.env"; set +a; }

mapfile -t new < <(jq -r --slurpfile seen "$SEEN" \
  '.alerts[] | "\(.project)|\(.why)" | select(. as $k | ($seen[0] | index($k)) == null)' "$SUMMARY" 2>/dev/null)
# 현재 경고 전체를 seen 으로 교체 (해소된 건 빠진다)
jq '[.alerts[] | "\(.project)|\(.why)"]' "$SUMMARY" > "$SEEN.tmp" 2>/dev/null && mv "$SEEN.tmp" "$SEEN"
[ ${#new[@]} -gt 0 ] || { echo "notify: no new alerts"; exit 0; }

total=$(jq -r '.alerts|length' "$SUMMARY")
when=$(date '+%Y-%m-%d %H:%M')
lines=$(printf -- '- %s\n' "${new[@]}" | sed 's/|/ — /')
text="⚠️ aidev 주의 필요 — 새 경고 ${#new[@]}건 (전체 ${total}건, $when KST)
$lines
대시보드: $SITE/"
echo "notify: ${#new[@]} new alert(s)"

# 1) GitHub Issue — 알림 겸 추적. (같은 계정의 활동은 GitHub 이 이메일로 알리지 않는다)
(
  cd "$REPO_DIR" || exit 0
  gh label create alert --color D93F0B --description "자율 개선 주의 필요" >/dev/null 2>&1 || true
  num=$(gh issue list --label alert --state open --json number --jq '.[0].number' 2>/dev/null)
  body="$lines

[대시보드]($SITE/) · [summary.json]($SITE/data/summary.json) · $when KST"
  if [ -n "$num" ]; then
    gh issue comment "$num" --body "### 새 경고 ${#new[@]}건
$body" >/dev/null 2>&1 && echo "notify: commented on issue #$num"
  else
    gh issue create --title "⚠️ 자율 개선 주의 필요 ($(date +%Y-%m-%d))" --label alert --body "$body" >/dev/null 2>&1 && echo "notify: issue created"
  fi
)

# 2) Slack
if [ -n "${AIDEV_SLACK_WEBHOOK:-}" ]; then
  curl -s -m 15 -X POST -H 'Content-type: application/json' \
    --data "$(jq -cn --arg t "$text" '{text:$t}')" "$AIDEV_SLACK_WEBHOOK" >/dev/null && echo "notify: slack sent"
fi

# 3) 이메일 — curl 의 SMTP 로 보낸다 (예: AIDEV_SMTP_URL=smtps://smtp.naver.com:465)
if [ -n "${AIDEV_SMTP_URL:-}" ] && [ -n "${AIDEV_MAIL_TO:-}" ]; then
  from="${AIDEV_MAIL_FROM:-$AIDEV_SMTP_USER}"
  subj="=?UTF-8?B?$(printf '%s' "[aidev] 주의 필요 ${#new[@]}건 ($when)" | base64 -w0)?="
  mail=$(printf 'From: %s\nTo: %s\nSubject: %s\nMIME-Version: 1.0\nContent-Type: text/plain; charset=UTF-8\n\n%s\n' "$from" "$AIDEV_MAIL_TO" "$subj" "$text")
  curl -s -m 30 --url "$AIDEV_SMTP_URL" --mail-from "$from" --mail-rcpt "$AIDEV_MAIL_TO" \
    --user "${AIDEV_SMTP_USER}:${AIDEV_SMTP_PASS}" -T <(printf '%s' "$mail") >/dev/null && echo "notify: mail sent to $AIDEV_MAIL_TO"
fi

# 4) Windows 토스트 (WSL 에서 powershell.exe 가 보일 때)
if command -v powershell.exe >/dev/null 2>&1; then
  msg=$(printf '%s' "${new[0]}" | sed 's/|/ — /'); [ ${#new[@]} -gt 1 ] && msg="$msg 외 $(( ${#new[@]} - 1 ))건"
  ps1=$(wslpath -w "$HERE/toast.ps1" 2>/dev/null)
  [ -n "$ps1" ] && powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$ps1" -Title "aidev 주의 필요 ${#new[@]}건" -Message "$msg" >/dev/null 2>&1 && echo "notify: toast shown"
fi
exit 0
