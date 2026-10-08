# 수정 과제 검증 — 81a8b5e

시작점 e54875c, 깨끗한 트리, x/text v0.40.0. 최종 저장소 변경은 go.mod/go.sum의 3줄 교체(2파일)뿐이고 8e877bf의 해당 두 파일과 일치한다. 버전 0.76.8 유지. 신규 시험/앱 코드 0개. 기존 PR 웹 변경은 시작 트리에 없으며 가져오거나 재구현하지 않았다.

## 원인과 반증

저장된 PR #170 실패 로그는 ci.yml의 Go vulnerability scan이며, 이 단계는 별도 스크립트 없이 govulncheck CLI를 실행한다. release.yml에는 이 스캔이나 npm test가 없으므로 릴리즈 실패 두 건으로 단정할 수 없다. 2026-10-07 공개된 [공식 GO-2026-6629 권고](https://pkg.go.dev/vuln/GO-2026-6629)는 secure/precis의 v0.41.0 미만을 대상으로 한다. 실제 스캔은 `Store.beginExternalLease → pgx.ConnectConfig → precis.Profile.String`을 검출했다. 사용자 입력으로 악용 가능함을 입증한 것은 아니다.

- Red: `impl-vuln-base.log`, v0.40.0, GO-2026-6629, go run exit 1 (내부 govulncheck exit status 3).
- Green: `impl-vuln-fixed.log`, v0.41.0, `go mod verify`의 `all modules verified`, 같은 스캔 exit 0.
- 반증: go.mod/go.sum만 원래 HEAD 내용으로 잠시 복구하여 같은 식별자/호출 경로와 exit 1 확인(`impl-vuln-reverted.log`); finally 블록에서 수정본 복구.
- 최종: `impl-vuln-final.log`, `No vulnerabilities found.`, 호출 가능한 취약점 0, exit 0. 비호출 패키지 1건/모듈 4건 경고는 남아 있다.

새 테스트 파일을 추가하지 말라는 과제서에 따라 기존 CI의 실제 모듈/호출 그래프 스캔을 Red/Green 검증으로 사용했다. 소스 문자열 검사나 가짜 PRECIS 객체를 행동 증거로 사용하지 않았다. 웹 설치 문제는 root package.json이 web에 위임하는 구조에서 생기므로 실제 `npm ci --prefix web` 이후 기존 루트 npm test를 실행해 해결했다. 가드/스캔 완화 없음.

## 실제 실행 결과

도구 버전: Go 1.26.7, Node v22.23.1, npm 10.9.8. 아래 모두 현재 수정 트리에서 실행했으며 스캔 원복 실험 이후 Go 검증, 이어 웹 설치/검증 순서다.

| 명령 | exit | 초 | 증거 |
|---|---:|---:|---|
| `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` | 0 | 4.66 | `impl-vuln-final.log` |
| `go vet ./...` | 0 | 0.99 | `impl-vet.log` |
| `go test -p 1 ./... -count=1` | 0 | 80.79 | `impl-go-tests.log` |
| `go test -race -count=1 ./internal/intelligence` | 0 | 6.23 | `impl-race.log` |
| `go build ./...` | 0 | 0.93 | `impl-go-build.log` |
| `./scripts/check-version.sh` | 0 | 0.02 | `impl-version.log` |
| `git diff --check` | 0 | 0.12 | `impl-diff-check.log` |
| `go test ./internal/store -run 'Test.*Integration' -count=1 -v` | 0 | 41.83 | `impl-store-integration.log` |
| `npm ci --prefix web` | 0 | 16.46 | `impl-npm-ci.log` |
| `npm test --silent` | 0 | 9.34 | `impl-npm-test.log` |
| `make test-web` | 0 | 13.43 | `impl-test-web.log` |

`go mod verify`: exit 0, impl-vuln-fixed.log. 최종 git diff --check 및 기존 8e877bf와 go.mod/go.sum 비교도 exit 0. Go 전체 15패키지 PASS, migrations는 시험 없음. 웹 시험은 두 실행 모두 23파일/240개 PASS. make test-web의 offline-queue/audit/typecheck/lint/i18n/test/build/verify:pwa 전부 실행; audit 0 vulnerabilities, i18n 1060키, PWA 150 assets. 미변경 src 파일 lint warning 50개, Prettier 통과.

## DB 증거와 한계

새로 만든 `umm-verify-20261009-040829` PostgreSQL 17.11 컨테이너, loopback 포트 62427과 tmpfs 데이터로 격리. 운영 DB/기존 DB를 사용하지 않았다. 전체 Go 시험과 verbose store 시험에 이 DSN을 지정했고 완료 뒤 컨테이너/볼륨 삭제 확인(`impl-db-environment.log`). store 최상위 시험 156 PASS, 3 SKIP, 0 FAIL이며 `TestLongLeaseCapacityDoesNotConsumeRequestPoolIntegration` 및 AI/webhook 하위 시험은 PASS.

SKIP을 DB 통과로 세지 않았다:

- TestAutoLinkSkipsNearDuplicatesIntegration: 의미 임베딩 백엔드가 필요.
- TestMergeInvalidatesTheStoredVectorIntegration: 저장된 벡터가 없어 무효화 동작 미검증.
- TestClustersAndRelatedSurviveASentenceEmbeddingBackendIntegration: 실제 임베딩 URL/모델 미설정.

원격 GitHub CI, 전체 CI의 DB smoke/drill, 제품 브라우저 E2E, Docker 이미지 빌드, 릴리즈는 이번에 실행하지 않았다. 의존성 두 파일 이외의 빌드/릴리즈 배선은 변경하지 않았다. 이미지 통과나 원격 실패 2건을 주장하지 않는다.

## 완료 확인과 후속 주의

요청한 technology:completion-verification, technology:systematic-debugging, technology:test-driven-development는 Skill 도구가 제공되지 않아 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/<name>/SKILL.md`를 읽고 적용했다. 실패 재현·최소 수리·원복 반증·현행 트리 전체 지정 검증·출력과 SKIP 검토를 수행했다.

작성자 hkjang, 트레일러 없는 커밋 81a8b5e. git add 전에 수정 두 파일만 존재하고 미추적 파일 없음을 확인했다. web/dist와 web/node_modules는 기존 ignore 규칙 적용으로 미커밋. 원격 조작/푸시 없음. 외부 러너는 루트 node_modules 존재 여부로 web 설치를 판단하지 말고 `npm ci --prefix web`를 선행해야 한다. ideas.json 기존 18개 항목을 유지하고 선택한 수정 과제만 done으로 갱신했다.
