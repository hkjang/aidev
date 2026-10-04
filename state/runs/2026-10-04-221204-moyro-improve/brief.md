# 과제서 (2026-10-04 정찰) — 수정 과제

> **이 정찰 세션의 한계를 먼저 밝힌다.** 이 세션에서는 `gh` 가 권한 거부되어
> (`gh run list`, `gh pr view` 모두 "requires approval", 비대화형이라 승인 불가)
> **릴리즈/CI 실패 로그를 직접 읽지 못했다.** `npm audit`, `scripts/fetch-plugin-test-fixtures.sh`
> 같은 네트워크 명령도 전부 권한 거부됐다. 따라서 아래 "왜" 의 원인 지목은
> **워크플로 파일·릴리즈 스크립트·릴리즈 커밋 메시지에서 읽어낸 추론**이고,
> 구현자는 **1단계로 실제 로그를 먼저 확인**해야 한다. 추론인 부분은 전부 "미확인" 으로 표시했다.

- 과제: v0.2.45 태그에서 두 번 연속 같은 이유로 실패한 릴리즈 게이트의 원인을 찾아 고치기 (가치 5 / 위험 2 / 작업량 S~M)

## 왜

`e5422a1 release: carry release markers to v0.2.45` 가 main 에 올라간 뒤 `Release offline image`
워크플로가 **같은 이유로 두 번** 실패했고, v0.2.45 는 아직 게시되지 않았다. 릴리즈가 막혀 있는 동안
머지된 세 커밋(`e101e7b` bulk-delete 413/400, `52a3812` 테스트 플레이크, `d20a46a` 주석)은
사용자에게 닿지 않는다. 운영자 규칙상 **워크플로를 느슨하게 만들어 통과시키는 것은 금지**이므로,
게이트가 가리키는 실제 결함(또는 실제로 갱신해야 할 핀)을 고쳐야 한다.

## 1단계 — 반드시 먼저: 실패한 단계를 로그에서 확정하라

정찰이 하지 못한 일이다. 구현자는 이것부터 한다.

```
gh run list --workflow="Release offline image" --limit 5
gh run view <failed-run-id> --log-failed | tail -120
# 두 번째 실패도 같은 단계·같은 출력인지 확인 (같아야 "같은 이유로 두 번" 이 성립한다)
gh run list --workflow=CI --branch main --limit 5
gh run view <ci-run-id> --log-failed | tail -80
```

`release.yml` 의 단계는 순서대로: Resolve and validate version → Check out release tag →
setup-go → setup-node → **Fetch plugin fixtures**(:89) → **Test Go modules**(:99) →
**Scan reachable Go vulnerabilities**(:110) → **Verify web dependencies and types**(:114) →
**Build release image**(:124) → verify-pages(:150) → **Verify release user interface**(:153) →
Export(:176) → **Verify offline archive**(:184) → Create or update GitHub release(:192).

실패한 단계 이름을 확정한 뒤, 아래 해당 분기만 따라간다.

## 2단계 — 가장 가능성 높은 원인 (순서대로)

### A. 라이브 취약점 DB 게이트 — `npm audit --audit-level=high` 또는 `govulncheck` (최우선 가설, 미확인)

`release.yml:118-119` 와 `release.yml:112`, 그리고 `ci.yml:99-100`·`ci.yml:112` 는
**매 실행마다 외부 실시간 DB를 조회**한다. 즉 **코드 변경 0으로도 깨지고, 재실행해도 계속 깨진다** —
"같은 이유로 두 번 실패" 와 "CI 도 실패" 를 동시에 설명하는 유일한 부류다.
전례가 있다: 2026-10-02 회차가 `undici` high advisory 11건으로 릴리즈 web 단계가
`npm test` 에 닿기도 전에 깨져 있던 것을 발견해 잠금파일 3줄로 고쳤다.
그리고 `e5422a1` 커밋 메시지는 "두 `npm audit` 모두 0 vulnerabilities, govulncheck 0" 이라고
**태그 작성 시점(10-04 20:15)에는 통과했다**고 적고 있다 — 그 뒤 새 advisory 가 공개됐으면 그대로 깨진다.

고치는 방법(이 순서로만):
1. `cd webapp && npm audit --omit=dev --audit-level=high` 과 `npm audit --audit-level=high` 를
   실제로 돌려 지금 깨지는지, 어떤 패키지·advisory 인지 확인한다.
2. **매니페스트 변경 없이** 잠금파일의 해당 transitive 항목만 패치 버전으로 갱신한다
   (2026-10-02 가 `undici` 를 8.11.2 로 올린 것과 같은 모양: `webapp/package-lock.json` 만, 줄 수 몇 줄).
   의존성 메이저 업그레이드·`overrides` 신규 추가는 이번 회차 범위 밖이다.
3. Go 쪽이면 `cd server && go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...` 로 확인하고
   `go get <module>@<patched>` + `go mod tidy` 로 최소 올림만 한다.
4. **금지**: `--audit-level` 완화, `|| true`, `continue-on-error`, `npm audit fix --force`,
   advisory 억제, govulncheck 버전 내리기, 단계 삭제. 전부 "워크플로를 느슨하게 만들기" 에 해당한다.

### B. 릴리즈 전용 게이트 (A 가 아니면 여기, 미확인)

`e5422a1` 커밋 메시지가 스스로 적어 둔 문장이 결정적이다:
> "The gates this machine cannot run are the ones the release job owns: the `linux/amd64` image
> build, `scripts/verify-release.sh` against the exported archive, and the Playwright
> product-UI pass. They run on the tag push."

즉 v0.2.45 는 **이 셋만 검증되지 않은 상태로 태그가 올라갔다**. 로그가 이 중 하나를 가리키면:

- `scripts/verify-release.sh:173-174` 는 `probe_get ... | grep -F ...` 이고 스크립트 머리에
  `set -Eeuo pipefail` 이 있어 **grep 실패 시 아무 메시지 없이 exit 1** 한다.
  릴리즈 로그에 설명 없는 실패가 보이면 이 두 줄이다. 174 는
  `/api/v4/config/client` 응답에서 `"Version":"v0.2.45"` 를 문자 그대로 찾는다 —
  이미지에 `--build-arg VERSION` 으로 박힌 값과 태그가 어긋나면 여기서 죽는다.
  **고칠 때 grep 을 지우지 말고**, 실패 시 실제 응답을 stderr 로 찍는 진단을 더하거나
  값이 어긋난 쪽(마커/빌드 인자)을 고친다.
- 로컬 재현(도커 필요):
  ```
  cd /home/hkjang/.cache/auto-improve-wt/moyro
  git checkout --detach refs/tags/v0.2.45   # 또는 현재 트리
  docker build --platform linux/amd64 --provenance=false \
    --build-arg TARGETOS=linux --build-arg TARGETARCH=amd64 \
    --build-arg VERSION=v0.2.45 --build-arg COMMIT="$(git rev-list -n 1 refs/tags/v0.2.45)" \
    --build-arg BUILD_DATE="$(git show -s --format=%cI refs/tags/v0.2.45)" \
    --build-arg SOURCE_DATE_EPOCH="$(git show -s --format=%ct refs/tags/v0.2.45)" \
    --tag moyro:v0.2.45 .
  docker save --output /tmp/img.tar moyro:v0.2.45
  gzip -n -9 < /tmp/img.tar > moyro-v0.2.45.tar.gz
  docker pull postgres:16-alpine@sha256:cf78e76683b9ca8c5733cbbdce6c9262b45b6767934dd0a95e671f9a0fc20685
  EXPECTED_COMMIT="$(git rev-list -n 1 refs/tags/v0.2.45)" \
    bash scripts/verify-release.sh moyro:v0.2.45 moyro-v0.2.45.tar.gz
  ```
  생성한 `moyro-v0.2.45.tar.gz` 는 커밋하지 말고 지운다(`.gitignore` 확인 필요 — 미확인).
- Playwright 쪽이면 `bash scripts/verify-product-ui.sh moyro:v0.2.45 <outdir>` 로 재현한다.
  `e5422a1` 은 `webapp/e2e/product-pages.spec.ts:1532` 의 버전 문자열도 함께 바꿨다
  (`"v0.2.45은 검증된 단일 오프라인 자산으로 배포"`). 이 기대 문자열이 실제 페이지와 어긋나면
  여기서 두 번 다 깨진다. 정찰이 `0.2.4x` 전수 grep 으로 확인한 범위에서는
  리포지터리 안 마커가 전부 `0.2.45` 로 일관됐고 남은 `0.2.44` 는 없었다 —
  따라서 **마커 누락은 가능성이 낮다**. 단 `docs/*.html` 안의 긴 줄은 전량 확인하지 못했다(미확인).

### C. 체크섬 핀이 깨진 플러그인 픽스처 (가능성 낮음, 미확인)

`scripts/fetch-plugin-test-fixtures.sh` 가 GitHub 릴리즈 에셋 4개를 sha256 핀으로 받는다.
업스트림 에셋이 교체되면 **CI 와 릴리즈가 동시에, 재실행해도 계속** 깨진다 — 증상 모양이 맞다.
다만 이 핀은 그 결함을 잡기 위해 존재하므로 **체크섬을 새 값으로 그냥 덮는 것은 금지**다.
업스트림이 정말 재게시한 것이라면 새 에셋 내용을 확인하고 핀 갱신 근거를 커밋 메시지에 남긴다.
정찰은 이 스크립트를 실행하지 못했다(네트워크 명령 권한 거부).

### D. `-race -p 1 ./...` 플레이크 (가능성 낮음)

직전 회차가 이미 하나 고쳤다(`52a3812`, settings 직렬화 테스트의 `secondEntered`/`completed` 순서).
"같은 이유로 두 번" 은 플레이크와 어울리지 않으므로, 로그가 명확히 테스트 실패를 가리킬 때만 본다.
그 경우 **같은 테스트가 두 실행에서 같은 줄에서 깨졌는지**를 먼저 확인하고,
`t.Skip` 이나 타임아웃 늘리기로 덮지 말고 경합의 원인을 고친다.

## 수용 기준

1. 실패한 릴리즈 단계와 그 출력이 로그에서 인용되어 원장에 남는다 (추측이 아니라 인용).
2. 두 번의 실패가 **같은 단계·같은 출력**이었음이 확인된다. 다르면 둘 다 적고 공통 원인을 밝힌다.
3. 그 단계와 **동일한 명령**이 수정 후 로컬에서 통과한다 (명령과 출력 요약을 원장에 적는다).
4. 수정 전에 그 명령이 **실패하는 것을 먼저 관찰**했음이 기록된다(RED → GREEN).
5. `.github/workflows/*.yml` 의 게이트가 **느슨해지지 않았다**: `--audit-level`, `-race`,
   `-p 1`, `continue-on-error`, `|| true`, 단계 삭제, 타임아웃 연장 중 어느 것도 없다.
   워크플로 파일 diff 가 생기면 그것이 왜 느슨해지는 변경이 아닌지 커밋 메시지에서 논증한다.
6. 릴리즈 경로를 건드렸으므로 **릴리즈 순서 그대로** 재검증한다:
   `cd server && go build ./... && go vet ./...`,
   DSN 을 준 `go test -race -p 1 -count=1 ./...`,
   `cd webapp && npm ci && npm audit --omit=dev --audit-level=high && npm audit --audit-level=high && npm test && npm run typecheck && npm run build`,
   `bash scripts/check-source-sizes.sh`, `node scripts/verify-pages.mjs`.

## 건드릴 파일 (파일 6개 안쪽 — 분기별로 하나만 고른다)

- A 분기: `webapp/package-lock.json` (해당 transitive 항목만) **또는** `server/go.mod` + `server/go.sum`.
  → 프로덕션 코드 0파일. 가장 좁다.
- B 분기: `scripts/verify-release.sh` (실패 메시지 추가 또는 어긋난 값 수정) **또는**
  `webapp/e2e/product-pages.spec.ts:1532` 부근 (어긋난 기대 문자열) — 둘 중 한쪽만.
- C 분기: `scripts/fetch-plugin-test-fixtures.sh:38-53` 의 핀 (근거를 커밋에 남길 때만).
- D 분기: 해당 Go 테스트 파일 1개.
- **공통 금지**: `.github/workflows/release.yml`, `.github/workflows/ci.yml` 는
  게이트를 느슨하게 하지 않는 변경(예: 실패 시 진단 출력 추가)만 허용. 완화는 금지.

여러 분기가 동시에 깨져 있으면 **릴리즈를 막는 가장 앞 단계 하나만** 이번 회차에 담고
나머지는 보류 아이디어로 적는다(파일 10개 이상 변경은 47% 가 재작업됐다).

## 검증 명령 (이 저장소에서 실제로 도는 것)

```
cd /home/hkjang/.cache/auto-improve-wt/moyro/server
go build ./... && go vet ./...
MOYRO_TEST_POSTGRES_DSN='postgres://...@127.0.0.1:55433/...?sslmode=disable' go test -race -p 1 -count=1 ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...

cd /home/hkjang/.cache/auto-improve-wt/moyro/webapp
npm ci
npm audit --omit=dev --audit-level=high
npm audit --audit-level=high
npm test
npm run typecheck
npm run build

cd /home/hkjang/.cache/auto-improve-wt/moyro
bash scripts/check-source-sizes.sh
node scripts/verify-pages.mjs
```

DB 컨테이너는 이전 회차가 `moyro-pg-improve` / 호스트 55433 으로 썼다 (이번 세션 미재확인).
`npm` 은 Windows 쪽에서만 `C:\Program Files\nodejs\npm.cmd` 를 쓴다. Linux 워크트리에서는
`webapp/scripts/run-vitest.mjs` 가 인터프리터를 고정하므로 `npm test` 를 그대로 쓴다.

## 위험과 피할 것

- **워크플로 완화 금지**가 이 과제의 핵심 제약이다. 운영자가 반복해 말한 것:
  "릴리즈·빌드 경로를 건드리는 변경은 릴리즈까지 통과하는 것을 확인할 것 —
  릴리즈를 반복해 깨뜨린 머지는 되돌림 PR 과 자율화 강등으로 이어졌다."
- **grep 을 증거로 제출하지 말 것**: "문자열이 있다" 는 그 게이트가 통과한다는 증명이 아니다.
  반드시 실패한 명령 자체를 돌려 RED→GREEN 을 보여라.
- `npm audit` 는 실시간 DB 조회라 **같은 트리에서도 날마다 결과가 달라진다**. 통과 출력에
  날짜/시각을 남겨라. 또 base 에서 이미 깨져 있을 수 있으므로 **자기 변경 탓으로 오인하지 말 것**.
- webapp `typecheck` 가 `useDraft.test.tsx:196` TS2345 로 깨지면 **자기 변경 탓이 아니다**:
  `webapp/tsconfig.json` 에 `"types"` 가 없어 `@types/node` 가 스코프에 끼는 기존 결함이다(교훈 2026-09-09).
- `gofmt -l server/internal` 은 기존 9파일을 출력한다. 독립 포맷 PR 금지, 자기가 고친 파일만 clean 하게.
- 보호 경로 회피: `server/internal/store/migrations`, `auth`/`session`/`oidcauth` 는 이 과제와 무관하다.
- 로컬에서 만든 `moyro-v0.2.45.tar.gz`, `/tmp/img.tar`, 도커 네트워크/볼륨은 커밋·잔류 금지
  (`verify-release.sh` 는 trap 으로 정리하지만 아카이브 파일은 남는다).
- `docker` 가 이 워크트리에서 쓸 수 있는지 미확인. 못 쓰면 B 분기는 로컬 재현이 불가능하니
  수용 기준 3 을 충족할 수 있는 A/C/D 분기를 고르고, B 는 그 사실을 적어 보류로 넘긴다.

## 차선 후보

로그가 "실은 릴리즈가 아니라 PR #33 의 CI 가 실패했다" 를 가리키고 그 원인이 이미 main 에서
해소됐다면(머지 커밋 `d89ebf2` 가 이미 들어와 있다), 이번 회차는 **릴리즈 전용 게이트를
태그 전에 돌릴 수 있게 만드는 것**으로 바꾼다: `scripts/verify-release.sh:173-174` 의
침묵하는 `| grep -F` 두 줄에 실패 시 실제 응답 본문을 stderr 로 찍는 진단을 추가한다
(파일 1개, 게이트는 그대로, 다음 회차가 로그 없이 추측하는 일이 없어진다).
그마저 성립하지 않으면 보류 목록의
`createCustomProfileField`/`patchCustomProfileField` (`compat_wave_handlers_final.go:1212/1238`)
의 `_ = decodeCappedBody` 를 직전 회차가 만든 `decodeOptionalCollectionBody` 로 바꾸는 과제
(가치 2 / 위험 2 / S)를 고른다 — 단 이 둘은 쓰기 배치가 아니라 단건 쓰기이므로
빈 본문 계약(EOF 를 400 으로 할지 통과시킬지)을 먼저 정해야 한다.
