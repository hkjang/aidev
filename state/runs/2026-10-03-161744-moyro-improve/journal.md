# 회차 노트 2026-10-03-161744-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:17] base pinned — main@4a1a03b
- [러너 16:17] autonomy release — 

## 구현 노트
- 무엇/왜: `POST /api/v4/teams/{teamID}/invite/email` 이 같은 인증 그룹에서 두 번 등록돼(router.go:756 게이트+감사 / 1054 게이트 없는 스텁) chi 가 뒤를 채택했고, 평범한 팀 멤버도 200 OK + 감사 행 0건이었다. 1054 한 줄 삭제 + 재등록 금지 주석. 프로덕션 코드 1파일(코드 -1줄).
- 확신 없는 곳: (1) 비관리자의 응답이 거짓 200 → 403 으로 바뀌는 것은 의도한 수정이지만 **외부 클라이언트 관점에선 계약 변경**이다. 핸들러 주석이 "(admin)" 이라 적고 webapp/e2e/docs/OpenAPI 에 소비자가 없음은 확인했으나, 저장소 밖 통합이 이 경로의 무조건 200 에 의존했다면 영향을 받는다. (2) chi 의 "뒤 등록이 교체" 동작은 소스를 읽어 추론한 게 아니라 실제 HTTP 요청으로만 확인했다 — 그 관찰은 확실하지만 chi 내부 구현을 읽어 교차 확인하지는 않았다.
- 검증 못 한 것: webapp 은 한 글자도 건드리지 않았으므로 npm typecheck/test/build 를 돌리지 않았다(알려진 @types/node TS2345 함정과 무관하게, 변경이 없어 생략).
- 일부러 하지 않은 것: 나머지 `h.channels.Get` 호출자 약 10곳의 404 접기와 `createIncomingWebhook` 404→500 은 범위 밖으로 뒀다 — 후자는 선행 확인까지 끝냈지만 같은 접근의 커밋 `0b2d41c` 가 머지되지 않고 남아 있어 일부러 피했다(ideas.json 에 근거째로 기록). 중복 등록 회귀 가드(테스트/스크립트)도 이번엔 넣지 않았다.
- 다음 역할이 조심할 것: 신규 `invite_email_route_postgres_test.go` 는 **DB 가 있어야 돈다**(`MOYRO_TEST_POSTGRES_DSN` 없으면 skip — ok 를 통과로 오인 금지). 이 패키지 최초로 프로덕션 `NewRouter` 를 띄우므로 실행이 느리고 `pluginhost.NewWithRuntime` + `t.TempDir()` + `httptest.NewServer` 를 쓴다. router.go 에 `/teams/{teamID}/invite/email` 을 다시 등록하면 이 테스트가 즉시 RED 가 된다(그게 의도다).
- [러너 16:35] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 승인. 구현자가 의심한 두 곳을 직접 시험했다: 중복 라인을 되살려 실제 DB로 돌리니 `status = 200, want 403` + `audit row = 0, want 1` 로 FAIL(7.22s), 되돌리니 3개 서브테스트 PASS(1.18s) — 테스트는 진짜 회귀 가드다. 실험 후 작업 트리 clean 복원 확인.
- 756/1054 가 **같은 requireAuth 그룹**임을 그룹 경계(438 시작, system_admin 하위그룹 876-939·940-996·997-1001·1066+ 가 1054 를 포함하지 않음)로 확인했다. 200→403 은 실제 Mattermost 동작과 같으므로 호환성 규칙에 **부합**하는 방향이고, 저장소 전체에서 이 경로의 소비자는 Go 주석 2개뿐이다(webapp/e2e/docs/OpenAPI 없음).
- 추가 확인: router.go 전체 중복 등록 재스캔({param} 이름 정규화 포함) 결과 **다른 중복 없음**; 형제 경로 `/teams/{teamID}/invite-guests/email` 은 정상 게이트됨; `go test ./internal/httpapi/`(실DB) ok 39.733s, build/vet/gofmt 깨끗, 범위 이탈 없음(프로덕션 -1줄).
- 남는 우려(차단 아님): 게이트 복구로 초대 이메일이 다시 audit_logs 에 쓰인다. 읽기는 system_admin 게이트지만 **audit_logs 보존·삭제 메커니즘이 전무**(마이그레이션·internal/audit 어디에도 없음) — 선행 결함이므로 별도 티켓.
- 다음 회차: `/teams/members/invite` 는 여전히 무게이트 200 no-op 이고 이번 테스트가 그 200 을 고정했다. 지금은 무해(상태 변화 없음)하나 저 스텁이 실제 초대 로직을 갖게 되면 테스트가 무게이트를 계속 승인한다 — ideas.json 에 남길 것. 릴리즈 노트에 403 전환을 명시할 것.
- [러너 16:40] review approved — 리뷰 승인 (risk=low)
- [러너 16:40] pr created — https://github.com/hkjang/moyro/pull/32
- [러너 16:53] ci passed — 검사 3개 모두 success
- [러너 16:53] merge done — 76c6c95
- [러너 17:14] release published — v0.2.44
- [러너 17:32] assets verified — v0.2.44 자산 1개 (이전 v0.2.43: 1)
