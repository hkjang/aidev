# git-ctx v0.77.16

이번 릴리스는 **파일 읽기와 컨텍스트 내보내기의 본문이 바이트 예산에서 잘릴 때 한글·이모지 등의 마지막 문자가 깨지던 문제**를 고칩니다. v0.77.15가 MCP 응답의 긴 줄에서 고친 것과 같은 문제를, 그 위의 검색 서비스 경로에서도 고칩니다. 기본 서비스 포트는 계속 `4747`이며 기존 API·MCP 호환성을 유지합니다.

> **재색인은 필요하지 않습니다.** 저장된 내용은 바뀌지 않으며, 응답을 자르는 과정에서 원문의 UTF-8 문자 경계를 보존합니다.

## 수정

### 같은 텍스트를 자르는 자리가 두 군데 더 있었습니다

- `internal/search/service.go`의 `ReadFile`은 본문이 192KiB를 넘으면 `body[:readFileByteBudget]`로, `ExportContext`는 200000바이트를 넘으면 `result[:200000]`으로 잘랐습니다. 한글은 한 글자가 3바이트라 예산이 글자 중간에 떨어지면 invalid UTF-8이 남았고, `encoding/json`이 그 꼬리를 `U+FFFD`로 바꿔 내보내므로 REST 플레이그라운드를 쓰는 호출자는 마지막 글자가 깨진 본문을 받았습니다.
- 두 자리 모두 새 `cutAtRuneBoundary`로 물러나 자르도록 고쳤습니다. 예산 상수(192KiB·200000바이트), `Truncated` 플래그, `truncated: returned lines …` 진단, 내보내기의 절단 공지 문구는 그대로입니다.
- 헬퍼는 `internal/mcp`의 `runeSafeCut`과 같은 계약이지만 의도적으로 공유하지 않았습니다 — `mcp`가 `search`를 임포트하므로 되돌려 임포트하면 순환이 됩니다. 사유는 주석에 적어 두었습니다.

## 검증

- 실제 `store.Open("sqlite", …)`와 실제 `Service.ReadFile`·`Service.ExportContext`를 거치는 회귀 시험 2건(`internal/search/truncation_test.go`). 수정 전 코드에서 실패함을 먼저 확인했습니다 — `the returned content is not valid UTF-8`, `the export is not valid UTF-8`.
- UTF-8 유효성 단언을 지운 채로도 JSON 단언만으로 수정 전 코드가 실패함을 확인했습니다(`the JSON response escapes a replacement character`). 사용자에게 나가는 출력이 실제로 바뀝니다.
- 예산이 3바이트 문자의 각 오프셋에 떨어지는 네 가지 배치, 4바이트 이모지, 예산 미만 본문(무변경), ASCII 본문(정확히 예산에서 절단)을 함께 확인합니다.
- 일반·FTS5 빌드와 전체 테스트, `./internal/search` race, `./internal/mcp`·`./internal/app` 테스트, `vet`·`gofmt` 통과
- 버전 정합성·회귀 시험, 빌드 모드 교차, Kubernetes Kustomize 렌더링과 콘솔 구문·계약 시험 통과
- `govulncheck ./...` (v1.7.0): `No vulnerabilities found`
- 로컬 검증 결과는 `docs/completion-audit.md`에 기록했습니다.
- PostgreSQL·pgvector·Vault 통합, Docker 이미지 빌드·오프라인 아카이브 검증 및 GitHub 공개는 태그 푸시 후 릴리스 워크플로에서 수행합니다.

## 업그레이드 참고

- 마이그레이션과 재색인은 필요하지 않습니다.
- 이번 수정은 절단 자리의 문자 경계만 다룹니다. 예산 값 자체나 최종 응답 전체의 바이트 상한에 대한 변경은 포함하지 않습니다.
- 예산이 글자 중간에 떨어질 때 본문이 최대 3바이트 짧아집니다. 잘린 본문의 바이트 길이에 의존하는 호출자는 없어야 합니다.

## 오프라인 Docker 이미지

릴리스 워크플로가 다음 두 파일을 만들어 검증 후 첨부합니다.

- `git-ctx-v0.77.16.tar.gz`
- `git-ctx-v0.77.16.tar.gz.sha256`

```bash
sha256sum -c git-ctx-v0.77.16.tar.gz.sha256
gzip -dc git-ctx-v0.77.16.tar.gz | docker load
docker image inspect git-ctx:v0.77.16 --format '{{.Os}}/{{.Architecture}} {{.Config.User}}'
```

기대 결과는 `linux/amd64 10001`입니다. 아카이브에는 `git-ctx:v0.77.16`과 `git-ctx:0.77.16` 태그가 포함됩니다.

**전체 변경 내역**: https://github.com/hkjang/git-ctx/compare/v0.77.15...v0.77.16
