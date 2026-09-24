# 과제서 (2026-09-24, sqlon @ 57f99b7)

- **과제**: 관리자 가이드 §3.2 의 DB 프로파일 등록 curl 을 실제 라우트·필드 계약(`POST /api/db-profiles`)으로 교정하고, 0바이트인 `docs/README.md` 를 실제 존재하는 문서만 가리키는 색인으로 채운다 (가치 4 / 위험 1 / 작업량 S)

## 왜
`docs/admin_guide.md:119` 이 안내하는 `curl -X POST http://localhost:6767/api/fleet/instances` 는 **지금 서버에서 절대 성공할 수 없다** — `internal/mcp/admin.go:72` 는 `GET /api/fleet/instances` 만 등록하므로 Go 1.22+ ServeMux 가 POST 에 405 를 돌려주고, 실제 등록 경로는 `internal/mcp/dbapi.go:49` 의 `POST /api/db-profiles` 다. 게다가 예제 JSON 의 필드는 `dbconn.Profile`(`internal/dbconn/profile.go:24`) 와 **하나도 맞지 않는다**(`profile_id`/`engine`/`host`/`port`/`database`/`password_secret`/`max_open_conns` → 실제는 `id`/`type`/`connect_string`/`password_ref`/`pool`). 이 문서는 "대외비 · 시스템 관리자/DBA 대상" 설치 가이드의 첫 설정 단계라, 이대로 따라 하면 신규 설치가 첫 프로파일 등록에서 막힌다. 동시에 루트 `README.md:72` 가 "상세 문서" 진입점으로 거는 `docs/README.md` 가 0바이트라 독자가 빈 화면을 만난다.

## 수용 기준
1. `docs/admin_guide.md` §3.2 의 curl 이 **실행하면 실제로 성공하는 명령**이 된다 — 경로 `POST /api/db-profiles`, 바디 필드가 `dbconn.Profile` 의 JSON 태그(`id`, `type`, `connect_string`, `username`, `password_ref`, `pool`)와 일치. 구현자는 문서를 고치기 전/후로 **실제 서버를 띄워 두 명령을 모두 실행**하고, 옛 명령이 405(또는 404)를 내고 새 명령이 200 + `{"saved":true,...}` 를 내는 것을 출력으로 확인해 보고한다(소스 문자열 대조만으로 갈음하지 않는다).
2. `docs/README.md` 가 **내용이 있는 문서만** 링크하는 색인이 된다 — `admin_guide.md`, `executive_report.md`, `docs/reports/` 하위의 실재 파일, 그리고 루트 `README.md`. 나머지 `docs/*.md` 27개가 아직 빈 자리표시자임을 한 줄로 솔직히 적고, **그 27개 파일에 내용을 새로 쓰지 않는다**.
3. 회귀가 없음을 보인다 — Go 코드를 건드리지 않았으므로 `go build ./...` 와 `go test ./... -count=1` 이 그대로 통과하고, `git diff --check` 가 깨끗하며, `git status` 에 `docs/admin_guide.md` 와 `docs/README.md` **두 파일만** 변경으로 나타난다(PDF·이미지·다른 md 가 함께 바뀌면 안 된다).

## 건드릴 파일
- `docs/admin_guide.md` — §3.2 「데이터베이스 프로필 등록」의 코드블록(현재 118~131행 부근). 경로·필드명 교정. 인증 헤더 설명은 **두 모드를 구분**해 한 줄 덧붙일 것: 단독 모드는 `Authorization: Bearer $SQLON_ADMIN_TOKEN`(`upsertProfile` → `requireAdmin`, `internal/mcp/dbapi.go:951`), 메타 DB 모드는 로그인 세션/MCP 키(`upsertProfileMeta`, `dbapi.go:984`, 마스터 토큰으로는 생성 불가 — `dbapi.go:989`).
- `docs/README.md` — 0바이트 → 문서 색인. 루트 `README.md:72` 가 이 파일을 가리킨다.
- `CHANGELOG.md` — `Unreleased` 에 한 줄(기존 관례대로). **주의**: `CHANGELOG.md` 는 줄바꿈이 혼합돼 있으니 파일 전체를 다시 쓰지 말고 해당 줄만 추가할 것(과거 회차에서 전체 CRLF 재작성으로 97줄 diff 가 난 적 있음).

### 참고 — 실제 계약 (소스에서 확인함)
- `internal/mcp/dbapi.go:49` `mux.HandleFunc("POST /api/db-profiles", ...)` → `s.upsertProfile(w, r, "", true)`
- `internal/mcp/admin.go:72` `mux.HandleFunc("GET /api/fleet/instances", ...)` — GET 만 존재
- `internal/dbconn/profile.go:24` `Profile` 의 JSON 태그: `id`, `name`, `type`(postgres|mysql|mariadb|oracle), `connect_string`(`host:port/dbname` 또는 `postgres://...`), `username`, `password_ref`(`env:NAME` | `file:PATH` | `plain:VALUE`), `pool`, `policy`, `environment`, `criticality`, `owner_team` 등
- **미확인**: `Pool` 구조체의 실제 필드명(`max_open` 인지 `max_open_conns` 인지). 구현자가 `internal/dbconn/profile.go` 의 `type Pool struct` 를 직접 읽고 그 이름을 쓸 것 — 추측 금지.
- **미확인**: 서버 기동 플래그 이름(`-admin-token` 으로 추정). `go run ./cmd/sqlon -h` 로 확인할 것.

## 검증 명령
```bash
# 1) 실제 동작 확인 (수용 기준 1의 증거). 플래그 이름은 -h 로 먼저 확인.
go run ./cmd/sqlon -h
go run ./cmd/sqlon -admin-token=devtoken &   # 포트 기본 6767
curl -i -X POST localhost:6767/api/fleet/instances -H 'Authorization: Bearer devtoken' \
  -H 'Content-Type: application/json' -d '{}'          # → 405/404 (옛 문서 명령)
curl -i -X POST localhost:6767/api/db-profiles -H 'Authorization: Bearer devtoken' \
  -H 'Content-Type: application/json' -d '<문서에 새로 쓴 그 바디>'   # → 200 {"saved":true,...}

# 2) 회귀 없음
go build ./...
go test ./... -count=1
git diff --check
git status --short      # docs/README.md, docs/admin_guide.md, CHANGELOG.md 만
```

## 위험과 피할 것
- **PDF 를 다시 굽지 말 것.** `docs/admin_guide.pdf` 는 `docs/reports/generate_docs.py` 산출물이고 이 환경에는 playwright 브라우저가 없다(과거 회차에서 시스템 Chrome 래퍼로 우회해야 했다). PDF 는 손대지 말고, CHANGELOG 한 줄에 "md 만 갱신, PDF 재생성 필요" 를 명시해 상태를 숨기지 말 것. PDF 를 손으로 편집하는 것은 금지.
- **빈 `docs/*.md` 27개를 채우지 말 것.** `auth.md`·`rest-api.md`·`security.md` 정본 작성은 별도 회차 과제다. 이번에 손대면 범위가 M→L 로 번지고 라우트 검증 없이 쓴 문서가 또 하나의 정본이 된다.
- **Go 코드를 건드리지 말 것.** 이번 과제는 문서를 소스에 맞추는 것이지 소스를 문서에 맞추는 것이 아니다. `POST /api/fleet/instances` 라우트를 새로 만들어 문서를 정당화하는 것은 명시적으로 금지 — 생성 경로가 둘이 되어 권한 계약이 갈라진다.
- **전체 gofmt·줄바꿈 일괄 변환 금지.** 저장소에 CRLF 파일이 다수 있어 무관한 diff 가 폭발한다.
- 보호 경로(`internal/mcp/auth.go`·`authapi.go`, `internal/meta/pg.go`, `.github/`)는 이번 과제와 무관하므로 열지 말 것.
- **정직성**: 새 curl 을 실제로 실행해 보지 못했다면 수용 기준 1을 충족했다고 쓰지 말고 무엇을 못 돌렸는지 그대로 보고할 것.

## 차선 후보
**`docs/README.md` 색인 작성만 단독 수행** — 만약 §3.2 교정이 PDF 정본 불일치 때문에 받아들이기 어렵다고 판단되면, `docs/README.md`(0바이트, 루트 `README.md:72` 의 진입점) 채우기만 하고 `docs/admin_guide.md` 는 열지 말 것. 이쪽은 PDF 가 없어 정본 문제가 생기지 않는다. 수용 기준 2·3 만 적용한다.
