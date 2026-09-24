저장 결과 조회·삭제가 실패할 때 원인과 맞지 않는 상태코드를 내던 문제를 고쳤습니다. 메타 DB 장애는 500, 없는 결과는 404, 잘못된 ID 는 400 으로 나갑니다.

**메타 DB 가 죽어도 "없는 결과"로 보였습니다.**

`GET /api/v1/query/saved/{id}` 는 모든 실패를 404 로, `DELETE` 는 모든 실패를 400 으로 냈습니다. 그래서 메타 DB 연결이 끊긴 것도, 저장본 JSON 이 깨진 것도 "저장된 결과를 찾을 수 없거나 권한이 없습니다"·"잘못된 요청"으로 나갔습니다. 모니터링은 서버 장애를 5xx 로 집계하지 못해 조용히 지나갔고, 쓰는 사람도 자기 결과가 지워진 줄로만 알았습니다. 두 핸들러의 분류가 서로 어긋나 있어 같은 원인(잘못된 ID)에 조회는 404, 삭제는 400 을 내기도 했습니다.

이제 없는 결과와 저장소 장애를 오류 문구가 아니라 센티널 값(`ErrNotFound`·`ErrInvalidID`)으로 가릅니다. 저장소가 `sql.ErrNoRows`·`RowsAffected()==0` 자리에서 그 값을 돌려주고, 핸들러가 `errors.Is` 로 상태코드를 나눕니다. 조회와 삭제가 같은 분류를 씁니다.

| 상황 | 이후 동작 |
| --- | --- |
| 없는 결과 · 남의 결과 | `404 DF_SAVE_NOT_FOUND` |
| 결과 ID 가 0 이하 | `400 DF_SAVE_INVALID_REQUEST` |
| 메타 DB 장애 | `500 DF_SAVE_GET_FAILED` / `DF_SAVE_DELETE_FAILED` |
| 저장본 JSON 손상 | `500 DF_SAVE_GET_FAILED` |
| 드라이버 원문(`Error 1452`, `write tcp …`) | 응답 본문에 싣지 않음 |

화면에 보이는 안내 문구는 한 글자도 바뀌지 않습니다 — `saved.js` 가 `cause.message` 를 그대로 띄우고 `layout.js` 의 `api()` 는 401 만 특별 취급하므로, 상태코드가 갈라져도 사용자가 읽는 문장은 그대로입니다.

**검증.** 실제 라우터(`New(logger, options)`)에 httptest 로 요청하는 서버 테스트 9개를 먼저 돌려 6개가 실패하는 것을 확인한 뒤 통과시켰고(DELETE 는 `csrfProtection` 때문에 세션의 실제 CSRF 토큰을 헤더에 넣습니다), sqlmock 으로 저장소 경계 테스트 4개를 더해 센티널을 되돌리면 2개가 다시 실패하는 것까지 확인했습니다. 드라이버 원문이 응답 본문에 섞이지 않는지도 테스트로 못 박았습니다. `gofmt -l .` 무출력, `go vet ./...`, `go test -race ./...` 38개 패키지 통과, `go build ./cmd/dartfly` 성공. 릴리스 이미지는 `deploy/build-release.sh` 가 `test/smoke/artifact.sh` 로 실제로 띄워 healthz/readyz·임베드 정적 자산·로그인·내장 마스터 키 질의·TZ 적용을 확인했습니다. 화면 동작이 바뀌지 않아 브라우저 스모크는 돌리지 않았습니다.

오프라인 배포: `gunzip -c dartfly-v2.74.0.tar.gz | docker load`
