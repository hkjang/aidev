# 회차 노트 2026-10-16-065823-appstore-improve — appstore
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:58] base pinned — main@15f6892
- [러너 06:58] autonomy release — 

## 정찰 노트
- 가이드 문서 조회 실패 숨김을 골랐다: 직전 회차의 차선이고, 사용자에게 보이는 출력이 실제로 바뀌며(없음 → 안내+재시도), 제품 변경이 `guide-documents.tsx` 한 파일의 분기 하나라 파일 수가 작다. 즐겨찾기 검색 안내는 미병합 `0898f7a` 와 중복 위험이 남아 또 제쳤고, app-form 검증 테스트·Field aria 는 순수 테스트 반려 이력과 공용 계약 영향 때문에 제쳤다.
- 확신 없는 것: 기준선 테스트 수(대상 11 / 전체 131)는 직전 회차 기록이며 이번에 실행하지 않았다(worktree 에 node_modules 없음). Playwright `route.fallback()` 로 두 번째 호출을 mock 에 넘기는 방식은 코드로 확인하지 않은 설계 제안이다 — 안 되면 route handler 안의 카운터로 1회차 500 / 2회차 200 을 직접 fulfill 할 것.
- 구현자가 조심할 것: 같은 파일의 `useGuideDocumentDraft`/`GuideDocumentsField` 는 469e89a 가 방금 고친 자리이니 건드리지 말고, 공용 `ErrorState` 대신 화면 로컬 문구를 쓸 것. 운영 QueryClient 는 `retry:1`(providers.tsx:149) 이라 실제 화면의 오류 표시는 2회 실패 후다.
- [러너 07:02] scout done — 앱 상세의 가이드 문서 조회 실패를 "첨부 없음"처럼 숨기지 말고 안내와 다시 시도를 보여 준다 (가치 3 / 

## 구현 노트
- AppGuideDocuments 에 `documents.isError` 분기만 추가했다(guide-documents.tsx:283~). 실패가 "첨부 없음" 과 구분되지 않던 것을 카드 유지 + "가이드 문서를 불러오지 못했습니다." + `refetch()` 하는 `다시 시도` 로 바꿨다. 성공·0건의 `return null` 과 목록 마크업은 그대로다.
- 확신 없는 곳: 오류 문단에 쓴 `className="muted mt-4"` 는 styles.css:848 의 `.muted` 와 기존 `mt-4` 조합으로, 실제 시각 간격을 사람 눈으로 확인하지 않았다(테스트는 텍스트·역할만 본다). 또한 오류 상태에서 `documents.isFetching` 중 버튼을 비활성화하지 않아 연타 시 중복 요청이 가능하다 — 일부러 손대지 않았다.
- 검증 못 한 것: Go race/build, 실제 DB·Keycloak·SecCheck 통합, Docker smoke, 전체 E2E. E2E 는 실제 번들·Chromium 이지만 HTTP mock 이다.
- 일부러 하지 않은 것: 공용 `ErrorState` 미사용(제목이 "화면을 불러오지 못했습니다" 라 페이지 전체 실패로 읽히고 401/403 분기를 끌고 온다), 같은 파일의 `useGuideDocumentDraft`/`GuideDocumentsField` 미접근(469e89a 자리, 계약 다름), 서버·API 계약·types.ts 미변경.
- 다음 역할이 조심할 것: 운영 QueryClient 는 `retry:1`(providers.tsx:149) 이라 실제 화면의 안내는 2회 실패 후다 — 그래서 새 E2E 는 `page.route` 로 첫 2회만 500 을 주고 이후 `route.fallback()` 한다. 거부 횟수를 1로 줄이면 retry 가 성공해 테스트가 무의미해진다.
- 다음 역할이 조심할 것: E2E 는 `npm --prefix web run build` 선행 필수이고 `(cd web && npx playwright install chromium)` 가 필요하다. desktop/mobile 두 프로젝트 모두 확인했다(8 passed).
- [러너 07:08] brief accepted — 채택 — HEAD 15f6892 의 오류 분기 부재, `Button`/`Card` 기존 import, 테스트·E2E 삽입 자리가 모두 과제서대로여서 지정 범위와 �
- [러너 07:08] verify passed — 검증 8개 통과 (auto)

## 비평 노트
- 확인: diff 3파일 전부, 대상 Vitest 13 passed 재실행, red.log/revert-red.log 의 실패 출력이 이번 증상과 일치, query-core 리듀서(query.js:309) 로 오류 시 data 유지 확인. 못 봄: E2E 재실행, Go 빌드·테스트, 실제 서버 오류 경로.
- 판정 approve / risk low / blocking 없음. 보안·법무 소견 없음(신규 엔드포인트·식별자·개인정보·의존성 없음, 고정 문구라 오류 상세 미노출).
- 남는 우려(릴리즈 노트에 적을 것): `guide-documents.tsx:285` 의 `if (documents.isError)` 는 data 유무를 안 봐서, 목록이 떠 있던 화면에서 배경 refetch 가 실패하면 보이던 내려받기 링크가 사라지고 오류 카드로 대체된다(스크래치 테스트로 재현, 변경 전에는 stale 목록 유지). 차단 안 한 이유: public-pages.tsx:449 가 이미 같은 패턴이고 두 쿼리가 staleTime 을 공유하며 `다시 시도` 1회로 복구된다.
- 다음 회차 후보: 위 조건을 `documents.isError && !documents.data?.length` 로 좁히기. 같이 볼 것 — 오류 문단에 `role="alert"` 없음, `isFetching` 중 `다시 시도` 미비활성화(연타 시 중복 요청).
- 신규 Vitest 2건 중 'waits for the first lookup...' 은 수정 전에도 통과하는 보조 테스트다(red.log). 변경을 증명하는 것은 나머지 1건과 E2E 1건.
- [러너 07:11] review approved — 리뷰 승인 (risk=low)
- [러너 07:11] pr created — https://github.com/hkjang/appstore/pull/43
- [러너 07:17] ci passed — 검사 2개 모두 success
- [러너 07:17] merge done — 7ad0d34
- [러너 07:30] release published — v2.11.17
- [러너 07:33] assets verified — v2.11.17 자산 1개 (이전 v2.11.16: 1)
