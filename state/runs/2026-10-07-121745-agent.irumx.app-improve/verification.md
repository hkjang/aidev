# 구현 검증 — 6b7ac2b

- 스킬 도구가 제공되지 않아 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/`의 completion-verification, systematic-debugging, test-driven-development SKILL.md를 직접 읽었다.
- 가설/원인: localToUtc는 없는 02:15를 15:45Z로, 정상 02:30을 15:30Z로 정확히 변환한다. nextRuns가 벽시계 첫 후보에서 n=1을 채우고 종료해 후자의 더 이른 UTC를 확인하지 못했다. 개별 변환값과 n=1/5 재현으로 시각 변환 오류를 배제했다.
- 수정: shift의 두 내부 루프만 날짜 끝까지 탐색한다. 기존 UTC 중복 제거, 마지막 정렬/slice, skip 조기 중단, 5년 상한, 파서·localToUtc·API 계약을 유지했다. 전체 5년의 분 후보를 모으지 않는다.
- 추가 테스트: 단위 6개(n=1/2/5 각각의 고정 기대값·note, 순차 조회와 일괄 조회 일치 및 엄격 증가, after 경계, 이동/정상 회차의 UTC 중복 제거); API 1개(person/addModel/publishAgent → POST schedules → pause → next_run_at 고정 → GET 상세의 5개 utcIso 검증).

## 실제 실행과 결과

1. 과제서의 `node --experimental-strip-types --input-type=module` 최소 재현: 수정 전 exit 1 (`actual: 1791042300000`, `expected: 1791041400000`), 수정 후 exit 0 (`Lord Howe n=1/n=5 prefix: PASS`).
2. `npm ci`: exit 0. 기존 의존성 경고: deprecated 2개 및 audit 8개(4 moderate, 4 high); lockfile·버전 변경하지 않음. `npm run check` 기준 검사 exit 0.
3. 수정 전 테스트 추가 후 `npm run build`: exit 0, 빌드 점검 통과. `npx playwright test --project=unit --project=api tests/unit-core.spec.ts tests/api-schedule.spec.ts --grep "30분 서머타임"`: 3 failed / 4 passed, 9.0s (`red-tests.log`). 실패는 n=1, 순차 조회, 실제 HTTP 상세이며 환경 오류가 아니다.
4. 수정 후 `npm run build`: exit 0, 빌드 점검 통과 (`build.log`). `npx playwright test --project=unit tests/unit-core.spec.ts`: 27 passed, 13.0s (`unit-tests.log`). 기존 KST/New York skip·shift/overlap_once 포함.
5. 원인 검증을 위해 두 중단 조건을 수정 전으로만 돌린 뒤 `npx playwright test --project=unit tests/unit-core.spec.ts --grep "30분 서머타임"`: 2 failed / 4 passed, 8.3s (`reverted-tests.log`). n=1과 순차 조회가 다시 실패했다. 이어 동일 수정을 복원했다.
6. 최종 `npm run check`: exit 0 (`check.log`). `npx playwright test --project=unit --project=api tests/unit-core.spec.ts tests/api-schedule.spec.ts`: 34 passed, 15.9s, skipped 없음 (`final-tests.log`). Worker 산출물은 4번에서 빌드한 최종 수정과 동일하며 매 테스트 실행 새 네임스페이스에서 서버를 시작했다.
7. `git diff --check`: exit 0. 커밋 직전 `git status --short --untracked-files=all`: 지정한 세 파일만 변경. `git add -A && git commit -m "예약 DST shift의 다음 실행 순서와 조회 개수 일관성 수정"`: 6b7ac2b 생성, hkjang 작성자 유지, 트레일러 없음.

## 테스트 실행 환경과 재현 명령

기존 8870 workerd는 다른 작업 트리(`/mnt/c/Users/USER/projects/agent.irumx.app`)에서 실행 중이었다. 종료하지 않고 Linux user/network 네임스페이스를 사용해 포트 재사용을 피했다. 공통 webServer는 매번 이 작업 트리의 새 dist와 `.wrangler/test-state`를 사용했다.

첫 Playwright 기동은 상위 디렉터리의 Node 20 실행 파일 때문에 실패했다(`Wrangler requires at least Node.js v22.0.0. You are using v20.19.2.`). 전역 파일을 바꾸지 않고 회차 경로의 `node22-shell.sh`로 npm 하위 셸의 PATH를 Node 22.23.1에 고정한 뒤 다시 실행했다. 이는 기능 실패와 구분한다.

```bash
npm_config_script_shell=/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-07-121745-agent.irumx.app-improve/node22-shell.sh npm run build
npm_config_script_shell=/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-07-121745-agent.irumx.app-improve/node22-shell.sh npm run check
npm_config_script_shell=/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-07-121745-agent.irumx.app-improve/node22-shell.sh unshare --user --map-root-user --net bash -c 'ip link set lo up && npx playwright test --project=unit --project=api tests/unit-core.spec.ts tests/api-schedule.spec.ts'
```

## 한계

전체 브라우저/다른 API 파일은 실행하지 않았다. 역사적 날짜 단위 타임존 전환의 정합성과 실제 운영 큐의 DST 실행은 이번 범위 밖이다. 실제 로컬 HTTP 상세까지 검증했고 기존 예약 큐 관련 API 테스트는 통과했다. LIVE=1·배포·원격 D1·운영 메일을 사용하지 않았다. 새 경고를 없애기 위해 의존성/바인딩 설정을 변경하지 않았으며, 테스트 출력에는 기존 NO_COLOR 및 AI 바인딩 안내가 있다(네트워크 네임스페이스는 루프백만 활성화).
