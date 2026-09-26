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
