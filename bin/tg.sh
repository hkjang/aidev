#!/usr/bin/env bash
# 텔레그램으로 한 줄 알린다.  bin/tg.sh "본문"  또는  echo 본문 | bin/tg.sh
#
# 토큰과 대화 id 는 ~/.auto-improve/notify.env 에 있다(저장소에 두지 않는다).
# 둘 중 하나라도 비어 있으면 조용히 넘어간다 — 알림이 없다고 회차가 실패하면 안 된다.
#
# 회차는 단계마다 이 스크립트를 부르므로 실패에 관대해야 한다: 타임아웃을 짧게 잡고,
# 텔레그램이 죽어 있어도 러너를 붙잡지 않는다.
set -uo pipefail
[ -f "$HOME/.auto-improve/notify.env" ] && { set -a; . "$HOME/.auto-improve/notify.env"; set +a; }
: "${AIDEV_TELEGRAM_TOKEN:=}"; : "${AIDEV_TELEGRAM_CHAT:=}"
[ -n "$AIDEV_TELEGRAM_TOKEN" ] && [ -n "$AIDEV_TELEGRAM_CHAT" ] || exit 0

text="${1:-}"; [ -n "$text" ] || text=$(cat)
[ -n "$text" ] || exit 0
# 텔레그램 한 통은 4096자까지다. 넘치면 잘라 보내되 잘렸다고 밝힌다.
[ "${#text}" -gt 3900 ] && text="${text:0:3900}
…(잘림)"

curl -s -m 10 -o /dev/null -X POST \
  "https://api.telegram.org/bot${AIDEV_TELEGRAM_TOKEN}/sendMessage" \
  --data-urlencode "chat_id=${AIDEV_TELEGRAM_CHAT}" \
  --data-urlencode "text=${text}" \
  --data-urlencode "disable_web_page_preview=true" || true
exit 0
