# v0.259.0 릴리즈 검증 기록

- 릴리즈 커밋: `2b2624826ecd91a73d7d4b73b83afc4781d1bf19` (`chore(release): 0.259.0`). detached HEAD 유지, 작성자 hkjang, 트레일러 없음.
- 이전 태그 v0.258.0/v0.257.0/v0.256.0의 stat·커밋·태그 객체를 확인했으며 모두 경량 태그, 동일한 11개 배포/문서 버전 파일 갱신 방식이다.
- 저장소 내 CHANGELOG/별도 릴리즈 노트 파일은 없다. 한국어 GitHub Release 본문은 `.github/workflows/release.yaml`에서 `scripts/release-catalog-images.sh notes`로 생성한다. 이번 변경의 한국어 본문은 `release-notes.md`에 보존했다.
- 이전 GitHub 자산 8종 모두 워크플로가 생성한다: 제어 이미지 tar.gz, SPDX JSON, Sigstore JSON, SHA256SUMS, offline-bundle.json, 정적 offline helper, offline compose, offline env 예제. 따라서 로컬 게시 자산 배열은 비운다. PDF는 기존 릴리즈 자산이 아니다.
- 워크플로는 태그 `v*.*.*` 푸시로 테스트·이미지 빌드/상태 확인·자산 생성·서명·GitHub Release 게시까지 수행한다. `github_release=false`는 러너가 별도 GitHub Release를 생성할 필요가 없다는 의미다.
- 런타임 버전 파일과 web/package.json·package-lock.json의 독립 버전 0.1.0, buildinfo/Dockerfile 개발용 기본값은 이전 관례대로 유지한다.

## 로컬 검증

- `go test -race ./cmd/... ./internal/...` 통과 (버전 수정 후 실행, release-go-test.log).
- `npm --prefix web ci` 통과. lint·test:sso(12/12)·build 통과 (release-web-node22.log).
- 최초 SSO 테스트 실패는 npm이 상위 `/home/hkjang/node_modules/.bin/node`의 Node 20을 선택한 환경 문제였다. 저장소 밖의 release-node22-shell.sh로 CI와 같은 Node 22.23.1을 고정해 해결했다. 저장소·상위 node 설치를 바꾸지 않았다.
- `make build` 통과 (release-build.log).
- 카탈로그 validate/check-versions 및 Kubernetes kustomize 통과.
- Docker Compose config와 offline compose의 DSN 누락 거절·서비스 2개·v0.259.0 이미지 일치 검증 통과. 실제 비밀 대신 저장소 CI의 공개 검증 fixture를 프로세스 내에서만 사용했고 Compose JSON은 출력·저장하지 않았다.
- `git diff --check` 통과. 기능 코드 변경 없음.

## 적용 스킬 및 운영 인계

- Skill 도구가 제공되지 않아 로컬 marketing:product-launch 및 technology:release-and-deployment의 SKILL.md와 references/sources.md를 읽었다. 이번 저장소 릴리즈에는 나열된 앱스토어 정책·광고 주장·라이선스 판정이 필요하지 않았다.
- 마케팅 분류: Tier 3, 릴리즈 노트만 준비. 대상은 Momento/Matomo 추적 설정 관리자이며, 300룬 초과 출처가 있는 기존 설정의 쓰기 실패와 수정 방법을 본문에 적었다. 외부 공지는 하지 않았다.
- DB 스키마 변경 없음. 실제 운영 배포나 기능 플래그 변경 없음.
- 배포 담당자는 최초 검증 환경에서 정상 300룬 출처 저장 성공, 301룬 출처 저장 거절을 확인하고 정상 설정 저장 실패가 1건이라도 생기면 확대를 중단한다. 판단 담당은 해당 환경 운영자이며, 이 세션에서 특정 개인을 임의 지정하지 않았다.
- 되돌리기는 직전 검증 이미지 v0.258.0으로 복귀하는 절차를 운영자가 선택한다. DB 마이그레이션은 없으나 되돌리면 과대 출처 거절도 사라진다. 이 세션에서는 실행하지 않았다.
- 배포 후 첫 24시간에 검증한 설정 저장 시나리오의 기대 동작 일치율 100%와 정상 출처의 저장 실패 0건을 확인하도록 인계한다. 실제 운영 측정 결과나 외부 사용자의 독립 검증을 수행했다고 주장하지 않는다.

## 미실행 범위

- 실제 DSN 제공 여부: 없음. DB live/실제 HTTP 저장 검증 및 외부 사용자 검증 미실행.
- 원격 main 포함 여부, 실제 GitHub 자산 조회·재사용 계획, Docker 이미지 빌드·컨테이너 상태 점검, SBOM/OIDC 서명/게시 검증은 태그 푸시 후 CI에서 수행한다. 모든 원격 전송은 사용자 지시에 따라 외부 러너에 맡겼다.

## 버전 인벤토리

```json
{
  "VERSION": "0.259.0",
  "BASE_VERSION": "0.26.0",
  "BROWSERCODE_VERSION": "0.2.0",
  "GOOSE_VERSION": "0.1.0",
  "HOLMES_VERSION": "0.2.0",
  "JUPYTER_VERSION": "0.1.0",
  "LANGFLOW_VERSION": "0.2.0",
  "N8N_VERSION": "0.2.0",
  "NODERED_VERSION": "0.1.0",
  "OPENCODEREVIEW_VERSION": "0.1.0",
  "OPENHANDS_VERSION": "1.43.1",
  "ORCA_VERSION": "0.5.0",
  "PI_VERSION": "0.1.0",
  "PRIMEAGENT_VERSION": "0.1.0",
  "QWENCODE_VERSION": "0.2.0"
}
```

## 최종 확인

- 커밋 후 check-versions/validate 및 `go test -count=1 ./internal/buildinfo` 통과.
- 경량 태그 v0.259.0이 릴리즈 HEAD를 가리키며 작업 트리는 깨끗하다.
- release.json을 released로 원자적으로 기록하고 JSON 재읽기를 검증했다.
