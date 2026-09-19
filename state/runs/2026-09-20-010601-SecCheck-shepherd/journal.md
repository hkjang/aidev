# PR 처리기 노트 2026-09-20-010601-SecCheck-shepherd — SecCheck PR #9
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-19-184351-SecCheck-improve)
# 회차 노트 2026-09-19-184351-SecCheck-improve — SecCheck
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:43] base pinned — main@eb2a3b0
- [러너 18:43] autonomy release — 

## 정찰 노트
- 자동 배정(수정 과제)을 그대로 잡았다. PR #8 = `auto/2026-09-19-0143`(0bcbbb8+693b733, 9파일)임을 로컬 refs 로 확인했으나 `gh`·`curl`·WebFetch 가 모두 거부되어 **실패한 워크플로·단계·로그는 미확인** — 과제서 0단계가 로그 확인이며, 그 전에는 고치지 말라고 적었다.
- 추측인 곳: 원인 순위표(govulncheck > trivy/grype > npm audit > go test -race > ZAP)는 "precheck 는 통과했는데 CI 만 실패" 와 "같은 이유로 두 번" 에서 유도한 것이지 증거가 아니다. 과제 문구의 "릴리즈 워크플로 두 번 실패" 도 이전 회차(09-19 새벽)가 release.yml 6건 전부 success 를 확인한 바 있어 전제 자체를 의심하고 확인하게 했다.
- 구현자가 조심할 것: 워크플로·severity·-race·audit-level 완화 금지, 테스트 skip 금지, 메이저 업그레이드 금지, PR #8 브랜치 위에서 작업(git stash 금지), 파괴적 확인 전 WIP 커밋. 프로필은 0일 전 것이고 코드와 어긋난 곳이 없어 다시 쓰지 않았다.
- [러너 18:47] scout done — PR #8(`auto/2026-09-19-0143`, 693b733)의 security-ci 실패 원인을 실패 로그에서 확정하고 원인을 고쳐 같은 게이트를

## 구현 노트
- 원인(로그로 확정): PR #8 의 유일한 run `security-ci` 35372823139 · job `test-build-scan` · step 7 `Go vulnerability scan` failure. 로컬 재현(`GOTOOLCHAIN=go1.26.6 govulncheck ./...` exit 3): `GO-2026-6452 … excelize/v2@v2.11.0 … Fixed in: N/A … internal/store/seed.go:66:25 store.ExtractWorkbookDefaults calls excelize.File.GetRows`. DB 등재 09-16T18:00Z, main 마지막 CI 09-16T17:01Z → main 도 지금 돌리면 같은 자리에서 죽는 환경 게이트. release.yml 실패 run 은 없음("두 번 실패" 는 오판).
- 바꾼 것(6424eb1): go.mod excelize v2.11.0 → master `v2.11.1-0.20260918021423-0434413565bf`(+ x/crypto·net·sync·text 마이너) — v2.11.0 은 temp-file 경로에서 실제로 `index out of range [-1]` panic(재현), upstream 은 태그 뒤 f98df08 로 고쳤고 이후 태그 없음. 회귀 테스트 `internal/web/import_upload_test.go` 는 진짜 서버·DB·업로드로 `/api/v1/templates/import(/preview)` 가 500 이 아니어야 함 — v2.11.0 에서 실패, 핀 뒤 통과.
- **확신 없는 곳·검증 못 한 것**: (1) govulncheck 게이트는 핀 뒤에도 exit 3 — vulndb 보고서에 `fixed:` 가 없어 어떤 버전도 못 넘음(golang/vulndb#6510·#6532 open, 답 없음). 저장소 안에서 초록으로 만들 방법은 워크플로 완화뿐이라 하지 않았다 → **CI 는 DB 정정 전까지 빨강**. (2) master 핀은 태그가 아니라 upstream 의 미공개 변경(수식 계산·그림 복사 등)을 함께 들여온다 — 우리 사용처(NewFile/SetCellValue/CoordinatesToCellName/OpenReader/GetRows)는 `go test -race ./...` 전 패키지(xlsx export·import 테스트 포함)로 확인했지만 그 밖의 동작 차이는 보지 못했다. (3) 실패 로그 본문은 admin 권한(403)이라 못 읽었고 step 이름·결론만 API 로 확인.
- 검증한 것: `go test -race ./...`(DSN, internal/web 204s) + `go vet` exit 0 / npm ci·test 11·build·`npm audit --audit-level=high` exit 0(moderate 2건은 vitest 메이저 필요) / `docker build` + `trivy --severity CRITICAL,HIGH --ignore-unfixed --exit-code 1` 0건 / `bash scripts/precheck.sh` exit 0(gitleaks no leaks) / `git diff eb2a3b0 -- .github/workflows/` 0바이트.
- 일부러 하지 않은 것: 워크플로·게이트 변경, 테스트 skip, GetRows 를 다른 API 로 바꿔 스캐너를 피하기(가짜 수정), vitest 메이저 업그레이드, vulndb 이슈 제출(외부 행위·gh 미인증). PR #8 브랜치로 전환하지 않음(브랜치 전환 금지) — 이 브랜치는 main@eb2a3b0 기반이라 PR #8 은 이것을 병합/체리픽(6424eb1 한 커밋)해야 같은 게이트를 받는다.
- 다음 역할이 조심할 것: 새 테스트는 `TEST_POSTGRES_DSN` 이 있어야 돌고 17MiB 공유문자열을 만들어 3~8초 걸린다. CI 가 빨강인 것은 이 변경의 결함이 아니라 DB 상태 — `curl -s https://vuln.go.dev/ID/GO-2026-6452.json | grep fixed` 로 정정 여부를 확인하고, 정정되면 재실행만으로 초록. excelize 다음 태그가 나오면 pseudo-version 을 태그로 되돌릴 것(ideas.json).
- [러너 19:03] brief accepted — 채택 — 0단계 로그 확인으로 후보 1(govulncheck)이 맞았고 방향(게이트가 아니라 취약점을 고침)대로 했으나, 과제서가 전�
- [러너 19:03] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: 새 테스트를 임시 Postgres 16 컨테이너로 직접 실행 — 핀(0434413)에서 통과, go.mod 를 v2.11.0 으로 되돌리면 templates.go:810 GetRows 에서 index[-1] panic → 500 으로 실패. 수정 전 코드에서 떨어지는 진짜 회귀 테스트다. 핀 소스 rows.go:361 의 index<0 가드, go mod verify, tidy -diff 비어 있음(x/* 상승은 excelize 가 요구), 워크플로 무변경, vet + store 전체 + web Excel 관련 21개 테스트 통과, BSD-3 라이선스 유지까지 봤다.
- 못 본 것: 전체 `go test -race ./...`(구현자 기록만 신뢰), 핀이 함께 들여온 upstream 미공개 변경의 우리 사용처 밖 동작, 실패 CI 로그 본문(403).
- 승인. 남는 우려: vuln.go.dev GO-2026-6452 는 지금도 fixed 이벤트가 없어 병합해도 govulncheck 게이트는 빨강 — 릴리즈 노트에 "패닉은 고침, 스캐너는 DB 정정 대기" 로 적고 `curl -s https://vuln.go.dev/ID/GO-2026-6452.json | grep fixed` 로 재확인 뒤 재실행. templates.go:810 이 GetRows 오류를 삼켜 적대적 워크북이 preview 에서 빈 200 으로 돌아오는 것은 다음 회차 후보.
- PR #8 은 6424eb1 을 체리픽/병합해야 같은 게이트를 받는다. excelize 다음 태그가 나오면 pseudo-version 을 태그로 되돌릴 것.
- [러너 19:06] review approved — 리뷰 승인 (risk=low)
- [러너 19:06] pr created — https://github.com/hkjang/SecCheck/pull/9
- [러너 19:13] ci failed — 성공이 아닌 검사: test-build-scan=failure

## 수리 노트
- 맞았던 지적: CI 는 실제로 빨강 — step 7 `Go vulnerability scan` 만 failure, 테스트·vet 은 success(API 로 step 결론 확인, 로그 본문은 403). 로컬 `govulncheck ./...` 도 같은 자리에서 exit 3.
- 틀렸던 지적: 변경의 결함이 아니다. vuln.go.dev GO-2026-6452 에 `fixed` 이벤트가 없어(GHSA 는 2.11.0 patched, 픽스 커밋 93f0b3c 는 v2.11.0 안에 있음) 어떤 핀도 걸린다. 같은 엔트리에 `fixed: 2.11.0` 한 줄만 더한 로컬 DB(`govulncheck -db file://`)로는 이 브랜치가 exit 0 — 코드는 DB 정정 즉시 초록.
- 고친 방법: 없음, 커밋 없음. 남은 길이 워크플로 완화나 excelize 읽기 API(전부 플래그됨) 제거뿐이라 규칙대로 손대지 않았다. 근거는 fix-summary.md.
- 확신 없는 곳: vulndb#6510·#6532 가 언제 닫힐지(09-19 갱신, 아직 open). 정정 전에 병합할지는 사람 판단 — 패닉 회귀 자체는 고쳐졌고 새 테스트가 v2.11.0 에서 떨어지는 것은 비평가가 확인했다.
