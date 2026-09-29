# 회차 노트 2026-09-29-100225-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:02] base pinned — main@c8bceef
- [러너 10:02] autonomy release — 

## 정찰 노트
- AI 사람 조회 실패를 404로 분리하는 과제 선택: 사용자 가치와 결정적인 실DB 재현이 rows.Err 중간 실패·반복 OpenAPI 차선보다 높음.
- 초안을 먼저 저장한 뒤 dataKeyVersion도 ErrNoRows를 낸다는 사실을 확인해 첫 조회 전용 sentinel로 보완; 작업량 M, 프로덕션 1파일.
- go test ./... 통과. 새 결함 실DB 재현·race/vet·웹은 미실행이며 과제서에 표시. 스킬 3개는 전용 도구 부재로 로컬 SKILL.md 읽어 적용.
- 구현자는 전역 AI 설정 복원, 타인 UUID 격리, 키 누락 500 유지, 실제 SSE 배선을 검증할 것. 13개 아이디어 유지/갱신(신규 2개), 반복 OpenAPI 후보 rejected.
- [러너 10:07] scout done — AI 스트림에서 조회할 수 없는 사람만 SSE 시작 전 404로 반환 (가치 3 / 위험 2 / 작업량 M)

## 구현 노트
- ai.go 첫 사람 JOIN 조회만 sentinel로 분류해 SSE 시작 전 404 JSON 반환; ai_db_test.go 추가(프로덕션 1파일).
- 실DB에서 없는/타인/관계 없는 사람 500 실패를 먼저 재현했고 수정 후 9개 AI 하위 시험 PASS. 수정 제거 시 같은 세 시험 재실패 확인.
- 정상/빈 ID의 실제 proxyAIStream meta/delta/done 및 호출 1회, 거부 요청 호출 0회, 기억 키/설정 누락 500 유지 검증.
- DB 포함 전체 race·집중 시험, vet, Go build 통과; gofmt 출력 없음. ai-red.log/ai-green.log/ai-revert-red.log/ai-focused.log/ai-full-race.log에 출력 보존.
- 확신 없는 곳·검증 못 한 것: 실제 외부 AI·브라우저·웹 빌드·릴리즈는 미실행. 이번 서버 오류 분류 범위 밖이며 HTTP 제공자만 시험 서버로 대체.
- 전역 설정은 전체 컬럼 백업/복원하고 기존 행 없음/있음 두 경우 확인. 격리 PostgreSQL 컨테이너는 제거함.
- 다음 역할: ORBIT_TEST_DATABASE_URL 없으면 DB 시험 SKIP하므로 반드시 격리 DB로 검증. 병렬 실행 금지; 릴리즈·auth·암호화·공유 파서는 변경하지 않음.
- [러너 10:12] brief accepted — 채택 — 첫 사람 조회와 기억 키 조회가 모두 ErrNoRows를 반환함을 실제 DB로 확인했으며 지정한 첫 조회 전용 분류와 모든 
- [러너 10:12] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- approve / low, blocking 없음: diff·커밋·원장 실패 로그와 첫 조회 전용 오류 분류, 인증·소유자 제한, 테스트 단언을 확인했고 실제 결함은 찾지 못함.
- 별도 격리 PostgreSQL로 전체 go test -race -count=1 ./... 및 go vet ./... 통과; 컨테이너 제거, 코드 변경 없음.
- 프런트엔드 JSON 오류 표시 경로 확인; 실제 외부 AI·브라우저·웹 빌드·릴리즈는 미검증.
- 잔여 우려: CI에서 DSN 없으면 DB 시험 SKIP, 전역 설정 시험은 격리 DB·비병렬 필수. 세 부서 스킬은 전용 도구 부재로 로컬 파일 적용.
- [러너 10:15] review approved — 리뷰 승인 (risk=low)
- [러너 10:15] pr created — https://github.com/hkjang/orbit/pull/15
- [러너 10:17] ci passed — 검사 1개 모두 success
- [러너 10:17] merge done — 35ca564
