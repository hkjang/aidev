- 과제: 첨부 업로드가 순서 조회 실패를 0번으로 저장하지 않게 한다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `uploadAttachments`는 다음 순서를 읽는 `Scan` 오류를 버려, 조회가 실패해도 새 이미지를 0번으로 저장하고 201 성공을 답한다. 실제 HTTP로 기존 이미지와 같은 0번이 생성됨을 확인했으므로, 저장 전에 조회 오류를 거절하면 잘못된 순서와 성공 응답을 막을 수 있다.
- 수용 기준:
  1) 다음 순서 조회 실패 시 HTTP 500 / `QUERY_FAILED`와 순서 조회 실패임을 설명하는 문구를 반환한다. 트랜잭션·이미지 파일 쓰기로 진행하지 않고 기존 행·순서·파일을 보존한다(빈 디렉터리 생성 여부는 기준 밖).
  2) 정상 업로드는 기존 201 응답 모양과 placement별 max+1 계약을 유지한다. 오류를 일으킨 기존 순서를 정상값으로 PATCH한 뒤 같은 이미지를 재시도하면 한 번만 저장되고 그룹 끝에 놓인다.
  3) 실제 PostgreSQL + `newTestServer` / `App.Handler()`로 재현한 시험이 수정 전 실패·후 통과해야 한다. 거절 뒤 목록의 ID·순서·건수와 디스크 파일 목록이 같고, 기존 이미지 GET은 200과 원본 바이트를 유지함을 증명한다. AFTER 정상 흐름과 BEFORE가 독립적으로 번호를 세는 회귀도 검사한다.
- 건드릴 파일:
  - `internal/app/attachments.go:uploadAttachments` (현재 269행) — `_ = ...Scan(&nextOrder)`를 명시적 오류 검사로 바꿔 500 `QUERY_FAILED` 후 return. 기존 count 조회 실패 처리(214행)와 같은 방식. 필요 시 logger에는 error·reportId·trace만 남긴다.
  - `internal/app/attachmentuploadorder_test.go` (신규) — 위 실패·보존·복구 및 그룹별 정상 순서 시험. `// guards: uploadAttachments`를 실제 해당 함수를 실행하는 시험에 단다.
  - 프로덕션 1파일 + 시험 1파일. 다른 변경은 이번 과제에 포함하지 않는다.
- 검증 명령: 아래 실행 순서를 따른다. 저장소 루트 기준이며 모든 DB 시험은 DSN 없이 SKIP이면 검증 실패로 취급한다.
- 위험과 피할 것: `updateAttachment`의 명시적 sortOrder 저장 계약, 자동 재번호 매김, 동시 업로드 직렬화·개수 제한, 전체 이미지 디코딩, deleteAttachment, imports.go를 함께 고치지 않는다. auth/session·migrations·workflows·프런트·의존성 변경은 범위 밖이다. 큰 순서를 금지하거나 bigint로 바꾸면 이번 시험만 피해 가고 일반 조회 오류 무시가 남는다. 기존 파일 정리·트랜잭션 경계를 옮길 이유도 없다.
- 차선 후보: `adminUsers`의 알 수 없는 role 필터를 400으로 거절 (가치 2 / 위험 1 / S) — `internal/app/admin.go:adminUsers`가 유효하지 않은 토큰을 전부 버려 필터를 제거함을 확인했다. 1순위 오류 무시가 이미 별도로 해결된 경우에만 선택하고, 유효한 단일·복수·소문자 role과 빈 필터의 기존 동작을 지킨다. reviewer 500명 상한은 별개 과제다.

범위와 재현 근거 (2026-10-09, main@f291f8c / v0.318.0):
- `internal/app/migrations/008_attachments.sql`의 sort_order는 PostgreSQL integer이고, `docs/openapi.yaml:647`의 PATCH sortOrder는 integer이며 상한이 없다. SQL/계약은 읽기만 했다.
- `internal/app/attachmentorder_test.go`의 `uploadCaptures`, `patchCapture`, `listCaptures`, `orderByFilename`을 재사용했다. `httpharness_test.go:newTestServer/request`는 scratch DB와 프로덕션 Handler를 통과한다.
- 순서: 보고서 생성 → a.png/b.png 업로드 → a=0, b=2147483647을 각각 HTTP PATCH → 새 c.png 업로드 → HTTP GET 목록 확인. `max(sort_order)+1`이 integer 범위를 넘는 입력이며 테이블 rename·mock·임의 의존성 주입은 없다.
- 실제 결과: POST **201**, 응답 c.png의 **sortOrder:0**, 목록 **a.png:0 / b.png:2147483647 / c.png:0**. 새 행이 3개라 거절·2개 보존을 요구하는 진단 시험이 실패했다. `probe-result.txt`에 원문이 있다.
- 기존 `TestASentSortOrderIsStoredAsSent`는 같은 실행에서 PASS, 새 `TestScoutUploadOrderQueryOverflow`는 FAIL, 패키지 3.190초/exit 1. 이는 수정 전 결함 증거이며 전체 시험 실패를 뜻하지 않는다.
- 보관한 `probe_test.go` + `overlay.json`은 진단용이다. 저장소 소스는 변경하지 않았다. overlay는 이 정찰 checkout의 절대 경로를 쓰므로 다른 checkout에서는 Replace 키를 현재 경로로 바꾸거나 위 절차로 정식 시험을 작성한다.
- 실제 UI 클릭/PPTX 출력은 이번에 미실행. 과제의 관찰 결과는 API 성공 오보·저장된 중복 순서이며 UI/PPTX 개선을 검증했다고 주장하지 않는다. 기존 이미지 바이트 보존과 복구 재시도는 구현 시험에서 추가해야 한다.

실행 계획 (구현 전부 미착수; 사람 승인 체크포인트 없음, 각 단계의 실행 결과로 다음 단계 진행):
1. [ ] 신규 시험 파일에 위 재현과 기존 파일 보존 단정을 작성한다. `python3 "$RUN/run-db-check.py" go test ./internal/app -run '^TestAttachmentUploadOrder' -count=1 -v`로 201/새 행 생성 때문에 실패함을 확인한다. 새 시험 이름은 이 prefix를 사용한다. 다른 이유로 실패하면 과제서를 먼저 정정한다.
2. [ ] uploadAttachments의 nextOrder 오류 처리만 구현한다. 같은 명령으로 PASS를 확인하고 정상값으로 복구한 재시도·placement별 순서·기존 이미지 GET을 확인한다. 오류 응답만 단정하는 시험으로 끝내지 않는다.
3. [ ] 관련 시험, 전체 Go 시험, 빌드·정적 검사·가드를 순서대로 실행한다. 실패하면 관련 원인을 조사하고 범위를 자동으로 넓히지 않는다. mutation/authz 도구는 이 회차에 실행하지 않는다.

검증 명령 상세:
```bash
RUN=/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-09-022839-weekly-improve
# weekly-test-pg는 현재 15434에서 실행 중. helper는 docker inspect로 자격을 읽어
# 자식 환경에만 DSN을 전달하며 자격을 출력/저장하지 않는다.
python3 "$RUN/run-db-check.py"  # 정찰 overlay 재현: 수정 전 exit 1이 기대 결과
python3 "$RUN/run-db-check.py" go test ./internal/app -run '^TestAttachmentUploadOrder' -count=1 -v
python3 "$RUN/run-db-check.py" go test ./internal/app -run 'Attachment|Capture|SortOrder|Placement' -count=1
python3 "$RUN/run-db-check.py" go test ./... -count=1
go vet ./...
go build ./...
python3 scripts/openapi-check.py
python3 scripts/paging-check.py
python3 "$RUN/run-db-check.py" python3 scripts/guard-check.py --changed f291f8c
git diff --check
```
- 신규 시험 명령은 구현 후 사용할 계획이다. 전체 Go 시험·vet·build·guard는 정찰에서 실행하지 않았다. 전체 DB 시험은 과거 149~182초였으며 이번 소요 시간은 미확인.
- 정찰에서 관련 Go 시험 `go test ./internal/app -run 'Attachment|Capture|SortOrder|Placement' -count=1`은 실제 DB로 PASS(7.360초), openapi-check(119경로)·paging-check(10목록)·git diff --check도 PASS.
- 처음 관련 시험은 TMPDIR를 /mnt/c 출력 폴더로 지정하여 `TestAttachmentWriteIsAllOrNothing`이 mode 0777/want 0600으로 실패했다. helper에서 임시 경로 강제를 제거하고 기본 Linux 임시 폴더로 재실행해 위 PASS를 확인했다. 이 환경 문제를 고치려고 첨부 권한 코드를 변경하지 않는다.
- helper가 가리키는 로컬 DB가 없으면 실행 환경의 시험용 DSN을 설정하고 `go test ...`/guard 명령을 직접 실행한다. 운영 DB는 사용하지 않는다.

대안 비교와 선택 가정:
- 선택: 조회 오류 시 즉시 거절. 1파일로 일반 오류도 처리하고 현재 PATCH/순서 계약을 보존한다.
- 입력 상한 제한 또는 자동 재번호: 경계값 입력 예방에는 유효하지만 PATCH 계약 변경·이미 저장된 데이터 처리 결정이 필요하고 일반 조회 실패는 해결하지 못한다.
- 부모 잠금/모든 첨부 쓰기 직렬화: 동시 상한·순서까지 해결할 때 검토할 대안이나 이번 오류 처리보다 범위·경합 위험이 크다.
- 현상 유지: 변경 위험은 없지만 이미 재현된 201 오보·중복 순서가 남는다.
- 핵심 가정: 순서를 읽지 못하면 임의 위치에 저장하는 것보다 실패로 답하는 편이 맞다. 이 정책은 같은 함수의 기존 첨부 count 조회 실패 처리와 일치한다.

작업량 산정:
- Bottom-up: 재현/회귀 시험 10분 + 오류 처리 4분 + 보존·복구·그룹 회귀 보강 8분 + 전체 검증/리뷰 8분 = 기본 30분.
- 알려진 불확실성 예비 8분(시험 helper 적응·DB fixture/가드 확인), 합계 38분. 예상 범위 30~43분, 신뢰 중간(판단 기반이며 통계적 신뢰구간 아님). 미지 범위용 관리 예비는 0분이며 새 범위는 다음 회차로 넘긴다.
- 유사 과거 회차는 프로덕션 1파일+실제 HTTP 시험으로 끝낸 첨부 순서 수정(2026-10-05). 당시 총 구현시간 기록은 없어 수치 추정의 독립 검증은 미확인이나 변경 면적은 비교 가능하다.
- 첫 실패 재현 후 가정을 재평가한다. 45분을 맞추려고 필수 검증을 생략하지 말고 범위가 달라지면 기록한다.

적용 스킬: headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md,
headcount/plugins/technology/skills/implementation-planning/SKILL.md,
headcount/plugins/technology/skills/solution-exploration/SKILL.md.
Skill 도구 부재로 `/mnt/c/Users/USER/projects/headcount/` 아래 실파일을 읽었으며, 산정 스킬의 references/sources.md도 확인했다. 외부 비용 추정 표준의 수치나 인용은 사용하지 않았다.
