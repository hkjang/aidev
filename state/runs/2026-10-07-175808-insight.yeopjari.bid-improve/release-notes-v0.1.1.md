일·월 단위 쓰기 제한이 알려 주는 재시도 시각을 한국 자정에 맞췄습니다.

## 고친 것

- `quota.ts:retryAfterSeconds` 가 다음 창 전환 시각을 `windowStart()`(KST 날짜의 **UTC** 자정 라벨)로 계산해,
  `day`·`month` 제한에 걸린 사람에게 실제보다 9시간 긴 재시도 시각을 알려 주고 있었습니다. 전환 시각을
  `kstDayStart(at) + 1일` / `Date.UTC(kstY, kstM, 1) - 9h` 로 직접 계산하도록 바꿨습니다.
- `day` 제한은 인증 코드·연구 신청·신고·메일 발송 상한에 함께 쓰여, `retry-after` 헤더를 받는 모든 곳이 한 번에
  바로잡힙니다.
- `windowStart()` 본문은 한 줄도 건드리지 않았습니다 — 그 값이 `insight_priv.quota_counters` 의 유일 키라서,
  바꾸면 운영 중인 모든 제한이 한 번 리셋됩니다. 테스트가 부를 수 있도록 `export` 만 붙였습니다.
- `rules.test.ts` 에 `describe('write limits')` 4개를 더해 day 2건·month 2건(10→11월, 12→1월) 고정값,
  `minute` 분기 불변, `windowStart()` 의 ISO 라벨 4개를 못 박았습니다.

## 검증

- `npm run check` — lint, 타입 검사 3개(core·server·web), 단위 테스트 44개 통과
- `npm run build` — 웹·서버 번들 통과
- `VERSION` 과 `wrangler.toml` 의 `INSIGHT_VERSION` 일치 — `scripts/deploy-pages.sh` 가 배포 전에 보는 검사
- e2e 와 브라우저 화면 검증은 실 PostgreSQL·Chromium 이 필요해 이 기계에서 돌리지 않았습니다. push 시
  `check` 워크플로가 PostgreSQL 16 을 띄워 전체를 돌립니다.

## 배포

운영 배포는 수동입니다: `bash scripts/deploy-pages.sh`
