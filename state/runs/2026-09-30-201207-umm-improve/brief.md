# 과제서 — 2026-09-30-201207-umm-improve

## 먼저: 자동 배정된 "우선 과제" 는 이미 해결되어 있습니다 (증거 있음)

러너가 적은 우선 과제는 "마지막 회차가 verify-failed, CI failed, PR open #162 / 릴리즈 워크플로가 같은
이유로 두 번 실패" 입니다. 이 정찰 회차에 `gh` 실행 권한이 없어 API 를 직접 못 불렀으나, **이전 회차들이
남긴 원장 파일** 로 다음을 확인했습니다:

1. 실패의 실체는 릴리즈 워크플로가 아니라 CI `verify` 잡의 **"Frontend dependency audit"**
   (`ci.yml:118-120`, `npm audit --audit-level=high`, `working-directory: web`) 이었습니다 —
   PR #162 의 Go 한 줄 변경과 무관한 **전이 개발 의존성** undici 8.10.0 의 high 권고 11건.
   출처: `…/2026-09-30-191150-umm-shepherd/fix-summary.md:3-4`.
2. 이미 고쳐졌습니다. 커밋 `0525de0` "Bump undici to 8.11.2 so the frontend dependency audit reports
   no high severity advisory". 현재 트리에서 확인: `web/package-lock.json:3426` 이 undici-8.11.2.tgz,
   `web/package.json` 에는 undici 선언이 없음(jsdom 경유 전이 개발 의존성뿐, `package-lock.json:2317`
   의 `"undici": "^8.9.0"` 범위 안). 수리 회차 게이트도 통과:
   `…/2026-09-30-191150-umm-shepherd/verify.gate.json` → `{"ok": true, "state": "verified"}`.
3. PR #162 은 열려 있지 않습니다 — `50af3b0 Merge pull request #162` 로 머지됐고, 그 위에
   `db53f8b Release v0.76.2` 가 올라가 v0.76.2 가 릴리즈됐습니다
   (`…/2026-09-30-192943-umm-approve/release.json` → `{"status":"released","tag":"v0.76.2"}`).
4. **릴리즈 커밋의 CI 는 네 검사 전부 초록입니다.**
   `…/2026-09-30-192943-umm-approve/ci-db53f8b9475d.json`:
   `verify` / `build` / `deploy` / `report-build-status` 모두 `"conclusion": "success"`
   (verify 는 10:35:58~10:45:52 완주).
5. 릴리즈 워크플로 자체는 v0.7.0 이후 한 줄도 바뀌지 않았습니다 —
   `git log --oneline -- .github/workflows/release.yml` 이 `0135431`, `18df4d8` 둘뿐이고
   그 사이 v0.73.0~v0.76.2 가 통과했습니다.
6. `scripts/check-version.sh` 의 다섯 검사가 통과할 입력입니다: `VERSION`=0.76.2,
   `web/package.json`=0.76.2, `web/package-lock.json` 첫 version=0.76.2, `compose.yaml:20`
   `VERSION: 0.76.2`, `compose.yaml:21` `image: umm:v0.76.2`, `docs/releases/v0.76.2.md` 존재.
   (스크립트를 읽고 각 입력을 대조해 확인. **미확인**: 스크립트 실행 자체는 이 세션에 bash 승인이
   막혀 못 돌렸습니다 — `npm audit` 재실행도 같은 이유로 미실행.)

**구현자에게**: 고칠 실패가 남아 있지 않으므로 워크플로·lockfile 을 다시 건드리지 마세요. 특히
`ci.yml:119` 의 `npm audit --audit-level=high` 를 `--production` 이나 낮은 임계로 바꾸는 것은 금지된
"워크플로 느슨하게 하기" 이며, 운영자가 되풀이해 말한 "릴리즈·빌드 경로를 건드리는 변경" 에 해당합니다.
그러므로 이번 회차는 아래 일반 과제를 하세요. 만약 착수 시점에 **새로** 빨간 CI 가 있으면 그것이
우선이고, 그때는 그 단계의 로그를 직접 확인한 뒤 원인을 고치세요(새 권고가 또 떴다면 lockfile 만 올리는
`npm audit fix --package-lock-only` 가 선례입니다 — 0525de0).

---

## 이번 과제

- 과제: 업로드한 그림의 이름 라벨이 경로를 통째로 이고 들어와 뭉개진다 — 마지막 조각만 쓰기 (가치 2 / 위험 2 / 작업량 S)
- 왜: `internal/store/attachments.go:150` 의 `safeFilename` 은 `strings.Map` 으로 `/` 와 `\` 를 **지우기만**
  하므로, 어떤 브라우저·클라이언트가 전체 경로를 보내면 `C:\사진\회의.png` 가 `C:사진회의.png` 로,
  `../etc/passwd.png` 가 `..etcpasswd.png` 로 붙습니다(경로 탈출은 막지만 사람이 읽을 이름이 아닙니다).
  이 문자열은 `Content-Disposition` 헤더까지 그대로 가므로, 마지막 조각만 취하면 내려받은 파일 이름이
  올린 파일 이름과 같아집니다.
- 수용 기준:
  1) `safeFilename("C:\\사진\\회의.png")` 가 `회의.png` 를, `safeFilename("a/b/c.png")` 가 `c.png` 를 돌려준다.
  2) 마지막 조각이 비면 라벨을 잃지 않는다 — `safeFilename("사진/")`, `safeFilename("a/b/")` 가 빈 문자열이
     아니라 뜻이 남는 값(예: 직전 조각 `b`)을 돌려준다. 이 선택을 주석에 한 줄로 적을 것.
  3) 기존 계약 중 **지키는 것**: `TrimSpace`(`attachments_test.go:36` 의 `"  화이트보드.png  "` → `화이트보드.png`),
     제어문자·`"` 제거(`\x00`, `"`), `textutil.LimitUTF8Bytes(…, 120)` 의 문자 경계 절단
     (`attachments_test.go:20-32` 의 경계 시험이 그대로 통과해야 한다).
  4) `TestSafeFilenameDropsSeparatorsAndControls`(`attachments_test.go:41-45`)의 단언은 **의도적으로**
     바뀐다: `"../etc/pass\"wd\x00.png"` → `"..etcpasswd.png"` 가 아니라 마지막 조각 기준 값
     (`pass"wd\x00.png` 를 정리한 `passwd.png`)이어야 한다. 시험 이름과 본문을 새 계약을 말하도록 고치고,
     **경로 구분자가 남지 않는다**(`strings.ContainsAny(got, "/\\")==false`)는 단언을 유지할 것.
- 건드릴 파일 (프로덕션 1개):
  - `internal/store/attachments.go:150 safeFilename` — `strings.Map` 전에 `/` 와 `\` 양쪽을 기준으로
    마지막 비어 있지 않은 조각을 취한다. `filepath.Base` 는 쓰지 말 것(리눅스에서 `\` 를 구분자로 보지
    않아 윈도 경로를 놓친다). `strings.LastIndexAny(s, "/\\")` 또는 두 구분자로 `strings.FieldsFunc` 후
    마지막 비지 않은 원소가 맞다. `\` 와 `/` 는 조각을 고른 뒤에도 `strings.Map` 에서 계속 지워
    (조각 안에 남을 수 없게) 심층 방어를 유지한다.
  - `internal/store/attachments_test.go` — 41-45 단언 갱신 + 윈도 경로·POSIX 경로·꼬리 구분자
    (수용 기준 1·2) 시험 추가.
- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/store -run TestSafeFilename -count=1 -v` — DB 불필요. SKIP 이 아니라 RUN/PASS 인지
    `-v` 로 확인할 것.
  - 고친 뒤 프로덕션 한 덩이만 `git show HEAD:internal/store/attachments.go` 로 되돌려 새 시험이
    **실제로 실패하는지** 확인하고 그 출력을 원장에 붙일 것(지난 두 회차가 쓴 방식).
  - `go vet ./...` · `gofmt -l internal/store` (무출력)
  - 전체: `POSTGRES_DSN=<격리 PostgreSQL 17 DSN> go test -p 1 ./... -count=1`.
    첨부 업로드 통합 시험이 이름 단언을 들고 있을 수 있으므로 `go test ./internal/store -run 'Test.*Integration' -count=1`
    은 반드시 DSN 을 붙여 돌릴 것. **미확인**: 통합 시험 쪽 파일 이름 단언 존재 여부는 이번에 확인하지 못했습니다.
    DSN 이 없으면 통합 시험은 SKIP 되고, SKIP 을 통과로 읽으면 안 됩니다(도커 `umm-test-pg`/`umm-e2e-pg` 는
    지금 떠 있지 않을 수 있음 — 프로필의 검증 함정 참고).
- 위험과 피할 것:
  - **`safeFilename` · `httpapi.dispositionSafe` · `attachmentDisposition` · `handoffFilename` 을 하나로
    합치지 말 것.** 계약이 서로 다르고, 운영자가 되풀이해 말한 "계약이 다른 파서를 통합하려 하지 말 것"
    에 정면으로 걸립니다. 이번에 바꾸는 것은 `safeFilename` 하나뿐입니다.
  - 이 값은 라벨이지만 `Content-Disposition` 까지 갑니다. 조각을 고른 뒤에도 구분자·제어문자·`"` 제거와
    120바이트 문자 경계 절단을 **없애지 말 것** — 없애면 헤더 주입·PostgreSQL UTF-8 거부가 되돌아옵니다.
  - `.github/workflows/`, `migrations/`, `internal/auth/`, `web/package-lock.json` 은 건드리지 마세요.
    이번 과제와 무관하고 전부 보호 경로입니다.
  - 마이그레이션 없음. 이미 저장된 옛 라벨을 소급 수정하지 말 것(데이터 변경은 이 과제 범위 밖).
  - 버전(`VERSION`, `web/package.json`, `compose.yaml`)을 올리지 마세요 — 개선 회차의 관례입니다.
- 차선 후보: `make test-go` 를 CI 와 같은 `-p 1` 직렬 실행으로 맞추기 (가치 3 / 위험 1 / S) —
  `Makefile:8` 이 `go test ./...` 인데 `ci.yml:52-58` 은 `app_settings` 의 단일 `ai_gateway` 행 경합
  때문에 `-p 1` 이고 그 이유가 주석에 적혀 있습니다. 로컬과 CI 가 다른 것이 문제의 전부이므로
  "시험이 무엇을 증명하나" 를 세우기 어렵고(`-p 1` 없는 경합은 재현이 확률적), Makefile 은 빌드 경로에
  가까워 위험이 올라갑니다 — 1순위가 성립하지 않을 때만 고르고, 그때는 `-p 1` 없이 3회 돌려
  경합을 먼저 재현한 뒤 고칠 것.
