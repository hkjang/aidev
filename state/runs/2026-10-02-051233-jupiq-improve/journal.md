# 회차 노트 2026-10-02-051233-jupiq-improve — jupiq
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:12] base pinned — main@92ff88c
- [러너 05:12] autonomy release — 
- [러너 05:18] scout done — `page_size` 상한이 OpenAPI(`maximum: 100`)와 실제 동작(`pageBounds` 200)으로 갈린 목록 엔드포인트를, **실제 HTTP 응�

## 구현 노트
- 0단계(배정 과제) 재현 실패 = 깨끗한 `npm --prefix web ci` 후 `npm run lint` **exit 0**, test 144개·build 통과. 그래서 web 소스는 손대지 않고 1단계를 구현했다. 바꾼 것: `openapi/openapi.yaml:324` 의 `maximum: 100` → `200` 한 토큰(문서가 외톨이 — `/audit`:559 가 이미 200, `pageBounds` 도 200) + 신규 통합 테스트 1개. `pageBounds`·핸들러·web 무변경.
- **과제서 정정(비평가가 먼저 볼 곳)**: 정찰이 `/users` 핸들러를 `localUsers`/`ListLocalUsers` 라고 적었지만 틀렸다. 실제는 라우터 `core_handlers.go:38` → `managedUsers`(:1254) → `pageQuery`(helpers.go:111, 기본 20) → `ListManagedUsersWithAccess`(hubs.go:628) → `pageBounds`(hubs.go:629). `localUsers` 는 `/local-users`(:40)다. 결론(상한 200)은 같다.
- **확신 없는 곳**: (1) 적재된 `npm run lint` 실패가 *다른 환경*(GitHub 러너)에서는 날 수도 있다 — 나는 로컬 Node 로만 확인했고 `gh` 로 원격 run 로그는 못 봤다. (2) `maximum` 을 200 으로 올리는 것이 "문서가 약속을 넓힌다" 는 뜻이므로, 100 이 의도된 제품 결정이었다면 방향이 거꾸로다. 코드·다른 문서·web 어디에도 100 을 쓰는 곳이 없어 문서 오타로 판단했지만 이것은 판단이다.
- **일부러 하지 않은 것**: 핸들러에 상한 100 을 주는 쪽(6개 공유 호출처를 건드리지 않으려고), `/audit` 에 같은 테스트를 더하는 것(fixture 격리 수단이 없어 범위를 넘김 — ideas.json 에 올렸다), `openapi` maximum 전체를 일괄 검증하는 계약 테스트(사례가 두 번 반복된 뒤에 할 일).
- **다음 역할이 조심할 것**: 새 테스트는 `JUPIQ_INTEGRATION_TEST_DSN` 이 있어야 돈다(없으면 skip). fixture 는 managed_users 260건 + hub 1 + 관리자 1이고 Hub CASCADE 로 정리한다 — DSN 없이 `go test ./...` 가 통과하는 것은 증거가 아니다. 상한을 openapi.yaml 에서 파싱해 쓰므로 문서의 그 값을 바꾸면 테스트가 따라 움직인다(의도된 것: 문서와 동작이 갈리면 실패).
- 검증 전부 통과: gofmt/vet/build, `go test ./...`, `-race ./...`, 계약 4개, `make test-integration`(store 4.560s/api 1.217s), `-race` 통합, 통합 `-v` SKIP 0건, fixture 잔여 0, check-version 1.8.7. 임시 컨테이너 제거.
- [러너 05:26] brief accepted — 채택 — 0단계는 정찰의 예상대로 통과했고(`npm run lint` exit 0), 1단계의 "미확인" 두 가지(`/users` path 와 그 핸들러가 `pageBoun
- [러너 05:26] verify failed — 실패한 검증: npm run lint   # CI에서 가져옴 (exit 1)
