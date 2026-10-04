- 과제: 첨부 정리 작업이 첫 업로드의 저장 중인 이미지를 지우지 않게 한다 (가치 4 / 위험 2 / 작업량 M)
- 왜: `cleanupAttachmentFiles`는 첨부 행이 0건인 보고서 디렉터리를 지우지만 `uploadAttachments`는 파일을 먼저 쓰고 행을 커밋하므로, 실제 DB로 재현한 첫 업로드가 201 성공 직후 이미지 GET에서 404 `ATTACHMENT_FILE_MISSING`을 냈다. 삭제된 보고서의 디렉터리만 정리하도록 기준을 바꾸면 살아 있는 보고서의 업로드를 보존하면서 보고서 삭제 후의 파일 정리도 유지할 수 있다.
- 수용 기준:
  1) 실제 `newTestServer`의 첫 첨부 POST가 DB INSERT에서 잠금 대기 중일 때 정리를 실행해도, 잠금 해제 후 201·첨부 목록 `available:true`·이미지 GET 200과 원본 바이트 일치가 모두 성립한다. 같은 시험이 수정 전에는 반드시 실패한다.
  2) 실제 HTTP로 보고서를 만들고 이미지를 올린 다음 `DELETE /api/v1/reports/{id}?version={현재버전}`으로 삭제한다. 첨부 행의 cascade 삭제를 확인하고 정리를 실행하면 해당 디렉터리가 없어지며, 다른 살아 있는 보고서의 파일과 이미지 GET은 정상이어야 한다.
  3) 보고서 존재 여부를 확인하지 못한 경우 파일을 지우지 않는다. 실제 업로드 뒤 취소된 context로 정리를 호출해 파일 보존을 검사하는 작은 사례로 검증한다. 빈 디렉터리나 첨부 0건이라는 이유만으로 살아 있는 보고서의 디렉터리를 지우지 않는다.
- 건드릴 파일:
  - `internal/app/attachments.go:cleanupAttachmentFiles` (604행) — `SELECT count(*) FROM report_attachments WHERE report_id=$1` 대신 `SELECT EXISTS(SELECT 1 FROM weekly_reports WHERE id=$1)`로 부모 존재를 읽는다. 오류 또는 존재 시 continue, 부모 부재를 확인한 경우에만 기존 RemoveAll을 수행한다. 함수 주석도 삭제된 보고서의 디렉터리를 정리한다는 계약으로 맞춘다. 프로덕션 파일은 이것 하나다.
  - `internal/app/attachmentcleanup_test.go` (신규) — 아래 실제 경합과 삭제 후 정리 시험. `// guards: cleanupAttachmentFiles`를 붙인다. 기존 `httpharness_test.go:newTestServer/uploadMany/request`, `attachments_test.go:samplePNG`, `attachmentupload_test.go:attachmentCount/attachmentFiles`를 재사용한다.
  - `docs/OPERATIONS.md` — 주기적 정리는 삭제된 보고서의 디렉터리를 대상으로 하고, 살아 있는 보고서의 실패 업로드 잔여 파일은 이 정리에서 보존됨을 짧게 적는다. 별도 HTML 재생성 대상 아님.
- 검증 명령 (저장소 루트, 실제 DB 환경 필수):
  - `go test ./internal/app -run 'Test.*Attachment|TestARefusedAttachmentUpload|TestTheMaintenanceSweep|TestMaintenanceSweeps' -count=1 -v -timeout 120s`
  - `go test ./... -count=1`, `go vet ./...`, `go build ./...`
  - `python3 scripts/openapi-check.py`, `python3 scripts/paging-check.py`, `python3 scripts/guard-check.py --changed 8ab3712`, `git diff --check`
  - 정찰의 저장소 무수정 재현을 다시 실행하려면 `python3 /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-05-035730-weekly-improve/run-probe.py`. 이 명령은 현 checkout의 프로덕션 코드에 Go overlay로 시험만 붙인다. 수정 전 exit 1은 기대 실패이며, 구현 후 exit 0이어야 한다.
- 위험과 피할 것: auth·migrations·workflows·업로드/삭제 API 계약은 건드리지 않는다. 업로드 전체 잠금, 프로세스 전역 mutex, mtime 유예, 파일별 새 GC는 추가하지 않는다. 기존 보고서의 실패 업로드 고아 바이트가 보고서 삭제 전까지 남는 비용을 받아들이는 작은 수정이며, 이를 해결하려고 이번 범위를 늘리지 않는다. 보고서 id를 운영 중 재사용하거나 수동으로 되살리는 흐름은 미확인·범위 밖이다. scratch DB만 사용하고 실제 공유 업무 DB에 잠금을 걸지 않는다. 하네스는 전역 파일 경로를 바꾸므로 t.Parallel 금지; goroutine·잠금 해제와 종료 대기를 실패 경로에도 보장한다. mutation/authz 검사는 소스를 제자리 변경하므로 다른 빌드와 병행하지 않는다.
- 차선 후보: `embeddingStatus`가 실패한 통계 질의를 임베딩 0건으로 답하지 않게 한다 (가치 3 / 위험 1 / 작업량 S). 1순위에 기존 미확인 보존 계약이 발견되어 성립하지 않을 때만 사용한다. 단순히 DB가 처음 연결되지 않는다는 이유로 전환하지 말고 아래 확인된 컨테이너를 사용한다. 과거 Confluence 수정과 같은 유형이라 이번에는 실제 파일 유실 방지를 우선한다.

실제 확인한 근거와 재현

- 기준 main@8ab3712, v0.316.0. 정찰 중 저장소 파일 수정/커밋 없음. 초안부터 작성한 뒤 이 최종판으로 덮어썼다.
- `attachments.go:uploadAttachments`의 MkdirAll → 파일 쓰기 → INSERT → Commit 순서와 `cleanupAttachmentFiles`의 첨부 count 0 → RemoveAll을 직접 읽었다. `app.go:maintenance/runMaintenance`는 기동 후 및 30분 간격으로 정리를 부른다. `migrations/008_attachments.sql`은 부모 삭제 시 ON DELETE CASCADE, `001_initial.sql`은 보고서 id bigserial이다. 마이그레이션 변경은 필요 없다.
- `/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-05-035730-weekly-improve/attachmentcleanup_probe_test.go`와 `probe-result.txt`에 실행한 진단 시험과 원문 결과가 있다. 손으로 만든 App이나 파일 대역을 쓰지 않고 실제 New, scratch PostgreSQL, Handler()를 사용했다.
- 결정적 순서: scratch DB의 별도 tx에서 `LOCK TABLE report_attachments IN SHARE MODE` → 실제 multipart POST를 goroutine으로 시작 → `pg_stat_activity`에서 현재 DB의 `INSERT INTO report_attachments%`가 `wait_event_type='Lock'`인 것을 제한 시간 내 확인 → 행 0건/파일 1개 확인 → 실제 `cleanupAttachmentFiles(ctx)` 호출 → tx Commit → POST 완료 → 실제 이미지 GET 검사. SHARE 잠금은 INSERT를 막으면서 정리의 SELECT는 허용했다. 10ms polling은 잠금 상태 관측용이지 임의 sleep 후 성공을 가정한 것이 아니다.
- 관찰: `before cleanup: rows=0 files=[5da9e0cd…png]`; POST `201`과 `available:true`; GET `404 ATTACHMENT_FILE_MISSING`. 새 진단 시험 0.39초 실패, 기존 `TestARefusedAttachmentUploadStoresNoneOfTheImages` 2.26초 통과, 패키지 2.705초. 이것이 선택 근거이며 문자열 검색을 동작 증거로 쓰지 않았다.
- OpenAPI 검사 119경로 통과, paging 검사 목록 10곳 통과. 전체 Go 시험·vet·build·프런트 검증·guard 검사 및 수정 후 성공은 정찰에서 미실행이다. 프런트 변경은 없으므로 npm은 이 과제의 필수 검증에 넣지 않았다.
- 셸의 `WEEKLY_TEST_POSTGRES_DSN`은 미설정이다. 실행 중인 `weekly-test-pg` (15434)의 docker inspect 환경에서 자격을 읽어 자식 프로세스 환경에만 DSN을 넣어 실행했다. 비밀번호는 파일·로그에 쓰지 않았다. DSN 없이 SKIP된 시험은 통합 검증 성공이 아니다.

해법 비교와 범위

- 선택: 부모 보고서 부재를 확인해 삭제한다. 기존 함수 한 개·DB 질의 한 개로 업로드 중인 파일을 지키며, 원래 주석의 보고서 삭제 후 정리 목적과 맞는다. 가장 중요한 가정은 살아 있는 보고서의 무참조 파일 회수보다 정상 업로드 보존을 우선한다는 것이다.
- 대안: 업로드/삭제/GC에 공통 잠금 도입. 살아 있는 보고서의 잔여 파일도 치울 수 있지만 여러 쓰기 경로의 잠금 순서·다중 프로세스까지 다뤄야 하므로 45분 범위를 넘긴다.
- 대안: 디렉터리 수정 시간에 유예를 둔다. 수정 파일 수는 적지만 느린 업로드와 오래된 빈 디렉터리를 정확히 구분하지 못해 보장할 수 없다.
- 유지: 코드 변경 없이 스윕을 끄거나 재업로드를 안내한다. 첨부 유실과 삭제 보고서 파일 누적을 해결하지 못하므로 선택하지 않는다.

구현 순서·검토 지점 (구현 세션에서 상태 갱신)

1. [완료] 신규 시험 파일에 위 재현을 옮겨 기존 코드에서 기대 404 실패를 확인한다. 증명: 첫 번째 go test 명령. 사람이 검토하는 체크포인트는 없고, 예상과 다른 실패면 먼저 과제서의 재현 부분을 갱신한다.
2. [완료] `cleanupAttachmentFiles`의 질의와 분기를 수정하고 삭제된 보고서 정리·생존 보고서 보존·질의 실패 보존 사례를 완성한다. 증명: 같은 go test가 성공하고 HTTP 바이트까지 일치. 다음 단계 조건은 시험 성공이다.
3. [완료] 운영 문서를 짧게 수정한 뒤 전체 Go·계약·guard 검증을 순서대로 실행한다. 증명: 위 검증 명령 결과를 기록. 사람 승인 대기 없이 끝내며, 미실행을 통과로 기록하지 않는다. 단계 완료 표시를 이 계획에 갱신한다.

추정 근거 (회사 스킬 적용)

- bottom-up 기본 30~35분: 재현 시험 정리 8~10분, 프로덕션 수정 3~4분, 삭제/보존 시험 8~9분, 문서·전체 검증 11~12분. known contingency 5~8분은 잠금/worker 정리와 DB 실행 변동에만 사용한다. 합계 35~43분, 80% 정도의 주관적 신뢰이며 실측 통계는 아니다. management reserve는 0분; 새 범위 발견 시 추가 작업을 끼워 넣지 않고 재계획한다.
- 교차 확인: 과거 첨부 요청 원자성 회차도 프로덕션 1개+시험 1개였으며 전체 DB 시험 148.655초였다. 이번 기본 변경은 더 작지만 경합 시험이 필요하다. 과거 총 구현 시간은 미확인이라 유사 사례에서 45분 완료율을 지어내지는 않는다. 이번 진단 2.705초와 실행 중인 DB가 추정의 환경 근거다.
- 적용한 원본: `/mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md`, `/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md`. Skill 전용 도구가 노출되지 않아 로컬 원본을 읽었다. estimating의 references/sources.md도 확인했으며 외부 원가 기준이나 통계 수치를 차용하지 않았다.

구현 완료 증거 (2026-10-05, 6dbab0b)
- 1단계: red-tests.txt, red-complete-tests.txt에서 첫 업로드 GET 404와 생존 보고서 빈 디렉터리 삭제 실패 확인.
- 2단계: green-tests.txt의 지정 시험 12개 통과(4.051s); revert-tests.txt의 원래 코드 재실패 후 수정본 복원.
- 3단계: all-tests.txt의 전체 Go 시험 통과(166.788s), vet.txt/build.txt/openapi.txt/paging.txt/guard.txt/diff-check.txt 모두 EXIT=0; probe-after.txt의 정찰 overlay도 EXIT=0.
- 프로덕션 1파일·시험 1파일·문서 1파일. 프런트/릴리즈 경로 무변경으로 npm/릴리즈는 실행하지 않았고 OPERATIONS.md는 HTML 재생성 대상이 아님.
