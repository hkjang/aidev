# 수리 결과: 고칠 결함을 찾지 못했음 — 커밋 없음

- **CI 4개 잡 전부 30928ef 에서 로컬 재현 실패(= 통과)**: `go build`·`go vet` ok, `TEST_DATABASE_URL` 를 준 `go test ./... -count=1 -p 1` 전부 ok(`internal/tools` 5.425s 로 DB 통합 테스트 실제 실행), `web: npm ci && npm run check && npm run build` ok, `plugin: mvn -B -ntp package` BUILD SUCCESS, `docker build` + 기동 점검(`/healthz`·`/api/config`) ok. CI 의 `go-version: '1.24'` 차이도 `golang:1.24.13` 컨테이너에서 빌드·vet·신규 테스트 9개 전부 통과로 배제했다.
- **실패는 스텝 실행 전에 났다**: 4개 잡이 모두 `2026-10-16T22:14:52Z` 에 동시 시작해 2~3초 만에 실패했고, 저장된 체크런(`../2026-10-17-065908-bamcp-improve/ci-*.json`)에 `실패 단계` 가 비어 있으며 잡마다 주석이 정확히 2개다. 즉 각 잡의 첫 스텝(`actions/checkout@v4`) 이전/직후에 러너 쪽에서 끊겼다는 신호다.
- **이 diff 로는 설명 불가**: 변경은 `internal/tools/{args.go,args_test.go,integration_test.go}` 3개뿐이라 `npm`·`mvn`·`docker build` 잡을 깨뜨릴 경로가 없다.
- **확인 못 한 것**: 실패 로그·주석 본문. 저장소가 private 이고 이 세션에 `gh` 인증도 외부 웹 접근도 없다(`gh auth` 미설정, API 404, WebSearch 권한 거부). 다음 역할은 토큰으로 `gh run view 38974581497 --log-failed` 또는 체크런의 `annotations_url` 을 읽어 확인하라. 가장 유력한 가설은 **deprecated 액션 버전 강제 실패** — 네 잡의 첫 스텝이 모두 `actions/checkout@v4` 인데 이 환경의 다른 저장소들은 이미 `checkout@v6/v7`·`setup-go@v7`·`setup-node@v7` 를 쓴다.
- **고치지 않은 이유**: 남은 후보 조치는 `.github/workflows/ci.yml` 의 액션 버전 상향뿐인데, 그것은 이 PR 범위 밖의 파일이고 원인을 로그로 확인하지 못한 상태에서 CI 를 손대 통과시키는 일이라 절대 규칙에 걸린다. 따라서 커밋 없이 끝낸다.
