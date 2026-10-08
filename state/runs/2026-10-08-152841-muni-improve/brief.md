- 과제: HWPX 가져오기 머리말·꼬리말의 200룬 절단에 AI 안내 문구가 붙는 것 고치기 (가치 3 / 위험 1 / 작업량 S)
- 왜: HWPX reader가 길이 제한 없이 돌려주는 머리말·꼬리말을 HTTP 계층이 AI 프롬프트용 `truncateRunes`로 잘라, 200룬을 넘으면 사용자가 쓰지 않은 줄바꿈과 「[…문서 컨텍스트가 길어 일부 생략됨…]」을 저장·반환한다. 문구 없는 절단기로 바꾸면 가져온 페이지에 안내 문장이 끼어들지 않고 일반 수정 API의 200룬 제한과도 맞는다.
- 수용 기준:
  1) 프로덕션 `hwpx.Build`로 만든 머리말 201룬·꼬리말 201룬 파일을 `POST /api/v1/import`로 올리면 DB `documents.page_header/page_footer`와 응답 `pageHeader/pageFooter`가 각각 앞 200룬이며 AI 문구·추가 줄바꿈이 없다. 한글처럼 멀티바이트 입력을 쓴다.
  2) 같은 파일을 `POST /api/v1/documents/{id}/import`로 넣으면 응답 `header/footer`도 같은 값이다. 이 경로는 편집기에 삽입할 데이터를 반환하는 경로이므로 대상 문서의 기존 머리말·꼬리말·본문·revision을 직접 변경하지 않아야 한다.
  3) 두 경로 모두 빈 문자열·짧은 값·정확히 200룬을 보존한다. 첫 199룬 뒤 공백, 그 뒤 글자가 오는 입력은 기존 `cutFilenameRunes` 계약대로 절단 후 끝 공백을 제거한다(결과 199룬). 제목·본문·용지 방향은 기존 동작을 유지한다.
  4) 실제 writer → parser → 라우트 → DB/응답을 지나는 회귀 테스트를 수정 전에 실패시키고 수정 후 통과시킨다. 프로덕션 네 호출만 되돌리면 긴 입력 단정이 다시 실패해야 하며, live 테스트 SKIP을 통과로 취급하지 않는다.
- 건드릴 파일:
  - `internal/httpapi/import_attachments.go:storeImportedDocument`(:184) — `doc.furniture.Header/Footer`의 `truncateRunes(..., 200)` 둘을 기존 `cutFilenameRunes(..., 200)`로 교체.
  - 같은 파일 `(*Server).importIntoDocument`(:352~353) — 응답 `header/footer`의 두 호출도 동일하게 교체. 이유를 설명하는 짧은 주석 추가. 프로덕션 변경은 이 1파일에 한정.
  - `internal/httpapi/import_name_length_live_test.go` — 머리말·꼬리말 회귀 테스트 2개 추가(새 문서 저장 / 기존 문서 삽입 응답). 이 파일의 `contextNoticeFragment`를 재사용한다. 테스트를 더 분리하고 싶으면 새 `import_furniture_length_live_test.go` 한 파일로 대신하되 둘 다 만들지는 않는다.
- 검증 명령:
  - 기존 형식 왕복 확인: `go test -count=1 -v ./internal/hwpx -run '^TestTheHeaderAndFooterAreWrittenAndReadBack$'` (정찰에서 실제 PASS).
  - 새 테스트 이름을 `TestImportedPageFurnitureIsStoredWithoutAContextNotice`, `TestInsertedPageFurnitureIsReturnedWithoutAContextNotice`로 정하고, 전용 DB의 `MUNI_TEST_DSN`을 설정한 셸에서 `go test -count=1 -v ./internal/httpapi -run 'Test(Imported|Inserted)PageFurniture'`.
  - 같은 환경에서 `go test -count=1 -v ./internal/httpapi`, `go test -count=1 ./...`, `go vet ./...`, `gofmt -l internal/httpapi/import_attachments.go internal/httpapi/import_name_length_live_test.go`, `scripts/check-webui-placeholder.sh`. 별도 테스트 파일을 택하면 gofmt 경로도 바꾼다.
  - 정찰 실행 결과: `go test -count=1 -v ./internal/docx ./internal/httpapi` exit 0, 최상위 PASS 270 / SKIP 72 / FAIL 0. MUNI_TEST_DSN 미설정으로 live는 미실행이며 Chromium은 존재한다. placeholder 검사 exit 0. 로그는 같은 회차의 `scout-tests.log`.
- 위험과 피할 것: DOCX로 실패 픽스처를 만들지 말 것 — `internal/docx/import.go:pageFurniture`(:711~712)가 별도 패키지의 문구 없는 `truncateRunes`로 이미 200룬을 만든다. HWPX reader와 writer는 고칠 필요 없다. AI용 `internal/httpapi/ai.go:truncateRunes`, 공용 `export.go:cutFilenameRunes`, 제목/첨부 240룬, 기존 DB 행, mail 오류 문구, auth·migrations·workflows·프런트는 범위 밖이다. `cutFilenameRunes`는 초과 입력을 자른 뒤 끝 공백도 제거하므로 '항상 200룬'이라고 단정하지 말 것. `make build`는 tracked webui 번들을 덮어쓰므로 실행하지 않는다. admin 워크스페이스에 다른 테스트 데이터가 남을 수 있으므로 자기 ID만 검증하고 생성 직후 자기 문서 정리를 t.Cleanup에 등록한다(공용 liveServer 개편 금지).
- 차선 후보: 운영 안내의 외부 PostgreSQL 백업·복구 명령 정정 (3/1/S) — `docs/OPERATIONS.md:42~86`만 우선 변경. README는 외부 PostgreSQL 준비를 명시하고 compose에는 muni만 있으며 Dockerfile도 DB 서버를 포함하지 않는다. 존재하지 않는 postgres 서비스·postgres_data 볼륨 가정을 제거하고, 외부 DB 접속과 별도 복구 대상에 맞춘 명령을 검증한다. 1순위의 HWPX 201룬 전달이 실제로 성립하지 않을 때에만 전환하며 두 과제를 함께 하지 않는다.

확인한 근거와 구현 순서

- 확인: `internal/hwpx/parts.go:builder.furniture`는 문자열을 XML로 escape할 뿐 길이를 자르지 않는다. `internal/hwpx/hwpx.go:importer.furnitureText`(:172)는 문단을 모아 공백으로 연결하고 Parse가 Meta에 그대로 담는다. `internal/httpapi/import_attachments.go:hwpxImport`(:379)는 Meta를 docx.Meta로 옮길 뿐이며, `storeImportedDocument`와 삽입 응답에서 처음 200룬 제한을 적용한다. HWP의 `internal/hwp/control.go:importer.furniture`도 길이 절단이 없다(이번 필수 픽스처는 HWPX만).
- 확인: `internal/httpapi/ai.go:truncateRunes`(:238)는 초과 시 실제로 문구와 줄바꿈을 붙인다. `documents.go`(:352~363)의 일반 수정은 collapseToLine 뒤 200룬 초과를 거절한다. 가져오기 화면에서 저장 버튼이 막히는 구체적 UI 증상은 미확인이다.
- 미확인: 201룬 HWPX의 실제 라우트 재현 및 DB 결과. 정찰은 코드를 수정하지 않고 기존 테스트만 실행했다. 기존 `internal/hwpx/hangul_writes_test.go:TestTheHeaderAndFooterAreWrittenAndReadBack`는 짧은 값의 writer/parser 왕복을 증명하지만 201룬까지 증명하지는 않는다.
- [pending] 1. `hwpx.Build(richdoc.Doc(richdoc.Paragraph(...)), hwpx.Options{Header: ..., Footer: ...})`로 실제 파일을 만든다. `hwpx.Parse` 결과가 201룬인지 먼저 단정하고, `newServerUnderTest`·`adminWorkspace`·`importFile`·`importIntoDocument`를 재사용해 위 두 테스트를 작성한다. 헬퍼 위치는 각각 `provision_live_test.go`, `import_word_live_test.go`, `import_markdown_live_test.go`, `import_into_document_live_test.go`이며 모두 읽었다. 증명: 위 새 테스트 명령으로 먼저 FAIL을 확보한다. 체크포인트: 사람 승인 없음; 입력이 parser에서 이미 잘리면 가정을 수정한 뒤 진행한다.
- [pending] 2. 네 호출을 바꾸고 같은 새 테스트 명령으로 PASS 및 경계·불변 감시자를 확인한다. 체크포인트: 사람 승인 없음; 공용 헬퍼 본문 변경 없이 해결되는지 확인한다.
- [pending] 3. 네 호출만 잠시 원복해 새 테스트가 다시 실패하는지 확인한 뒤 수정을 복구한다. 위 패키지·전체 Go·vet·gofmt·placeholder 명령을 실행하고 PASS/SKIP/FAIL을 보고한다. 체크포인트: 사람 승인 없음; 예상 밖 실패는 환경/회귀를 구별하고 계획에 기록한다. 프런트 검사는 프런트를 변경하지 않는 이번 범위에서 제외한다.

대안과 추정 근거

- 선택: HTTP의 네 호출만 기존 문구 없는 절단기로 교체. 새 추상화 없이 저장·응답 계약 두 곳을 함께 맞추며 프로덕션 1파일로 끝난다.
- 대안: 각 형식 reader에서 200룬 제한을 통일하면 이 결함도 막지만 HWP/HWPX 라이브러리 계약을 함께 바꾸고 새 reader의 HTTP 방어가 남지 않는다. 별도 page-furniture 정규화 계층은 이후 요구가 커질 때 재검토하고 이번에는 제외한다. 현상 유지도 가능하나 사용자 원문에 AI 안내가 들어가는 경로를 남긴다.
- 가장 중요한 가정: HWPX writer/parser가 201룬을 그대로 전달한다. 정적 코드가 이를 뒷받침하며 첫 테스트에서 실행으로 확정한다. DOCX는 이 가정이 성립하지 않는다.
- bottom-up 예상: 입력·회귀 테스트 12~17분, 호출 수정·설명 3~5분, 인과·회귀 검증 10~13분 = 기본 25~35분. 알려진 환경 변동(전용 PostgreSQL 준비·테스트 조정)에 contingency 5~10분, 합계 30~45분. 경험적 중간 확신의 범위이며 통계적 80% 신뢰구간은 아니다. 관리 예비시간은 0분(45분 밖의 추가 범위는 다음 회차).
- 비교 기준: 2026-09-29의 여섯 이름 절단 호출 수정과 동일한 결함·헬퍼·live 검증 구조이며 이번은 네 호출·프로덕션 1파일이다. 과거 소요시간 데이터가 없어 유사 사례로 분량만 교차 확인했고 분당 생산성은 가정하지 않았다. DB나 201룬 재현이 막히면 첫 단계에서 재추정한다.
- 적용 스킬: `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration`. Skill 호출 도구가 없어 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/` 아래 해당 SKILL.md를 직접 읽었다. 추정은 정찰자의 코드 기반 판단이며 외부 비용 기준을 적용하지 않았다.
