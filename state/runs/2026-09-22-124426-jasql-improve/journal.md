# 회차 노트 2026-09-22-124426-jasql-improve — jasql
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:44] base pinned — main@e9f3fb2
- [러너 12:44] autonomy release — 

## 정찰 노트
- 실제 eval CLI에서 컬럼 누락·SQL 오류가 [ok]로 나와 문서의 MISS 필터에 안 잡힘을 재현하여 선택; 문서 정정·성능 추정보다 직접 관찰 가능한 효과가 크다.
- 한정 컬럼 비교 변경은 docs/evaluation.md의 베어 정규화 계약 때문에 rejected; 기존 9개 후보는 재확인 후 pending 유지.
- 실 Oracle exec:/rows: 오류 재현은 미확인. Missing 공통 규칙만 재사용하고 대역으로 실 DB 검증을 주장하지 말 것; catalog 지표·종료 코드·실데이터·이전 성공 과제는 변경 금지.
- 요청한 부서 스킬/Skill 도구를 제공 목록·로컬 검색에서 찾지 못해 원문 적용 미확인. build/vet/test 통과; 코드·커밋 변경 없음.
- [러너 12:48] scout done — jasql-eval 상세 출력에서 컬럼·SQL 등 진단이 있는 케이스를 MISS로 표시 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- caf7a00: verbose 상태에 len(Missing)>0를 추가하고 MISS 범위와 요약 종료 기준 차이를 문서화했다.
- 실제 CLI를 한 번 빌드하고 t.TempDir JSON으로 7개 fixture × verbose/비verbose 실행; 수정 전 column_only/sql_only만 실패, 수정 후 모두 통과.
- 정상·expected_sql/profile 생략·테이블/지표/조인 실패, 케이스별 상태·missing 보존·JSON 요약·table miss 종료 1 검증.
- go test ./cmd/jasql-eval -count=1, go build ./..., go vet ./..., go test ./..., git diff --check 통과; 기존 패키지는 캐시 결과.
- 확신 없는 곳·검증 못 한 것: 실 Oracle exec:/rows: 재현은 미검증. 요청한 technology 스킬 3개/Skill 도구는 제공 도구·로컬 검색에서 없어 원문 절차·반환 형식 적용 불가.
- 일부러 하지 않은 것: 카탈로그 지표·종료 정책·실데이터·보호 경로·이전 성공 과제·릴리즈 변경은 범위 밖이라 유지.
- 다음 역할 주의: 신규 테스트는 DB 없이 실제 CLI 배선을 검증한다. exec:/rows:는 공통 Missing 조건의 코드 근거만 있으며 실 DB 검증을 주장하지 말 것.
- [러너 12:51] brief accepted — 채택 — 현재 CLI 표시 조건과 공통 Missing 생산 경로가 정찰 근거와 일치하여 카탈로그·요약·종료 정책 변경 없이 구현했
- [러너 12:51] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음: 변경 3파일과 Missing 생성·소비 경로, 문서·커밋 일치, 범위·보안·개인정보·revert 가능성을 확인; 실제 결함 없음.
- 실제 CLI 7개 fixture × verbose/비verbose 테스트 및 diff --check 통과; 수정 전 column_only/sql_only 실패는 조건·단언 대조로 확인(별도 실행 안 함).
- 실 Oracle exec:/rows: 실행은 미검증; 공통 Missing 경로만 확인했으므로 릴리즈에서 실 DB 검증으로 표현하지 말 것.
- 요청한 부서 스킬 3개와 Skill 도구는 제공 목록·로컬 검색에 없어 원문 절차·반환 형식 적용 불가. 코드 변경 없음.
- [러너 12:53] review approved — 리뷰 승인 (risk=low)
- [러너 12:53] pr created — https://github.com/hkjang/jasql/pull/4
- [러너 12:57] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
