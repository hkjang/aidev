# 회차 노트 2026-10-05-100726-appstore-improve — appstore
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:07] base pinned — main@e9aee8a
- [러너 10:07] autonomy release — 

## 정찰 노트
- `/my/apps/:id/edit` 의 not-found 처리를 골랐다 — 코드로 결함을 확인했고(queryFn 의 undefined 반환 → ErrorState 가 `error.message` 를 그대로 노출), 프로덕션 파일 1개로 끝나며 영문 내부 문구가 한국어 UI 에 보이는 눈에 보이는 피해다. 차선인 `/favorites` 빈 문구는 review-pending 인 `0898f7a` 와 같은 `public-pages.tsx` 라 미뤘고, aria-invalid·rate limit·Page Size 는 설계나 공개 API 계약이 선행돼 한 세션에 안 끝난다.
- 확신 없는 곳: TanStack Query v5 가 `undefined` 반환을 오류로 바꾸는지 **실행해 확인하지 못했다**. `web/node_modules` 가 없고 `npm ci` 가 권한 승인에 막혔다(프로필의 기존 함정 재발). 과제서에 미확인으로 적고, 관측된 실제 증상이 다르면 그것을 커밋에 적으라고 지시해 두었다.
- 구현자가 조심할 것: not-found 분기는 `isPending`·`error` 분기 **뒤**에 둘 것(로딩 중 깜빡임). 401/500 같은 실제 HTTP 오류는 지금처럼 `ErrorState` 여야 한다 — "없음" 과 "오류" 를 섞지 말 것.
- `app-form-page.test.tsx` 는 존재하지 않는다(ls 확인) — 이 과제가 그 파일의 첫 테스트가 되므로 하네스는 `personal-pages.test.tsx` 것을 따라 쓰고 손으로 만든 대역은 쓰지 말 것. 프로필은 2일 전이고 코드와 어긋나는 점이 없어 다시 쓰지 않았다(HEAD 표기만 `e9aee8a` 로 낡음).
- [러너 10:10] scout done — `/my/apps/:id/edit` 에서 앱을 못 찾으면 영문 라이브러리 오류가 한국어 UI 에 그대로 보인다 (가치 3 / 위험 1 /

## 구현 노트
- `/my/apps/:id/edit` 에서 내 앱 목록에 그 id 가 없을 때 `queryFn` 이 `undefined` 를 반환해 TanStack Query v5 가 오류로 바꾸고 `ErrorState` 가 그 문구를 그대로 노출했다. `?? null` + `isPending`·`error` **뒤**의 not-found 분기(`EmptyState` + `/my/apps` 링크)로 고쳤다. 프로덕션 1파일, 커밋 `598c188`.
- 과제서 전제를 실행해 확인했는데 화면에 실제로 나온 문구는 `Query data cannot be undefined…` 가 아니라 ``["my-app","<id>"] data is undefined`` 였다(전문은 stderr 로만). 수정안은 그대로 성립하며 커밋 메시지에 관측값을 적었다.
- 확신 없는 곳: 저장 성공 후 `complete` 화면이 새 분기보다 **뒤**에 있다 — 수정 모드에서 `existing.data` 는 저장 후에도 캐시에 남으므로(`["my-apps"]` 만 invalidate) 가로채지 않는 것을 코드로 확인했지만 저장 플로우를 테스트로 직접 돌려 보지는 않았다. 비평가가 여기를 먼저 보면 좋겠다.
- 일부러 하지 않은 것: `ui.tsx` 공용 기본값(bare `<EmptyState />` 공유 화면 다수)과 `:81`·`:172` 의 `existing.data` 가드는 `null` 에도 그대로 동작하므로 손대지 않았다. E2E 는 공개 화면 변화가 없어 미실행.
- 테스트 하네스 주의: `app-form-page.test.tsx` 는 프로덕션 라우트 2개(`/my/apps/:id/edit`·`/my/apps`)를 마운트하고 `fetch` 만 스텁한다. DB 불필요. `/my/apps` 의 실제 제목은 "내가 등록한 앱" 이다("내 앱" 아님 — 처음 이걸로 썼다가 고쳤다).
- 검증: Vitest 103건(기준선 99 직접 실행 확인)·lint·prettier·build·check-offline-assets·check-env-contract·check-docs·`go build ./cmd/server` 모두 exit 0. 프로덕션 파일만 되돌려 같은 2건 재실패 확인. `server` 산출물 삭제, 트리 깨끗.
- [러너 10:18] brief accepted — 채택 — 지목한 `app-form-page.tsx:71-75`·`:128-141`·`:12-22` 와 `ui.tsx` 의 `EmptyState`/`ButtonLink`/`ErrorState:292`, `app-form-page.test.tsx` 부�
- [러너 10:19] verify passed — 검증 8개 통과 (auto)

## 비평 노트
- 승인. 테스트 유효성을 직접 재현했다 — 프로덕션 파일만 main 판으로 되돌리니 not-found 2건이 `Unable to find … "앱을 찾을 수 없습니다"` + 화면의 `["my-app","dddd…"] data is undefined` 로 실패하고, 나머지 2건은 전후 모두 통과하는 회귀 가드다. 커밋이 적은 관측 문구와 실패 출력이 일치한다.
- 구현자가 의심한 자리(저장 성공 후 `complete` 가 새 분기에 가로채이는지)를 임시 probe 로 실행해 확인했다 — 안전하다. `invalidateQueries(["my-apps"])` 가 prefix 매칭에서 `["my-app", id]` 와 안 맞아 캐시가 남기 때문. probe 삭제, 트리 깨끗.
- 다음 회차가 알 것: 그 안전성은 분기 순서에 의존한다. `app-form-page.tsx:147`(not-found)이 `:157`(complete)보다 앞이라, 나중에 저장 시 `["my-app"]` 를 무효화하면 성공 화면이 "앱을 찾을 수 없습니다" 로 바뀐다. 지금은 정상이라 차단 안 함.
- 선행 결함(이번 변경이 만든 것 아님): `personal_handlers.go:30` 의 `pagination(r, 50, 100)` 탓에 앱 50개 초과 소유자가 첫 페이지 밖 앱의 edit URL 로 직접 들어가면 새 문구가 사실과 다르게 뜬다. UI 경로로는 도달 불가(목록도 50건). 공개 API 계약이라 함께 안 고친 판단이 옳다.
- 못 본 것: E2E 미실행(로그인 필요 라우트, 공개 화면 변화 없음)·Go 테스트 생략(Go 변경 0). 보안·법무 차단 없음 — `/me/apps` 가 서버에서 소유자 범위라 새 IDOR 없고, 문구가 "없음"/"내 것 아님" 을 구분하지 않아 열거 불가. 직접 돌린 게이트: vitest 103/103·tsc·lint·build·check-offline-assets 전부 exit 0.
- [러너 10:27] review approved — 리뷰 승인 (risk=low)
- [러너 10:28] pr created — https://github.com/hkjang/appstore/pull/38
- [러너 10:33] ci passed — 검사 2개 모두 success
- [러너 10:33] merge done — 3cbef71
