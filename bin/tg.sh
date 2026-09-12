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

# 같은 문장을 되풀이해 보내지 않는다.
#
# 승인 스윕은 10분마다 돌면서 아직 머지되지 못한 PR 을 다시 확인한다. 아무것도
# 달라지지 않았으므로 러너는 그것을 회차로 기록하지 않는데, 알림은 매번 나갔다.
# CI 워크플로가 없는 저장소 하나가 같은 문장을 12번 보냈다 (2026-09-12 Kkiit).
#
# 상황이 이어지는 동안 사람이 알아야 할 것은 "그렇다" 한 번이지 열두 번이 아니다.
# 창을 넘기면 다시 보낸다 — 아직 안 풀렸다는 사실 자체는 다시 알릴 값이 있다.
QUIET_SECONDS=${AIDEV_TG_QUIET_SECONDS:-21600}
seen_dir="$HOME/.auto-improve/.tg-seen"
mkdir -p "$seen_dir" 2>/dev/null
key=$(printf '%s' "$text" | md5sum | cut -c1-16)
stamp="$seen_dir/$key"
if [ -f "$stamp" ]; then
  age=$(( $(date +%s) - $(stat -c %Y "$stamp" 2>/dev/null || echo 0) ))
  [ "$age" -lt "$QUIET_SECONDS" ] && exit 0
fi
: > "$stamp"
# 오래된 기록은 치운다 — 매 단계마다 파일이 하나씩 생긴다.
find "$seen_dir" -type f -mmin +1440 -delete 2>/dev/null || true
# 텔레그램 한 통은 4096자까지다. 넘치면 잘라 보내되 잘렸다고 밝힌다.
[ "${#text}" -gt 3900 ] && text="${text:0:3900}
…(잘림)"

curl -s -m 10 -o /dev/null -X POST \
  "https://api.telegram.org/bot${AIDEV_TELEGRAM_TOKEN}/sendMessage" \
  --data-urlencode "chat_id=${AIDEV_TELEGRAM_CHAT}" \
  --data-urlencode "text=${text}" \
  --data-urlencode "disable_web_page_preview=true" || true
exit 0
