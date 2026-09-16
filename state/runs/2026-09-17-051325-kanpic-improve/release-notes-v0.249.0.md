# v0.249.0

## 넘겨받은 워크북이 어디서 왔는지 편집기에서 보입니다

v0.246.0 부터 다른 사내 서비스가 kanpic 으로 문서를 보내면 새 워크북이
만들어지고 그 워크북으로 넘어옵니다. 어디서 왔는지는 남아 있었지만
`GET /api/v1/workbooks/{id}/handoff` 로 **물어보아야만** 알 수 있었습니다.
화면에서는 방금 열린 워크북이 내가 만든 것인지 ptium 이 보낸 것인지 알 길이
없었습니다.

이제 편집기 제목 아래 저장 상태 줄(`저장됨 · v12`) 끝에 **ptium 에서 받음**
이 붙습니다. 마우스를 올리면 오리진·파일 이름·받은 시각을 말합니다. 출처는
바뀌지 않으므로 워크북을 열 때 한 번만 묻고, 넘겨받은 것이 아닌 워크북(404)
에는 아무것도 붙이지 않습니다 — 대부분의 워크북이 그렇습니다.

## 이름은 허용 목록에서 옵니다

출처에는 오리진(`https://ptium.internal`)만 남습니다. 사람이 부르는 이름은
서버가 답할 때 `handoff.peers` 에서 찾아 `service` 로 함께 줍니다. 그 사이
그 서비스를 허용 목록에서 지웠으면 이름 없이 오리진만 답하고, 화면은
호스트(`ptium.internal 에서 받음`)로 부릅니다. 관리자 가이드의
**다른 서비스와 문서 주고받기** 절에 화면 표시와 `service` 를 적고 PDF 를
다시 구웠습니다.

## 확인

- `internal/httpapi/handoff_test.go` — 두 서버 사이 실제 보내기 → 받기 → 출처
  확인에 `service` 가 허용 목록의 이름으로 오는지 더했습니다
- `web/src/lib/handoff.test.ts` — 이름·호스트·주소가 아닌 값의 표기와 툴팁,
  404 → null · 200 → 값 · 500 → 오류

`gofmt`·`go vet`·`go build`·`go test ./...`(전체 통과),
`npm run lint`·`npm test`(70 파일 498개 통과)·`npm run build`,
`scripts/check-release-docs.sh`, `scripts/check-commit-identities.sh` 를
돌렸습니다.

## 올리실 때

마이그레이션과 새 설정은 없습니다. `GET /api/v1/workbooks/{id}/handoff` 응답에
`service` 가 **더해졌을 뿐** 기존 필드는 그대로입니다. v0.248.0 에서 올라오시는
분은 이미지만 바꿔 다시 띄우시면 됩니다.
