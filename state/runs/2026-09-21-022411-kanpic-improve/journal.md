# 회차 노트 2026-09-21-022411-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:24] base pinned — main@1f9dfcc
- [러너 02:24] autonomy release — 

## 정찰 노트
- 불리언 타입 불일치는 두 호출부에서 확인되고 기존 HTTPS/업로드 비교 테스트를 재사용할 수 있어, API 결합·서식 정책·구분자 확장보다 범위와 위험이 작다.
- 실제 main의 b52a394가 XLSX 15자리 제한을 되돌린 사실을 반영했다. XLSX·trim·아포스트로피·숫자 파서 확장은 금지한다.
- Go 전체 테스트와 릴리즈/신원 스크립트 통과; 불리언 새 실패 테스트·웹/DB 실행은 미확인. 신규 UTF-8/여러 줄 CSV 아이디어도 실행 재현 전이다.
- 세 스킬은 전용 도구 부재로 headcount의 SKILL.md 원문을 읽어 적용했다. 추정 30–45분은 통계 보장이 아닌 판단이며 실제 결과에 맞춰 계획 상태를 갱신한다.
- [러너 02:29] scout done — IMPORTDATA도 업로드처럼 true/false를 불리언으로 읽는다 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 변경: delimited.Boolean으로 기존 업로드 판정만 추출하고 IMPORTDATA 원문에 연결; 719aec5 커밋, 작업 트리 깨끗함.
- 단계 1 완료: 공통 함수/업로드 연결 뒤 go test ./internal/delimited ./internal/importexport 통과.
- 단계 2 완료: TestImportDataReadsBooleansLikeUpload 실제 TLS→Resolve/Parse 비교에서 여섯 bool 타입 실패 확인→연결 후 통과→분기 제거 시 동일 실패→복원.
- 단계 3 완료: go test ./internal/delimited ./internal/external ./internal/importexport 및 go test ./... 통과(기존 숫자/XLSX 17자리 회귀 포함); go vet ./...·go build ./... exit 0, gofmt -l ./cmd ./internal ./pkg 출력 없음.
- 검사: check-release-docs.sh는 release docs ok: v0.251.0; check-commit-identities.sh HEAD는 커밋 전후 exit 0; git diff --check 통과. 세 technology 스킬은 전용 도구 부재로 SKILL.md 원문 적용.
- 확신 없는 곳·검증 못 한 것: 웹/npm·브라우저·실 DB 통합 테스트 미실행(서버 CSV 판정에 한정된 변경); 실제 외부 서비스는 TLS 시험 서버로 대체.
- 일부러 제외/다음 역할 주의: XLSX·숫자·trim·unguard·SSRF/캐시 정책 유지. 업로드 빈칸은 미저장, 원격 빈칸은 빈 문자열/누락은 nil이며 숫자 공백·아포스트로피 계약 차이도 의도적으로 유지했다.
- [러너 02:32] brief accepted — 채택 — 현재 두 파서의 차이가 과제서와 일치했고 숫자·XLSX·공백 정책을 넓히지 않고 해결했다.
- [러너 02:32] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음: 네 변경 파일·호출부·보안 경계·되돌리기 검토, 코드 수정 없음.
- 네 Go 패키지 캐시 없는 테스트 통과; main fetcher를 Go overlay로 적용해 새 테스트의 여섯 불리언 실패 재현.
- 웹/npm·브라우저·실 DB/격리 통합·실제 외부 서비스 왕복은 미검증; 세 부서 스킬은 전용 도구 부재로 원문 적용.
- 비차단: fetcher.go:388 주석 갱신 필요. 릴리즈에는 IMPORTDATA 불리언 타입 변경과 기존 공백·아포스트로피 계약 유지를 명시할 것.
- [러너 02:34] review approved — 리뷰 승인 (risk=low)
- [러너 02:34] pr created — https://github.com/hkjang/kanpic/pull/28
- [러너 02:42] ci passed — 검사 2개 모두 success
- [러너 02:42] merge done — 719aec5
- [러너 02:42] release missing — 릴리즈 결과 없음/손상: missing
