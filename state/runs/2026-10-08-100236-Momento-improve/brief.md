- 과제: [수정 과제] 반복 verify-failed의 보안 검사 누락을 막는 로컬 `make verify` 진입점을 만든다 (가치 5 / 위험 1 / 작업량 S)
- 왜: 두 회차의 CI 실패는 각각 web source-map-js 감사와 Go x/text govulncheck였으며, 두 의존성 수정은 이미 main@0beff4c에 병합되어 현재 릴리즈도 성공했다. 그러나 Makefile:test와 README 개발 절차는 두 보안 검사를 모두 생략하므로, CI와 같은 보안 검사를 먼저 실행하는 표준 로컬 검증 명령으로 재발 가능성을 줄인다.
- 수용 기준: 1) 루트의 `make security-check`가 CI와 같은 `go run golang.org/x/vuln/cmd/govulncheck@latest ./cmd/... ./internal/...`, sdk의 `npm audit`, web의 `npm audit`를 순서대로 실제 실행하고 어느 하나 실패하면 비영 종료한다. 2) `make verify`는 security-check가 성공한 뒤 기존 `make test`를 실행한다. `make -j verify`에서도 보안 검사 완료 전에 test를 시작하지 않는 의존 관계를 사용한다. 기존 `make test`의 독립 실행 방식과 CI·릴리즈 워크플로는 유지한다. 3) 현재 의존성에서 보안 검사와 verify가 통과하고, 아래 실제 과거 파일을 사용하는 실행에서 각각 해당 취약점 때문에 security-check가 실패한다. 소스 문자열 단언이나 명령 대역만으로 실패 전파를 증명하지 않는다. 4) README의 명령을 저장소 루트에서 순서대로 실행할 수 있고, 의존성 설치·네트워크·DB 통합 테스트 조건과 이 명령이 Docker/offline smoke까지 대신하지 않는다는 범위가 분명하다.
- 건드릴 파일: `Makefile:1,20-24` — `.PHONY`에 `security-check verify` 추가, security-check의 세 실제 검사와 `verify: security-check` 아래 `$(MAKE) test` 추가(단순히 `verify: security-check test`로 두면 병렬 make에서 순서를 보장하지 못함). `README.md:98-112 개발` — 설치를 `(cd sdk && npm ci)`·`(cd web && npm ci)`처럼 루트로 돌아오는 서브셸로 안내하고 `make verify`를 표준 명령으로 설명한다. 프로덕션/빌드 파일 1개 + 문서 1개, 신규 패키지·영구 테스트 파일 불필요.
- 검증 명령: 아래 실행 계획을 따른다. 기존 상태에서 실제 실행한 검사는 govulncheck, 각 패키지 npm audit, `go test ./internal/version`이며 모두 통과했다. `make verify`·`make security-check`는 새로 만들 대상이므로 아직 실행 성공한 명령이 아니다.
- 위험과 피할 것: `.github/workflows/*`, Dockerfile, auth, migrations, go.mod/go.sum, package.json/lock을 변경하지 않는다. audit의 수준을 낮추거나 devDependencies 제외, `|| true`, Make recipe의 실패 무시 접두사, vulnerability DB 고정·오래된 도구 고정으로 통과시키지 않는다. `npm audit fix`/의존성 재생성/메이저 업그레이드를 하지 않는다. 원장에 '같은 원인이 두 번 발생했다'고 쓰지 않는다. 보안 DB는 변하므로 구현 시 새 경고가 생기면 실제 출력과 범위 변경을 기록하고 이 과제에 업데이트를 묶지 않는다.
- 차선 후보: 새 타깃을 넣기 어렵다면 README 개발 절차에 동일 세 보안 검사를 순서대로 명시하는 문서 한 조각과 이미 해결된 실패의 원장 정정으로 한정한다. UI 개선·이미 적용된 의존성 변경으로 전환하지 않는다.

확인한 사실과 근거

- 자동 적재의 'PR open'은 오래된 상태다. https://github.com/hkjang/Momento/pull/29 는 merged=true, 2026-10-07T23:19:37Z 병합, head fcf17ce4fef4cea8e9ad67127ebd886ed074dfe8이다.
- PR #29 실패: https://github.com/hkjang/Momento/actions/runs/37687217169 (job 113018016038), govulncheck 단계. x/text v0.39.0의 GO-2026-6629, 수정 버전 v0.41.0. 로그의 호출 경로는 실제 읽은 `internal/database/database.go:Open` 49행 → pgxpool.NewWithConfig → precis.Profile.String. fcf17ce가 x/text v0.41.0 및 필요한 x/sync v0.22.0으로 이미 고쳤다.
- 앞선 실패: https://github.com/hkjang/Momento/actions/runs/37545738339 (job 112549181628), web npm audit 단계. source-map-js 1.0.0~1.2.1의 GHSA-68fv-2mgg-jv7q이며 97e3063이 잠금을 1.2.2로 고쳤다. Go 실패와 동일 취약점이 아니다.
- 현재 main CI: https://github.com/hkjang/Momento/actions/runs/37702077833 성공. 현재 v0.34.61 Offline image release: https://github.com/hkjang/Momento/actions/runs/37702752117 성공. 실제 release.yml에는 빌드·저장·로드·체크섬·DB readiness·version/commit·콘솔·bootstrap 로그인 검증이 있다. 이 워크플로를 손댈 근거가 없다.
- 별도로 최근 실패한 Release reconciliation 37363965959·37370594454는 job conclusion=cancelled, steps=null이고 로그 API가 404다. 같은 스크립트 원인이라는 주장은 미확인이다. 해당 두 실패를 추측으로 '수정'하지 않는다.
- 현재 `make -n test`에는 보안 검사가 없고 `make -n verify`는 No rule to make target 'verify'로 종료 2였다. `README.md` 개발 절차도 보안 검사와 sdk test가 없고 `cd ../web` 뒤 make docker가 web에서 실행되는 문제까지 있다. 문서 갱신은 이 절 안에서 끝낸다.

실행 계획과 검증 지점 (사람 승인 대기 없음)

1. [정찰 완료] 위 로그·현재 SHA·이미 적용된 수정 확인. 실제 과거 상태를 작업 트리가 아닌 회차 assets 아래 복사하여 동일 명령으로 red를 확인했다.
   - `assets/before-go`: `git archive 5bf68a6 go.mod go.sum cmd internal` 추출본. 그 디렉터리에서 CI의 govulncheck 명령 → GO-2026-6629, go run 종료 1(도구 exit status 3).
   - `assets/before-web/web`: `git archive 90019ca web/package.json web/package-lock.json` 추출본. 여기서 `npm audit` → source-map-js high 1건, 종료 1.
   - 현재 루트 govulncheck → affected 0, 종료 0. 현재 sdk/web에서 각각 npm audit → found 0 vulnerabilities, 종료 0. npm에는 로그/cache 위치만 회차 assets로 지정했고 감사 조건은 바꾸지 않았다. npm ci 없이 committed lock을 감사했으며 전체 npm 빌드 게이트 통과로 주장하지 않는다.
2. [구현 대기] Makefile 두 타깃과 README만 수정. `make security-check`로 실제 세 검사 실행·종료 0 확인. `make -n verify`로 배선 보조 확인(실행 증명의 대체 아님).
3. [구현 대기] 실제 수정한 Makefile을 사용하는 임시 검증 트리를 준비하여 실패 전파를 검증한다. 첫 트리는 before-go에 수정한 Makefile만 복사하고 `make security-check` → GO-2026-6629 실패를 확인한다. 둘째는 현재 cmd/internal/go.mod/go.sum, sdk manifests, 과거 before-web의 web manifests 및 수정한 Makefile을 복사하고 `make security-check` → Go와 SDK 통과 후 source-map-js 실패를 확인한다. 손으로 취약 버전 문자열을 고치거나 명령을 대역으로 바꾸지 않는다. 검증 트리는 커밋하지 않는다.
4. [구현 대기] 루트에서 `(cd sdk && npm ci)` 및 `(cd web && npm ci)` 후 `make verify`, `go test ./internal/version` 실행. MOMENTO_TEST_POSTGRES_DSN 없을 때 DB 통합 테스트가 skip되는 한계를 별도 기록한다. 필요하면 제공된 PostgreSQL 테스트 환경을 이용하되 기존 DB를 지우지 않는다. `.github/workflows/ci.yml`의 race·Docker 및 release.yml smoke는 추가 릴리즈 게이트로 유지된다.
5. [구현 대기] `git diff --check`, `git diff --stat`로 두 파일 범위 확인. journal의 구현 항목에 '수정 과제', 과거 실패/현재 통과 결과, 실제 make 경로의 red/green, 미실행 검증을 구분해 남긴다.

대안·추정 (요청한 세 스킬 적용)

- 권고: 기존 Makefile에 최소 진입점 추가. 실행 가능한 공통 명령이 생기며 런타임 의존성이나 워크플로를 늘리지 않는다.
- 문서만 보완: 가장 작은 차선이나 명령 누락을 실행 경로로 막지 못한다.
- 전체 CI runner·Docker 오케스트레이터 신규 작성: DB와 이미지 수명 관리까지 번져 45분 범위를 벗어나므로 이번엔 제외.
- 아무 코드도 고치지 않고 stale 실패만 종료: 취약점 자체에는 올바르지만 현재 확인된 로컬 보안 검사 누락은 남는다.
- Bottom-up 추정: Makefile 4~6분, README 3~5분, 실제 과거 파일 red/현재 green 6~10분, 의존성 설치 및 기존 test 8~12분, 정리 2~3분 = 기본 23~36분. 알려진 네트워크·캐시 변동 여유 4~8분을 별도로 두어 총 27~44분. 통계적 P80이 아닌 중간 확신의 작업 추정이며, 별도 관리 예비분/새 범위는 포함하지 않았다. 비교 가능한 과거 작업의 실제 소요시간이 없어 유사사례 숫자는 만들지 않았다.
- 가장 큰 가정: 구현 시점에도 세 보안 검사가 현재 잠금에서 통과한다. 새 advisory가 나오면 이 추정을 다시 한다. 외부 통계나 비용 모델은 사용하지 않았다.
- 읽은 스킬: `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md` 및 `references/sources.md`, 같은 marketplace의 `technology/skills/implementation-planning/SKILL.md`, `technology/skills/solution-exploration/SKILL.md`. Skill 전용 도구는 제공되지 않아 로컬 원문을 읽었다.
