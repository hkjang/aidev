# 회차 노트 2026-10-17-235854-multica-improve — multica
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:58] base pinned — main@18fa9f7
- [러너 23:58] autonomy release — 

## 정찰 노트
- 지정된 수정 과제가 명확했다: 이전 회차 `verify.txt` 115~161줄이 원인을 그대로 보여준다 — pnpm 이 `--silent` 를 turbo 에 넘기고 turbo 2.9.5 가 `unexpected argument` 로 죽어 테스트가 0개 실행됐다. 새 아이디어를 고르지 않고 이것만 적었다.
- 미확인(추측으로 적은 것): 이 샌드박스는 `pnpm` 실행 권한이 없어 직접 재현하지 못했다. 특히 **인자 없는 `turbo test --filter=... --`(빈 패스스루)를 turbo 2.9.5 가 받아들이는지 미확인** — 구현자가 가장 먼저 확인해야 하고, 거부하면 접근을 버리고 차선으로 갈 것.
- 구현자가 조심할 것: `typecheck`/`lint` 에 같은 `--` 를 달아도 tsc/eslint 가 `--silent` 를 모르므로 아무것도 고쳐지지 않는다(ideas.json 에 rejected 로 기록). 건드릴 파일은 루트 `package.json` 한 개면 끝난다.
- 증거 제출은 실제 실행 출력으로. `package.json` 에 `--` 가 있는지 보는 문자열 검사 테스트를 추가하지 말 것(운영자 교훈).
- 프로필이 없어 `profile.md` 를 새로 썼다. 검증 함정 절에 이번 `pnpm test --silent` 결함과 turbo 캐시 적중 함정을 남겼다.
- [러너 00:04] scout done — 루트 `test` 스크립트가 전달 인자를 받지 못해 `pnpm test --silent` 가 turbo 사용법 오류로 죽는 것을 고친다 (가

## 구현 노트
- 루트 `package.json` 의 `scripts.test` 만 `turbo test --filter=!@multica/mobile --` 로 끝맺었다(1줄). pnpm 이 스크립트 뒤 인자를 turbo 자신의 플래그 자리에 붙여 `--silent` 가 turbo 를 죽이던 것 — 이제 vitest 로 전달된다.
- 정찰이 미확인으로 남긴 **빈 패스스루**를 먼저 검증했다: turbo 2.9.5 는 인자 없는 trailing `--` 를 받고(exit 0), `--dry-run=json` 의 6개 `#test` 해시가 `--` 유무와 바이트 단위 동일 → 인자 없는 경로·기존 캐시 무영향.
- 확신 없는 곳: 없음에 가깝다. 네 명령 모두 `0 cached` 로 실제 실행됐고, 일부러 깨뜨린 임시 프로브로 `--silent` 하에서도 실패가 exit 1 + 테스트 이름으로 남는 것과 프로브의 `console.log` 가 억제되는 것(= 인자가 vitest 까지 실제 도달)을 함께 확인한 뒤 프로브를 삭제했다.
- **비평가가 먼저 볼 것(의도된 부수효과)**: 뒤따르는 인자가 이제 turbo 가 아니라 vitest 를 향한다. 그래서 `pnpm test --force` 는 vitest 가 `Unknown option --force` 로 거부한다(실측). 패스스루를 여는 어떤 방법으로도 둘을 동시에 만족시킬 수 없는 트레이드오프이며, turbo 자신의 플래그는 `pnpm exec turbo test ...` 를 쓰면 된다. `.github/workflows/ci.yml` 은 `pnpm exec turbo test` 를 직접 쓰므로 CI 는 무영향(249줄은 이미 `-- --shard=` 로 같은 패스스루를 쓴다).
- 일부러 하지 않은 것: `typecheck`/`lint`/`build` 에는 `--` 를 달지 않았다(tsc·eslint 에 대응 플래그가 없어 실패 지점만 옮겨간다). `turbo.json` 과 CI 도 건드리지 않았다. `package.json` 에 `--` 가 있는지 보는 문자열 검사 테스트는 운영자 교훈에 따라 추가하지 않았다 — 증거는 실행 출력이다.
- 다음 역할이 조심할 것: Go(`make test`)와 E2E 는 이 변경과 무관하여 **실행 안 함**. `pnpm test` 전체는 약 1분 7초(views 가 479파일/6070테스트로 대부분)이고, 재확인 시 turbo 캐시 적중에 속지 말 것 — 요약의 `Cached: 0 cached` 를 볼 것.
- [러너 00:35] brief accepted — 채택 — 과제서의 진단(pnpm → turbo 인자 누출)과 지정 파일 1개가 지금 코드와 정확히 일치했고, 미확인으로 남긴 빈 패스
- [러너 00:35] verify failed — 실패한 검증: pnpm run typecheck --silent (exit 1)
