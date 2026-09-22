# git-ctx v0.77.15

이번 릴리스는 **MCP 응답의 긴 줄이 바이트 예산에서 잘릴 때 한글·이모지 등의 마지막 문자가 깨지던 문제**를 고칩니다. 기본 서비스 포트는 계속 `4747`이며 기존 API·MCP 호환성을 유지합니다.

> **재색인은 필요하지 않습니다.** 저장된 내용은 바뀌지 않으며, 응답을 자르는 과정에서 원문의 UTF-8 문자 경계를 보존합니다.

## 수정

### 긴 줄을 자른 뒤에는 원래 문자 경계를 알 수 없었습니다

- `cutAtBoundary`는 적절한 줄·공백 경계를 찾지 못하면 `runeSafeCut`으로 잘랐습니다. 이미 바이트 단위로 잘린 `window`를 넘겨 원본 문자의 나머지 바이트를 볼 수 없었고, JSON 응답에서 깨진 문자가 `U+FFFD`로 나타났습니다.
- fallback에 **원본 `text`와 `limit`**를 넘기도록 한 줄을 수정했습니다. 줄·공백 경계 선호, 예산 활용률, 코드 펜스와 Notes 동작은 유지합니다.
- 실제 SQLite → MCP `tools/call(read-file)` → JSON 경로에서 캐시 미스와 히트 모두를 회귀 시험합니다. 2·3·4바이트 문자와 ASCII 혼합 입력, 3000~3011 연속 예산에서 반환 본문이 원문 접두부이며 대체 문자가 없는지 확인합니다.

## 검증

- 운영 함수의 모든 바이트 경계와 절단 없는 입력, 경계 선호·60% 활용·펜스·Notes 회귀 시험
- 실제 read-file HTTP 경로의 UTF-8 보존과 캐시 감사 기록 검사
- 일반·FTS5 빌드와 전체 테스트, FTS5 전체 race·MCP 비캐시 테스트, vet·gofmt 통과
- 버전 정합성·회귀 시험, 빌드 모드 교차·기존 릴리스 DB 업그레이드, Kubernetes Kustomize 렌더링과 콘솔 구문·계약 시험 통과
- `govulncheck ./...` (v1.7.0): `No vulnerabilities found`
- 로컬 검증 결과는 `docs/completion-audit.md`에 기록했습니다.
- PostgreSQL·pgvector·Vault 통합, Docker 이미지 빌드·오프라인 아카이브 검증 및 GitHub 공개는 태그 푸시 후 릴리스 워크플로에서 수행합니다.

## 업그레이드 참고

- 마이그레이션과 재색인은 필요하지 않습니다.
- 이번 수정은 응답 절단의 문자 경계만 다룹니다. 최종 응답 전체의 엄격한 바이트 상한이나 파일 읽기 자체의 192KiB 제한에 대한 변경은 포함하지 않습니다.

## 오프라인 Docker 이미지

릴리스 워크플로가 다음 두 파일을 만들어 검증 후 첨부합니다.

- `git-ctx-v0.77.15.tar.gz`
- `git-ctx-v0.77.15.tar.gz.sha256`

```bash
sha256sum -c git-ctx-v0.77.15.tar.gz.sha256
gzip -dc git-ctx-v0.77.15.tar.gz | docker load
docker image inspect git-ctx:v0.77.15 --format '{{.Os}}/{{.Architecture}} {{.Config.User}}'
```

기대 결과는 `linux/amd64 10001`입니다. 아카이브에는 `git-ctx:v0.77.15`과 `git-ctx:0.77.15` 태그가 포함됩니다.

**전체 변경 내역**: https://github.com/hkjang/git-ctx/compare/v0.77.14...v0.77.15
