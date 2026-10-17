# 과제서 — 2026-10-17-191337-moyro-improve

> **먼저 읽을 것 (전제 확인 절차).** 적재된 오류 문구는 `error: interrupted before PR` 다.
> 이것은 **러너 세션이 PR 전에 끊긴 것**이지 릴리즈 워크플로의 빨간불이 아니다.
> 2026-10-16 회차도 똑같이 "릴리즈가 같은 이유로 두 번 실패" 전제를 받았으나, 실제로는
> 실패 1건 + 이미 수리 + 릴리즈 성공이었고 **예산의 절반이 반증에 들어갔다**.
> 이번 정찰 세션은 `curl`·`WebFetch` 호출이 **권한 거부**되었고 `node`/`bash scripts/*` 실행도
> 거부되어, 빨간 런의 유무를 **확인하지 못했다(미확인)**.
> 구현자는 **첫 10분 안에** 아래를 실행해 전제를 출력으로 확정하라(공개 저장소라 인증 불필요):
>
> ```
> curl -s 'https://api.github.com/repos/hkjang/moyro/actions/runs?per_page=15'   # id/name/conclusion/head_sha
> curl -s 'https://api.github.com/repos/hkjang/moyro/actions/runs/<실패id>/jobs' # 실패 "스텝 이름"까지 보인다
> ```
>
> - **빨간 런이 실제로 있으면** 그 스텝을 로컬 재현해 고치는 것이 1순위다(아래 과제보다 우선).
>   워크플로를 느슨하게 만들어 통과시키는 것은 금지.
> - **빨간 런이 없으면**(10-16 처럼 전제가 거짓이면) 아래 과제를 그대로 구현하라. 아래 과제도
>   "릴리즈가 같은 이유로 반복해서 깨지는" 실제 구조를 제거하는 **수정 과제**다.

---

- 과제: 릴리즈 게이트 `scripts/verify-pages.mjs` 가 기대 버전을 **문자열 리터럴로 박아 두어** 매 릴리즈마다 손으로 고쳐야 하고, 한 군데만 어긋나면 CI·릴리즈가 **결정적으로, 재실행해도 똑같이** 깨지는 것 제거 (가치 3 / 위험 2 / 작업량 S)

- 왜: `scripts/verify-pages.mjs:266` 은 `if (!html.includes("v0.2.51"))` 로 기대 버전을 **리터럴**로 들고 있다(실제로 열어 확인했다). 이 리터럴은 버전의 권위 있는 출처가 아니라 **복사본**이고, 같은 복사본이 `webapp/package.json:3`, `plugin-sdk/js/package.json:3`, `docs/openapi-v4.yaml:4`, `docs/openapi-moyro.yaml:4`, `docs/USER_GUIDE.md:3`, `webapp/src/features/system/SystemInfoContext.tsx:6`, 그리고 docs/ HTML·MD 본문 수십 곳에 흩어져 있다. 직전 릴리즈 커밋 `a7b87e6 release: carry release markers to v0.2.51` 은 이 동기화를 위해 **21개 파일을 손으로 고쳤다**(`git show --stat a7b87e6` 로 확인; 그중 `scripts/verify-pages.mjs` 4줄 = 게이트 자신). 즉 게이트가 **자기가 검사할 정답을 매번 사람이 받아 적는 구조**다. docs 쪽만 올리고 `:266` 을 빼먹거나 그 반대면 `node scripts/verify-pages.mjs` 가 ci.yml·release.yml **두 곳에서** 실패하고, 같은 태그를 재실행해도 **바이트 동일하게 다시 실패한다** — "같은 이유로 두 번 실패" 의 정확한 모양이다. 기대 버전을 선언된 출처에서 **유도**하게 바꾸면 이 실패 모드가 사라지고, 출처들끼리의 drift 가 새로 **검출된다**(게이트가 느슨해지는 게 아니라 강해진다).

- 수용 기준:
  1) `scripts/verify-pages.mjs` 안에 `v0.2.51` / `0.2.51` 같은 **현재 버전 리터럴이 하나도 남지 않는다**(`grep -n '0\.2\.5' scripts/verify-pages.mjs` 출력 0줄). 기대 버전은 `webapp/package.json` 의 `version` 에서 읽는다.
  2) 지금 작업 트리에서 `node scripts/verify-pages.mjs` 가 **수정 전과 똑같이 exit 0** 이다(기존 통과 상태를 깨지 않는다).
  3) **RED 증명(커밋하지 말 것)**: `webapp/package.json` 의 version 을 `0.2.52` 로 임시 변경하면 게이트가 **실패**하고, 실패 메시지가 **어느 파일이 어긋났는지 이름으로** 말한다(네 릴리즈 페이지 + 버전 출처들). 되돌리면 다시 exit 0. 이 RED/GREEN 출력 두 줄을 회차 노트에 인용하라.
  4) **새 검사 추가(drift 검출)**: `webapp/package.json`, `plugin-sdk/js/package.json`, `docs/openapi-v4.yaml`, `docs/openapi-moyro.yaml` 네 곳의 선언된 버전이 서로 다르면 실패한다. 이 중 하나만 `0.2.52` 로 임시 변경해도 RED 가 되는 것을 확인하라.
  5) **기존 단언을 하나도 없애지 않는다**: `:267` 의 stale `v0.1.x` 금지, `:268` 의 `"dateModified": "2026-09-07"`, `:275` 의 screens.html `v0.2.1` + `2026-08-30`, `:284-286` 의 sitemap lastmod 개수(2026-09-07 ×4, 2026-08-30 ×1)는 **그대로 남는다**. screens.html 의 `v0.2.1` 과 sitemap 의 날짜들은 **의도적으로 과거 캡처 자산을 가리키는 값이므로 유도 대상이 아니다** — 리터럴로 유지하라. (`:286` 메시지 문자열 안의 `v0.2.51` 만 유도된 버전으로 치환한다.)
  6) `.github/workflows/` diff **0줄**, `webapp/`·`server/`·`docs/` **무수정**. 건드리는 프로덕션 파일은 `scripts/verify-pages.mjs` **단 1개**.

- 건드릴 파일:
  - `scripts/verify-pages.mjs` — ① 파일 상단(기존 `readFileSync`/`join` import 가 이미 있다)에서 `webapp/package.json` 을 읽어 `releaseVersion = "v" + pkg.version` 을 만든다. ② `:266` 의 `"v0.2.51"` 두 군데(조건식 + 메시지)를 `releaseVersion` 으로 치환. ③ `:286` 메시지의 `v0.2.51` 을 `releaseVersion` 으로 치환. ④ 새 drift 검사 블록(수용 기준 4) 추가 — `plugin-sdk/js/package.json` 은 JSON 으로, 두 openapi YAML 은 `/^\s{2}version:\s*(\S+)\s*$/m` 수준의 좁은 정규식으로 `info.version` 을 뽑아 `pkg.version` 과 비교(YAML 파서 의존성 추가 금지; 두 파일 모두 `info:` 바로 아래 `  version: 0.2.51` 한 줄이다 — 실제로 확인했다). 읽기 실패·매치 실패는 **조용히 넘기지 말고 failures 에 push** 한다.
  - (테스트 파일 없음 — 이 스크립트에는 전용 테스트 하네스가 없다. 수용 기준 3·4 의 임시 변경 → 출력 관찰 → 되돌리기가 증명이다. 작업 트리에 변경을 남기지 말 것.)

- 검증 명령:
  - `cd /home/hkjang/.cache/auto-improve-wt/moyro && node scripts/verify-pages.mjs` (주 게이트, 즉시 끝남)
  - `grep -n '0\.2\.5' scripts/verify-pages.mjs` → 출력 0줄 (수용 기준 1)
  - `bash scripts/check-source-sizes.sh` (size ratchet)
  - `git diff --check` / `git diff --stat .github/` → 0줄
  - 릴리즈 순서 영향 확인용(범위 밖이지만 싸다): `cd server && go build ./...` — 서버 무수정이므로 통과해야 정상.
  - **서버 테스트는 이 과제에 불필요하다**(Go 파일 무수정). 돌릴 거면 DSN 을 자식 프로세스 환경에만 주고 `cd server && go test -race -p 1 -count=1 ./...`; httpapi 가 0.2초로 끝나면 DB 테스트가 안 돈 것이다.

- 위험과 피할 것:
  - **게이트 완화 금지.** 이번 변경은 "검사를 지우는" 것이 아니라 "정답의 출처를 바꾸는" 것이다. 기존 단언 5종(수용 기준 5)을 하나라도 지우면 이 회차는 실패다.
  - `screens.html` 의 `v0.2.1` 과 sitemap 의 `2026-08-30`/`2026-09-07` 은 **과거 캡처 자산의 고정 값**이다. "일관성" 을 이유로 유도 대상에 끌어들이지 말 것 — 그렇게 하면 다음 릴리즈에서 게이트가 거짓 실패한다.
  - `.github/workflows/` 는 보호 경로다. 워크플로는 `node scripts/verify-pages.mjs` 를 인자 없이 부르므로(release.yml:151, ci.yml 동일) **스크립트 시그니처를 바꾸지 말 것**.
  - 스크립트는 저장소 루트에서 불린다. 경로는 기존 `root`/`docs` 계산 관례를 그대로 쓰고 새 cwd 가정을 만들지 말 것.
  - npm 의존성 추가 금지(YAML 파서 포함). 이 스크립트는 node 표준 모듈만 쓴다.
  - 과거 교훈: 10-16 회차는 **전제가 뒤집힌 과제서**를 받아 예산 절반을 날렸다. 위 "전제 확인 절차" 를 **먼저** 하라.

- 차선 후보:
  1) **`scripts/verify-product-ui.sh:151` 의 상대 경로 `bash scripts/contract-test.sh`** — 같은 스크립트 `:97` 은 docker build 용으로 `BASH_SOURCE` 에서 `repo_root` 를 계산하는데, 계약 테스트만 cwd 의존이다. 저장소 루트 밖에서 부르면 "No such file" 로 죽는다. `:97` 의 `repo_root` 를 재사용해 절대 경로로 부르는 1줄 수정(가치 2 / 위험 1 / S). CI 는 항상 루트에서 부르므로 **이것이 릴리즈 실패 원인은 아니다** — 가치가 낮은 이유다.
  2) **`listReactions`(`server/internal/httpapi/handlers.go:2491-2499`)가 `ChannelForPost` 의 `pgx.ErrNoRows` 와 진짜 DB 장애를 둘 다 404 로 접는 것** — `errors.Is(err, pgx.ErrNoRows)` 분기만 넣으면 S. 다만 httpapi 상태코드 재분류 계열이 10-04/10-05/10-08/10-16 네 회차를 먹었으므로 방향 중복이다(가치 2 / 위험 2 / S).
