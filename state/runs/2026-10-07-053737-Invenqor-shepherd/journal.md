# PR 처리기 노트 2026-10-07-053737-Invenqor-shepherd — Invenqor PR #33
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-07-042745-Invenqor-improve)
# 회차 노트 2026-10-07-042745-Invenqor-improve — Invenqor
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 04:27] base pinned — main@263afa6
- [러너 04:27] autonomy release — 

## 정찰 노트
- 고른 이유: 지난 회차가 "다음 회차 가장 익은 후보" 로 지목한 `classification.go:421` relationID 원문 UPDATE 를 열어 보니 근거가 그대로 맞았다 — `openapi.yaml:1677` 의 `format: uuid`, `:1685-1688` 의 200/400/404(500 없음), postgres/sqlite `001_initial.sql:245` 의 UUID/TEXT 차이. 프로덕션 1파일 S 규모이고 직전 네 회차가 같은 패턴(canonicalUUID 적용)으로 전부 release-ready 였다.
- 제친 후보: 관계 삭제의 부모 assetID 소속 제한(가치 4)은 incoming/outgoing 부모 정책 확정이 선행이라 차선으로 남겼다. internalError 교착·rows.Err() 는 재현 수단이 없어 수용 기준을 쓸 수 없으므로 여섯 회차 연속 미선택.
- 확신 없는 곳: **수정 전 PostgreSQL 의 500 과 대문자 UUID 폴딩을 이번 정찰에서 재현하지 않았다**(컬럼 타입과 앞선 회차 실측에 기댄 추론). 구현자가 먼저 돌려 확인할 것 — 과제서 수용 기준 4) 에 명시했다.
- 조심할 것: 응답 코드 집합을 늘리지 말 것(형식 오류는 기존 404 `PROPOSAL_NOT_FOUND` 로 합치고 `openapi.yaml` 은 고치지 말 것). `canonicalUUID` 를 느슨하게 바꾸면 merge·split 의 기존 테스트 계약까지 번진다. 감사에는 정규화된 id 만 넘길 것.
- 실측: 좁은 SQLite suite `go test ./internal/httpapi/ -run 'Relation|Proposed|Classification' -count=1` ok 3.252초. PostgreSQL 포트는 `ss -ltn` 으로 55531 점유·55532·55533 빈 것 확인(브리프에 55533 제안). PostgreSQL·전체 suite·vet·web/Rust 는 이번 회차 미실행.
- [러너 04:32] scout done — 제안 관계 심사(`POST /api/v1/assets/relations/{relationID}/{decision}`)의 relationID 를 기존 canonicalUUID 계약으로 검증 (�

## 구현 노트
- `classification.go:425 reviewProposedRelation` 이 `chi.URLParam("relationID")` 을 날것으로 `UPDATE ... WHERE id=$6` 에 넘기던 것을 `decision` 검사 직후 `canonicalUUID` 로 통과시키고 실패 시 404 `PROPOSAL_NOT_FOUND` 로 반환하게 했다. 이후 SQL·감사에는 정규화된 id 만 쓴다. 프로덕션 1파일 + 신규 테스트 1파일(`classification_relation_review_test.go`), `openapi.yaml` 은 건드리지 않았다(응답 코드 집합 불변).
- 수정 전 PostgreSQL 실측이 과제서의 추론보다 나빴다: `not-a-uuid`·`urn:uuid:` 는 500 이지만 **중괄호형·하이픈 없는 32자는 200 으로 실제 심사까지 성공**했고(감사 기록도 남았다), 대문자 36자는 200 이면서 `audit_logs.resource_id` 에 대문자를 그대로 저장했다. SQLite 는 네 철자 404·대문자 404. 비평가는 이 네 철자 × approve/reject 조합이 두 방언에서 같은 404 인지를 먼저 보면 된다.
- 확신 없는 곳: 없음에 가깝다 — 두 방언 전체 suite 를 수정 후 실제로 돌렸다(SQLite httpapi 28.852s, PostgreSQL httpapi 74.122s). 다만 `git diff` 를 되돌려 재실패시키는 역검증은 따로 돌리지 않고 수정 전 실측 출력으로 갈음했다.
- 일부러 하지 않은 것: 차선 후보(관계 삭제의 부모 assetID 소속 제한)는 incoming/outgoing 부모 정책 확정이 선행이라 손대지 않았다. `mcp.go`·`users.go`·`api_keys.go`·`agents.go` 의 나머지 UUID 경로도 범위를 넘겨 다음 회차로 넘겼다. 버전 범프·릴리즈 노트·PDF 는 섞지 않았다.
- 다음 역할이 조심할 것: 새 테스트는 SQLite 로 그냥 돈다. PostgreSQL 검증에는 Docker 가 필요하고 **회차 고유 컨테이너 이름**이 필수다(스크립트가 지정 이름을 먼저 삭제한다). 이번에 쓴 포트 55533/55532 는 비어 있었고 컨테이너는 둘 다 정리됐다. `go build ./cmd/invenqor-server` 가 `server/invenqor-server` 바이너리를 남기므로 커밋 전에 지울 것(이번에는 지웠고 `git status` 로 확인했다).
- [러너 04:38] brief accepted — 채택 — 지목한 행·헬퍼·테스트 템플릿·검증 명령·제안 포트(55533)가 모두 그대로 맞았고, '미확인' 으로 남긴 PostgreSQL �
- [러너 04:39] verify passed — 검증 8개 통과 (auto)

## 비평 노트
- 확인한 것: `classification.go` 만 263afa6 으로 되돌려 **양 방언 역검증을 직접 돌렸다** — SQLite 는 대문자 테스트가 `404, want 200` 으로 실패, PostgreSQL 은 404 테스트 8조합 전부 실패(brace·unhyphenated 가 approve/reject 모두 200 에 status 실제 변경, not-a-uuid·urn 은 500, 대문자는 audit resource_id 에 대문자 저장). 구현 노트가 "역검증은 안 돌렸다" 고 비워 둔 자리를 메웠고, 주장·주석·커밋 메시지가 측정과 전부 일치했다. 수정 후 SQLite narrow 7.523s·PostgreSQL httpapi 9.998s 통과, `gofmt -l` 빈 출력, `go vet ./...` 무출력, 트리 clean, 컨테이너(55541·55542) 잔여 없음.
- 못 본 것: web/Rust/e2e(Go 핸들러 1개 변경이라 생략), PostgreSQL 전체 `-run` 없는 완주.
- 승인이어도 남는 우려 — **릴리즈 노트에 동작 변화를 한 줄 써야 한다**: 수정 전 PostgreSQL 에서 중괄호형·하이픈 없는 relationID 로 보낸 심사가 실제로 제안을 승인·거절하고 감사까지 남겼고, 이제 404 다. 그리고 404 테스트의 절반은 SQLite 에서 수정 전에도 통과하므로 그 경로 회귀는 PostgreSQL CI 만 잡는다.
- 검토 부서: security·legal 모두 차단 없음. 신뢰 못 할 경로 파라미터를 SQL 전에 allowlist 로 좁히고 감사에 정규형만 남기는 개선이며, 인가(requireCSRF + relations.write)·개인정보·의존성·외부 계약(openapi.yaml 무변경)은 불변이다.
- 다음 회차: 남은 UUID 미점검 경로는 `mcp.go`·`users.go`·`api_keys.go`·`agents.go`.
- [러너 04:43] review approved — 리뷰 승인 (risk=low)
- [러너 04:43] pr created — https://github.com/hkjang/invenqor/pull/33
- [러너 04:48] ci failed — 성공이 아닌 검사: server=failure · 실패한 검사: ? 잡: server 

## 수리 노트
- 지적은 맞았다(CI 실패는 실재). 다만 **PR 의 코드 결함이 아니다** — `npm audit --omit=dev --audit-level=high` 가 신규 권고 GHSA-68fv-2mgg-jv7q 로 넘어졌고, PR 디프는 Go 2파일뿐이라 main 에도 있는 실패다(vite 가 `dependencies` 라 prod 감사 경로에 postcss→source-map-js 가 들어온다).
- 고친 방법: `npm audit fix` 로 `web/package-lock.json` 의 source-map-js 1.2.1 → 1.2.2 만 올렸다(3줄). 워크플로·테스트·단언 무수정. ci.yml:112 주석이 "신규 권고는 고쳐라" 라고 정책을 못박고 있다.
- 재현·검증 모두 실측: 수정 전 audit exit 1(CI 와 동일 출력) → 수정 후 "found 0 vulnerabilities". 추가로 npm test 150건, build 후 dist 무변화, redocly lint, go test ./... 전체 ok, vet/gofmt/build 모두 통과.
- 확신 없는 곳: `server-postgres` 잡은 로컬 미실행(락파일 변경이라 영향 없고 그 잡은 이번 CI 에서 실패하지 않았다). dev 전용 권고는 CI 게이트 밖이라 손대지 않았다 — 나중에 `npm audit` 전체가 게이트가 되면 다시 걸린다.
- 비평가가 볼 것: e2df1f1 의 Go 변경은 이번 회차에 전혀 건드리지 않았다. 새 커밋은 94cc5f2 하나뿐.

## 심사 노트
- 확인한 것: 양 방언 역검증을 직접 돌렸다 — `classification.go` 만 origin/main 으로 되돌리니 PostgreSQL 에서 brace/unhyphenated 가 approve·reject 모두 실제로 심사 성공(200, status 변경), not-a-uuid/urn 은 500, 대문자는 `audit_logs.resource_id` 에 대문자 저장으로 전부 실패했고 SQLite 는 대문자 테스트만 실패했다. 수정 후 양 방언 통과. 비평가가 비워 둔 **PostgreSQL 전체 suite 완주**를 돌려 통과(httpapi 68.056s, EXIT=0). 락파일은 `rm -rf node_modules && npm ci` 로 integrity 실검증(source-map-js@1.2.2 설치), prod audit 0건, npm test 150건, build 후 `git diff --exit-code -- server/internal/webui/dist` 무변화. gofmt/vet 무출력, 트리 clean, 컨테이너(55571·55572·55573) 잔여 없음.
- 회귀 없음 근거: `asset_relations.id` 는 세 삽입 경로(`classify/relate.go:82,180`, `assets.go:658`) 전부 `uuid.NewString()` 이고 마이그레이션에 시드 행이 없다 → 기존 id 는 모두 canonicalUUID 를 통과한다. 콘솔은 `classificationPage.tsx:150` 에서 API 가 돌려준 `proposal.id` 를 그대로 보내므로 404 로 막힐 철자를 만들지 않는다.
- 못 본 것: Rust/cargo audit/govulncheck/e2e(각각 Rust·Go 의존성·워크플로 무변경), redocly lint(openapi.yaml 무변경).
- 권고 근거: 인가(requireCSRF + relations.write, `server.go:399`) 불변, 새 공개 경로·비밀값·개인정보·마이그레이션 없음, 응답 코드 집합은 openapi 가 선언한 200/400/404 안으로 **좁아졌다**. 감사에는 정규화 id 만 들어간다. revert 로 완전히 되돌아온다 → approve / merge.
- 남는 주의(차단 아님): 비평가 지적대로 릴리즈 노트에 "PostgreSQL 에서 중괄호형·하이픈 없는 relationID 로 보낸 심사가 전에는 실제로 승인·거절되었고 이제 404" 한 줄이 필요하다. 락파일 3줄은 Go 디프와 무관하지만 ci.yml:112 가 정책으로 못박은 CI 게이트 수리이므로 범위 이탈로 보지 않는다.
