# v0.318.0 — 임베딩 현황이 읽지 못한 집계를 "0건" 으로 답하지 않습니다

이 릴리즈는 관리자 화면의 임베딩 카드와 "다시 만들기" 가 **질의가 실패했을 때 할 일이 없다고 말하던** 결함을 담습니다. 프로덕션 Go 는 `internal/app/semantic.go` 한 파일이고, `migrations/` 는 한 줄도 바뀌지 않았습니다. 새 환경변수·설정 변경도 없습니다.

## 읽지 못한 숫자와 0 건이 글자 그대로 같았습니다

`embeddingStatus`(`internal/app/semantic.go`)는 `items`·`embedded`·`stale` 세 개의 count 를 한 번에 읽는데, 그 `Scan` 의 오류를 `_ =` 로 버렸습니다. 질의가 실패하면 세 값은 **제로값 그대로** 남은 채 `200` 으로 나갔습니다.

관리자 화면에서 그 응답은 "임베딩 0/0건" 입니다. 그런데 같은 화면, 같은 숫자가 두 가지를 뜻합니다.

- **임베딩할 항목이 하나도 없다** — 할 일이 없습니다. 운영자는 아무것도 하지 않는 것이 맞습니다.
- **집계를 읽지 못했다** — 가서 고쳐야 합니다. 운영자는 그 사실을 들은 적이 없습니다.

같은 파일의 `pendingEmbeddingCount`(`446행`)도 같은 모양으로 0 을 돌려줬습니다. 그리고 그 값은 `rebuildEmbeddings` 가 **화면에 "남은 건수" 로 쓰는 숫자**입니다. 그래서 backlog 가 10만 건이어도 집계 질의가 실패하면 "다시 만들기" 는 이렇게 보고했습니다:

> 임베딩 4건을 생성했습니다. **남은 항목이 없습니다.**

backlog 가 남아 있는데 끝났다고 말하는 문장입니다. 이것이 운영자가 버튼을 다시 누르지 않게 만드는 한 문장입니다.

### 읽지 못했으면 읽지 못했다고 말합니다

- `status` 에 `countsUnread bool` 을 `omitempty` 로 더하고 `a.logger.Warn` 을 남깁니다. 이 저장소의 기존 관례(`rollup_handlers.go:227` 의 `Unread` 플래그)와 같은 이름·같은 모양입니다.
- 실패하면 세 값을 **모두 0 으로 되돌립니다.** 부분 `Scan` 은 도달한 값만 남기는데, 읽은 한 값이 읽지 못한 두 값 옆에 서면 같은 거짓말이 작은 글씨로 남습니다.
- `vectorAvailable`·`enabled`·`model` 은 **그대로 보냅니다.** 여기서 `500` 으로 바꾸면 그 세 값이 같이 사라져, 읽지 못한 집계와 **pgvector 가 없는 배포**가 화면에서 다시 같아집니다 — 가서 고칠 것이 서로 다른 두 가지입니다.
- `pendingEmbeddingCount` 를 `(int, error)` 로 바꿔 실패를 호출자에게 넘깁니다. 0 은 호출자가 **행동하는** 답("backlog 가 비었다")이므로, 거절된 집계와 구분되어야 합니다.
- `rebuildEmbeddings` 는 실패하면 `remaining` 을 **키째로 빼고** `remainingUnread: true` 를 담습니다. 응답은 `200` 을 유지합니다 — 방금 생성한 건수는 사실이고, 그것은 보고할 값입니다.
- 정상 경로의 응답 본문은 **그대로입니다.** 새 필드는 둘 다 실패 시에만 나오고, 시험이 정상 카드의 키 집합(`[embedded enabled items model stale vectorAvailable]`)을 직접 단정해 고정합니다.

### 화면도 같이 고쳤습니다

이 응답을 읽는 자리가 저장소 안에 **실제로 두 곳** 있었습니다 — `AdminPage.tsx:44` 의 카드와 `:47` 의 "다시 만들기" 알림입니다. 손대지 않으면 서버가 사실을 보내도 화면에 조용히 틀린 문장이 남으므로, 두 곳에 분기를 더하고 `types.ts` 에 선택 필드를 더했습니다.

## 검증

구현 단계에서 새 시험 2개를 **손으로 만든 대역 없이** 프로덕션 `app.Handler()` 와 실제 PostgreSQL(pgvector) 위에서 돌렸습니다.

- `internal/app/embeddingstatusunread_test.go` — 현황 쪽은 `report_item_embeddings` 를 rename 해 **집계만** 실패시킵니다. 다시 만들기 쪽은 더 어렵습니다: 두 질의가 같은 테이블을 읽으므로 그냥 rename 하면 배치가 먼저 실패해 `502` 가 되고, 고치려는 "배치는 성공하고 뒤따르는 집계만 실패하는" 모양이 만들어지지 않습니다. 그래서 **배치가 임베딩 게이트웨이에 가 있는 동안** `report_items` 를 rename 합니다 — 응답의 `embedded: 4` 가 배치의 성공을 증명하므로 실패한 단계가 집계임이 고정됩니다. 둘 다 `t.Cleanup` 으로 복원합니다.
- 고치기 전 두 시험이 각각 `the counts could not be read and the card does not say so: map[embedded:0 enabled:true items:0 model:status-1 stale:0 vectorAvailable:true]` 와 `the backlog count failed and the answer reports remaining=0 as a fact: map[embedded:4 model:unread-1 remaining:0]` 로 **실패합니다.** 고친 뒤 `result.CountsUnread = true` 와 `answer["remainingUnread"] = true` 를 각각 한 줄씩 되돌려 **같은 실패가 그대로 다시 나는 것**까지 확인하고 원복했습니다.

릴리즈 단계에서도 실제 PostgreSQL 위에서 `go test ./... -count=1` 전체를 통과했습니다. `gofmt`·`go build ./...`·`go vet ./...`, 버전·OpenAPI·쪽넘김·모달 검사, 프런트엔드 시험과 타입 검사·프로덕션 빌드, `scripts/build.sh` 를 통과했습니다. 로드맵 HTML·PDF 와 두 안내서 HTML 을 기존 스크립트로 다시 만들었습니다.

검증하지 않은 것을 밝힙니다. **프런트엔드 두 곳은 타입 검사와 프로덕션 빌드, 기존 vitest 까지만 확인했습니다** — 이 저장소에 `AdminPage` 렌더 시험이 없어 화면 문구 자체는 UI 행동으로 증명하지 않았습니다. 그 공백을 메우는 일은 다음 회차로 남깁니다. `remaining` 을 **키째로 빼는** 설계이므로 이 키를 필수로 읽는 **저장소 밖의** 클라이언트가 있으면 `undefined` 를 봅니다 — 저장소 안 소비자는 위 한 곳뿐이고 그곳은 고쳤지만 외부 통합은 확인할 길이 없어 OpenAPI 설명에 적어 뒀습니다. 같은 파일의 다른 `_ =`(`searchSemantic` 등)는 이번 범위 밖이라 그대로입니다. 제자리에서 소스를 고치는 `mutation-check`·`authz-check` 는 과거 회차가 시간을 넘겨 돌리지 않았습니다. `gh` 를 사용할 수 없는 세션이므로 원격 이력을 보는 advisory `release-check.sh` 와 docker 빌드가 필요한 `install-check.sh` 는 실행하지 않았습니다.

## 업그레이드

마이그레이션·새 환경변수·설정 변경은 없습니다. 기존과 같이 PostgreSQL 과 상태 볼륨을 백업한 뒤 v0.318.0 이미지를 올립니다.

**정상적으로 돌고 있는 배치에서는 달라지는 것이 없습니다.** 새 필드는 집계 질의가 실패할 때만 나오고, 그 전까지 응답 본문은 이전과 같은 키를 같은 값으로 담습니다.

올린 뒤에 볼 것은 로그입니다. `embedding status counts` 와 `embedding backlog count` 두 `WARN` 이 **이제 남습니다** — 이전에는 같은 실패가 아무 자취도 없이 0 으로 나갔습니다. 이 줄이 보이면 화면의 "0건" 을 믿지 말고 그 오류를 보십시오. 반대로 이 줄이 없는 동안의 "임베딩 0/0건" 은 **진짜로 할 일이 없다는 뜻**입니다 — 이 릴리즈가 만드는 차이가 정확히 그것입니다.

되돌려야 하면 v0.317.0 이미지로 내립니다. 스키마가 바뀌지 않았으므로 데이터 되돌림은 필요하지 않고, 되돌리면 결함도 함께 돌아옵니다.
