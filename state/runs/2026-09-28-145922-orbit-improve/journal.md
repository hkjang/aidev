# 회차 노트 2026-09-28-145922-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:59] base pinned — main@fc11b43
- [러너 14:59] autonomy release — 

## 정찰 노트
- 고른 이유: 지난 회차의 `looksLikeUUID` 가드가 `data.go` 여섯 곳에만 붙어 **같은 결함이 남은 두 REST 경로**(`workflow.go:187` 의 `NULLIF($2,'')::uuid`, `ai.go:85` 의 `p.id=$1`)를 실제로 열어 확인했다. 프로덕션 2파일·기존 헬퍼 재사용·nil store 로 실패 재현이 되는 것이 결정적이었다.
- 제친 후보: `openapi.go` 렌더 테스트는 다섯 회차 연속 차선이라 가치 신호가 낮아 차선으로만 남겼고, `mcp.go` uuid 가드는 `mcp.go:224~231` 을 읽어 보니 **이미 `isError:true` 로 곱게 떨어져 500 이 아니라** 가치가 메시지 충실도뿐임을 확인해 내렸다.
- 확신 없는 곳: `22P02`→500 을 실제 postgres 로 이번에 재확인하지는 않았다(2026-09-27 회차가 `psql` 로 확정한 사실에 기댔다). `internalError` 가 500 을 내는 것은 코드로 확인.
- 구현자가 조심할 것: `streamAI` 의 가드는 `s.readSetting`(ai.go:35) **앞**이어야 nil-store 시험이 성립하고, 그 대가로 "AI 비활성 + 깨진 person_id" 의 응답이 503→400 으로 바뀐다 — 주석으로 근거를 남길 것. `createMemory`(workflow.go:75)는 이미 400 이니 건드리지 말 것.
- 새로 기록한 것: `rows.Err()` 를 안 보는 목록 경로 다섯 곳(data.go:137·mcp.go:151·personal.go:70/282/355·settings.go:207)을 ideas.json 에 넣었다 — 결정적 실패 재현 수단이 없어 M 으로 두었다.
- [러너 15:04] scout done — uuid 모양이 아닌 `person_id` 로 `GET /api/v1/memories/` 와 `POST /api/v1/ai/stream` 이 500 을 내는 것을 DB 앞에서 400 으로

## 구현 노트
- 무엇/왜: `listMemories`(workflow.go)와 `streamAI`(ai.go)에 기존 `looksLikeUUID` 가드를 붙여, uuid 모양이 아닌 `person_id` 가 postgres `22P02` → `internalError` 500 으로 새는 것을 `validation_error` 400 으로 막았다. 새 슬러그·새 헬퍼 없음, 프로덕션 2파일.
- 확신 없는 곳: 없음에 가깝다 — 실제 postgres(docker, 포트 55537)에서 두 실제 핸들러를 불러 가드 전 500 / 가드 후 400, 정상 uuid 는 양쪽 200 동일을 직접 봤다. 다만 그 DB 탐침은 일회용이라 커밋하지 않았으니 재현하려면 다시 띄워야 한다.
- 일부러 하지 않은 것: `queryMemoriesLimit`·`searchMemories` 의 시그니처·SQL 무변경(MCP·AI 와 공유). `createMemory` 는 이미 400 이라 손대지 않음. `mcp.go`·`docs/` 도 범위 밖(400 은 openapi.go 의 `operation()` 이 이미 선언).
- 비평가가 먼저 볼 자리: `streamAI` 의 가드를 `readSetting` **앞**에 두어, AI 비활성 + 깨진 person_id 가 겹치면 응답이 503 `ai_disabled` → 400 `validation_error` 로 바뀐다. 의도된 변경이고 근거를 코드 주석에 남겼다.
- 다음 역할이 조심할 것: 커밋된 새 테스트(`workflow_test.go`·`ai_test.go`)는 `&Server{store: nil}` 로 돌아 DB 가 필요 없다 — CI 에서 실제로 실행된다. "정상 uuid 는 DB 까지 간다" 쪽 시험은 nil 포인터 패닉을 `recover` 로 잡아 성공 판정하므로, 가드 위치를 뒤로 옮기면 조용히 통과할 수 있다는 점만 유의.
- 남은 500: 정상 uuid 지만 없는 사람으로 `/ai/stream` 을 부르면 `pgx.ErrNoRows` 가 여전히 500 이다 — ideas.json 에 다음 후보로 남겼다.
- [러너 15:08] brief accepted — 채택 — 근거(`queryMemoriesLimit` 의 `::uuid` 캐스팅, `relationshipContext` 의 `p.id=$1`, `createMemory` 는 이미 400, 가드 자리가 `readSetting
- [러너 15:09] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 프로덕션 2파일만 main 으로 되돌려 새 테스트를 실제로 빨갛게 만들었다 — 원장의 실패 재현(`ai_test.go:36`·`workflow_test.go:47`, 거부 하위 시험 8개)이 그대로 재현됐고 증상도 이번 수정이 고치는 것과 일치한다. 복구 후 gofmt·go vet·`go test -race ./...` 전부 초록, 작업 트리는 HEAD 와 동일하게 되돌려 두었다. 라우팅(server.go:56·62)·`queryMemoriesLimit` 의 `::uuid`·`relationshipContext` 의 `p.id=$1`·openapi 의 400 선언·웹 두 호출처까지 열어 봤다.
- 못 본 것: 실제 postgres 로 22P02→500 을 이번 세션에서 재확인하지 않았다(구현자가 포트 55537 로 확인했다는 기록에 기댔다). 웹 빌드·vitest 미실행 — 웹 변경이 없어 필요 없다고 봤다.
- 승인이어도 남는 우려 ①: 두 `LetsWellFormedPersonIDReachDB` 는 고치기 전에도 PASS 이고, `callWithoutStore` 의 `recover()` 가 패닉 종류를 가리지 않아 가드를 `readSetting` 뒤로 옮기면 조용히 통과한다. 가드 위치를 붙잡는 것은 거부 쪽 시험뿐이다.
- 승인이어도 남는 우려 ②(릴리즈 노트감): `looksLikeUUID` 는 `{uuid}`·하이픈 없는 32자리를 거부하지만 postgres 는 받아 준다 — 그 모양으로 `?person_id=` 를 보내던 외부 API 키 클라이언트는 200→400 이다. main 이 이미 data.go 여섯 곳에 같은 관례를 적용했으므로 결함은 아니나 문서화되지 않은 축소다. `streamAI` 의 503→400 우선순위 변경은 주석·커밋·과제서가 일치해 의도로 받았다.
- 보안·법무 차단 없음: 새 경로·권한 확대·비밀값·개인정보 수집 없음, 사용자 격리(`m.user_id=$1`·`p.user_id=$2`) 유지, 질의는 이전에도 파라미터 바인딩(주입 주장도 하지 않는다), 마이그레이션 없어 revert 로 완전 복귀. 다음 회차: `/ai/stream` 의 `pgx.ErrNoRows` 500 과 `mcp.go` 의 `args.PersonID` 만 남았다.
- [러너 15:12] review approved — 리뷰 승인 (risk=low)
- [러너 15:12] pr created — https://github.com/hkjang/orbit/pull/14
- [러너 15:14] ci passed — 검사 1개 모두 success
- [러너 15:14] merge done — 50b5d07
- [러너 15:20] release published — v0.7.4
- [러너 15:21] assets verified — v0.7.4 자산 1개 (이전 v0.7.3: 1)
