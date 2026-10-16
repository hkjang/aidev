# 회차 노트 2026-10-16-230953-weekly-improve — weekly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:10] base pinned — main@37efe44
- [러너 23:10] autonomy release — 

## 정찰 노트
- 고른 이유: `deleteAttachment`(attachments.go:498)의 참조 수 Scan 오류 무시는 유일하게 **되돌릴 수 없는 피해**(아직 쓰이는 이미지 파일 삭제)를 내는 미해결 후보이고, 프로덕션 1파일·S 다. 검토 책임자 선택기(limit=500 잘림, 가치 3)는 프런트 렌더 시험 기반이 없어 "실행 테스트로 증명" 요구를 만족시킬 수 없어 제쳤고, 동시 업로드 상한·잘린 이미지·메일 재큐잉은 계약 결정이 선행돼야 해 한 세션에 들어가지 않는다.
- 추측으로 적은 것: **재현 기법 전체가 설계이지 관측이 아니다.** `weekly-test-pg` 가 Exited(255)이고 비대화형이라 `docker start`/`docker exec` 승인이 거부돼 DB 를 한 번도 띄우지 못했다. 트리거 안의 `ALTER TABLE ... RENAME` 허용 여부가 과제 성립 조건이라, 과제서에 30초 확인용 SQL 과 실패 시 대체 2안·포기 조건을 명시했다. PPTX 누락 피해도 코드 유추다.
- 구현자가 조심할 것: 응답을 500 으로 바꾸지 말 것(행 DELETE 는 이미 커밋됨 — 거짓말이 된다). 시험이 만든 트리거·rename 은 반드시 `t.Cleanup` 으로 되돌릴 것(공유 DB). `t.Parallel()` 금지, TMPDIR 을 /mnt/c 로 두지 말 것, 감사 details 에 `stored_path` 금지.
- 반복 우려: 이것으로 "질의 오류를 버리는 Scan" 계열이 여섯 번째다. 비평이 반복을 지적할 수 있으니 커밋 메시지·PR 에 피해가 표시 오류가 아니라 **파일 소실**임을 분명히 적을 것.
- [러너 23:16] scout done — 첨부 삭제가 참조 수를 읽지 못하면 아직 쓰이는 이미지 파일을 지우지 않게 한다 (가치 3 / 위험 1 / 작업�

## 구현 노트
- `deleteAttachment`(attachments.go:498)의 참조 수 `Scan` 오류 무시를 명시적 검사로 바꿨습니다. 집계가 실패하면 `os.Remove` 를 건너뛰고 `attachmentId`·`error`·`trace` 만 `Warn` 으로 남깁니다 — 행 DELETE 는 이미 커밋됐으므로 응답은 204 그대로입니다. 프로덕션 1파일 + 시험 1파일(신규 `attachmentdeletekeepsfile_test.go`), 커밋 c9feef4.
- 과제서가 "미확인(가장 중요)" 으로 남긴 재현법을 먼저 실측했습니다: `docker start weekly-test-pg` 후 임시 테이블로 확인한 결과 `BEFORE DELETE ... FOR EACH ROW` 트리거 안의 `ALTER TABLE ... RENAME` 은 PG16 에서 허용되고 `DELETE ... RETURNING` 은 성공하며 뒤따르는 `count` 만 실패합니다. 대체안 ①②는 불필요했습니다.
- **확신 없는 곳 / 검증 못 한 것**: ① UI 클릭 경로와 PPTX 출력은 이번에도 보지 않았습니다 — 수용 기준의 "덱에서 캡처 누락" 은 여전히 코드 유추이고, 시험이 단정한 피해는 이미지 GET 404·목록 `available:false` 까지입니다. ② 시험의 마지막 단정(`stored_path` 가 로그에 없음)은 고치기 전에는 *다른 핸들러*(`serveAttachment` 의 `read attachment` ERROR, `storedPath=` 를 로그에 씁니다)가 찍은 이름 때문에 실패했습니다. 고친 뒤에는 파일이 멀쩡해 그 줄이 아예 안 나오므로 통과합니다 — 즉 이 단정은 제 새 경고의 내용을 고정하지만, **기존 `serveAttachment` 가 `storedPath` 원문을 로그에 쓰는 것 자체는 손대지 않았습니다**(감사 `details` 는 아니고 로그이며, "넓히지 말고 좁히는" 지시에 따라 별건으로 남겼습니다 — 비평가가 문제로 볼 수 있는 자리입니다). ③ 트리거가 테이블을 숨긴 짧은 창 동안 기동된 worker 가 `report_attachments` 를 읽으면 실패할 수 있습니다. 전체 suite 와 지정 범위를 각각 돌려 재현된 적은 없지만 구조적으로 배제한 것은 아닙니다.
- 일부러 하지 않은 것: 응답을 500 으로 바꾸지 않았습니다(행 DELETE 가 커밋돼 "삭제되지 않았다" 는 거짓이 됩니다). `cleanupAttachmentFiles`·`updateAttachment`·업로드 경로, `migrations/`, 프런트는 한 줄도 건드리지 않았습니다(그래서 `npm` 검증 미실행). 차선 후보 `adminUsers` 역할 필터는 ideas.json 에 pending 으로 남겼습니다.
- 다음 역할이 조심할 것: 새 시험 3개는 **DB 가 있어야 돕니다**(`weekly-test-pg`, pgvector/pgvector:pg16, 15434 — 이번에 `docker start` 로 올렸고 지금 Up). DSN 없이 초록을 통과로 읽지 마세요. TMPDIR 을 `/mnt/c` 아래로 두지 마세요(`attachmentwrite_test.go` 가 파일 모드로 깨집니다). 커밋 전 `mutation-check`/`authz-check` 금지(소스 제자리 변경).
- 검증 실측: `gofmt -l internal/app` 빈 출력, `go build ./...`, `go vet ./...`, 지정 범위 11.643s 통과, 전체 `go test ./... -count=1` → `ok github.com/hkjang/weekly/internal/app 212.702s` EXIT=0, openapi-check 119경로, paging-check 10목록, `guard-check.py --changed 37efe44` 17개 도달(새 시험 3개 모두 `deleteAttachment` 62/67/67%), `git diff --check` 통과. 고친 줄을 되돌려 같은 실패 재현까지 확인하고 원복했습니다.
- [러너 23:26] brief accepted — 채택 — 과제서의 근거(498행 `_ =`, 내용 주소 공유 파일, 204·무로그)가 현재 코드와 실제 HTTP 응답에 그대로 일치했고, 과�
- [러너 23:30] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 고친 줄만 되돌린 일회용 복제본에서 시험 1이 실패하는 것을 직접 재현했습니다(공유 파일 os.Remove, available:false, GET 404 ATTACHMENT_FILE_MISSING) — 수정 전에는 통과하지 않고 실패 증상이 고치는 피해와 일치합니다. 리뷰어가 직접: gofmt/vet 깨끗, 지정 3개 통과(2.68s), 전체 `go test ./...` → ok internal/app 184.442s EXIT=0(실제 DB), openapi 119·paging 10·guard 17/17 도달(SKIP 0).
- 못 본 것: UI 클릭 경로와 PPTX 출력(여전히 코드 유추), 프런트 검증(변경 없음), 트리거 창에서의 worker 경합(세 번의 실행에서 재현 안 됨, 구조적 배제는 아님).
- 승인이어도 남는 우려: ① 집계 실패로 건너뛴 파일이 마지막 참조였다면 영구 고아가 됩니다 — `cleanupAttachmentFiles`(attachments.go:637-664)는 보고서 행이 사라진 디렉터리만 회수합니다(다음 회차 후보). ② `testServer.logged` 는 줄 단위가 아니라 버퍼 전체 부분문자열 매칭이라 시험 1의 로그 단정이 읽히는 것보다 약합니다(결정적 단정은 견고). ③ `serveAttachment` 는 아직 `storedPath` 원문을 로그에 씁니다(별건).
- 릴리즈 노트: "덱에서 캡처 누락" 을 측정된 사실로 쓰지 마세요 — 증명된 범위는 파일 소실·이미지 404·목록 available:false 입니다.
- 판정 approve / risk low / blocking 없음(인가·개인정보·비밀값 변화 없음, 질의 파라미터 바인딩 유지).
- [러너 23:36] review approved — 리뷰 승인 (risk=low)
- [러너 23:36] pr created — https://github.com/hkjang/weekly/pull/34
- [러너 00:00] ci passed — 검사 1개 모두 success
- [러너 00:00] merge done — c9feef4

## 릴리즈 노트
- v0.320.0 을 냈습니다 — 커밋 848f6b9(`chore: v0.320.0 을 냅니다`), 주석 태그 `v0.320.0`. detached HEAD 에서 커밋·태그했고 **원격에 아무것도 보내지 않았습니다**. 증가 패턴(0.317→0.318→0.319)을 따라 마이너만 올렸습니다.
- 버전 아홉 곳(VERSION, frontend/package.json+lock, docs/openapi.yaml, deploy 3개, README, 두 안내서) + `.github/release-notes/v0.320.0.md` 신규 + 로드맵 본문/HTML/PDF + 두 안내서 HTML. 파일 집합이 v0.319.0 릴리즈 커밋과 정확히 같은 16개입니다.
- **비평 노트의 지시를 반영했습니다**: "덱에서 캡처 누락" 을 측정된 사실로 쓰지 않고 코드 유추로 명시했습니다(릴리즈 본문·로드맵·커밋 메시지 세 곳). 또 처음 초안에 "남은 파일은 cleanupAttachmentFiles 가 정리할 수 있습니다" 라고 잘못 적었던 것을 비평 노트 ①(637-664행은 보고서 행이 사라진 디렉터리만 회수)에 맞춰 "영구 고아로 남는다, 회수 보강은 범위 밖" 으로 고쳤습니다. `serveAttachment` 의 `storedPath` 로그 별건도 범위 밖으로 적었습니다.
- 검증 실측: 이 릴리즈만을 위해 만든 별도 DB(`weekly_rel320`, weekly-test-pg:15434)에서 전체 `go test ./... -count=1` → `ok internal/app 170.386s` EXIT=0. 새 시험 3개를 `-v` 로 다시 돌려 SKIP 아닌 PASS 확인(0.93s/0.36s/0.33s). go vet·build, gofmt, `git diff --check`, version-check(아홉 곳), openapi-check(119경로), paging-check(10목록), modal-close-check, `guard-check.py --changed 37efe44`(23개 도달), 프런트 npm ci·lint·두 시간대 각 173개·prod build·`scripts/build.sh` 통과. backup-check 는 호스트에 psql 이 없어 `docker cp` 로 시험 컨테이너에서 돌려 통과(7/7).
- 함정 하나: `scripts/build.sh` 가 **추적되는** `cmd/weekly/web/index.html` 의 자산 해시를 고칩니다(assets/ 는 ignore). 이전 릴리즈 커밋들에 없는 파일이라 되돌려 제외했습니다 — 그대로 커밋하면 저장소에 없는 자산을 가리킵니다. 끝나고 dist·release·frontend/dist·web/assets 를 지워 트리는 깨끗합니다.
- 실행하지 않은 것: mutation/authz(제자리 소스 변경), `release-check.sh`(gh 없음, advisory), install-check·로컬 docker build(이전 릴리즈도 안 함 — 태그 워크플로가 이미지와 자산을 만듭니다).
- **assets 는 빈 배열, github_release 는 false** 입니다. `.github/workflows/release.yaml` 이 태그 푸시로 돌아 VERSION·본문 존재를 검증하고, buildx 로 이미지를 만들어 `weekly-v0.320.0.tar.gz` 를 `gh release create` 로 올립니다. 사람/러너가 할 일은 커밋과 태그 푸시뿐입니다.
- 임시 DB(`weekly_rel320`)와 컨테이너의 `/tmp/relcheck` 는 지웠습니다. `weekly-test-pg` 는 Up 그대로 둡니다.
- [러너 00:24] release published — v0.320.0
- [러너 00:25] assets verified — v0.320.0 자산 1개 (이전 v0.319.0: 1)
