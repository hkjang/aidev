# 회차 노트 2026-10-15-150834-review.yeopjari.bid-improve — review.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:08] base pinned — main@6ea982c
- [러너 15:08] autonomy release — 
- [러너 15:14] scout done — PR #7 의 CI 실패 원인을 저장소 안에서 끝까지 배제한 뒤, 배제되면 **CI 의 `perf` 단계가 여유 6.3KB 로 터지기

## 구현 노트
- 바꾼 것: `scripts/perf.mjs` 한 파일. `WARN_AT_PCT = 95` 를 두어 첫 방문 JS 가 예산의 95% 이상이면 현재·예산·백분율·남은 KB 를 담은 한 줄을 찍고 **exit 0 을 유지**한다. 지금 코드에서 실제로 `near budget: 223.7 KB of 230 KB (97.3%), only 6.3 KB left` 가 찍힌다. 실패 기준(`let failed = total > BUDGET_KB`, 기본 `230`)은 숫자 하나도 안 건드렸다 — `git diff` 에 그 두 줄의 `-` 가 없다.
- 우선 과제: PR #7 의 CI 실패는 저장소 밖이다. PR #7 내용을 main 위에 올려 `npm run check` 와 `TZ=UTC npm run check` 둘 다 `6 files / 187 tests` 통과, build + perf exit 0 — 과제서 기대치와 전부 일치. 중복 PR 금지라 그 파일은 되돌렸고(`118 passed` = main 기준선) 워크플로는 무수정이다.
- 확신 없는 곳: (1) 경고 문구를 **영어**로 썼다 — `perf.mjs` 자체 출력이 전부 영어라 파일 내 일관성을 택했지만, `a11y.mjs`·`e2e.mjs` 의 운영자 노출 verdict 줄은 한국어다. 저장소 관례를 한국어로 읽는다면 여기만 바꾸면 된다(동작 무관). (2) 예산 **초과** 시 메시지를 한 줄 더 붙였다 — 과제서에 없던 추가다(그전엔 아무 말 없이 exit 1 만 했다). 실패 기준은 불변이지만 범위를 넘는다고 보면 그 `if (failed)` 분기만 떼어도 수용 기준 1~4 는 그대로 충족된다.
- 일부러 안 한 것: 번들을 실제로 줄이는 일(여유 6.3KB 는 그대로 — ideas.json 에 `[3/2/M]` 신규로 적었다). `WARN_AT_PCT` 을 env 로 열지 않았다(설정 표면을 늘리지 않으려고). 차선 후보 `slugify` 는 본 과제가 성립해서 손대지 않았다.
- 다음 역할이 조심할 것: 검증은 전부 이 머신에서 실제 실행했다 — `npm run check`(118/6), `node scripts/perf.mjs`(223.7KB exit 0), `node scripts/e2e.mjs` **218 passed / 0 failed**(`dev-reset.sh` 직후, 기준선과 동일). e2e 는 `127.0.0.1:55443` PostgreSQL 이 있어야 돈다. 실패 경로는 env(`INITIAL_JS_BUDGET_KB=100`)와 리터럴 `?? 100` 두 방식 모두로 exit 1 을 확인하고 즉시 되돌렸다 — 커밋된 diff 에 `230` 그대로 있다. 95% 경계도 env 양쪽으로 찍었다(235→경고, 236→침묵). `perf.mjs` 는 어느 열린 PR 과도 겹치지 않는다(네 브랜치 `git diff --name-only` 로 직접 확인).
- [러너 15:20] brief accepted — 채택 — 0단계 기대 수치(187 tests / 6 files, 223.7KB / 230KB, 237+/1-)가 실행 관측과 전부 일치했고, 분기 B 판정도 그대로 성립했�
- [러너 15:20] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인한 것: `git diff main...HEAD` 는 `scripts/perf.mjs` 10+/1- 한 파일뿐. 실패 기준 두 줄에 변경 없음을 diff 로 확인하고, `node scripts/perf.mjs` 를 230/235/236/100 네 예산으로 직접 돌려 97.3% 경고+exit 0 / 95.2% 경고 / 침묵 / over budget+exit 1 을 모두 재현했다. `npm run check` 6 files · 118 tests 통과, `eslint scripts/perf.mjs` 통과 — 구현 노트 수치와 전부 일치.
- 못 본 것: e2e(218 passed)·ui-flows·a11y 는 재실행하지 않았다. 변경이 빌드 산출물 크기를 읽어 로그만 찍고 exit code 를 바꾸지 않으므로 런타임·UI 경로와 무관하다고 판단했다.
- 판정: approve / risk low / blocking 없음. 보안·법무 접점 없음(개인정보·비밀값·인증·권한 미접촉, 신뢰할 수 없는 입력 경로 없음).
- 남는 우려(릴리즈 노트·다음 회차): ① 경고는 GitHub Actions 에서 성공한 단계의 접힌 로그에만 남아 "터지기 전에 보인다" 가 약하게만 달성된다 — 다음 회차에 `::warning::`/step summary 로 올릴 가치가 있다(저장소에 그 관례가 아직 없다). ② README.md:102·engineering/architecture.md:70 이 여전히 실패 조건만 적어 95% 경고가 문서에 없고, 같은 줄의 218.8KB 는 실측 223.7KB 와 어긋난다(묵은 값). ③ 여유 6.3KB 는 그대로이므로 번들 축소 과제(ideas.json `[3/2/M]`)가 다음 적자다.
- [러너 15:22] review approved — 리뷰 승인 (risk=low)
- [러너 15:22] pr created — https://github.com/hkjang/review.yeopjari.bid/pull/8
- [러너 15:23] ci failed — 성공이 아닌 검사: check=failure · 실패한 검사: ? 잡: check 
