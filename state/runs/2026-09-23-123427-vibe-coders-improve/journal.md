# 회차 노트 2026-09-23-123427-vibe-coders-improve — vibe-coders
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:34] base pinned — master@d67267b
- [러너 12:34] autonomy release — 

## 정찰 노트
- work_mem 을 골랐다: 검증 훅(`settingDef.validate`)이 이미 있어 S 로 끝나고, 값을 읽는 경로가 env·DB 둘인데 어느 쪽도 검사하지 않는 자리라 운영자 규칙(모든 경로가 같은 값을 같게 읽기)에 맞는다. pnpm 옵션 복원은 네 번째 반복이라 제쳤고(러너 범위), MCP SSO·메일 후보는 선행 기능이 이 HEAD 에 없어 성립하지 않는다.
- 추측으로 적은 것: "잘못된 `SET LOCAL` 이 트랜잭션을 abort 시켜 뒤 쿼리가 실패한다" 는 PostgreSQL 문서 지식이다. 이 환경에 PostgreSQL 이 없어 실행으로 확인하지 못했다 — 구현자는 이것을 증명하려 하지 말고, 검증 가능한 두 가지(저장 거부, SQL 미생성)로만 수용 기준을 채울 것.
- 조심할 것 1: 같은 함수의 30s 고정 타임아웃은 09-20 회차 수정이 미병합이라 그대로 남아 있다. 눈에 띄어도 이번 diff 에 섞지 말 것.
- 조심할 것 2: 검증 정규식을 넓게 잡으면 정상 설치의 저장이 거부된다. 저장 시점에만 걸고, 이미 저장된 값의 읽기 경로는 건드리지 말 것.
- 확인한 것: `internal/proxy` 컴파일·테스트 실행 형식 exit 0. 코드 수정·커밋 없음, 작업 트리 깨끗.
- [러너 12:38] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 무엇을/왜: `text2sql.work_mem` 에만 `settingDef.validate` 가 없어 어떤 문자열도 저장됐고, 그 값은 `executeReadOnlyQuery` 가 `SET LOCAL work_mem = '<v>'` 로 이어 붙인다. PostgreSQL 이 메모리 GUC 로 받는 형태만 허용하는 검증기를 레지스트리에 붙였다(커밋 f8b3b42).
- **확신 없는 것 1(가장 먼저 볼 곳)**: `memSizeRe` 가 PostgreSQL 이 실제로 받는 집합과 정확히 일치하는지는 **실행으로 확인하지 못했다** — 이 환경에 PostgreSQL 이 없다. 정규식은 PG 문서·`parse_int` 지식으로 썼다(단위 b/kb/mb/gb/tb, 대소문자 무관, 공백 허용, 단위 없는 수는 base unit=kB, PG12+ 소수 허용). 너무 좁으면 **정상 설치의 저장이 거부**된다 — 그게 이 변경의 유일한 실질 위험이다.
- **확신 없는 것 2**: 정찰이 적은 "잘못된 SET LOCAL 이 트랜잭션을 abort 시킨다" 는 증명하지 않았고 증명하려 하지도 않았다. 이 커밋의 근거는 그게 아니라 "저장 시점에 아무 신호가 없다" 뿐이다.
- 일부러 안 한 것 1: **읽기 경로(`executeReadOnlyQuery`)를 건드리지 않았다.** 이미 저장된 값의 런타임 동작은 완전히 불변 — 정규식이 틀려도 돌던 설치가 죽지 않는다. 대신 읽기 경로와 같은 작은따옴표 제거를 검증 전에 적용해 두 경로가 값을 같게 읽는다. 쓰기 경로 5곳(PUT·bulk·import·change-set dryrun/apply)이 전부 `validateSettingValue` 한 곳을 지나는 것은 grep 으로 확인했다.
- 일부러 안 한 것 2: **미병합 pnpm `--silent` 호환 수정(964d2b0/3a17c4f)을 cherry-pick 하지 않았다** — 정찰이 러너 범위로 제쳤고 벌써 네 번 반복된 접근이다. 따라서 러너 `run_verify` 가 `pnpm run typecheck --silent` 를 만들면 이 회차도 TS5072 로 verify-failed 가 날 수 있다. 이건 이 diff 의 결함이 아니다 — 러너/수리가 판단할 것.
- 다음 역할 주의: 새 테스트 3개는 실제 `NewServer().Routes()` + 실제 modernc SQLite 로 돌아 별도 DB 가 필요 없다. web 게이트는 `web/node_modules` 가 이번에 설치됐지만 커밋 대상이 아니다(gitignore). race 전체(`-race ./internal/...` 약 19분)·Docker smoke·브라우저 E2E·SBOM/vuln 게이트는 이번 회차에서 돌리지 않았다.
- [러너 12:48] verify failed — 실패한 검증: cd web && pnpm run typecheck --silent (exit 1)
