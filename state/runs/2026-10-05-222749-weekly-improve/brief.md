- 과제: 첨부 이미지의 삽입 위치를 바꿀 때 목적지 그룹의 끝으로 보내, 두 이미지가 같은 순서 번호를 갖지 않게 한다 (가치 3 / 위험 2 / 작업량 S)
- 왜: `updateAttachment` 는 `placement` 만 바뀐 요청에서 `sort_order` 를 그대로 둔다. 업로드는 `placement` 를 그 그룹 안에서만 `max(sort_order)+1` 로 세므로 한 번 빠져나간 번호는 다시 쓰이고, 그 번호를 들고 돌아온 이미지가 목적지에 이미 있는 이미지와 **같은 `sort_order`** 가 된다. 그러면 화면과 내보낸 덱의 슬라이드 순서가 사용자가 고른 값이 아니라 `id` 로 결정되고, AttachmentPanel 의 위/아래 이동은 같은 값끼리 맞바꾸므로 **사용자가 그 순서를 고칠 수도 없다**.

- 수용 기준:
  1) 아래 "실패 경로" 4단계를 실제 HTTP 로 거친 뒤, 한 보고서의 한 `placement` 안에 **중복된 `sortOrder` 가 없다**. 마지막에 옮겨 온 이미지가 그 그룹의 가장 큰 `sortOrder` 를 갖는다.
  2) 같은 `placement` 안에서 `sortOrder` 만 보내는 기존 맞바꾸기 PATCH 는 **동작이 변하지 않는다**(보낸 값이 그대로 저장된다). `placement` 와 `sortOrder` 를 함께 보내면 보낸 `sortOrder` 가 이긴다.
  3) 요청한 `placement` 가 **현재 값과 같을 때는** `sort_order` 를 다시 세지 않는다(같은 그룹으로 다시 보내는 PATCH 가 이미지를 맨 뒤로 밀어내지 않는다). 2)와 3)이 회귀를 막는 자리다.

- 실패 경로 (결정적 · 경합 없음 · 전부 프런트가 실제로 내는 요청):
  1. 이미지 2장 업로드 → `AttachmentPanel.tsx:41` 이 항상 `placement=AFTER` 로 보내므로 AFTER: A=0, B=1.
  2. B 를 BEFORE 로 옮긴다(`AttachmentPanel.tsx:146` 의 select — **placement 만** 보낸다) → BEFORE: B=1 / AFTER: A=0.
  3. 이미지 1장 더 업로드 → `attachments.go:269` 가 AFTER 안에서만 `max+1` 을 세므로 C=1 → AFTER: A=0, C=1.
  4. C 를 BEFORE 로 옮긴다 → **BEFORE: B=1, C=1 (중복)**. 이 뒤 `move()`(AttachmentPanel.tsx:85-93)는 `sortOrder` 를 서로 맞바꾸는데 둘이 같은 값이라 **무효 조작**이 되어 사용자가 순서를 바꿀 길이 없다.

- 건드릴 파일 (프로덕션 1개):
  - `internal/app/attachments.go:updateAttachment` (432행의 단일 `UPDATE`) — `sort_order` 를 `CASE` 로 고친다: 보낸 `sortOrder` 가 있으면 그것, 없고 **`placement` 가 현재 값과 다르면** 목적지 그룹의 `coalesce(max(sort_order),-1)+1`(업로드 경로 269행이 이미 쓰는 식), 그 밖에는 `sort_order` 그대로. 한 문장 안에서 끝내면 추가 왕복도 경합도 없다 — PostgreSQL 의 `UPDATE` 는 `SET` 의 하위질의를 변경 전 스냅숏으로 읽으므로 옮기는 행은 아직 출발지에 있어 자기 자신을 세지 않는다. 하위질의는 `report_id` 와 목적지 `placement` 로 좁힐 것.
  - 시험 1개 추가(`internal/app/attachmentorder_test.go` 같은 새 파일). `// guards: updateAttachment` 줄을 붙일 것.

- 검증 명령:
  - `gofmt -l internal/app` (빈 출력), `go build ./...`, `go vet ./...`
  - 지정: `go test ./internal/app -run 'Attachment' -count=1`
  - 전체: `go test ./... -count=1` — **DSN 필수**. 셸에 `WEEKLY_TEST_POSTGRES_DSN` 이 없고, 없이 돌린 SKIP 은 성공이 아니다. `weekly-test-pg` 가 15434 에서 돌고 있으니 `docker inspect` 로 자격을 읽어 자식 프로세스 환경에만 넣고 비밀번호는 출력하지 말 것. internal/app 전체는 약 150~170초.
  - `python3 scripts/openapi-check.py`, `python3 scripts/paging-check.py`
  - `python3 scripts/guard-check.py --changed f7488dd` — **`8ab3712` 아님**. 프로필의 기준 커밋은 낡았고 이번 회차의 base 는 `main@f7488dd` 다.
  - 고친 뒤 `CASE` 를 잠시 원래대로 돌려 같은 실패가 다시 나는 것까지 확인하고 원복할 것.

- 위험과 피할 것:
  - **`migrations/` 를 건드리지 말 것.** `(report_id, placement, sort_order)` 에 UNIQUE 제약을 새로 거는 것이 더 "올바른" 수정처럼 보이지만, 008 은 출하된 체크섬이고 기존 데이터에 이미 중복이 있을 수 있어 마이그레이션이 깨진다. 이번 수정은 앞으로 중복을 만들지 않는 것까지다 — 기존 중복 데이터 정리는 범위 밖.
  - **프런트를 바꾸지 말 것.** `move()` 의 맞바꾸기와 select 가 placement 만 보내는 것은 그대로 두고 서버가 올바른 번호를 주는 것으로 고친다. 프런트를 한 줄도 안 바꾸면 `npm` 검증이 필요 없다. 두 경로(업로드의 `max+1` 과 이동의 `max+1`)가 **같은 식**을 쓰는지 꼭 맞출 것 — 한쪽만 고치면 같은 입력이 두 값으로 읽힌다.
  - `caption`/`placement` 의 `coalesce` 동작과 `RowsAffected()==0` → `ATTACHMENT_NOT_FOUND` 분기, 응답 본문(`{"id":…}`), OpenAPI 는 바꾸지 말 것. 소유권 검사(407행)도 그대로.
  - `$3::int IS NOT NULL` 같은 형 지정을 빼면 pgx 가 매개변수 형을 못 정해 질의가 통째로 실패한다. 세 매개변수 모두 명시적으로 형을 붙일 것.
  - 보호 경로(auth/session/migrations/.github/workflows)는 이번 과제에서 전혀 닿지 않는다.
  - 미확인: 운영 DB 에 이미 중복 `sort_order` 가 있는지는 확인하지 못했다. 또 `AttachmentPanel` 을 렌더하는 프런트 시험이 이 저장소에 있는지 확인하지 않았으므로, 프런트 쪽 주장은 소스를 읽은 것까지다(`move`/select 의 요청 본문).

- 차선 후보: `embeddingStatus`(semantic.go:431)가 실패한 통계 질의를 `items:0, embedded:0, stale:0` 으로 답하지 않게 한다 (가치 3 / 위험 1 / 작업량 S). 재현은 이 저장소의 관례대로 `ALTER TABLE report_item_embeddings RENAME TO …`. 단 500 을 돌려주면 프런트 `loadEmbedding` 이 실패를 `undefined` 로 바꿔 화면이 멈추므로 AdminPage 의 실패 문구까지 같이 해야 하고(Confluence 회차가 쓴 `errorText` + `다시 시도` 관례를 따를 것), 그러면 같은 모양의 수정이 세 번째가 된다.
