# 수리 요약 — 커밋 없음 (고칠 결함을 찾지 못함)

- **재현 실패**: 워크트리에 `npm ci` 후 CI 의 `check` 잡 명령을 그대로 돌렸다 — `npm run check` 는 typecheck exit 0, **19 files / 155 tests 전부 통과**, build 성공(자산 경고는 main 에도 있는 기존 것)이고, 이어지는 `node --check dist/extension.js && node --check dist/extension.core.js` 도 통과했다. CI 가 실패했다는 그 자리에서 로컬은 녹색이다.
- **테스트는 대상을 실제로 검증한다**: 프로덕션 파일만 `origin/main` 으로 되돌리니 새 테스트 2건이 반환값 AssertionError 로 실패했다(`go test ././...`+`go test ./...` 중복, `pytest` 통짜 폴백). 되돌리기를 풀자 13/13 통과. 빈 단언이 아니다.
- **이 PR 은 CI 를 건드릴 수 없다**: `git diff origin/main...HEAD` 는 `src/features/verification.ts`(2줄)와 `tests/unit/verification.test.ts` 뿐 — `.github/`, `scripts/`, `package.json`, `package-lock.json` 변경 0건.
- **실패 양상이 코드 실패와 맞지 않는다**: 두 워크플로가 **각 3초**에, 러너가 캡처한 실패 로그는 **양쪽 다 비어 있다**. `UI snapshots` 는 `npm ci`→`build`→pwsh VSIX 내려받기→`apt-get`→`playwright install chromium` 을 지나야 프로젝트 코드에 닿는데 3초로는 1단계도 못 끝낸다. 잡이 아예 시작되지 못한 계정·인프라 레벨 실패(결제/쿼터 등) 신호이며, 이 저장소에는 과거 결제 실패 전력이 있다.
- **그래서 커밋하지 않았다**: 녹색으로 만들 길은 워크플로를 고치는 것뿐인데 절대 규칙이 금지한다. 다만 **확증은 못 했다** — 이 환경에 `gh` 인증이 없고(비공개 저장소라 비인증 API 는 404) 원격 로그를 읽지 못했다. 결정적 확인은 `main` 의 최신 CI 도 같은 3초 실패인지 보는 것이며, 그렇다면 PR 무관이 확정된다.
