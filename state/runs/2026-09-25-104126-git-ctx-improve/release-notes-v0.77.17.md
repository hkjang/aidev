# git-ctx v0.77.17

이번 릴리스는 **MCP 도구 호출의 캐시 키를 만드는 과정이 호출자의 ACL 주체 목록을 제자리에서 정렬해 버리던 문제**를 고칩니다. 캐시 키 계산은 읽기만 해야 하는 자리인데, 제한된 권한의 주체에서는 요청이 계속 들고 다니는 슬라이스를 직접 건드렸습니다. 기본 서비스 포트는 계속 `4747`이며 기존 API·MCP 호환성을 유지합니다.

> **재색인도 마이그레이션도 필요하지 않습니다.** 저장되는 내용과 검색 결과의 접근 판정은 바뀌지 않습니다.

## 수정

### 캐시 키를 계산하면 요청의 ACL 주체 순서가 바뀌었습니다

- `internal/mcp/cache.go`의 `cacheKey`는 `principals := principalACLs(p)` 로 받은 슬라이스를 그대로 `sort.Strings` 했습니다. `principalACLs`(`internal/mcp/args.go:170`)는 `p.ACLPrincipals` 가 비어 있지 않으면 그 슬라이스를 그대로 넘기고, `search.WithUnrestricted`(`internal/search/service.go:230-233`)도 제한 있는 역할에서는 받은 슬라이스를 그대로 돌려줍니다. 그래서 캐시 키 하나를 만드는 동작이 호출자의 `auth.Principal.ACLPrincipals` 를 제자리 정렬했고, 같은 요청에서 그 뒤에 주체 목록을 읽는 구간은 재배열된 순서를 봤습니다.
- 같은 함수의 바로 다음 줄이 근거였습니다 — `repositories := append([]string(nil), p.AllowedRepositories...)` 는 복사한 뒤 정렬하는데 주체 쪽만 그러지 않는 비대칭이 남아 있었습니다. 주체도 같은 관용구로 복사한 뒤 정렬하게 한 줄 고쳤고, 두 슬라이스를 모두 복사하는 이유를 주석에 적었습니다.
- `sort.Strings` 는 그대로 뒀습니다. 주체 집합이 같고 순서만 다른 두 요청이 같은 캐시 항목을 맞히는 것은 의도된 동작이며, 이번 수정은 그 정렬이 호출자에게 새는 것만 막습니다.

## 영향 범위

- **잘못된 응답이 나가던 사례는 확인되지 않았습니다.** ACL 판정과 캐시 키는 모두 순서에 무관한 집합 연산이므로, 재배열된 순서를 읽는 구간에서 다른 답이 나오는 경로를 찾지 못했습니다. 이 릴리스는 "키를 계산하는 함수가 입력을 변경하지 않는다"는 계약을 되돌리는 수정입니다.
- 접근 판정, 캐시 적중률, 캐시 키 값 자체는 바뀌지 않습니다.

## 검증

- 실제 `store.Open("sqlite", …)` fixture 의 실제 `mcp.Server` 와 실제 `auth.Principal` 로 `s.cacheKey` 를 호출하는 회귀 시험(`internal/mcp/cache_test.go`). 대역 타입을 쓰지 않습니다.
- 제한된 주체와 무제한 주체 두 분기를 표로 두고, `p.ACLPrincipals` 의 순서·내용이 그대로 남는지와 "순서만 다른 같은 주체 집합 → 같은 키" 를 함께 단언합니다.
- 수정 전 코드에서 제한 분기만 실패함을 먼저 확인했습니다 — `cacheKey mutated the caller's ACL principals: got [alice middle zeta], want [zeta alice middle]`. 이미 복사 경로인 무제한 분기는 수정 전에도 통과합니다.
- 일반·FTS5 빌드와 전체 테스트, 전체 race, `vet`·`gofmt` 통과
- 버전 정합성·회귀 시험, Kubernetes Kustomize 렌더링과 콘솔 구문·계약 시험 통과
- `govulncheck ./...` (v1.7.0): `No vulnerabilities found`
- 로컬 검증 결과는 `docs/completion-audit.md`에 기록했습니다.
- PostgreSQL·pgvector·Vault 통합, Docker 이미지 빌드·오프라인 아카이브 검증 및 GitHub 공개는 태그 푸시 후 릴리스 워크플로에서 수행합니다.

## 업그레이드 참고

- 마이그레이션과 재색인은 필요하지 않습니다.
- 설정·환경변수·API 스키마 변경은 없습니다.
- `search.WithUnrestricted` 가 입력 슬라이스를 그대로 돌려주는 성질 자체는 이번 릴리스에서 건드리지 않았습니다. 호출자 세 곳을 함께 검토해야 하는 변경이라 한쪽만 맞는 상태를 만들지 않았습니다.

## 오프라인 Docker 이미지

릴리스 워크플로가 다음 두 파일을 만들어 검증 후 첨부합니다.

- `git-ctx-v0.77.17.tar.gz`
- `git-ctx-v0.77.17.tar.gz.sha256`

```bash
sha256sum -c git-ctx-v0.77.17.tar.gz.sha256
gzip -dc git-ctx-v0.77.17.tar.gz | docker load
docker image inspect git-ctx:v0.77.17 --format '{{.Os}}/{{.Architecture}} {{.Config.User}}'
```

기대 결과는 `linux/amd64 10001`입니다. 아카이브에는 `git-ctx:v0.77.17`과 `git-ctx:0.77.17` 태그가 포함됩니다.

**전체 변경 내역**: https://github.com/hkjang/git-ctx/compare/v0.77.16...v0.77.17
