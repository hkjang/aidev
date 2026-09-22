- 과제: Makefile test에 CI와 같은 Go race 검사 적용 (가치 2 / 위험 1 / 작업량 S)
- 왜: 루트 Makefile의 test recipe는 `go test ./...`를 실행하지만 `.github/workflows/ci.yml`의 Source tests는 `go test -race ./...`를 실행하여 로컬 표준 검증이 데이터 경쟁 검사를 빠뜨립니다. 로컬 명령에도 race 검사를 적용하면 구현자가 CI 전에 같은 종류의 동시성 오류를 발견할 수 있습니다.
- 수용 기준: 1) 루트 `make test`가 backend에서 `go test -race ./...`와 `go vet ./...`를 순서대로 실행하고 성공한다. 2) 기존 frontend npm ci → lint → test → VERSION 주입 build 실행이 그대로 이어지고 성공하며, test의 도움말에 race 검사 포함을 짧게 명시한다. 3) 새로운 소스 문자열 검사나 가짜 go 실행기를 만들지 않고 실제 `make test` 로그로 기존 Go 테스트가 race 옵션으로 실행됐음을 확인한다. DB 통합 테스트의 실행/skip 여부와 각 단계 결과를 따로 기록한다.
- 건드릴 파일: `Makefile:test` — 첫 recipe의 `go test ./...`에 `-race` 추가, 해당 target 도움말 갱신. 이 파일에는 함수가 없으며 target 이름이 구현 지점이다. 근거로 읽은 `.github/workflows/ci.yml:source`의 Go formatting, tests and vet 단계는 비교만 한다.
- 검증 명령: 저장소 루트에서 `make test`, `make fmt`, `make check`, `git diff --check`. 명령 배선 확인 보조로 `make -n test` 사용 가능하지만 이것만으로 통과 판정하지 않는다. Go만 집중 확인하려면 `cd backend && go test -race ./...`.
- 위험과 피할 것: auth·session·store/migrations·.github/workflows·Dockerfile·배포 태그·의존성을 변경하지 않는다. Docker 빌드의 CGO_ENABLED=0 정책까지 확대하지 않는다. race에는 지원되는 Go 플랫폼 및 C 컴파일러가 필요하며 실패 시 옵션을 조용히 빼거나 자동 fallback하지 않는다. 현재 정찰 Linux 환경에서는 race 실행 성공을 확인했다. 기존 테스트 실패가 나오면 원인을 기록하고 이번 과제를 무관한 기능 수정으로 넓히지 않는다. frontend 단계의 실패를 숨기거나 생략하지 않는다. 보호 경로 수정과 DB 서비스 신설은 이번 범위가 아니다.
- 차선 후보: e2e README 기본 테스트 실행 순서를 package.json과 일치시키기 — 1순위가 구현 시 이미 적용되어 있거나 지원 대상 환경의 race 실행 불가가 확인된 경우만 선택. `e2e/README.md`의 “기본 테스트는 전체 route 브라우저 smoke 뒤에…” 문장이 실제 `e2e/package.json:scripts.test`의 visual → accessibility → smoke 및 바로 앞 문단과 충돌한다. 그 문단만 실제 순서로 고치고 `make check`와 `git diff --check`로 검증한다. npm test 순서·시각 기준 이미지·로그인 배선은 바꾸지 않는다.

근거와 실행 계획
- 기준: main@c336a30, VERSION v0.1.35. 작업 트리 변경 없음. CLAUDE.md·AGENTS.md·명시적 ROADMAP/TODO 파일 및 backend/frontend/src/scripts/e2e TODO/FIXME 검색 결과 없음.
- 정찰 실행: `make -n test`에서 race 누락 확인. `cd backend && go test -race ./...` exit 0(대부분 캐시), `make check` exit 0(OpenAPI 120 method/path), `make fmt` exit 0. MOINA_TEST_POSTGRES_DSN unset이라 DB 통합 테스트는 미실행/skip이며 DB까지 검증했다는 의미가 아니다. 전체 make test·frontend·e2e는 정찰에서 실행하지 않았다.
- 예상: 변경·diff 검토 5분, 실제 make test 및 계약 검증 15~25분, 결과 기록 5분, 환경/설치 지연 예비 10분으로 최대 45분. 이는 정찰 추정이며 frontend 전체 소요 시간은 미확인이다.
- 탐색 비교: 동영상 dimension 파싱은 포맷별 검증이 커지고, 캐시 변경은 미디어 접근권한·재검증 계약에 닿으며, 화면 캡처는 앱·DB·브라우저 준비가 필요하다. 미디어 거절 통합 테스트도 가치가 있지만 이번 회차에는 기존 CI 명령과 직접 대조할 수 있는 작은 누락을 선택했다. 최근 채택된 multipart 413·파일명 정규화·updatePost 오류 분리는 이미 반영되어 반복하지 않는다.
- 스킬 가용성: 요청한 `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration`은 callable 도구/리소스 카탈로그와 /home/hkjang/.codex·/home/hkjang/.claude 파일 검색에서 찾지 못했다. Skill 도구도 제공되지 않아 로드 및 고유 반환 형식 준수는 미확인이다. 위 추정·계획·후보 비교는 사용자 프롬프트에 따른 정찰이며 해당 스킬을 실행했다는 뜻이 아니다.
