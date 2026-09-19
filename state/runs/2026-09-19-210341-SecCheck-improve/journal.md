# 회차 노트 2026-09-19-210341-SecCheck-improve — SecCheck
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:03] base pinned — main@eb2a3b0
- [러너 21:03] autonomy release — 

## 정찰 노트
- 선택 이유: 자동 배정이 "릴리즈 워크플로 두 번 실패" 라 했지만 지난 회차가 확정한 실패 지점은 ci.yml `Go vulnerability scan`(GO-2026-6452, vulndb 에 fixed 없음)이고 PR #9(`auto/2026-09-19-1843`, 6424eb1 = excelize 핀 + panic 회귀 테스트)는 이미 올바른 코드 수정을 담고 있다. 저장소 안에서 게이트를 초록으로 만들 정당한 수단이 없으므로 새 코드 후보(admin-guide.md 제거 등)보다 "오늘의 vulndb 로 재판정 → 확정 → 같은 게이트를 precheck 에 넣어 다음 회차가 푸시 전에 보게" 를 골랐다.
- 확신 없는 곳(미확인): 이 정찰 환경은 curl·gh·go 실행이 승인 거부돼 vulndb 의 현재 상태(fixed 가 생겼는지)·PR #9 의 실패 step 이름·excelize 새 태그 유무·operations.md 가 docs_test/PDF 목록에 포함되는지를 직접 보지 못했다. 과제서 0단계가 그것을 먼저 확인하도록 했다.
- 구현자가 조심할 것: (b) 상태에서 precheck 의 새 단계는 정당하게 실패한다 — 통과시키려고 단계를 빼거나 조건부로 만들지 말 것. 워크플로 완화·`replace` 로 모듈 경로 바꾸기 금지. 6424eb1 은 손으로 다시 쓰지 말고 cherry-pick. 파괴적 확인 전 `git add -A && git commit`.
- 프로필은 오늘 것이라 새로 쓰지 않았다. 다음 갱신 때 "검증 함정" 에 "govulncheck 는 vulndb 보고서에 fixed 가 없으면 어떤 버전으로도 초록이 안 된다(GO-2026-6452)" 를 더할 것.
- [러너 21:07] scout done — PR #9(`auto/2026-09-19-1843`, 6424eb1 "Pin excelize past the negative shared-string index panic")의 security-ci 실패를 **오늘의 vulnd

## 구현 노트 — 0단계 판정 (2026-09-19 21:0x, 구현자)

`curl -s https://vuln.go.dev/ID/GO-2026-6452.json | jq '.modified, [.affected[] | {module: .package.name, ranges: .ranges}]'`
```
"2026-09-16T18:00:43Z"
[
  {
    "module": "github.com/xuri/excelize/v2",
    "ranges": [
      {
        "type": "SEMVER",
        "events": [
          {
            "introduced": "0"
          }
        ]
      }
    ]
  }
]
```
→ `fixed` 없음. `modified` 도 09-16 그대로(그 뒤 손대지 않음). **판정: (b)**.

`gh run list …` 는 이 세션에 gh 인증이 없어(exit 4) 공개 API 로 대신 확인:
`curl -s "https://api.github.com/repos/hkjang/SecCheck/actions/workflows/ci.yml/runs?branch=auto/2026-09-19-1843&per_page=3"`
```
{ "id": 35436507400, "head_sha": "6424eb11ec52370247a265e79c717c6f680b961e", "status": "completed", "conclusion": "failure", "created_at": "2026-09-19T10:06:46Z" }
```
`…/actions/runs/35436507400/jobs` → job `test-build-scan` failure, failed_steps: `["Go vulnerability scan"]` — 여전히 같은 단계.

로컬 재현(go1.26.7, govulncheck@latest, cherry-pick 6424eb1 뒤 트리): `govulncheck ./...` exit 3 —
`GO-2026-6452 … Found in: github.com/xuri/excelize/v2@v2.11.1-0.20260918021423-0434413565bf / Fixed in: N/A / internal/store/seed.go:66:25: store.ExtractWorkbookDefaults calls excelize.File.GetRows`.
`go list -m -versions github.com/xuri/excelize/v2` 마지막 태그 v2.11.0 — 새 태그 없음, pseudo-version 핀 유지.

**확정(수정 과제): 코드 변경으로 해결 불가 — vulndb 결함, 워크플로 완화 금지. 운영자 결정 항목: golang/vulndb 에 `fixed: 2.11.0`(in-memory 경로) 또는 `v2.11.1-0.20260918021423-0434413565bf`(temp-file 경로 f98df08 포함) 를 더하는 PR 을 낼지.** 그래도 6424eb1 은 실제 panic 을 막는 변경이라 cherry-pick 했다(a08e347).

## 구현 노트
- 무엇/왜: (1) `git cherry-pick 6424eb1` → a08e347 (go.mod·go.sum·import_upload_test.go, 손으로 안 씀). (2) 9b4b7aa: `scripts/precheck.sh` 에 `step "Go 취약점 스캔"` 을 비밀정보 스캔 바로 앞에 추가(`command -v govulncheck` 있으면 `govulncheck ./... || fail …`, 없으면 건너뜀 문구) + `docs/operations.md` `## 릴리즈 게이트` 끝에 `### govulncheck 가 \`Fixed in: N/A\` 로 막힐 때` 소절(확인 명령·의미·하지 말 것·할 것·GO-2026-6452 사례). CI 가 막히는 판정을 푸시 전에 같은 명령으로 보게 하는 것이 목적.
- 검증: 임시 Postgres 16(:55433 — :55432 는 다른 세션의 `hunter-ci-pg` 가 점유) DSN + govulncheck PATH 로 `bash scripts/precheck.sh` → gofmt·go vet·go test 전 패키지(internal/web 124s, 새 회귀 테스트 PASS 0.84s)·vitest 11개·vite build·가이드 그림·PDF·gitleaks "no leaks found" 모두 통과하고 **오직 `Go 취약점 스캔` 만 GO-2026-6452 / Fixed in: N/A 로 `fail`** → exit 1 (정답). govulncheck 없는 셸에서는 `건너뜀: govulncheck 가 없어 …` 문구. `go test ./internal/web -run 'Docs|Guide' -count=1` ok. `git diff origin/main(eb2a3b0) -- .github/workflows/` 0바이트 — 주의: 이 worktree 의 로컬 `main` 은 c32eec7 로 stale 이라 `git diff main` 은 기존 커밋(npm test 줄·1.0.146 버전)이 보인다; 러너가 pin 한 base 는 eb2a3b0.
- 확신 없는 곳: operations.md 는 PDF 목록(`build_docs_pdf.sh --list`)에 없어 PDF 를 굽지 않았음 — 맞다고 봄. docs_test 는 operations.md 의 알림·마이그레이션·업그레이드 표만 보고 새 소절은 검사하지 않음(통과가 소절의 정확성을 증명하진 않음). PR #9 재실행(a 경로)은 (b) 라 하지 않았고 gh 인증도 없음.
- 일부러 하지 않은 것: 워크플로 변경 없음, `replace` 없음, 새 단계를 조건부로 만들지 않음, PDF 재생성 없음(대상 아님), excelize 태그 전환 없음(태그 없음).
- 다음 역할 주의: precheck 는 이 브랜치에서 **의도적으로 exit 1** 이다(govulncheck 단계). 그것을 "실패" 로 읽지 말 것 — 다른 모든 단계는 초록. CI 의 security-ci 도 vulndb 가 fixed 를 기록할 때까지 같은 단계에서 계속 실패한다. 임시 컨테이너는 지웠다.
- [러너 21:13] brief accepted — 채택 — 0단계 판정이 (b) 로 나와 과제서의 3)·4)·5)·6) 을 그대로 수행했고, 과제서의 전제(실패 지점 = ci.yml govulncheck, vuln
- [러너 21:13] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: 임시 Postgres 16(:55434)으로 `TestAHostileWorkbookIsRejectedRatherThanCrashingTheImport` 를 HEAD 에서 PASS(preview 200·import 422), go.mod/go.sum 을 eb2a3b0(excelize v2.11.0)으로 되돌려 같은 테스트 FAIL(`/templates/import/preview` 500 INTERNAL_ERROR) — 테스트가 수정을 실제로 핀한다. 모듈 캐시의 `rows.go:361` 에 `index < 0 ||` 하한 검사 있음. `go mod verify` 통과, `go mod tidy` 무변경, x/crypto·x/net·x/text·x/image·mscfb 상향은 모두 excelize 핀의 go.mod 가 요구하는 MVS 결과(범위 이탈 아님). docs_test(Docs|Guide) ok. precheck 새 단계는 govulncheck 부재 시 건너뜀 문구만 출력(failed=0)·명령은 ci.yml 49-52 와 동일. `.github/workflows` 무변경. 인증·마이그레이션·개인정보 무접촉.
- 못 봄: 이 세션에 govulncheck 이 없어 precheck 의 fail 분기를 직접 돌리진 못했고(구현 노트의 재현 결과를 신뢰), operations.md 의 `golang/vulndb#6510` 이슈 번호는 네트워크 없이 검증 불가.
- 승인이어도 남는 우려: 머지하면 main 의 security-ci 가 vulndb 에 `fixed` 가 기록될 때까지 `Go vulnerability scan` 에서 계속 빨갛고 release.yml 2번 게이트가 릴리즈를 막는다 — 이는 문서가 설명하는 의도된 상태이며 운영자 결정 항목(vulndb PR). 릴리즈 노트에는 "excelize 음수 shared-string 인덱스 panic 차단(CVE-2026-59162)" 과 "게이트 대기 중" 을 함께 적을 것.
- 다음 회차: vulndb 보고서에 `fixed` 가 생기면 코드 변경 없이 CI 재실행만으로 초록이 되어야 한다; 그때 `curl -s https://vuln.go.dev/ID/GO-2026-6452.json` 으로 먼저 확인.
- [러너 21:16] review approved — 리뷰 승인 (risk=low)
- [러너 21:16] pr created — https://github.com/hkjang/SecCheck/pull/10
- [러너 21:21] ci failed — 성공이 아닌 검사: test-build-scan=failure
