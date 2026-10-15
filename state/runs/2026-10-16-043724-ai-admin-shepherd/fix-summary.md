# 수리 요약 — 커밋 없음 (CI 실패를 재현하지 못함)

- CI 가 가리킨 lint·test 를 브랜치 HEAD(83f264c) 그대로 로컬에서 전부 돌렸고 **모두 통과**했다: `gofmt -l .`(출력 없음), `go vet ./...`, `./scripts/verify-version.sh`(1.2.37 일치), 전용 `postgres:16-alpine`(포트 55617) 로 `TEST_POSTGRES_DSN=… go test -race -count=1 ./...`(internal/server 157.4s ok, 전 패키지 ok), `TestApprovalNULWrites` 14 subtest `--- PASS`, `cd web && npm ci && npm test && npm run build`, `docker build --build-arg VERSION=v1.2.37`, `git diff --check`.
- 따라서 이번 변경(workflow.go 두 검증 지점 + 테스트 + docs)에는 고칠 결함이 보이지 않는다. 고치지 않고 커밋 없이 끝낸다.
- 실패 원인 추정: lint·test **두 잡이 모두 2초** 만에 실패했고 러너가 실패 로그를 하나도 캡처하지 못했다. 2초는 저장소별 첫 스텝(gofmt / Keycloak 기동)에 닿기 전이며, 두 잡이 그 시점에 공유하는 것은 `actions/checkout` 과 `actions/setup-go@v5.6.0 (go-version 1.26.6, cache: true)` 뿐이다 — 코드가 아니라 런 셋업/인프라 실패로 보인다.
- 확인 못 한 것: `gh` 자격증명이 없어(`gh auth status` 미로그인, GH_TOKEN 없음, 공개 API 404) 실제 CI 로그를 읽지 못했다. `TEST_KEYCLOAK_ISSUER` 없이 돌려 OIDC E2E 1건은 SKIP 이다(평소대로). 워크플로 파일은 규칙대로 손대지 않았다.
- 권장: CI 를 **재실행**해 같은 2초 실패가 반복되는지 보고, 반복되면 `.github/workflows/ci.yml` 의 setup-go 쪽(또는 Actions 쿼터)을 별도 변경으로 다룰 것. 이 PR 의 코드 변경과는 무관하다.
