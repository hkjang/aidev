## 2026-09-29
- 선택: sqlsafe PostgreSQL WITH ORDINALITY 뒤 컬럼 별칭 오탐 수정 (가치 2 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `FROM unnest(...) WITH ORDINALITY g(value,position)`의 g를 비허용 함수로 오인해 정상 질의를 차단하던 경로에, `) WITH ORDINALITY` 접미사와 기존 FROM 문맥을 모두 요구하는 별칭 판별을 추가했다(프로덕션 1파일 +8줄, 테스트 2파일; 커밋 7cda7f5). 새 단위 테스트(정상 8·차단 8)와 실제 PostgreSQL 17 Manager.Validate/Execute 페이징 테스트 6개로 red→green→프로덕션 파일 되돌려 red를 확인했고, 142질의×2방언 비교에서 정상 7개만 허용으로 변경·Oracle 판정 불변·새 오탐 0을 확인했다. `go test ./...` 28패키지, `go test -race -p=1 -tags=integration ./... -count=1` 30패키지, make lint·Go build·통합 태그 vet·웹 134테스트/빌드·브라우저 E2E 14개·프로덕션 npm audit가 통과했으며 Oracle 실서비스/릴리즈 이미지는 미검증, 컨테이너·생성 산출물은 제거했다.
- 실패 재현: `sqlsafe_test.go:268: PostgreSQL WITH ORDINALITY alias was rejected: sqlsafe.Analysis{StatementType:"SELECT", ReadOnly:false, MultipleStatements:false, Risk:"blocked", Reasons:[]string{"PostgreSQL SELECT의 비허용 함수 호출 g()이 포함되어 있습니다."}, Tokens:[]string{"SELECT", "FROM", "UNNEST", "ARRAY", "WITH", "ORDINALITY", "G", "VALUE", "POSITION"}}` / `manager_integration_test.go:83: WITH ORDINALITY execution failed: SQL violates the target database read-only policy: PostgreSQL SELECT의 비허용 함수 호출 g()이 포함되어 있습니다.`
- 보류 아이디어: legacyapi의 pool.Close/픽스처 Cleanup 순서 역전 확장 (3/2/M) — 실제 잔존 행 재현 후 별도 진행.
- 보류 아이디어: store 통합 테스트의 명시적 전용 DSN/연결 실패 Fatal 전환 (3/2/M) — 초기 Skip 경로가 현재도 남음.
- 보류 아이디어: [신규] 스키마 없는 FROM table alias(column) 오탐 조사 (2/2/S) — 현재 점 표기 면제와 달라 실DB 비교 선행.
- 보류 아이디어: [신규] 다수 CTE 컬럼 목록의 분석 비용 측정 (3/2/M) — 역방향 반복 탐색의 선형성부터 벤치마크로 확인.
