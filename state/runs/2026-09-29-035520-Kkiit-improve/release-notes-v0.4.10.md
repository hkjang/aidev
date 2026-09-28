Kkiit v0.4.10 — 포트폴리오 저장 실패 안내 수정

v0.4.9 이후 들어온 수정 하나를 담는다.

- `updateMyPortfolio` 와 `deleteMyPortfolio`(`discovery.go`)가
  `UPDATE`·`DELETE portfolios` 의 모든 오류를 `err != nil ||
  tag.RowsAffected() == 0` 한 덩어리로 묶어 404 `portfolio_not_found`
  로 돌려주던 것을 고친다. 바로 위 `createMyPortfolio` 는 같은 표의
  INSERT 실패를 이미 500 `create_failed` 로 보고하고 있어 비대칭이었고,
  목록에서 자기 포트폴리오를 보고 있는 소유자가 수정만 하면
  "포트폴리오를 찾을 수 없습니다." 라는 안내를 받아 무엇이 잘못됐는지
  알 수 없었다. 저장 실패를 500 `portfolio_save_failed`
  "포트폴리오를 저장하지 못했습니다."(`createMyPortfolio` 문구와 동일)
  로 가르고 `RowsAffected()==0` 만 404 로 남긴다. `DELETE` 의 동일한
  패턴도 같은 형태로 갈라 500 `portfolio_delete_failed` 로 보고한다 —
  이 경로는 파라미터가 uuid 두 개뿐이라 HTTP 입력으로 DB 오류를 만들 수
  없어 새 500 갈래의 도달은 증명하지 못했고, 회귀(없는 id → 404,
  본인 것 → 204)만 확인했다. `validate()` 와 조회 경로는 건드리지
  않았다. 버릴 PostgreSQL 16 에 붙는 통합 테스트로 404 를 먼저 재현한 뒤
  고쳐 통과시켰고, 임시 프로브로 실패 경로의 오류가
  `*pgconn.PgError{Code:"22021"}`(`title` 의 NUL 거절)이며 그 행이 실제로
  존재하는데도 RowsAffected 가 0 이라는 것을 확인한 뒤 프로브를 제거했다.
  분기를 두 방향으로 변이시켜(항상 500 / 항상 404) 각각 다시 실패하는
  것도 확인했다. 프런트는 이미 `error.message` 를 그대로 띄우므로 `web/`
  과 `internal/ui/dist` 는 건드리지 않았고, `openapi.yaml` 의
  PUT /me/portfolios/{id} 응답에 400·500, DELETE 응답에 404·500 을 더해
  실제 동작과 맞췄다.

버전 표기(VERSION·compose·web/package·openapi·README)를 0.4.10 으로 올린다.

gofmt · go build · go vet · go test, web lint · test · build 전부 통과.
버릴 PostgreSQL 16 으로 `make test-integration` 도 통과(88.2초).
