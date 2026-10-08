# 회차 노트 2026-10-09-022839-weekly-improve — weekly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:28] base pinned — main@f291f8c
- [러너 02:28] autonomy release — 

## 정찰 노트
- nextOrder 오류 무시를 선택: 공개 PATCH 경계값으로 201/중복 0/새 행 3개를 실제 HTTP 재현했고, 프로덕션 1파일로 좁힐 수 있어 동시 상한·UI 시험 기반 확장보다 위험이 낮다.
- 미확인: UI/PPTX 출력, 새 시험의 기존 이미지 바이트 보존·복구 재시도는 구현 단계에서 증명. 관련 Go 시험은 7.360초 PASS, 전체 Go 시험은 이번 미실행.
- 주의: PATCH 상한 제한/재번호 매김으로 우회하지 말고 Scan 오류를 처리할 것. 테이블 rename·제자리 mutation·t.Parallel은 불필요. /mnt/c TMPDIR는 권한 시험을 깨므로 기본 Linux 임시 폴더 사용.
- brief 초안을 먼저 저장한 뒤 HTTP 재현 결과로 덮어썼으며, 기존 아이디어 12건 유지·재평가와 신규 후보 3건 및 과거 해결 2건을 기록했다. 코드 변경·커밋 없음.
- [러너 02:37] scout done — 첨부 업로드가 순서 조회 실패를 0번으로 저장하지 않게 한다 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- cdc51f0: nextOrder Scan 실패 시 HTTP 500 QUERY_FAILED로 즉시 거절; attachments.go 1개와 attachmentuploadorder_test.go 1개만 변경.
- 공개 PATCH sortOrder=2147483647의 max+1 오버플로로 재현. red-tests.txt → green-tests.txt → 원본 복원 reverted-tests.txt에서 같은 실패를 재확인하고 수정본 복구.
- 실제 PostgreSQL/newTestServer/Handler로 거절 전후 ID·순서·건수·파일 목록·기존 이미지 200/바이트 보존, 정상 PATCH 후 같은 이미지 단일 재시도, BEFORE/AFTER 독립 순서·201 응답을 검증.
- 전체 Go 177.632s, 관련 8.947s, 신규 1.622s PASS; vet/build/OpenAPI 119경로/paging 10목록/guard 12개/gofmt/diff 검사 PASS(implementation-checks.txt 및 *-tests.txt).
- 확신 없는 곳·미검증: UI 클릭/PPTX 출력은 별도 검증하지 않았음. 정수 오버플로 외 DB 장애 종류는 따로 유발하지 않았으며 같은 Scan 오류 반환 분기가 처리함.
- 일부러 제외: PATCH 상한·재번호·동시 업로드 직렬화·개수 상한·전체 디코딩·삭제·imports·프런트·의존성·릴리즈 변경; 이번 오류 거절 범위를 유지.
- 다음 역할: 실제 시험용 DSN 필수(SKIP은 검증 아님), 기본 Linux 임시 경로 사용, t.Parallel 금지. 지정 세 스킬은 Skill 도구 부재로 headcount의 실파일을 읽어 적용했고 mutation/authz 도구는 실행하지 않음.
- [러너 02:45] brief accepted — 채택 — 현재 코드의 Scan 오류 무시와 실제 HTTP의 201·중복 0번 저장이 정찰 근거와 일치했고 지정 2파일 범위로 수용 기준
- [러너 02:48] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음: diff·커밋·원장 실패 재현·세 부서 스킬·권한/데이터 경로를 확인했고 실제 결함을 찾지 못함.
- 실제 시험용 PostgreSQL에서 신규 2개 포함 관련 25개 PASS(8.160초, SKIP 없음); 오류 시 행·파일·바이트 보존, 복구 재시도, placement 및 타인 접근 차단 확인.
- UI 클릭·새 오류 시나리오 PPTX·오버플로 외 장애 주입은 미검증; 전체 Go/build/vet/guard는 구현 기록만 확인. 저장소 코드는 수정하지 않음.
- 릴리즈/다음 회차: PATCH 최대 순서값 허용·동시 업로드 문제는 기존 범위로 남으며 이번 보장은 순서 조회 실패 시 저장 중단임.
- [러너 02:50] review approved — 리뷰 승인 (risk=low)
- [러너 02:50] pr created — https://github.com/hkjang/weekly/pull/33
