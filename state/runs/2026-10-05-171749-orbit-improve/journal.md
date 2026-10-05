# 회차 노트 2026-10-05-171749-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:17] base pinned — main@1a382c4
- [러너 17:17] autonomy release — 

## 정찰 노트
- MCP person_id 가드를 골랐다: 지난 회차가 다음 1순위로 지목한 것이고, 세 자리(mcp.go:178·workflow.go:195·mcp.go:258)를 직접 열어 무검사 통과를 확인했으며 DB 없이 증명되는 S 과제다. 차선이던 docs/API.md 문서화는 코드 결함이 남아 있는 동안 미뤘고, pages 테스트·orbitAt 성능은 한 세션 범위를 넘어 또 제쳤다.
- 과제서에서 추측인 것: (1) 22P02 가 실제로 세 도구 각각에서 일반 메시지로 나가는 것은 **코드 경로만 읽고 판단**했고 실 DB 로 재현하지 않았다(REST 쪽 같은 경로는 2026-09-28 에 확정됨). (2) `callWithoutStore` 를 MCP 핸들러(`s.mcp`)에 쓰면 nil store 패닉이 그대로 잡힌다고 봤지만 실행해 보지 않았다 — 안 되면 `mcpCall` 을 직접 부르는 쪽으로 물러서라.
- 기준 2(빈 person_id → get_relationship)는 `p.id=''` 가 22P02 라는 전제에 기댄다. 만약 postgres 가 다르게 받으면 그 하위 시험만 떼고 기준 1·3 으로 과제를 성립시켜라.
- 구현자가 조심할 것: 대문자 uuid·빈 person_id(list/create)는 반드시 통과해야 한다(지금 되던 것이 사라지면 회귀다). 기존 일반 메시지 문구와 HTTP 200 + isError 봉투는 mcp_db_test.go 가 문자열로 고정하고 있으니 바꾸지 말 것.
- [러너 17:22] scout done — MCP `tools/call` 의 세 도구가 uuid 모양이 아닌(또는 빈) `person_id` 를 uuid 컬럼에 그대로 넘겨, 외부 에이전트가

## 구현 노트
- 무엇/왜: `mcpCall` 세 arm 에 `mcpPersonIDError(personID, required)` 가드를 넣어 모양이 어긋난(또는 get_relationship 의 빈) `person_id` 를 DB 앞에서 구체적 한국어 isError 로 막고, `mcpCreateMemory` 의 `err != nil || !exists` 를 갈라 DB 장애가 "대상을 찾을 수 없습니다." 로 보이지 않게 했다. `tools/list` 의 list/create 스키마에 `format: uuid` + 설명 추가. 프로덕션 1파일(`internal/server/mcp.go`).
- 확신 없는 곳: 메시지 문구는 내가 새로 정한 것이라 어느 시험도 전문을 고정하지 않는다 — `mcp_test.go` 는 "person_id" 와 "uuid" 가 들어 있는지만 본다(문구는 바꿔도 초록). 기존 두 일반 문구는 `mcp_db_test.go` 가 전문으로 고정하고 있어 그대로 두었다. `orbit_get_relationship` 의 `person_id` 에는 설명을 넣지 않았다(과제서가 116·117 두 줄만 지정) — 세 도구 설명이 비대칭인 것은 알고 남겨 둔 것이다.
- 검증 못 한 것: 없음. 정찰이 추론으로만 남긴 22P02→일반 메시지를 격리 postgres(포트 55683)로 네 경로 각각 확인했고, 가드별로 한 줄만 되돌려 해당 시험만 빨개지는 것을 세 번 따로 확인했다. 웹(vitest/vite)은 손대지 않았으므로 돌리지 않았다.
- 일부러 하지 않은 것: 기존 isError 세 자리(mcp.go:133·229·237)를 헬퍼로 바꾸는 리팩터, `queryMemories`/`personConnections`/`escapeLike` 시그니처·SQL 변경, `orbit_search_people` arm 의 `rows.Err()` 가드와 오류 매핑, `mcpCreateMemory` 의 title/content 누락 메시지(다음 회차 1순위로 ideas.json 에 올렸다).
- 다음 역할이 조심할 것: `mcp_test.go` 는 DB 없이 CI 에서 돈다(24개 하위 시험). `mcp_db_test.go` 의 새 `TestMCPCreateMemorySeparatesLookupFailureFromMissingPerson` 은 DSN 이 있어야 돌고 뷰 DDL 을 쓰므로 t.Parallel 금지·`t.Cleanup` 원복에 의존하며, 그 원복은 같은 파일 뒤쪽 `TestMCPSearchPeopleProbeRestoresSchema` 가 함께 검사한다(선언 순서를 그 앞으로 둔 이유).
- 함정 한 가지: `breakPeopleRowStream` 식 뷰 주입은 질의가 그 컬럼을 실제로 읽지 않으면 플래너가 식을 가지치기해 오류가 안 난다 — `SELECT EXISTS` 에 `display_name` 을 깨뜨린 뷰는 exists=true/err=nil 로 멀쩡히 성공했다. 그래서 새 헬퍼는 WHERE 가 반드시 읽는 `user_id` 를 깨뜨린다.
- [러너 17:32] brief accepted — 채택 — 지정한 자리(세 arm 의 무검사 통과, `mcp.go:256~259` 의 `err != nil || !exists`, 116·117 의 스키마, `looksLikeUUID` 시그니처, `c
- [러너 17:32] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 격리 postgres(포트 55711, 신규)로 `go test -race ./...` 전부 ok·gofmt·vet 무출력. 작업 트리를 /tmp 로 복사해 `mcp.go` 만 main 것으로 되돌려 돌린 결과, 새 시험 중 가드를 겨누는 것은 전부 FAIL 하고 구현자가 "수정 전에도 PASS" 라고 적은 회귀 가드만 PASS — 시험이 실제로 바뀐 경로를 지난다. 스코프 검사가 가드보다 앞이고 세 질의의 user_id 범위가 그대로여서 보안·개인정보 차단 소견 없음.
- 못 본 것: 웹(vitest/vite) — 변경 없음이라 미실행. 22P02 가 실 운영에서 나는 빈도는 측정하지 않았다.
- 승인이어도 남는 우려 ①: `tools/list` 의 list/create person_id 가 `format: uuid` 를 걸면서 description 은 "비우면 …" 을 안내한다(스키마 자기 모순). format 을 단언하는 클라이언트는 `person_id: ""` 를 막는다 — 키를 빼는 우회로가 있어 차단은 아니지만 `pattern: "^$|<uuid>"` 가 정확하다. TestMCPToolsDeclarePersonIDFormat 이 지금 이 모순을 고정 중.
- 우려 ②: mcp.go:244~250 은 여전히 무로깅이라 새로 갈라낸 "조회 실패" 가 서버 로그에 한 줄도 남기지 않는다 — 주석이 말하는 "원인" 은 운영자 쪽에 아직 없다(slog 관례 위반, 기존 상태). 다음 회차 후보.
- 릴리즈 노트에 적을 동작 변경: `orbit_get_relationship` 의 빈/누락 person_id 응답 문구가 "요청을 처리하지 못했습니다." → "person_id가 필요합니다…" 로 바뀐다. docs/ 와 main 시험 어디에도 고정돼 있지 않아 계약 파손은 아님.
- [러너 17:37] review approved — 리뷰 승인 (risk=low)
- [러너 17:37] pr created — https://github.com/hkjang/orbit/pull/20
- [러너 17:39] ci passed — 검사 1개 모두 success
- [러너 17:39] merge done — 303b930

## 릴리즈 노트
- 판정: released. `0.7.9` → `0.7.10`, 주석 태그 `v0.7.10`(주석 본문 `Orbit v0.7.10`), 릴리즈 커밋 `chore(release): v0.7.10` (f73bbe9) — detached HEAD 에서 커밋·태그했고 브랜치는 하나도 건드리지 않았다. 원격에 아무것도 보내지 않았다(푸시·릴리즈 생성·이미지 푸시 전부 없음).
- 왜 패치인가: v0.7.0 이후 열 번 연속 패치만 올려 왔고(0.7.0~0.7.9) 이번 변경은 MCP 오류 메시지 충실도 한 가지라 마이너를 올릴 근거가 없다. 0.7.9 다음은 0.7.10 (0.8.0 아님).
- 이전 방식 확인한 것: 최근 태그 3개(v0.7.9·v0.7.8·v0.7.7)가 모두 주석 태그 + 본문 `Orbit <tag>` 이고, 각 릴리즈 커밋은 **`VERSION` 한 파일만** 1줄 바꿨다. CHANGELOG.md·docs/RELEASE*.md 는 저장소에 없고, 버전이 적힌 곳은 `VERSION` 뿐이다(`web/package.json` 의 `0.1.0` 은 역대 릴리즈가 한 번도 올리지 않았으므로 그대로 뒀다. 저장소 전체에서 `0.7.9`/`0.7.8` 문자열 검색 결과 0건 — 다른 동기화 지점 없음).
- 자산·GitHub Release 를 내가 만들지 않은 이유: `.github/workflows/release.yml` 이 `v*.*.*` 태그 푸시에 반응해 세만틱 태그 검증 → `docker build --platform linux/amd64` → `docker save | gzip -9 > orbit-<tag>.tar.gz` → manifest 검증 → `gh release create --verify-tag --generate-notes --title "Orbit <tag>"` 까지 전부 한다. 과거 자산 `orbit-v0.7.9.tar.gz` 와 제목·본문 양식이 이 워크플로 산출물과 정확히 일치한다. 그래서 `release.json` 은 `github_release:false`·`assets:[]`·`notes_file:""` (노트는 `--generate-notes` 가 PR 제목에서 만든다).
- 릴리즈 전 검증(이 기계, 전부 초록): `gofmt -l ./cmd ./internal` 무출력 · `go vet ./...` · `go build ./...` · `go test -race -count=1 ./...`(config·secure·server·scripts ok, DSN 없어 DB 시험 SKIP) · CI 와 같은 웹 경로 `npm ci` → `npm run test -- --run`(16파일 121시험 통과) → `npm run build`. 추가로 **릴리즈 워크플로의 빌드 단계를 같은 인자로 재현**해 `docker build --platform linux/amd64 --build-arg VERSION=v0.7.10 …` 이 성공하는 것까지 확인했다(태그 푸시 후 자산이 안 붙는 사고를 미리 막는 유일한 관문) — 검증용 로컬 이미지 `orbit:v0.7.10-relcheck` 는 지웠다(`docker images orbit` 로 부재 확인 — 남아 있는 `orbit:release-check-v0.7.5`·`orbit:v0.7.0`·`orbit:ci` 는 이전 회차 것이라 건드리지 않았다). Dockerfile 은 버전을 build-arg 로만 받고 `VERSION` 파일을 읽지 않으므로 이번 커밋이 이미지 빌드를 깨뜨릴 수 없다.
- 다음 사람이 알아야 할 것: 비평 노트가 "릴리즈 노트에 적을 동작 변경" 으로 남긴 것 — `orbit_get_relationship` 의 빈/누락 `person_id` 응답 문구가 "요청을 처리하지 못했습니다." → "person_id 가 필요합니다…" 로 바뀐다. 자동 생성 노트는 PR 제목만 싣기 때문에 이 문구 변경은 GitHub Release 본문에 드러나지 않는다. MCP 클라이언트가 그 문장을 문자열로 보고 분기했다면 영향이 있지만 docs/ 와 main 시험 어디에도 고정돼 있지 않아 계약 파손은 아니다.
- [러너 17:46] release published — v0.7.10
- [러너 17:47] assets verified — v0.7.10 자산 1개 (이전 v0.7.9: 1)
