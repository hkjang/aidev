# Postra v0.25.6 — 갓 내려받은 체크아웃에서 프런트엔드 게이트가 돌아갑니다

**개발 환경 전용 유지 보수 릴리즈입니다.** `make frontend-test` 가 의존성이 설치되지 않은 깨끗한 체크아웃에서 `tsc: not found` 로 즉시 죽던 것을 고치고, README 의 govulncheck 예시를 CI 가 실제로 설치하는 고정 버전에 맞췄습니다. **프로덕션 Go/TypeScript 코드 변경은 0줄입니다** — 기능 변경, 새 설정 키, 새 환경변수, DB 마이그레이션, 화면 변경이 **없습니다**.

**운영자가 할 일은 없습니다.** 실행 파일과 컨테이너 이미지의 동작은 v0.25.5 와 같습니다. 이번 변경이 닿는 사람은 저장소를 내려받아 프런트엔드 검사를 돌리는 개발자와 CI 를 재현하려는 사람뿐입니다.

## 깨끗한 체크아웃에서 깨지던 `make frontend-test`

- `Makefile` 의 `frontend-test` 는 `npm run typecheck` 와 `npm test` **두 줄뿐**이었고, 의존성을 설치하는 것은 `frontend` 타깃만이었습니다. 그래서 `web/node_modules` 가 없는 상태 — 갓 clone 한 저장소, 새 러너 워크트리 — 에서는 첫 줄에서 바로 죽었습니다: `sh: 1: tsc: not found`, `make: *** [Makefile:16: frontend-test] Error 127`.
- 실제 피해는 멈춘 검사가 아니라 **잘못된 신호**입니다. 프런트엔드 게이트를 재현하려면 `make frontend` 를 먼저 돌아야 한다는 사실이 어디에도 없었고, 첫 실패는 "내 기계가 깨졌나" 처럼 보였습니다.
- 고친 방법은 멱등 가드 한 줄입니다: `@test -d web/node_modules || (cd web && $(NPM) ci --no-audit --no-fund)`. 깨끗한 트리는 한 번 설치하고, 이후 실행은 곧장 typecheck 로 넘어갑니다.
- **무조건 `npm ci` 를 앞세우지 않은 이유**가 있습니다. 그러면 호출마다 수 분이 걸려 빠른 내부 반복 명령으로서의 쓸모가 사라집니다. 플래그는 `frontend` 타깃과 같게 맞췄고, 설치가 실패하면 레시피가 **그 자리에서 중단**됩니다 — 돌아갈 수 없는 typecheck 로 흘러내려 조용히 통과하는 것이 원래 결함보다 나쁘기 때문입니다.
- 가드는 디렉터리 존재만 봅니다. 따라서 `web/node_modules` 가 있지만 `package.json` 이 바뀌어 내용이 낡은 경우는 다시 설치하지 않습니다. 이것이 의도한 계약이고, `frontend`·`frontend-check` 는 여전히 무조건 `npm ci` 를 하므로 **CI 재현성은 그대로입니다**.

## README govulncheck 예시의 고정 버전

- README 보안 절은 "보안 스캐너도 고정 버전으로 설치해 재현성을 확보합니다" 라고 적어 두고, 두 줄 아래의 로컬 예시는 `govulncheck@latest` 를 불렀습니다. 한 절 안의 자기모순이었고, 바로 옆 `make lint-security` 줄은 이미 CI 가 쓰는 gosec 버전을 적고 있었습니다.
- 예시를 `ci.yml` 이 설치하는 `@v1.6.0` 으로 맞췄습니다. 쓰기 전에 그 버전을 직접 돌려 확인했습니다 — 이 코드에 영향을 주는 취약점 0건, 종료 코드 0.

## 검증

- **되돌려 실패를 먼저 확인했습니다**: `web/node_modules` 가 없는 같은 워크트리에서 가드 없이는 `tsc: not found` / Error 127, 가드를 넣으면 종료 코드 0. 단일 변수 비교입니다.
- **멱등성**: 수정 후 첫 실행은 의존성을 설치하고(`added 249 packages`) 50개 파일 426개 테스트를 통과, 이어진 재실행은 설치 줄 없이 곧장 typecheck 로 들어가 8초에 끝났습니다. 테스트를 건너뛰거나 `--passWithNoTests` 로 약화시키지 않았고 426개 전부 그대로 실행됩니다.
- **설치 실패를 삼키지 않음**: `make frontend-test NPM=false` → `Error 1` 로 중단, typecheck 로 흘러내리지 않습니다.
- `go test -race -count=1 -timeout=900s ./...` 25개 패키지 실패 0건
- `go build ./...`, `go vet ./...`, `gofmt` 무지적, `go mod tidy` 무변경, canonical API 계약 검사(`postra-contracts -check`)
- `gosec -severity medium` — 141개 파일 지적 0건, `govulncheck@v1.6.0` — 영향 취약점 0건
- `make frontend-check` 통과 후 `internal/transport/spa/assets` 변화 없음 — 태그에 담긴 `/app` 번들이 최신입니다(릴리즈 워크플로가 같은 검사를 다시 합니다)
- 외부 PostgreSQL(`POSTRA_TEST_PG`)과 브라우저 e2e 는 돌리지 않았습니다 — 프로덕션 코드가 바뀌지 않아 범위 밖입니다.

## 업그레이드 및 호환성 주의사항

DB 마이그레이션, 새 설정 키, 새 필수 환경변수가 없습니다. 실행 파일 또는 컨테이너 이미지를 교체한 뒤 재시작하면 끝이며, **교체하지 않아도 잃는 것이 없습니다** — 운영 동작은 v0.25.5 와 동일합니다. 되돌릴 때도 v0.25.5 로 그대로 내려갈 수 있습니다.

- 저장소를 내려받아 검사를 돌리는 개발자는 이제 `make frontend-test` 한 줄로 프런트엔드 게이트를 재현할 수 있습니다. 첫 실행은 의존성 설치 때문에 느리고, 이후는 빠릅니다.
- `package.json` 을 바꾼 뒤에는 `make frontend` 또는 `make frontend-check` 로 의존성을 다시 설치하세요. `frontend-test` 의 가드는 디렉터리가 이미 있으면 설치하지 않습니다.
- v0.25.5 의 거부된 `LIST` 크기 선별 경고, v0.25.4 의 크기 제한 없는 계정 본문 잘림 수정, v0.25.3 의 POP3 `fetch` 진단 교정, v0.25.2 의 OpenTelemetry v1.45.0, v0.25.1 의 "계정당 인바운드 연결 하나"와 `mail.idle_enabled`, v0.25.0 의 동기화 실패 진단은 바뀌지 않았습니다.

## 배포 파일

- `postra-0.25.6-linux-amd64`: 정적 단일 실행 파일
- `postra-0.25.6.tar.gz`: 폐쇄망 `docker load` 이미지, `postra:0.25.6` (linux/amd64)
- `postra-0.25.6-sbom.cdx.json`: Go CycloneDX SBOM
- `postra-0.25.6-frontend-sbom.cdx.json`: 프런트엔드 CycloneDX SBOM
- `SHA256SUMS.txt`: 배포 파일 SHA-256
