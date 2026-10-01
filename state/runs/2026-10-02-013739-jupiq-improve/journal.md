# 회차 노트 2026-10-02-013739-jupiq-improve — jupiq
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:37] base pinned — main@92ff88c
- [러너 01:37] autonomy release — 

## 정찰 노트
- 지난 회차가 남긴 최고 가치 보류 항목(writeJSON)을 골랐다. 실제 httptest 서버로 200+빈 본문을 재현했고 프로덕션 파일 1개로 끝나며, 지난 릴리즈(v1.8.7)의 프런트 수정과 짝이 맞는다. page_size 불일치·store 순수 헬퍼 테스트는 세 회차 연속 차선에 머문 저가치라 제쳤다.
- 확신 없는 곳(과제서에 추측이라고 적어 뒀다): NaN이 응답까지 가는 **실제 유입 경로를 못 찾았다**. prometheus.go의 validPrometheusSample이 수집 입구를 막고, numberFromAny의 "NaN" 통과는 호출자가 모두 int64 캐스팅이라 무해하다. 이번 과제는 "지금 터지는 버그"가 아니라 계약 방어다 — 구현자는 실제 버그인 척 쓰지 말 것.
- PostgreSQL double precision 열에 'NaN'을 실제로 넣어 보지는 않았다(DDL만 확인). Go 테스트도 이번에 돌리지 않았다.
- 구현자가 조심할 것: Encode는 끝에 개행 1바이트를 붙이고 Marshal은 붙이지 않는다. 이 차이를 먼저 테스트로 고정하지 않으면 성공 경로 회귀로 깨진다. 그리고 큰 응답을 io.Writer로 흘리게 "최적화"하면 과제 목적이 사라진다.
- 보호 경로(auth·migrations·workflows)와 SSE 스트리밍은 이 과제에 전혀 필요 없다. helpers.go 한 파일 + 테스트 한 파일로 끝내라.
- [러너 01:43] scout done — 서버 `writeJSON`이 인코딩 실패를 버려 **200 + 빈 JSON 본문**을 내보내는 것을 500 오류 봉투로 바꾼다 (가치 3 /

## 구현 노트
- `internal/api/helpers.go`의 `writeJSON`을 marshal-first로 바꿨다(프로덕션 1파일, 커밋 35a86dd). 실패하면 500 + const 고정 error 봉투를 쓰고 `slog.Default().Error`로 원래 오류를 남긴다. 성공 경로는 끝 개행 1바이트까지 종전과 동일하며 신규 테스트 4개가 그 바이트를 고정한다.
- 확신 없는 곳: NaN이 응답까지 가는 **실제 유입 경로는 이번에도 못 찾았다** — 정찰과 같은 결론이고 계약 방어로만 다뤘다. PostgreSQL `double precision` 열에 `'NaN'`이 실제로 들어가는지는 여전히 **미확인**(DDL만 봤다). 프런트가 이 새 500을 어떻게 보이는지는 브라우저로 확인하지 않았다(`client.ts`가 500 + `error.code`를 ApiError로 바꾸는 경로는 기존 테스트가 덮는다).
- 과제서에 없던 추가 두 가지: ① 실패 시 `slog` 로깅(안 하면 오류가 완전히 사라진다. `cmd/jupiq/main.go:25`가 `slog.SetDefault`를 하므로 설정된 로거로 간다) ② 봉투를 `var []byte`가 아닌 `const` 문자열로 둠(불변).
- 일부러 안 한 것: 실패 봉투에 `request_id` 담기(시그니처·호출자 7곳을 바꿔야 해서 ideas.json에 가치 1로 기록), 출구 유한성 가드(계약 결정이 남아 위험 3), 차선 후보 page_size.
- 다음 역할이 조심할 것: 새 테스트 `internal/api/write_json_test.go`는 **DB 없이 돈다**(`httptest.NewRecorder`만 씀). `make test-integration`은 DSN이 필요하고, 나는 임시 `postgres:16-alpine`(포트 5434)로 돌려 통과시킨 뒤 컨테이너를 지웠다 — 비평가가 다시 돌리려면 컨테이너를 새로 띄워야 한다. 릴리즈 통합은 PostgreSQL **14**인데 거기서는 확인하지 않았다. 프런트·npm·`make release-check`는 실행하지 않았다(Go만 바뀌었다).
- [러너 01:48] brief accepted — 채택 — 과제서의 진단(helpers.go:29가 WriteHeader를 먼저 하고 Encode 오류를 버린다, 호출자 7곳, 함정은 끝 개행 1바이트)이 코
- [러너 01:49] verify failed — 실패한 검증: cd web && npm test --silent (exit 1)
