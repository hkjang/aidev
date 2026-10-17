# 회차 노트 2026-10-18-015851-multica-improve — multica
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:58] base pinned — main@18fa9f7
- [러너 01:58] autonomy release — 

## 구현 노트
- 수정 과제: `pnpm run typecheck --silent` 가 turbo 옵션 파서에서 죽던 것. 루트 build/typecheck/test/lint 를 새 `scripts/turbo-run.mjs` 로 보내 `--silent`/`--quiet` 만 turbo 의 `--output-logs=errors-only` 로 번역하고 나머지는 그대로 전달한다(첫 bare `--` 뒤는 태스크 인자로 보존). 베이스 main@18fa9f7 에는 지난 회차의 `--` 패스스루가 없어 네 스크립트 전부가 대상이었고, tsc/eslint 는 `--silent` 를 거부하므로 같은 접근을 다시 쓰지 않았다.
- 확신 없는 곳: (1) Windows. wrapper 는 `node` 로 `turbo/bin/turbo`(Node 스크립트)를 직접 띄워 shell·PATH 를 타지 않게 했지만 Windows 에서 실행해 보지는 못했다. (2) `--silent --output-logs=full` 조합은 돌려서 exit 0 을 확인했다(번역 생략 경로). 단 이 경로에는 테스트가 없다. (3) turbo 의 `--dry-run=json` 출력에서 JSON 시작을 `indexOf("{")` 로 찾는다 — pnpm/turbo 배너에 `{` 가 생기면 테스트가 깨진다.
- 일부러 하지 않은 것: `.github/workflows/ci.yml` 은 turbo 글로벌 해시에 들어 있어 손대지 않았다(전 태스크 캐시 무효화). 그래서 회귀 테스트는 이미 설치된 의존성으로 도는 `.github/actions/frontend-quality/action.yml` 복합 액션에 붙였다. AGENTS.md/문서는 명령 이름이 그대로여서 수정하지 않았다. `clean` 스크립트의 같은 결함은 러너가 호출하지 않아 ideas.json 으로 넘겼다.
- 다음 역할이 조심할 것: `scripts/turbo-run.test.mjs` 는 `pnpm run <script> --dry-run=json` 을 실제로 띄우므로 **node_modules(특히 turbo)가 설치된 체크아웃**에서만 돈다. 게이트 잡의 `node --test` 단계(의존성 없음)에 옮기면 반드시 실패한다. 테스트는 태스크를 실행하지 않고 계획만 읽으므로 20초 안에 끝난다.
- [러너 02:15] verify failed — 실패한 검증: cd server && go test ./... (exit 1)
