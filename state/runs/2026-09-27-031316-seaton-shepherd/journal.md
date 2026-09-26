# PR 처리기 노트 2026-09-27-031316-seaton-shepherd — seaton PR #36
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-27-023236-seaton-improve)
# 회차 노트 2026-09-27-023236-seaton-improve — seaton
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:32] base pinned — main@c0735bf
- [러너 02:32] autonomy release — 

## 정찰 노트
- 고른 이유: 좌석맵이 URL 쿼리를 읽기만 하는 것(`SeatMapPage.tsx:324` setter 없음)을 직접 확인했고, 링크를 만드는 네 화면과 09-26 의 returnTo 가 이미 그 반대편을 깔아 놓아 한 화면·두 파일로 닫힌다. rows.Err() 건은 파일 10개 안팎 + DB 테스트 하네스 부재로 또 제쳤고, '구역 불일치 문구 통합'은 두 자리를 읽어 보니 형식이 다른 별개 문장이라 rejected 로 내렸다.
- 확인한 것: 324/350/384/636/679/683-693/1402/1441-1446/1530 과 chooseMap 호출부 4곳(708/755/766/1270), `web/e2e/seed.mjs:101-124`(건물·층·도면 각 1개), 링크 생산자 4곳(MapsPage:432, AppShell:506, EmployeesPage:404, DashboardPage:168).
- 추측으로 적은 것: E2E 스펙을 어디에 넣을지(seatmap.spec.ts 추가 vs 신규 파일)와 helpers.ts 로그인 헬퍼의 정확한 이름 — 열어 보지 않았다. 도면 전환은 씨드가 도면 1개라 E2E 로 증명 못 하니 vitest 로만 증명하라고 적었다.
- 구현자가 조심할 것: 제출로 `?q=` 를 쓰면 `:683` useEffect 가 다시 깨어나 같은 검색이 두 번 나간다 — `lastSearchRef` 키를 제출 경로에서 먼저 세울 것. 필터·색상모드·`edit` 쓰기는 범위 밖(ideas.json 에 별 항목으로 남겼다).
- 프로필은 0일 전 것이 지금 코드와 맞아 새로 쓰지 않았다.
- [러너 02:39] scout done — 좌석맵의 검색어·도면 선택을 주소에 반영해 새로고침·링크 공유에도 남게 하기 (가치 3 / 위험 2 / 작업�

## 구현 노트
- 좌석맵이 `?q=`·`?map=` 을 읽기만 하던 것을 쓰기도 하게 했다(커밋 cda1e3f). 쿼리 규칙은 `web/src/lib/seatMapLink.ts` 한 곳(`readSeatMapParams`·`writeSeatMapParams`·`seatSearchKey`)에 두고 SeatMapPage 의 읽는 세 자리·쓰는 두 자리가 함께 쓴다. 프로덕션 파일 2개.
- 검색 두 번 나가는 함정은 주석 규칙이 아니라 공유 함수 `seatSearchKey` 로 굳혔고, 실제로 나간 `/api/v1/employees?q=` 요청 수를 브라우저에서 세는 E2E 로 1건임을 확인했다(대역 없음).
- 확신 없는 곳: (1) `map` 쪽 URL 쓰기는 씨드에 도면이 하나뿐이라 선택란이 화면에 없어 **E2E 로 증명하지 못했다** — vitest 왕복과 `chooseMap` 코드 경로만 본 것이다. (2) 제출 시 `lastSearchRef` 를 먼저 세우므로, 좌석이 아직 안 실린 상태에서 제출하면 effect 의 "좌석 도착 후 재시도"가 그 한 번은 막힌다(좌석 0개면 effect 자체가 막혀 있어 실제 영향은 없다고 판단했지만 검증하지는 않았다).
- 일부러 안 한 것: `edit` 쓰기·필터·색상모드·확대 상태의 URL 반영(범위 밖, ideas.json 에 남김), 씨드 확장, 서버(`internal/app/`) 변경, PDF 재생성(09-17 이후 관례), mcp-oauth·tracking spec(가짜 IdP·수집기 부재로 아예 돌리지 않음 — 실패로 본 것이 아니라 미실행).
- 다음 역할이 조심할 것: `web/e2e/seat-deeplink.spec.ts` 는 Docker 이미지 + PostgreSQL 16 + 씨드가 있어야 돈다. `PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright` 가 필요했다(HOME 이 run 디렉터리라 기본 경로에 브라우저가 없다). 검증에 쓴 컨테이너·네트워크·`seaton:e2e-before` 이미지는 모두 지웠다.
- [러너 02:54] brief accepted — 채택 — 과제서가 지목한 다섯 자리(324/350/384/636/679-693)와 `lastSearchRef` 재사용 가정이 모두 지금 코드와 맞았고, 씨드에 도
- [러너 02:54] verify passed — 검증 7개 통과 (auto)
- [러너 03:08] review timeout — 단계 제한 시간 초과
- [러너 03:08] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 03:08] pr created — https://github.com/hkjang/seaton/pull/36

## 심사 노트
- 확인한 것: web 에서 `npm ci` → `npm run lint`(tsc -b, e2e 프로젝트 포함) · `npm test` 131/131 · `npm run build` 전부 통과. 제출 경로와 `?q=` 감시 effect 가 `seatSearchKey` 한 함수를 공유해 중복 검색이 구조적으로 막히는 것, loadBase 가 `[]` deps 라 `setSearchParams` 되먹임 루프가 없는 것, 기존 e2e 의 URL 단언이 전부 pathname 기준이라 회귀가 없는 것을 파일에서 읽어 확인했다.
- 확인한 것: 구현 노트의 우려 (2) 는 실제 영향이 없다 — 제출 시 mapId 가 아직 "" 면 키가 `:검색어` 라 좌석 도착 후 `m1:검색어` 로 effect 가 다시 돌고, mapId 가 정해졌는데 좌석이 0개인 경우는 effect 자체가 `!seats.length` 로 막혀 있어 잃는 재시도가 없다.
- 확인한 것: 보호 파일 미접촉(Go·migrations·auth 변경 0), 새 공개 경로·의존성·비밀값 없음, `editMode` 는 여전히 manager 게이트. 검색어가 서버 로그로 새지 않는다 — server.go:155 는 `r.URL.Path` 만 찍는다. `git revert cda1e3f` 로 온전히 되돌아온다.
- 못 본 것: Playwright 를 돌리지 않았다(Docker + PG16 + CV 시드로 10분 이상). spec 이 기준 코드에서 붉을 것과 셀렉터·시드가 모두 실존함을 읽어 확인하고 CI 가 spec 지정 없이 전체를 돌리는 것에 기댔다. `?map=` 주소 쓰기는 시드 도면이 1개라 브라우저로 증명하지 못했고, 배선 2줄(SeatMapPage.tsx:649-651)과 lib 왕복 테스트만 봤다 — 틀려도 loadBase 가 기본 도면으로 떨어지는 실패 방식이다.
- 권고 근거: approve/merge. 결함을 찾지 못했고 남은 것은 notes 뿐이다(무관한 괄호 서식 SeatMapPage.tsx:759, 저장 뒤 주소에 `map=` 이 붙는 의도된 변화, 추적 스니펫을 켠 경우에만 성립하는 개인정보 우려 — 기본 꺼짐·새 수집/보존 없음·공격 경로 없음이라 차단 아님).
