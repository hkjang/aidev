- 과제: DOCX 신규 가져오기에서 확정 제목과 같은 첫 H1만 본문에서 제거하기 (가치 3 / 위험 1 / 작업량 S)
- 왜: DOCX 내보내기 `internal/docx/export.go:Build`는 제목을 첫 `Title` 스타일 문단으로 쓰고 `internal/docx/import_blocks.go:importer.paragraph`는 이를 level-1 heading으로 읽지만, 신규 가져오기 `importDocument`는 현재 `parsed.markdown`일 때만 중복 제목을 제거한다. DOCX에도 기존의 정확한 일치 규칙을 적용하면 재가져온 문서의 본문과 검색 텍스트에 제목이 중복되지 않는다.
- 수용 기준: 1) `docx.Build`의 실제 출력(제목 회의록, 본문 문단·H2·목록)을 `회의록.docx`로 신규 업로드하면 제목은 회의록, 첫 블록은 원래 본문이고 `content_text` 및 최초 revision의 본문에도 제거한 제목이 없다. 2) 폼 제목이 다르거나 첫 블록이 H2/문단이거나 뒤쪽에 같은 H1이 있으면 보존하며, 기존 문서에 끼워 넣는 `importIntoDocument`는 첫 제목을 보존한다; 제목만 있는 DOCX는 유효한 빈 문단이 남는다. 3) 실제 writer 출력으로 만든 단위 왕복 및 인증된 HTTP live 회귀 테스트가 변경 전 실패·변경 후 성공을 증명하고 Markdown/넘겨받기 회귀 테스트도 통과한다; 저수준 `docxImport`는 여전히 제목을 읽어야 한다.
- 건드릴 파일: `internal/httpapi/import_attachments.go:upload,parseUpload,importDocument` — `markdown`을 의미에 맞게 `titleInBody`로 바꾸고 현재 참인 md/markdown에 docx만 추가, 최종 제목 결정 뒤·검색 텍스트 추출 전 기존 `dropLeadingTitle` 호출 유지; `internal/httpapi/handoff.go:storeHandoff` — 플래그 이름 참조만 변경; `internal/httpapi/import_markdown.go:dropLeadingTitle` — 필요시 설명만 일반화하고 판정 로직 유지; `internal/httpapi/import_export_test.go` — DOCX 실제 writer 왕복에서 제거한 첫 블록 외 JSON 보존·제목만 있는 파일 검증 추가; `internal/httpapi/import_markdown_live_test.go`의 `importFile`, `firstBlockType` 헬퍼를 재사용하는 새 `internal/httpapi/import_docx_live_test.go` — 실제 파일 업로드·다른 폼 제목·끼워 넣기·저장 검색 텍스트 검증; `docs/USER_GUIDE.md` — 가져오기 안내에 DOCX의 정확히 같은 첫 제목 제거 규칙 한 구절.
- 검증 명령: 저장소 루트에서 `go test ./...`, `go vet ./...`, `gofmt -l internal/httpapi`, `scripts/check-webui-placeholder.sh`. HTTP 검증은 전용 PostgreSQL의 `MUNI_TEST_DSN`을 설정한 상태에서 `go test -count=1 -v ./internal/httpapi`로 실행하고 추가 테스트가 SKIP되지 않았는지 확인한다. DB 준비 예시는 기존 `internal/httpapi/provision_live_test.go` 주석의 `docker run -d --name muni-test -e POSTGRES_PASSWORD=muni -e POSTGRES_DB=muni -p 5433:5432 postgres:16-alpine` 및 `MUNI_TEST_DSN='postgres://postgres:muni@127.0.0.1:5433/muni?sslmode=disable' go test -count=1 -v ./internal/httpapi`이다(이름·포트 충돌은 별도 이름/빈 포트로 조정). 정찰은 DSN 없이 `go test ./...`와 placeholder 검사 통과; live 실행 및 Docker 가용성은 미확인이다.
- 위험과 피할 것: 기준 HEAD는 1547faa이며 HTML 개선 36dccc6과 직전 ZIP 개선은 현재 코드에 없다. 그 과제를 다시 구현하거나 HTML 플래그까지 켜지 말고 DOCX만 추가한다; 이후 병합된 작업 트리라면 기존 titleInBody를 재사용한다. `docxImport` 자체에서 제거하면 기존 왕복 테스트 및 끼워 넣기 계약이 깨진다(`TestDOCXExportImportRoundTrip`는 저수준 파서에 제목이 남는 것을 기대). 파일명 정규화/240자 절단으로 제목이 다르면 그대로 보존하고, 첫 블록을 찾아 건너뛰거나 연속 H1을 모두 지우지 않는다. auth·migrations·workflows·DOCX reader/writer·제목 후보 우선순위·PDF/HTML 정책은 수정하지 않는다. 전용 DB만 쓰고 sealer 공유/테스트 인프라 개편을 섞지 않는다.
- 차선 후보: Markdown 표의 마지막 셀에서 이스케이프된 끝 파이프 보존 (가치 3 / 위험 2 / 작업량 S) — `internal/httpapi/import_markdown.go:splitTableRow`는 `TrimSuffix("|")`를 이스케이프 판단 전에 수행한다. `A | B` / `--- | ---` / `x | tail\|` 입력에서 마지막 셀의 `tail|` 보존을 `internal/httpapi/import_text_test.go`에 먼저 재현하고, 선택적 끝 구분자와 이스케이프된 실제 문자 구별만 고친다. 실제 동적 재현은 정찰에서 미확인; 재현되지 않으면 임의로 고치지 말고 근거를 기록한다.

범위 밖: HTML/PDF 동작, DOCX 파서/내보내기 변경, 제목 추론, auth/migrations/workflows, DB 테스트 기반 재설계. 이번 목표는 신규 DOCX 업로드의 제목 중복만 없애는 것이다.

대안 비교(solution-exploration):
- 선택: 업로드 결과의 기존 플래그를 일반화하여 DOCX에 기존 dropLeadingTitle 적용. 작은 수정이며 Markdown과 같은 사용자 계약을 재사용한다. 첫 H1이 확정 제목과 같으면 제목의 반복으로 취급한다는 기존 규칙이 DOCX에도 적합하다는 전제에 의존한다.
- 파서에서 Title 스타일의 원본 정보를 별도로 전달하여 그 스타일만 제거: 일반 H1 보존을 더 엄격히 할 수 있지만 DOCX Meta/파서 인터페이스 확장과 회귀 검증이 필요하다. 이번 규칙보다 원본 스타일 구분이 제품 요구로 확인될 때 후속 검토한다.
- DOCX 내보내기에서 본문 제목 자체 생략: 가져오기 수정 없이 중복을 막지만 Word로 읽는 사람의 출력물을 바꾸므로 채택하지 않는다.
- 현상 유지와 수동 삭제 안내: 코드 위험은 없지만 모든 왕복에서 사용자가 다시 삭제해야 하므로 선택하지 않는다.

구현 순서(implementation-planning, 모두 pending; 사람이 지켜보지 않는 회차이므로 단계별 사람 승인은 없음):
1. pending — import_export_test.go 및 새 import_docx_live_test.go에서 실제 docx.Build 출력으로 재현한다. 증명: 전용 MUNI_TEST_DSN을 설정한 `go test -count=1 -v ./internal/httpapi -run 'DOCX|Docx'`. 컴파일은 성공하고 신규 중복 제거 assertion만 현재 실패하는지 확인하는 체크포인트; 불일치면 코드를 바꾸기 전에 과제 근거와 상태를 고친다. 이 의도적 실패 확인은 커밋하지 않는다.
2. pending — import_attachments.go의 플래그 일반화와 docx 활성화, handoff.go의 참조 이름을 한 단위로 변경한다. 증명: 같은 DOCX 명령 및 `go test ./internal/httpapi`; 체크포인트: 빌드와 신규 동작/기존 Markdown 동작 성공 후 다음 단계로 간다. 플래그 선언만 바꾸고 참조를 깨뜨린 중간 상태로 남기지 않는다.
3. pending — 다른 제목·H2·뒤쪽 H1·제목만 있는 문서·끼워 넣기·최초 revision 보존 검증을 완성하고 USER_GUIDE 안내를 갱신한다. 증명: 전용 MUNI_TEST_DSN을 설정한 `go test -count=1 -v ./internal/httpapi`; 체크포인트: 신규 live 테스트 SKIP 없음, 파서 수준 제목 보존 테스트 성공.
4. pending — `go test ./...`, `go vet ./...`, `gofmt -l internal/httpapi`, `scripts/check-webui-placeholder.sh` 실행. 체크포인트: 오류/미검증 항목을 기록하고 후속 비평 단계로 넘긴다. 구현자는 완료 여부를 이 계획의 상태와 회차 노트에 갱신한다.

추정 근거(estimating-and-contingency): 정찰이 읽은 파일과 기존 Markdown 개선의 유사 범위를 근거로 bottom-up 추정한다. 실패 재현 8분 + 플래그/참조 수정 7분 + 보존 테스트/문서 10분 + 전체 검증 10분 = 기준 35분. 환경이 준비된 숙련 구현자 기준 실무 예상 범위는 30~45분, 신뢰도 중간이며 통계적 확률을 추정한 값은 아니다. 유사 Markdown/HTML 회차는 실제 writer/HTTP/보존 검증 방식이 비슷하지만 소요시간 기록이 없어 시간 수치의 독립 교차 검증은 불가능하다.
알려진 불확실성 예비(contingency)는 최대 10분: 전용 DB 연결/포트 조정 5분, fixture 정렬/보존 assertion 조정 5분이며 기준 작업시간에 중복 포함하지 않았다. 관리 예비(management reserve)는 이 회차에 배정하지 않는다(0분); 새로운 파서 기능이나 인프라 수리가 필요하면 45분에 억지로 끼우지 말고 범위/추정을 갱신한다. 첫 writer 재현과 첫 DB 연결 성공 시 추정을 재평가한다. DB가 없으면 SKIP를 성공 근거로 쓰지 않고 미검증으로 기록한다. DOCX 재현이 성립하지 않거나 이미 해결됐다면 차선 후보로 전환한다.

적용한 스킬: 전용 Skill 도구는 노출되지 않았으나 파일 검색으로 다음 원문을 찾아 읽고 위 계획에 적용했다.
- [pmo:estimating-and-contingency](/mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md) 및 references/sources.md: 작업 분해, 추정 근거/범위, 알려진 불확실성과 관리 예비 분리. 외부 비용모형이나 편익 수치는 사용하지 않았다.
- [technology:implementation-planning](/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md): 단계별 변경·명령·체크포인트·상태.
- [technology:solution-exploration](/mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md): 관찰 가능한 목표, 다른 접근 비교, 결정의 전제.
