# 회차 노트 2026-09-24-020433-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:04] base pinned — main@db190ed
- [러너 02:04] autonomy release — 

## 정찰 노트
- `leaveMoim`(social.go:583)을 고른 이유: `RowsAffected()==0`이 소유자·유령 slug·이미 탈퇴를 한 덩어리로 409 "Moim 소유자는 나갈 수 없습니다"로 보고하고, `MoimsPage.tsx:207`이 서버 message를 그대로 토스트에 띄워 **사용자에게 사실과 다른 문구가 실제로 보인다**. 차선인 `getMoim`·`followTopic`의 500/404 분리는 updatePost·joinMoim에서 두 번 채택된 것과 같은 유형이라 세 회차 연속을 피했고, Makefile `-race`는 2회 연속 no-change로 끝나 `rejected`로 내렸다.
- 추측으로 적은 것: (a) `moims.visibility`가 public/private 두 값뿐이라는 것은 `createMoim`의 검증에서만 읽었고 migration CHECK는 열지 않았다 — 과제서는 `== "public"` 양성 비교를 쓰게 해 제3의 값이 와도 404로 떨어지게 했다. (b) "공개 Moim 비회원 → 204(멱등)" 대 "비공개 비회원 → 404(존재 은폐)"는 join의 멱등 200과 `getMoim`의 비공개 404에서 유추한 **제품 계약 판단**이지 기존 코드에 적힌 규칙이 아니다. 구현자가 다르게 판단하면 과제서가 아니라 이 결정을 근거와 함께 뒤집을 것.
- 구현자가 조심할 것: `joinMoim`을 같이 건드리지 말 것(c8d3628로 막 릴리즈된 계약). OpenAPI는 `description` 한 줄만 더하고 `responses`를 늘리지 말 것(3b95053 교훈 — middleware가 내는 응답은 handler에 없어도 실재한다). 증명은 반드시 DSN을 준 `go test -race -count=1 ./...`로 `--- SKIP` 0줄을 확인할 것.
- 이번 정찰에서 실제로 실행한 것: `git log`, 코드 열람(social.go·server.go·MoimsPage.tsx·openapi.yaml·moim_join integration test), `owner_cannot_leave` 저장소 전체 grep(= social.go 1곳). `go vet`·`go test`는 이 세션에서 **권한 거부로 미실행** — 과제서의 검증 명령은 지난 회차들이 실제로 돌린 것을 옮긴 것이다.

- [러너 02:08] scout done — `leaveMoim`이 "없는 Moim"·"애초에 회원이 아님"까지 409 `owner_cannot_leave`("Moim 소유자는 나갈 수 없습니다")로 �

## 구현 노트
- 무엇을 왜: `leaveMoim`(social.go:583)의 `RowsAffected()==0`이 소유자·없는 slug·비회원을 한 덩어리로 409 `owner_cannot_leave`로 보고해 나가기를 두 번 누른 회원에게 "Moim 소유자는 나갈 수 없습니다"가 토스트로 보였다. 0행 경로에서만 판별 조회 1회를 돌려 404/204/409/500으로 나눴다(d329380). 문구는 전부 기존 것 재사용, 성공 경로 쿼리 수 불변.
- **확신 없는 곳·검증 못 한 것**: (1) 판별 조회 **자체**가 실패하는 500 분기는 integration 미커버 — 평 PostgreSQL에서 특정 행의 SELECT만 실패시킬 방법이 없다(트리거는 SELECT 불가, RLS는 FORCE가 필요해 공유 `moims`를 전역 변경하고 `go test ./...`의 패키지 병렬 실행과 충돌). 대신 DELETE 실패 500을 실제 `BEFORE DELETE` 트리거로 덮었다. (2) "공개 비회원 → 204, 비공개 비회원 → 404"는 정찰이 join 멱등 200·getMoim 비공개 404에서 유추한 **제품 계약 판단**이고 기존 코드에 적힌 규칙이 아니다 — 그대로 채택했으나 뒤집을 여지가 있는 유일한 설계 결정이다. (3) `make lint`의 frontend eslint 구간은 node_modules 미설치로 실행 불가(Go 구간 exit 0); 프런트 무변경이라 vitest·e2e도 미실행.
- 정찰이 "미확인"으로 남긴 `moims.visibility`는 확인했다: `001_initial.sql:163` `NOT NULL CHECK(visibility IN ('public','private'))`. 지시대로 안전한 방향(`!= "public"` → 404) 비교를 유지했다.
- 일부러 안 한 것: `joinMoim`의 `_ = Scan(&exists)`(c8d3628로 막 릴리즈된 계약), `getMoim`·`followTopic`의 같은 유형 분리, OpenAPI `responses` 확장(3b95053 교훈), 프런트 수정(서버 message를 그대로 띄우므로 불필요) — 전부 ideas.json에 pending으로 남겼다.
- 다음 역할이 조심할 것: 새 `moim_leave_postgres_integration_test.go`는 **`MOINA_TEST_POSTGRES_DSN`이 없으면 조용히 SKIP**된다. 검증에 쓰려면 throwaway `postgres:17-alpine` DSN을 주고 `--- SKIP` 0줄을 확인할 것(이번에 그렇게 했다: 전 패키지 ok, SKIP 0, `TestPostgreSQL*` 35건 PASS). 테스트는 `BEFORE DELETE ON moim_members` 트리거를 만들었다가 `t.Cleanup`으로 지운다.
- [러너 02:14] brief accepted — 채택 — 과제서의 근거가 현재 코드와 정확히 일치했고(`social.go:583`의 0행 통합 보고, `MoimsPage.tsx:207`의 DELETE 토글, `owner_ca
- [러너 02:15] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve(low, 차단 없음). 실제 대상은 db190ed..HEAD = d329380 3파일 — 이 worktree의 로컬 `main`은 v0.1.28로 낡아 `git diff main...HEAD`는 무관한 76파일을 보여준다. 다음 회차는 diff 전에 main을 fetch할 것.
- 테스트가 정말 검증하는지 직접 증명했다: db190ed를 `git archive`로 /tmp에 펼치고 새 테스트만 얹어 돌리자 missing_slug_404·twice_204·non-member_public_204·non-member_private_404 네 subtest가 `409 owner_cannot_leave`로 FAIL, 수정본에서는 PASS. throwaway postgres:17-alpine DSN으로 backend 전 패키지 `go test -race` 13/13 ok(SKIP 0), `go vet` 0, OpenAPI 120 route 통과.
- 구현자가 의심한 세 자리를 확인: (1) 판별 SELECT의 500 분기는 여전히 미커버 — DELETE 실패 500은 실제 트리거로 덮여 있어 잔여 위험 낮음. (2) public 204 / private 404 비대칭은 새 노출이 아니다(public Moim은 이미 listMoims·getMoim이 전원에게 공개, private은 없는 slug와 동일한 404). (3) 001_initial.sql:159/163/170을 직접 열어 slug UNIQUE·visibility CHECK·role NOT NULL을 확인, NULL role·제3 visibility 경계 없음.
- 못 본 것: 프런트 vitest·e2e 미실행(프런트 무변경), docs pdf·캡처 무변경. 소유자 판정이 `moims.owner_id`가 아닌 `moim_members.role='owner'` 사본에 의존하는데 저장소에 `UPDATE moims`가 한 곳도 없어 현재는 안전 — 소유권 이전 기능이 생기면 이 handler를 같이 고칠 것.
- 릴리즈 노트에 넣을 것: 비공개 Moim을 나간 뒤 한 번 더 누르면 404 `Moim을 찾을 수 없습니다`가 뜬다(getMoim과 일관되나, 두 번 누름 오해 해소는 공개 Moim 한정).
- [러너 02:18] review approved — 리뷰 승인 (risk=low)
- [러너 02:18] pr created — https://github.com/hkjang/moina/pull/30
- [러너 02:27] ci passed — 검사 2개 모두 success
- [러너 02:27] merge done — d329380
- [러너 02:41] release published — v0.1.37
- [러너 02:47] assets verified — v0.1.37 자산 1개 (이전 v0.1.36: 1)
