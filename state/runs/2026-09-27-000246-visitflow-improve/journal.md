# 회차 노트 2026-09-27-000246-visitflow-improve — visitflow
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:02] base pinned — main@15456db
- [러너 00:02] autonomy release —

## 정찰 노트
- 고른 이유: v2.8.4~v2.8.7 이 신청 화면의 서버 경계 미러링(회사명·일정·이름/전화·인원/반복)을 모두 닫아 그 광맥은 고갈됐다. 남은 최대 결함은 `VisitFormPage.tsx:45` 의 reference-data 실패 — `ref=null` 로 굳고 상단 Alert 은 `onClose` 로 닫히므로 닫는 순간 이유도 복구 수단도 없는 빈 양식만 남는다(제출은 `siteId==""` 로 영구 잠김). 프로덕션 파일 1개로 끝난다.
- 제친 것: "방문자 0명 방어" 는 도달 불가로 확인해 기각(`import.go:267` 이 0명 미리보기를 오류로 돌려보내고 삭제 버튼은 `visitors.length===1` 에서 잠긴다). 미머지 3개 브랜치와 겹치는 메일·MCP OAuth·가져오기 계열은 이번에도 전부 보류.
- 미확인(과제서에 추측으로 적은 것): reference-data 5xx 를 실제로 재현해 보지는 않았다 — 소스만 읽었다. 구현자는 수정 전 증상을 브라우저로 먼저 재현할 것. 재시도 성공 뒤 `useEffect([siteLobbies, lobbyId])` 의 로비 자동 선택이 다시 도는지도 미확인.
- 조심할 것: 기존 `error` state 를 재사용하면 수용 기준(닫아도 복구 수단이 남을 것)을 못 지킨다 — 별도 state 로 둘 것. vitest 는 `.tsx` 를 수집하지 않으니 억지 순수 함수 대신 실제 서버+dist+Chromium 확인을 증거로 쓸 것. `internal/app/visits.go` 는 손대지 말 것.
- [러너 00:06] scout done — 방문 신청 화면에서 기준 정보(reference-data) 로드 실패를 복구 가능하게 만들기 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇을 왜: `VisitFormPage.tsx:45` 의 reference-data 로드를 이름 붙은 `loadReference()` 로 빼고 공용 `error` 와 **분리된** `refError`/`refLoading` state 를 추가해, 실패 시 닫히지 않는 Alert + `다시 불러오기`(진행 중 disabled)를 띄운다. 공용 Alert 을 재사용하면 닫는 순간 복구 수단이 사라져 수용 기준 1)을 못 지킨다. 프로덕션 파일 1개(469a8b4) + 가이드 한 문장, PDF 미재생성.
- 확신 없는 곳·검증 못 한 것: (1) `siteScope` 경로는 **추가로 실측해 닫았다** — 두 번째 사업장/로비를 만들고 `siteScope=[두 번째 사업장]` 인 role=lobby 계정으로 `/lobby/walk-in` 에서 재시도했더니 `sites` 의 첫 항목인 본사가 아니라 "범위검증사업장 · 두 번째 사업장"이 자동 선택되고 "범위검증로비"까지 채워졌다(즉 재시도 경로에서도 범위 필터가 돈다). 다만 `loadReference` 는 매 렌더 새로 만들어져 **재시도 시점의** `siteScope` 를 쓴다 — 로그인 중 범위가 바뀌는 상황은 검증하지 않았고, 그 편이 맞다고 판단했다. (2) 중복 요청 방지는 버튼 `disabled` 하나에만 의존한다 — `loadReference` 안에 in-flight 가드는 없다. 키보드/프로그램 호출 경로가 새로 생기면 뚫린다. (3) 응답 역전(느린 첫 요청이 빠른 재시도보다 늦게 도착)은 취소·순번 없이 그대로 두었다.
- 일부러 하지 않은 것: vitest 단위 테스트 — `include` 가 `src/**/*.test.ts` 라 `.tsx` 는 조용히 0개가 되고, 이 결함은 React state/effect 배선이라 순수 `.ts` 로 뽑으면 정작 바뀐 경로를 지나지 않는다(과제서 지시). testing-library/jsdom 추가는 의존성 확대라 범위 밖으로 뒀다. `ScannerPage.tsx:21`(catch 없음)·`LobbyPage.tsx:27`(삼키기)은 같은 결함이지만 파일 수를 늘리지 않으려 ideas.json 으로 넘겼다. `submit()`·`importVisitors()`·기존 사전 검사 배선·`internal/app/visits.go` 미변경.
- 다음 역할이 조심할 것: 단위 테스트로는 이 변경이 전혀 커버되지 않는다 — 증거는 `/tmp/vf-refdata-check.mjs`(repro|fixed) 뿐이고 **PostgreSQL + `web/dist` 를 `cmd/visitflow/webdist` 로 복사한 실제 서버 + `/usr/bin/google-chrome` 이 있어야 돈다**. 검증 뒤 webdist 스텁은 임시 복사본으로 되돌렸고(`git checkout --` 미사용) `git status` 가 의도한 2개 파일만 보이는 것을 확인했다. 확인용으로 만든 부서("복구검증부서")·사업장("범위검증사업장")·로비·계정("scopelobby")과 방문 1건은 /tmp 의 임시 DB(docker `vf-pg-0928`, 정리함)에만 있고 저장소에는 남지 않았다.
- [러너 00:31] brief accepted — 채택 — 지정한 파일·근거·수용 기준 5개가 지금 코드와 정확히 맞았고 프로덕션 파일 1개로 끝났다. 과제서가 미확인으
- [러너 00:31] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: `git diff main...HEAD` 전부(프로덕션 1파일 + 가이드 1줄), `VisitFormPage.tsx` 1-124, `api.ts` 1-60, `App.tsx` 라우트 가드, `web/package.json` 스크립트, `server.go:440 writeError`, `/tmp/vf-refdata-check.mjs` 전문, `verify.json`, 워크트리·webdist 스텁 상태. 판정 **approve** (risk low, blocking 없음).
- 구현자의 "확신 없는 곳" 3개는 모두 코드로 닫혔다: `App.tsx:65` 가 `loading` 동안 렌더를 막아 마운트 시점 `siteScope` 가 확정이고(스테일 클로저 없음), `refError` 가 빈 동안에는 Alert·버튼 자체가 없어 in-flight 중복이나 응답 역전을 UI 로 만들 수 없다 — 순번·취소 미구현이 무해한 이유다.
- 못 본 것: 브라우저를 직접 띄우지 않았다(`/tmp/vf-refdata-check.mjs` 재실행 안 함 — Postgres·dist·Chromium 준비 비용). 하네스가 repro/fixed 를 실제로 갈라내는 것은 코드로 확인했다.
- 승인이어도 남는 우려: **회귀 테스트가 저장소에 없다.** /tmp 하네스가 사라지면 누군가 이 Alert 에 `onClose` 를 다시 붙여도 vitest 47개는 전부 통과한다. 릴리즈 노트에 "화면 배선이라 자동 커버리지 없음" 을 남길 것.
- 다음 회차용: 401 이면 재시도가 영구 실패(재로그인 유도 없음), `fetch` 자체 실패 시 꼬리에 영어 "Failed to fetch", 범위 내 사업장 0건이면 여전히 말없이 잠김 — 셋 다 이번 변경이 만든 것은 아니다.
- [러너 00:34] review approved — 리뷰 승인 (risk=low)
- [러너 00:34] pr created — https://github.com/hkjang/visitflow/pull/26
- [러너 00:38] ci passed — 검사 2개 모두 success
- [러너 00:38] merge done — 469a8b4
- [러너 00:46] release published — v2.8.8
