# fix-summary — vibe-code PR #14 (bbee934)

- **CI 실패는 코드 결함이 아니다 — 고치지 않았다.** `check` job 이 **3초**에 죽었고 배정된 실패 로그 끝부분이 **비어 있다**(`steps=[]` 패턴). `npm ci`+`npm run check` 만 로컬에서 ~15초라 3초에는 스텝이 실행될 수 없다. 이 저장소에서 이미 관측된 GitHub Actions 과금/한도 실패("The job was not started because recent account payments have failed…", run 36518589343)와 같은 모양이다. `gh` 미인증으로 원격 annotation 은 이번에도 직접 못 읽었다 — 이 한 줄은 **추정**이다.
- **재현 시도(모두 exit 0):** `bbee934` 를 `/tmp/ci-repro` 로 **클린 클론**해 CI `check` job 3스텝을 그대로 돌렸다 — `npm ci`=0, `npm run check`=0, `node --check dist/extension.js && dist/extension.core.js`=0. CI 와의 유일한 환경 차이인 Node 라인 드리프트도 양쪽에서 배제했다: **20.19.2**(engines 핀)와 **20.20.2**(CI `node-version: 20`) 모두 11 files / 88 tests 통과. diff 는 `exitCode` 필드에 대한 순수 로직뿐이라(fs·날짜·로케일·네트워크·플랫폼 의존 없음) CI 에서만 깨질 기전이 없다.
- **고친 것은 하나, 커밋 `bf12461`(주석 전용, 5/1줄):** 비평 노트 우려 ① 이 맞다는 것을 직접 확인했다 — `journal-summary.ts:44` JSDoc 의 "이 값을 읽는 **모든** 경로가 이 술어를 쓴다" 는 거짓이다. `command-audit.ts:51/73` 이 같은 감사 JSONL 의 `d.exitCode` 를 직접 읽어 미상 줄에 `! 종료 undefined` 를 찍는 **네 번째 경로**로 남아 있다(grep 으로 src/ 전체 읽기 지점 확인). 주석을 "세는 세 경로" 로 좁히고 네 번째 경로를 명시했다. **동작 변경 0, 테스트 기대값 변경 0.**
- **수정 후 재검증:** Node 20.19.2 에서 `npm run check` exit 0(11 files / 88 tests), `node --check` 둘 다 exit 0.
- **여전히 미확인:** 원격 CI 상태·PR #14↔브랜치 대응·windows `package` job·Extension Host. CI 의 빨간 불은 이 브랜치에서 고칠 수 없다 — 재실행이나 과금 한도 확인이 필요하다.
