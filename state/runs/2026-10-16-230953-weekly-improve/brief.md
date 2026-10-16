- 과제: 첨부 삭제가 참조 수를 읽지 못하면 아직 쓰이는 이미지 파일을 지우지 않게 한다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `deleteAttachment`(internal/app/attachments.go:498)가 `SELECT count(*) FROM report_attachments WHERE stored_path=$1` 의 `Scan` 오류를 `_ =` 로 버려, 질의가 실패하면 `remaining` 이 0 으로 남고 바로 아래 `os.Remove(safeAttachmentPath(storedPath))` 가 실행된다 — "아무도 참조하지 않는다" 와 "참조 수를 읽지 못했다" 가 같은 값이 되어 **되돌릴 수 없는 파일 삭제**가 일어난다. 저장 경로는 `reportID/sha256.ext`(attachments.go:258)로 내용 주소라서 같은 보고서에 같은 이미지를 두 번 올리면 두 행이 한 파일을 공유하고, 그 중 하나를 지우는 요청에서 질의가 실패하면 **살아 있는 다른 행의 이미지가 사라진다**(이미지 GET 이 `ATTACHMENT_FILE_MISSING` 404, 목록은 `available:false`, PPTX 내보내기에서 해당 캡처 누락). 삭제 응답은 204 이고 로그도 남지 않아 운영자는 원인을 알 수 없다.
- 수용 기준:
  1) 참조 수 질의가 실패하면 파일을 지우지 않는다. 디스크의 `<attachments>/<reportID>/<sha256>.<ext>` 가 그대로 남고, 남은 행의 이미지 GET 이 200 과 **원본 바이트**를 돌려주며 목록의 `available` 이 true 다.
  2) 행 삭제 자체는 이미 커밋됐으므로 응답 계약은 깨지 않는다 — 204(또는 구현자가 택한 대안)이고, 질의 실패는 `a.logger.Warn`/`Error` 로 `attachmentId`·`error` 와 함께 남는다(식별자만. `stored_path` 원문은 details/감사 기록으로 넘기지 말 것 — 운영자 반복 지시).
  3) 정상 경로는 그대로다: 참조가 남아 있으면 파일 보존, 마지막 행을 지우면 파일 삭제, 이미 파일이 없으면 `os.ErrNotExist` 를 무시하고 204. 세 경로를 시험이 각각 단정한다.
  4) 시험은 수정 전에 실패하고 수정 후 통과하며, 고친 뒤 그 한 줄을 되돌리면 같은 실패가 다시 난다(이 저장소의 기존 절차).
- 건드릴 파일:
  - `internal/app/attachments.go:deleteAttachment`(477~506행) — 498행 `_ = a.db.QueryRow(...).Scan(&remaining)` 를 명시적 오류 검사로 바꾸고, 오류면 `os.Remove` 를 건너뛰고 경고를 남긴다. 다른 줄(소유권 검사·DELETE·audit·204)은 건드리지 않는다.
  - `internal/app/attachmentdeletekeepsfile_test.go`(신규) — 아래 "검증 명령" 의 시험. 실제 `newTestServer`/`app.Handler()` 와 실제 PostgreSQL 로 HTTP 를 거친다(손으로 주입한 대역·소스 문자열 검사 금지 — 운영자 반복 지시).
  - 프로덕션 파일은 이 1개뿐. 프런트·`migrations/`·auth 는 한 줄도 건드리지 않는다(프런트 변경이 없으면 `npm` 검증은 불필요).
- 질의 실패를 만드는 방법(정찰이 실제로 확인함 — 아래 "확인된 것" 참조):
  - 보고서에 **같은 이미지 바이트를 두 번** 업로드해 `stored_path` 를 공유하는 두 행을 만든다(업로드 경로가 sha256 로 이름을 정하므로 두 번째 업로드도 같은 상대 경로를 쓴다. attachments.go:255-259).
  - 삭제 요청 전에 `report_attachments` 에 `BEFORE DELETE ... FOR EACH ROW` 트리거를 걸고, 트리거 함수가 `EXECUTE 'ALTER TABLE report_attachments RENAME TO report_attachments_hidden'` 을 수행하게 한다. 그러면 **DELETE ... RETURNING stored_path 는 성공**하고(같은 문장·같은 트랜잭션), 커밋 뒤 이어지는 `SELECT count(*) FROM report_attachments ...` 만 `relation ... does not exist` 로 실패한다 — 10-06 회차가 쓴 "앞 단계는 성공하고 뒤따르는 집계만 실패" 모양을 DDL 로 만든 것이다.
  - `t.Cleanup` 에서 트리거·함수를 drop 하고 테이블 이름을 되돌린다. (주의: 단순한 테이블 rename 만으로는 DELETE 가 먼저 실패해 404 가 되므로 재현이 성립하지 않는다.)
  - 복구 뒤 **남은 행의 이미지 GET** 으로 바이트 보존을 확인하는 것이 핵심 단정이다.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `gofmt -l internal/app` (빈 출력), `go build ./...`, `go vet ./...`
  - `go test ./internal/app -count=1 -run 'AttachmentDelete|Attachment|Capture|SortOrder|Placement'` — 지정 범위. 정찰에서 이 패턴군은 약 8~9초.
  - 전체: `go test ./... -count=1` (DSN 필요. `internal/app` 가 160~180초)
  - `python3 scripts/openapi-check.py` (119경로), `python3 scripts/paging-check.py` (10목록), `python3 scripts/guard-check.py --changed 37efe44` (DB DSN 필요 — 새 시험이 `deleteAttachment` 를 실제로 지나는지 확인)
  - `git diff --check`
  - DSN: 시험 DB 컨테이너 `weekly-test-pg`(pgvector/pgvector:pg16, 15434). **이번 회차 확인: 이 컨테이너는 `Exited (255) 3 days ago` 이므로 먼저 `docker start weekly-test-pg` 가 필요하다**(정찰 세션에서는 승인을 받을 수 없어 띄우지 못했고, 그래서 이번 회차에 Go 시험을 한 번도 돌리지 못했다). 자격은 `docker inspect` 로 **자식 프로세스 환경에만** 넘기고 파일로 저장하지 말 것. DSN 이 없으면 DB 시험은 전부 SKIP 되므로 "통과" 로 읽지 말 것.
  - TMPDIR 을 `/mnt/c` 아래로 지정하지 말 것 — `attachmentwrite_test.go:TestAttachmentWriteIsAllOrNothing` 이 파일 모드(0777 vs 0600)로 실패한다. 기본 리눅스 임시 경로를 쓸 것.
- 위험과 피할 것:
  - `newTestServer` 가 전역 상태 디렉터리를 바꾸므로 `t.Parallel()` 금지. 기동한 worker 때문에 유지보수 스윕(`cleanupAttachmentFiles`)이 돌 수 있다 — 이 시험은 **부모 보고서를 지우지 않으므로** 스윕은 디렉터리를 건드리지 않는다(10-05 회차 수정). 그래도 디렉터리 존재를 가정하는 단정은 피하고 파일 경로를 직접 확인할 것.
  - 응답을 500 으로 바꾸고 싶은 유혹을 피할 것: 행 DELETE 는 이미 커밋돼 돌이킬 수 없으므로 500 은 "삭제되지 않았다" 는 거짓말이 된다. 바꾸려면 응답 본문·OpenAPI 계약(`scripts/openapi-check.py`)까지 같이 맞춰야 하니, 기본은 204 유지 + 파일 보존 + 경고다.
  - `cleanupAttachmentFiles`·`updateAttachment`·업로드 경로를 같이 고치지 말 것. 같은 파일이지만 각각 다른 계약을 지키고 있고, 과거 지시는 "넓히지 말고 좁히는 방향으로만" 이다.
  - `migrations/` 의 출하 SQL 수정 금지(체크섬 보호). 시험이 만드는 트리거·rename 은 **시험 안에서만** 하고 반드시 `t.Cleanup` 으로 되돌릴 것 — 공유 DB 에 남으면 다른 시험이 통째로 깨진다.
  - 감사 기록 `details` 에 `stored_path` 나 파일 내용을 넣지 말 것(식별자만).
  - 커밋 전에 `mutation-check`/`authz-check` 를 돌리지 말 것(소스 제자리 변경 도구 — 과거 교훈).
- 확인된 것 / 미확인:
  - 확인: attachments.go:489-503 의 현재 코드(위 인용은 이번 회차에 직접 읽음), attachments.go:255-259 의 `reportID/sha256.ext` 내용 주소 경로, `safeAttachmentPath`(524행), 현재 HEAD `37efe44`(v0.319.0), 작업 트리 깨끗함.
  - **미확인(가장 중요)**: 트리거 안의 `ALTER TABLE ... RENAME` 이 Postgres 16 에서 허용되는지 **정찰에서 실행해 확인하지 못했다**. 시험 DB 컨테이너 `weekly-test-pg` 가 3일 전부터 Exited 상태이고 이 세션은 비대화형이라 `docker start` 승인을 받을 수 없었다. 즉 위 재현 기법은 **설계이지 관측이 아니다.** 구현자는 먼저 이것만 30초 안에 확인하라(임시 테이블로 충분, `BEGIN; ... ROLLBACK;`):
    `create temp table t(id int, p text); insert into t values(1,'a'),(2,'a'); create function pg_temp.f() returns trigger language plpgsql as $$ begin execute 'alter table t rename to t_x'; return old; end $$; create trigger tg before delete on t for each row execute function pg_temp.f(); delete from t where id=1 returning p; select count(*) from t;`
    `delete` 가 성공하고 뒤 `select` 가 `relation "t" does not exist` 로 실패하면 기법이 성립한다.
    - 거절되면("cannot ALTER TABLE ... because it is being used by active queries in this session" 류) 차례로 시도하라: ① 같은 트리거에서 `alter table ... rename column stored_path to stored_path_x` (DELETE 문장은 이미 계획·튜플 서술자가 고정돼 있어 통과하고, 뒤따르는 `WHERE stored_path=$1` 만 `column ... does not exist` 로 실패할 가능성이 있다), ② `after delete ... for each statement` 트리거로 같은 DDL, ③ 셋 다 실패하면 **이 과제를 버리고 아래 차선 후보로 교체하라** — 질의 실패를 손으로 주입한 대역이나 소스 문자열 검사로 "증명" 하지 말 것(운영자 반복 지시). 정상 3경로 시험만 남기고 수정을 올리는 것도 금지: 수정 전 실패를 못 보이면 그 수정은 근거가 없다.
  - 미확인: UI 클릭 경로와 PPTX 출력은 이번에 보지 않았다. 수용 기준의 피해 서술 중 PPTX 누락은 코드 경로(`attachmentSlides` 의 `open` 이 파일 부재를 보고함, attachments.go:535)에서 유추한 것이다.
- 차선 후보: `adminUsers` 의 알 수 없는 역할 필터를 조용히 버리지 않는다 (가치 2 / 위험 1 / 작업량 S) — `internal/app/admin.go:481-486` 가 `validRole` 이 거짓인 값을 그냥 건너뛰고, 남은 `roles` 가 비면 `WHERE` 절이 사라져 **전체 디렉터리 첫 페이지**가 돌아온다. 호출자가 검토 책임자 picker(`frontend/src/pages/AdminPage.tsx:207`, `role=TEAM_LEADER,ORG_MANAGER,ADMIN`)라서, 역할 이름이 한 번 어긋나면 일반 사용자가 검토 책임자 후보로 올라온다. 좁은 수정은 "알 수 없는 역할이 하나라도 있으면 400 `INVALID_ROLE`" 하나뿐이고, 빈 `role=` 파라미터(필터 없음)는 지금 동작을 유지해야 한다. HTTP 재현은 정찰에서 하지 않았다.
