# 회차 노트 2026-10-05-193727-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:37] base pinned — main@b3ec457
- [러너 19:37] autonomy release — 

## 정찰 노트
- 골랐다: `make frontend-test` 의 의존성 보장(S/위험1/파일1). POP3 `retrBody`·MaxMessageBytes 두 1순위급은 PR #22 사람 반려 여부가 여전히 미확인(gh 미인증)이라 규칙대로 제외했고, 새로 조사한 IMAP 후보 둘(`ensureIndex` 캐시 잠금, Atoi 오류 무시)은 코드를 읽어 **관찰 가능한 결함이 없음을 확인해 `rejected`** 로 닫았다 — 그래서 확인된 재현이 있는 유일한 S 과제가 이것이었다.
- 추측으로 적은 것: 실패 메시지 원문. 이 워크트리에 `web/node_modules` 가 없는 것은 실측했지만 `make frontend-test` 실행 승인이 나지 않아 출력을 못 봤다 — "`npm run typecheck` 단계에서 깨진다" 는 추정이다. 구현자는 첫 단계로 출력을 남길 것.
- 조심할 것: `web/` 아래에 **새 파일을 만들지 말 것**(Tailwind v4 자동 탐지 → 자산 드리프트 33건, 2026-10-02 실측). `Makefile` 한 파일로 끝낼 수 있고, `frontend`/`frontend-check`/`build`/워크플로는 건드리지 않는다.
- 멱등성이 수용 기준의 핵심이다 — 무조건 `npm ci` 를 선행시키면 매 호출이 수 분 걸려 DX 가 오히려 나빠진다. 가드(`test -d web/node_modules`) 없이 제출하면 비평에서 걸릴 자리다.
- 프로필은 0일 전 것이라 새로 쓰지 않았다. 다만 두 줄은 이번에 갱신됐다: README govulncheck `@latest` 는 **확인됨**(README.md:309 vs 302 자기모순), `sync.max_message_bytes` 문서화 항목은 **done**.
- [러너 19:41] scout done — `make frontend-test` 가 깨끗한 워크트리에서 성립하게 한다 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇/왜: `Makefile` 의 `frontend-test` 에 멱등 가드 한 줄을 선행시켜 깨끗한 트리에서 성립하게 했다(ae0c4a2). 재현 실측: `sh: 1: tsc: not found` / Error 127. 덧붙여 README 의 `govulncheck@latest` 를 CI 핀 `@v1.6.0` 에 맞췄다(6a989bf) — 쓰기 전에 직접 돌려 exit 0 확인.
- 확신 없는 곳: 가드는 `test -d web/node_modules` 뿐이라 **디렉터리가 있지만 내용이 낡거나 불완전한 경우는 재설치하지 않는다**(package.json 변경 후 등). 과제서가 정한 계약이 이것이고 `frontend`/`frontend-check` 는 여전히 무조건 `npm ci` 를 하므로 CI 재현성은 그대로지만, 비평가가 볼 자리라면 여기다. 또 `make frontend-test` 는 이 박스에서 engine-node 래퍼가 npm PATH 가 가린 node v20 을 v22 로 되돌리는 경고를 매번 출력한다 — 기존 동작이고 내 변경과 무관하다.
- 일부러 하지 않은 것: `frontend`/`frontend-check`/`build`/`build-offline`/워크플로/`web/package.json`/`styles.css` 미접촉(릴리즈 경로 + Tailwind 자산 드리프트 교훈). `web/` 아래 새 파일 0개. 두 타깃의 `npm ci` 중복을 공통 타깃으로 묶지 않았다 — `frontend` 는 무조건 설치가 옳고 가드를 공유하면 CI 재현성이 흔들린다.
- 다음 역할 주의: 돌려 보려면 `web/node_modules` 를 지워야 수정 전 상태가 재현된다. 자산 드리프트는 0건이고 프런트 소스를 안 바꿨으니 `spa/assets` 재생성은 불필요. Go 코드 미변경이라 전체 race 스위트는 돌리지 않았다(`go build`+`go vet` 만 통과 확인) — PostgreSQL·브라우저 e2e 도 미실행.
- [러너 19:47] brief accepted — 채택 — 근거가 전부 코드와 맞았고(`frontend` 만 `npm ci`, `frontend-test` 는 typecheck+test 만, 이 워크트리에 `node_modules` 없음), 정
- [러너 19:48] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 판정 **approve** (risk low, blocking 없음). 가드 양쪽 분기를 실측했다: 웜(`node_modules` 있음) → `npm ci` 건너뛰고 typecheck + 50 files/426 tests, exit 0. 콜드(`node_modules` 이동 + `NPM=false`) → 가드 발화 후 `make: *** [Makefile:18] Error 1`(exit 2), 설치 실패를 삼키지 않는다. 테스트 후 복원, 워크트리 clean.
- 원장의 `- 실패 재현:` 이 고치는 증상과 일치한다. 기록된 `Makefile:16` 은 수정 전 줄번호, 내 실측은 `Makefile:18` — 가드 2줄 차이와 맞아 출력이 진짜임을 뒷받침한다.
- 구현자가 의심한 자리(낡은 `node_modules` 를 재설치 안 함)는 결함 아님으로 닫았다: `ci.yml` 이 `make frontend-test` 를 **호출하지 않고**(`npm ci` 직접) `frontend`/`frontend-check` 는 무조건 설치 → CI 재현성 영향 0. 남는 건 로컬뿐 — lockfile 변경 후 낡은 의존성으로 초록이 날 수 있다. 다음 회차 선택지: `node_modules/.package-lock.json` 대조 가드(이번 계약 밖).
- README.md:309 govulncheck `@v1.6.0` 은 `ci.yml:173` 과 일치함을 확인했다(302줄 자기모순 해소). 과제 밖이지만 1줄·문서·별도 커밋·공급망 위험 감소 방향이라 범위 이탈로 반려하지 않았다. 보안·법무 차단 소견 모두 없음(인증/비밀값/개인정보/라이선스 미접촉).
- 내가 **못 본 것**: 전체 `go test -race`(Go 미변경), PostgreSQL 검사, 브라우저 e2e, 콜드 경로의 실제 `npm ci` 완주(분기 발화만 증명).
- [러너 19:51] review approved — 리뷰 승인 (risk=low)
- [러너 19:51] pr created — https://github.com/hkjang/postra/pull/35
