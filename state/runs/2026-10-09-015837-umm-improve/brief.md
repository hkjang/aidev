- 과제: dev·preview·e2e에서도 설치 누락을 기존 가드로 설명하기 (가치 2 / 위험 1 / 작업량 S)
- 왜: 설치 없는 현재 체크아웃에서 `npm --prefix web run dev`·`preview`는 `vite: not found`/exit 127로 끝나고, `e2e`는 상위 디렉터리 Playwright를 실행한 뒤 `@playwright/test` 누락 스택/exit 1로 끝나 설치 방법을 안내하지 않는다. 이미 검증된 `require-installed.mjs`를 세 npm 진입점에 연결하면 누락된 패키지·해석 기준·설치 명령을 즉시 알려 줄 수 있다.
- 수용 기준: 1) 설치 없는 실제 npm 진입점 세 곳에서 dev/preview는 vite, e2e는 @playwright/test를 명명하고 해당 package.json 경로와 `npm ci --prefix <root>`를 안내하며 exit 1로 종료한다. 2) 기존 `dev`·`preview`·`e2e` 명령 본문은 그대로이고 설치된 환경에서는 가드가 조용히 통과하여 실제 Vite/Playwright CLI까지 도달한다. 3) 실제 manifest와 가드를 복사한 미설치 fixture에서 npm을 자식 프로세스로 실행해 세 배선의 메시지와 종료를 증명하고, 실제 설치 환경의 정상 대조군도 통과한다.
- 건드릴 파일: `web/package.json:scripts` — `predev: node scripts/require-installed.mjs vite`, `prepreview: node scripts/require-installed.mjs vite`, `pree2e: node scripts/require-installed.mjs @playwright/test` 세 항목만 추가. `web/scripts/require-installed.test.mjs:runNpmScript/checkoutWithoutInstall`와 기존 it.each — 미설치 세 명령을 기존 표에 추가하고, 설치된 실제 npm CLI 도달 대조군을 보강. 프로덕션 1파일 + 테스트 1파일, 새 공용 모듈 없음.
- 검증 명령: 저장소 루트에서 `npm ci --prefix web` 선행 후 `npm --prefix web test -- scripts/require-installed.test.mjs`; `npm --prefix web test`; `npm --prefix web run typecheck`; `npm --prefix web run lint`; `npm --prefix web exec -- prettier --check scripts/require-installed.test.mjs`; `npm --prefix web run dev -- --help`; `npm --prefix web run preview -- --help`; `npm --prefix web run e2e -- --list`; `node web/scripts/check-i18n.mjs`; `./scripts/check-version.sh`. 일반 프런트 검증이 필요하면 설치 후 `make test-web` 사용(네트워크 audit·build 포함). --help/--list는 서버·브라우저를 시작하지 않는 CLI 배선 확인이며 실제 E2E 제품 검증을 했다고 쓰지 않는다.
- 위험과 피할 것: auth·migrations·workflows·Dockerfile·Makefile·engines·VERSION·락파일·루트 package.json·가드 본문·Node 래퍼는 변경하지 않는다. 설치를 자동 수행하거나 PATH/전역 설정을 바꾸지 않는다. 기존 fixture의 npm_* 제거와 node_modules PATH 제거를 보존한다. 실제 개발 서버를 테스트 안에서 계속 켜 두지 않는다. 파일 이름 통합(PR #167 회귀 기록), Go 게이트 확장, outline 수정은 이번 범위 밖이며 재제출하지 않는다.
- 차선 후보: `web/src/lib/edge-vocabulary.test.ts` 추가로 알려진 관계 6개·출처 6개의 ko/en 라벨과 로케일 전환 시 재조회 계약을 실제 EdgeRelation/EdgeOrigin·setLocale/translate로 검증 — 프로덕션 0파일, 가치 2 / 위험 1 / S. 미등록 서버 값이 실제 버그라는 주장은 금지(DB CHECK가 값 제한). 1순위가 이미 해결됐거나 정상 CLI를 막는 것으로 확인될 때만 선택한다.

근거와 검증 상태 (main@e54875c / VERSION 0.76.8)
- 초안 작성 뒤 실제 명령을 실행하여 위 실패를 확인했다. 특히 e2e는 exit 127이라는 이전 아이디어의 추측과 달랐다. 오류 스택은 `/home/hkjang/node_modules/playwright/`를 경유한다.
- `node web/scripts/require-installed.mjs vite`와 `node web/scripts/require-installed.mjs @playwright/test` 모두 정확한 누락 문장/exit 1을 출력했다. 새 파서나 새 패키지가 필요하지 않다.
- `web/scripts/require-installed.test.mjs`는 현재 typecheck/lint/build의 실제 npm 훅을 시험하며 설치된 경우의 조용한 성공을 직접 가드 호출로 확인한다. 기존 함수들을 재사용하되 새 정상 대조군은 npm 진입점까지 통과시킨다.
- 정찰의 `npm test --silent`는 vitest 미설치 안내/exit 1이었다. 설치 후 Vitest·typecheck·CLI --help/--list 성공 여부는 **미확인**이며 구현자가 위 명령으로 확인해야 한다. 정찰은 저장소에 설치나 코드 변경을 하지 않았다.
- `./scripts/check-version.sh` exit 0, `node web/scripts/check-i18n.mjs` exit 0(1060키). `make -n test-go`는 vet와 go test 두 줄만 출력했다.
- `.github/workflows/ci.yml`은 npm ci 후 프런트 8게이트를 돌리고 E2E는 `npx playwright test`를 직접 실행한다. 따라서 새 pree2e는 CI의 그 호출에는 적용되지 않는다. 이번 과제는 로컬 npm 명령의 진단 개선이다.

구현 순서와 체크포인트 (모두 미착수; 사람 확인 없이 각 단계 검증 후 진행)
1. 설치 후 기존 가드 테스트를 한 번 실행해 기준선을 확인한다. 실패가 있다면 새 변경 탓으로 처리하지 말고 원인을 기록한다.
2. 위 두 파일을 한 묶음으로 수정한다. 미설치 테스트의 출력에 패키지명·설치 명령이 있고 실제 exit 1인지 단언한다. 배선 세 줄이 없을 때 새 테스트가 실패하고, 복원하면 통과하는지 구현 작업 중 확인한다. 테스트 코드를 소스 문자열 검사로 대체하지 않는다.
3. 설치된 실제 npm의 dev/preview --help와 e2e --list를 실행한다. stdout에 실제 CLI 도움말/테스트 목록이 있고 비영 종료나 require-installed 오류가 없는지 확인한다. helper에 인자 전달을 추가할 경우 미설치 fixture 정리와 기존 timeout을 유지한다.
4. 위 전체 웹 시험·타입·포맷·버전 검증을 완료하고 실제 실행 결과와 미검증 한계를 기록한다. 변경 파일이 지정한 두 개를 넘으면 범위를 다시 좁힌다.

선택과 추정 근거
- 대안 A(선택): 기존 pre* 가드 재사용 — 새 런타임 동작을 만들지 않고 실제 재현된 진단 공백을 메운다.
- 대안 B: 모든 npm 명령을 공용 런처로 전환 — 향후 명령은 관리하기 쉽지만 이미 다른 계약인 Node 래퍼와 가드가 결합되어 이번 45분 범위를 넘는다.
- 대안 C: README에 설치 문장만 추가/현상 유지 — 코드 위험은 없지만 이미 설치 방법을 모른 채 실패한 터미널에는 안내가 나오지 않는다.
- 가장 큰 전제: 미설치 fixture에서 상위 설치와 npm 환경의 영향을 분리하는 기존 helper를 그대로 유지할 수 있다는 것. 실제 경로의 e2e와 fixture의 원래 실패 모양은 다를 수 있으므로 수정 전 오류 문구를 하나로 가정하지 않는다.
- 작업량은 bottom-up: 기준선·설치 5–10분, 세 배선·테스트 10–15분, 검증·기록 5–10분 = 기본 20–35분. 알려진 변동(설치/자식 npm 실행)에 contingency 5분을 한 번 더해 25–40분, 확신 중간의 공학적 예상 범위이며 통계적 보장은 아니다. 관리 예비는 0분; 외부 서비스/도커/새 기능으로 범위를 늘리지 않는다. 10월 7일 가드 작업과 범위를 유추 비교하면 새 가드 구현 없이 배선과 기존 시험만 늘리는 더 작은 과제다(당시 실측 소요시간은 없어 수치 유추는 하지 않음).
- 적용 스킬: pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration의 로컬 SKILL.md를 읽었다. Skill 호출 도구는 제공되지 않아 파일로 읽었으며, 추정은 저장소 근거에 따른 자체 판단이다.
