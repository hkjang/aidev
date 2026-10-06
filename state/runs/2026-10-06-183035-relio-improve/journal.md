# 회차 노트 2026-10-06-183035-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:30] base pinned — main@8630401
- [러너 18:30] autonomy low-risk — 롤백 PR 

## 정찰 노트
- 고른 이유: 실패 단계를 추측하지 않고 특정했다 — 지난 회차 디렉터리의 `ci-failure-e00f2446444f.txt` 가 `잡: test / 실패 단계: Frontend security audit` 다. 즉 깨진 것은 코드가 아니라 `npm audit --audit-level=high` 이고, 이 명령은 `ci.yml:26`·`release.yml:38` **두 곳에만** 있고 `Makefile:13-16`·루트 `package.json`·`scripts/` 에는 없다(grep 확인) — 그래서 `make test` green → PR → CI red 가 반복된다. 그래서 게이트를 느슨하게 하는 대신 **로컬 경로에 같은 문구로 복제**하고, lockfile 에만 손으로 박힌 1.2.2 를 `overrides` 로 선언하는 것을 골랐다. 보류 목록의 release.yml·esbuild·net.OpError·ClientIP 는 우선 과제가 아니어서 제쳤다.
- 확신 없는 곳(과제서의 추측): **현재 HEAD 에서 `npm audit --audit-level=high` 가 통과하는지 확인하지 못했다** — 이 세션은 네트워크·npm 실행이 허가되지 않았다(curl·npm audit 모두 차단). 구현자가 가장 먼저 돌려야 하는 명령이며, exit 0 이면 기준 2 는 할 일이 없고 기준 3·4 만 남는다. 원격 CI/release 이력도 이번에 조회하지 못했다("두 번 실패" 는 러너 전달 사항이지 내 실측이 아니다).
- 실측한 것: HEAD `8630401` 에서 `go test ./...` **23 ok / FAIL 0**(Go 쪽은 무죄) · `git show --stat 6b95f32` = `web/package-lock.json` 1파일 3+/3- 뿐이고 `web/package.json` 에 1.2.2 요구가 없음 · lockfile 의 `sha512-KGj/8Y43x35aZVDtt…` 가 `~/.npm/_cacache` 의 실제 `source-map-js-1.2.2.tgz` 항목과 **정확히 일치**(버전·해시 날조가 아니므로 `npm ci` 는 통과한다) · `lockfileVersion: 3` 이라 중복 `dependencies` 절 없음 · postcss 의 범위가 `^1.2.1` 이라 1.2.2 가 범위를 만족.
- 구현자가 조심할 것: `.github/workflows/` 는 **0줄** — `--audit-level` 하향·`|| true` 는 금지다. `npm install` 이 lockfile 을 대폭 재생성하면 `git diff --stat web/package-lock.json` 을 보고 `--package-lock-only` 로 최소화하거나 기준 3 을 버릴 것. `web/package.json:10` 의 `"$npm_node_execpath"` 와 `**` 글롭은 손대지 말 것(2026-10-02 의 중첩 npm v20 그림자). esbuild 선언은 같은 회차에 섞지 말 것(lockfile diff 판독 불가).
- 프로필은 1일 전 것이고 이번에 읽은 자리와 어긋나지 않아 새로 쓰지 않았다(검증 함정 항목에 'audit 게이트가 로컬 경로에 없다' 가 빠져 있는 것은 누락이지 모순이 아니다). 코드·커밋 변경 없음, `git status --short` 빈 출력.
- [러너 18:37] scout done — 수정 과제 — CI·release 를 두 번 깨뜨린 `npm audit --audit-level=high` 게이트를 **로컬 검증 경로에 재현 가능하�

## 구현 노트
- 무엇을 왜: `npm audit --audit-level=high` 가 `ci.yml:30`·`release.yml:34` 에만 있고 `Makefile` 에 없어 `make test` green → CI red 가 반복됐다. CI 와 **같은 문구로** `Makefile:16` 에 복제하고(게이트 느슨화 없음, `.github/workflows/` 0줄), `6b95f32` 가 lockfile 에만 손으로 박은 source-map-js 1.2.2 를 `web/package.json` 의 `overrides` 로 선언했다. 커밋 7bb7120, 프로덕션 2파일 + README 1문장.
- 확신 없는 곳·검증 못 한 것: (1) **`web/package-lock.json` 은 0줄 건드렸다** — 과제서 기준 3 의 `npm install --package-lock-only` 를 실제로 돌려 보니 override 가 lockfile 루트에 **기록되지 않고**(npm 10.9.8·11.21.0 둘 다) 무관한 `libc` 메타데이터 42줄만 퇴행해서 되돌렸다. `npm ci` 가 깨지지 않는 것은 npm 10 과 **CI 가 쓰는 npm 11.21.0** 양쪽에서 exit 0 으로 확인했지만, CI 의 실제 Node 24 이미지에서 돌려 본 것은 아니다 — 비평가는 여기를 먼저 보라. (2) override 는 오늘 **no-op** 이다(`^1.2.1` 이 이미 1.2.2 로 풀린다). 값은 장래 fresh resolution 의 바닥을 막는 것이고, 블록이 실제로 집행된다는 것만 섭동(override=1.2.0 → 1.2.0 으로 해석됨)으로 증명했다. (3) Docker 빌드·오프라인/업그레이드 컨테이너 테스트·`go test -race` 는 이번에 돌리지 않았다(변경이 빌드 레시피와 매니페스트 선언뿐).
- 일부러 하지 않은 것: `.github/workflows/` 수정(명시 금지) · `--audit-level` 하향이나 `|| true`(게이트 느슨화 금지) · esbuild devDependency 선언(같은 회차에 섞으면 lockfile diff 판독 불가, 다음 회차로) · `web/package.json:10` 의 `"$npm_node_execpath"`·`**` 글롭(한 글자도 안 건드림).
- 다음 역할이 조심할 것: **`make test` 는 이제 네트워크가 있어야 돈다** — `npm ci` 에 더해 `npm audit` 가 레지스트리에서 advisory 를 받아온다. 이것은 CI 와 같아지기 위한 **의도된 비용**이니 회피 플래그를 넣지 말 것. 네트워크 없는 환경에서는 `go test ./...`·`go vet ./...` 를 따로 돌려라. 그리고 **lockfile 을 만지는 다음 회차는 로컬 npm 10.9.8 을 쓰지 말 것** — rollup 선택적 바이너리의 `libc` 42줄을 조용히 지운다(`npm install --prefix /tmp/npm11 npm@11` 로 npm 11 을 띄워 쓰는 방법을 이번에 썼다).
- [러너 18:47] brief accepted — 채택 — 과제서의 실측이 코드와 전부 일치했다: 실패 단계가 `test / Frontend security audit`, audit 명령이 `ci.yml:30`·`release.yml:34
- [러너 18:47] verify passed — 검증 10개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 구현자의 1번 자기 의심을 직접 재현해 해소했다: web/package.json+package-lock.json 만 복사한 깨끗한 디렉터리에서 `npm ci` 가 npm 10.9.8·npm 11.21.0(CI Node 24 의 npm) 양쪽 exit 0, 실행 후 lockfile 바이트 동일. 섭동(overrides=1.2.0 → `npm install --package-lock-only` 가 1.2.0 으로 해석, 루트 overrides 는 여전히 미기록)으로 블록이 집행되는 것과 lockfile 0줄 diff 가 npm 의 정상 동작임을 둘 다 독립 확인했다.
- 워크트리에서 `make test` 전체 exit 0(go test·vet·npm ci·새 audit 게이트·typecheck·node --test 27건·vite build). 빌드 후 `git status --short` 빈 출력 — 임베드 자산 재현·앵커 보존. `npm audit --audit-level=high` 는 현재 0 vulnerabilities.
- 못 본 것: CI 의 실제 Node 24 러너 이미지, Docker 빌드, 오프라인/업그레이드 컨테이너 테스트, `go test -race`. Dockerfile:6-8 이 `npm ci`·`npm run build` 만 쓰는 것은 읽어서 확인했으므로 오프라인 이미지 경로는 이번 변경에 영향받지 않는다.
- 남는 우려(릴리즈 노트에 적을 것): (1) README.md:162 의 "make test 통과 → CI audit 단계 통과" 는 advisory DB 가 시점 의존이라 일반적으로 거짓 — 거절 사유로는 약하지만 다음에 문구 조건화 권장. (2) `make test` 는 이제 레지스트리 접근 필수(audit 는 캐시로 대체 불가) — 에어갭 개발 환경에서 새로 깨지며 README 에 미기재. (3) Makefile:16 ↔ ci.yml:30 ↔ release.yml:34 문구 일치를 고정하는 가드가 없어 같은 드리프트가 재발 가능 — 다음 회차 후보.
- 부수 관찰: npm 11 의 `npm ci` 가 esbuild@0.28.2 postinstall 이 allowScripts 미승인이라는 warn 을 낸다. 지금은 exit 0 이지만 강제되면 CI 가 깨진다.
- [러너 18:51] review approved — 리뷰 승인 (risk=low)
- [러너 18:51] pr created — https://github.com/hkjang/relio/pull/45
- [러너 18:55] ci passed — 검사 2개 모두 success
- [러너 18:55] merge done — 7bb7120
- [러너 18:55] release skipped — 자율화 단계 low-risk — 릴리즈는 사람이
