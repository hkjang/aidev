# 회차 노트 2026-10-07-211810-eval.irumx.app-improve — eval.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:18] base pinned — main@202c416
- [러너 21:18] autonomy release — 
- [러너 21:18] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- 무엇을·왜: `.github/` 가 없어 러너 PR 이 전부 "CI 검사 없음" 으로 막혀 있었다. 러너가 로컬에서 돌리는 두 명령(`npm run build` → `npm test`)을 `.github/workflows/ci.yml` 한 파일로 그대로 옮겼다. 바꾼 파일 1개, 프로덕션·시험·문서 변경 0개.
- 검증: CI 와 같은 조건(Cloudflare 자격증명 없음 · 빈 임시 홈 · `npm ci` 로 새로 받은 의존성)에서 실제로 돌렸다 — build 종료 0("빌드 점검 통과"), `npm test` → `50 passed (1.5m)`(unit·api·desktop·mobile). 워크플로 YAML 은 `yaml.safe_load` 로 파싱·`on`·steps 를 확인했다.
- **확신 없는 곳**: GitHub 러너에서 실제로 돌려 보지는 못했다(이 세션은 푸시 권한이 없다). 남은 차이는 셋뿐 — ① `--with-deps` 의 `sudo apt-get`(ubuntu-latest 에서는 표준) ② `actions/setup-node` 의 `node-version: '22'` 가 주는 22.x 가 engines(>=22.12.0)를 만족하는지(현재 22 최신은 만족) ③ 러너가 이 WSL 보다 느릴 때 wrangler dev 기동 180초·시험 240초 한도. 시험 전체가 여기서 1.5분이라 여유는 있다고 본다.
- 일부러 안 한 것: `push` 를 `branches: [main]` 으로 좁히지 않았다 — 과제서가 "push 와 pull_request 에서" 라고 했다. 같은 저장소 브랜치를 밀면 두 번 돈다(ideas.json 에 후보로 적었다). 브라우저 캐시(actions/cache)도 넣지 않았다 — 한 번 돌려 실제 소요를 본 뒤 판단할 일.
- 다음 역할이 조심할 것: 이 워크플로는 보호 경로라 사람 승인이 필요하다. 또 로컬에서 `npm test` 를 두 번째 돌릴 때 `tests/helpers.ts:94` 가 `me.workspaces[0].id` TypeError 로 죽는 기존 문제를 이번에도 맞았다(서버 재기동이 로컬 D1 을 비우는데 `.wrangler/test-sessions` 쿠키는 남아서) — 이 회차 변경과 무관하고, `.wrangler/test-state`·`.wrangler/test-sessions` 를 지우고 다시 돌리면 깨끗하게 통과한다.
- [러너 21:26] verify failed — 실패한 검증: npm test --silent (exit 1)
