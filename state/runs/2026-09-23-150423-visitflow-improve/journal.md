# 회차 노트 2026-09-23-150423-visitflow-improve — visitflow
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:04] base pinned — main@4e0481d
- [러너 15:04] autonomy release — 

## 정찰 노트
- 방문자 이름·휴대전화 사전 검사를 골랐다. 회사명(v2.8.4)·일정(v2.8.5)과 **같은 모양의 마지막 남은 구멍**이고(서버 visits.go:454-457 은 이름 trim·전화 7자리를 요구하는데 화면 117행은 빈 값만 본다), 표시할 메시지에 줄 번호를 붙일 수 있어 서버 400 보다 실제로 나아진다. 차선이던 "100명 상한"은 도달 빈도가 훨씬 낮아 뒤로 미뤘고, "기준 정보 재시도"는 같은 파일을 두 회차 연속 건드리게 돼 뺐다.
- 확신 없는 곳: `npm ci`·`npm test`·빌드를 이번 정찰에서 **한 번도 실행하지 않았다**(node_modules 부재만 확인). 테스트 23개라는 숫자는 파일을 읽어 센 것이다. TemplatesPage 의 자주 방문자 전화 규칙(templates.go:158)이 방문 신청과 같은지도 미확인이라 범위에서 제외했다.
- 구현자가 조심할 것: **빈 칸을 빨갛게 칠하지 말 것**(첫 화면 회귀 — 이 과제가 반려될 가장 큰 이유). 기존 `!x.name || !x.phone || !x.consent` 항을 지우지 말고 더하기만 할 것. 테스트는 `.ts` 로(vite.config.ts 의 include 가 `.tsx` 를 버린다). 서버 `visits.go` 와 `normalizePhone` 은 손대지 말 것.
- [러너 15:08] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 방문자 이름·휴대전화 사전 검사를 넣었다(f28c58d). 순수 모듈 `web/src/visitors.ts` 가 서버 경계(이름 TrimSpace, normalizePhone 숫자 7자리)를 계산하고, 칸 `error`/`helperText`·버튼 `disabled`·`submit()` 가드가 모두 `visitorFieldErrors` 한 값에서만 나온다. 메시지에 방문자 번호를 붙였다.
- 확신 없는 곳: `docker build` 와 `npm run test:e2e`(Playwright 정식 스위트)는 **돌리지 않았다** — 대신 실제 서버(go run + postgres 컨테이너)에 실제 dist 를 임베드해 google-chrome 으로 16항목을 확인했다. e2e·screenshots 스펙은 이름/휴대전화 칸에 유효한 값만 채우고 helperText 를 단언하지 않는 것을 확인했다(grep). TemplatesPage 의 자주 방문자 전화 규칙(templates.go)이 같은 경계인지도 확인하지 않았다.
- 일부러 하지 않은 것: 서버 `visits.go`·`normalizePhone` 무수정(공용 계약), 100명 상한 사전 차단(별 과제로 남김), 회사명/동의 항 수정, PDF 재생성(USER_GUIDE.md 한 문장만 수정).
- 다음 역할이 조심할 것: `internal/app/visitor_contract_test.go` 는 **DB(VISITFLOW_TEST_DSN)가 있어야 실제로 돈다** — DSN 없으면 SKIP 이라 PASS 가 증거가 아니다. 브라우저 확인을 재현하려면 `web/dist` 를 `cmd/visitflow/webdist` 에 복사해야 하는데 **끝나고 `index.html` 을 원래 스텁으로 되돌리지 않으면 빌드 산출물이 커밋된다**(이번에는 /tmp 백업으로 복원했고 커밋 전 `git status` 로 확인함).
- [러너 15:21] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 구현자가 못 돌렸다고 한 것부터 돌렸다: 실제 PostgreSQL(docker weekly-test-pg:15434)로 `go test ./... -count=1` 전부 ok(internal/app 44.6s), 새 `TestVisitorNameAndPhoneBoundary` 가 SKIP 이 아니라 실제 실행되어 PASS. `tsc -b`·`vitest run`(38개, visitors.test.ts 15개 수집됨)·`npm run build` 모두 통과. 워크트리 clean, `cmd/visitflow/webdist/index.html` 은 main 과 동일해 빌드 산출물 유출 없음.
- 못 본 것: Playwright 정식 스위트·`docker build`·실제 브라우저 확인은 재현하지 않았다(구현자 보고를 근거 없이 뒤집을 이유를 못 찾음).
- 승인이어도 남는 우려 ①: `docs/USER_GUIDE.md:45` 가 가져오기 맥락에서 '이름이 공백뿐이면 칸에 이유가 표시' 라고 쓰지만 `import.go:230 cell()` 이 TrimSpace 하므로 가져오기는 빈 이름('')만 만들고 빈 값은 일부러 표시하지 않는다 — 이름이 빠진 가져오기 행은 버튼만 잠기고 어느 행인지 칸에 안 보인다. 휴대전화 쪽은 문장대로 동작.
- 승인이어도 남는 우려 ②(릴리즈): `USER_GUIDE.pdf` 재생성이 v2.8.4·v2.8.5 에 이어 세 회차째 밀려 .md/.pdf 차이가 누적된다. 또 휴대전화 칸은 1~6번째 글자 동안 계속 빨갛다(빈 칸은 조용하므로 첫 화면 회귀는 아님).
- 보안·법무 차단 없음: 새 엔드포인트·인가·비밀값·의존성·마이그레이션 없고, 새 문구는 방문자 번호만 담아 이름·전화번호를 노출하지 않는다. 미머지 `origin/auto/2026-09-21-0654` 와 USER_GUIDE 행이 겹치지 않는 것도 확인했다.
- [러너 15:25] review approved — 리뷰 승인 (risk=low)
- [러너 15:26] pr created — https://github.com/hkjang/visitflow/pull/24
- [러너 15:29] ci passed — 검사 2개 모두 success
- [러너 15:29] merge done — f28c58d
- [러너 15:37] release published — v2.8.6
- [러너 15:39] assets verified — v2.8.6 자산 1개 (이전 v2.8.5: 1)
