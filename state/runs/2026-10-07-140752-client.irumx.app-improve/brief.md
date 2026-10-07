- 과제: 수정 과제 — 요청 진행 알림 API 시험의 메일 판정을 벽시각에서 새 메일 ID로 전환 (가치 5 / 위험 2 / 작업량 S)
- 왜: 직전 릴리즈 검증은 tests/api-bulk.spec.ts:20의 작업 시작 메일 대기에서 실패해 118 passed / 1 failed / 3 did not run으로 끝났다. 실제 메일이 있어도 시각이 2초 넘게 뒤로 찍히면 현재 waitMail은 실패하므로, 기존 ID 기반 도우미로 새 메일 여부를 엄밀하게 판정해 발송 검증을 안정화한다.
- 수용 기준: 1) 실제 요청 API를 통해 범위 판단·작업 시작·납품·완료 알림 네 통을 확인하며 본문, 수신자, 중복 없음, 내부 변경 무알림, 알림 끔·가져온 요청 무알림의 기존 단정을 모두 유지한다. 2) 네 번의 대기와 전체 네 통 집계에서 Date.now()/m.at/2초 여유를 제거하고, 변경 직전의 ID 집합과 정확한 제목·수신자를 사용한다; 이미 있던 같은 제목 메일이 새 메일로 인정되지 않아야 한다. 3) 실제 업무 API를 거치는 api-bulk 전체 시험과 npm test --silent가 새 빌드·새 시험 서버에서 exit 0이고, 실패/skip/retry 증가로 통과시키지 않는다. 당시 메일 미발송이 확인되면 시각 가설을 철회하고 아래 차선 조사로 과제서를 수정한다.
- 건드릴 파일: tests/api-bulk.spec.ts: 첫 번째 '내 요청의 진행' 시험 — waitNewMail import, 각 알림 유발 동작 전 mailIds(p) 스냅샷, waitMail 네 곳 대체, 전체 집계의 시각 필터 제거; docs/qa.md: 시험 중 찾은 것 — 관찰한 실패·수정·검증 결과를 짧게 추가. tests/api.ts:mailIds/waitNewMail 및 tests/mock-resend.mjs는 기존 구현을 사용하며 기본적으로 수정하지 않는다. 프로덕션 파일 0개, 기본 수정 파일 2개.
- 검증 명령: 준비 npm ci --no-audit --no-fund; node --version; npm exec -- node --version; 필요 시 npx playwright install chromium; npm run check; npm run build; npm test -- --project=api tests/api-bulk.spec.ts; npm test --silent; git diff --check. 직전 명령의 exit와 passed/failed/did-not-run을 기록한다. 실행 때 8798/8799가 이번 빌드의 시험 서버인지 확인한다.
- 위험과 피할 것: auth/migrations/프로덕션 outbox·메일 정책·.github/workflows·외부 러너를 변경하지 않는다. 대기 시간을 늘리거나 시각 허용치를 넓히거나 실패 단정을 제거하거나 retry/skip을 추가하는 해결 금지. 이전 붙여넣기 수정 09e7133 및 혼합 입력 변경을 다시 구현/체리픽하지 않는다. 테스트의 기준을 '메일이 언젠가 있었음'으로 낮추지 않는다. 전체 메일 집계는 수신자 p의 ID 집합만으로 다른 수신자의 메일을 걸러서는 안 된다.
- 차선 후보: 실제 요청 상태 전이→notifyRequesters→processOutbox 미발송 원인 조사 — 해당 메일이 mock 기록에도 없다는 증거가 있을 때만 전환; 구현 전 범위를 다시 적고 6개 이내 프로덕션 파일 제한을 지킨다. unrelated 새 기능으로 전환하지 않는다.

확인한 근거와 한계
- base는 main@155a4ee. 현재 tests/kakao.spec.ts는 과거 30개 경계 시험이 없는 상태이고 requests.tsx에는 옛 300개 안내가 남아 있다. 직전 검증은 09e7133에서 수행됐으며 현 checkout과 시험 개수가 다를 수 있다. 122개라는 수치를 현재 목표 개수로 고정하지 않는다.
- ../2026-10-07-132809-client.irumx.app-improve/verify.json: 설치 확인 exit 0, npm test --silent exit 1(235초). verify.txt의 유일한 실패는 tests/api-bulk.spec.ts:20 → tests/api.ts:54 '메일 없음 ... /요청 #1 작업을 시작했습니다/'. 이어지는 같은 serial 묶음 세 시험이 실행되지 않았다. 직전 npm-test.log의 전체 성공 기록과 후속 verify 실패는 구분해야 한다. 두 차례 같은 실패라는 운영자 보고 중 독립적인 두 번째 실패 로그는 이번 정찰에서 확인하지 못했다.
- 저장소 .github/AGENTS.md/CLAUDE.md/별도 ROADMAP·TODO 파일은 검색 결과 없다. README와 docs/qa.md, 최근 git log -30(실제 12개), playwright.config.ts, scripts/db-local.mjs, scripts/deploy.mjs를 읽었다. TODO/FIXME 주석 검색 결과 없음. 문의 후속 계획은 이전 프로필 기준이고 이번에 명세 전체 재독은 하지 않았다.
- 실제 릴리즈 게이트는 /mnt/c/Users/USER/projects/aidev/bin/run.sh:run_verify(845행 이후). package.json의 test를 찾아 설치 확인→npm test --silent→build 순서로 실행하고 첫 실패에서 중단한다. 저장소 workflows YAML은 없다. scripts/deploy.mjs는 운영 배포 스크립트로 이번 실패를 발생시킨 단계가 아니며 실행하지 않는다. 검증 전에 로컬 빌드를 준비해야 한다는 기존 계약을 지킨다.
- tests/api.ts:waitMail은 performance.now로 대기하지만 메일 선택은 x.at >= since - 2000. mailIds/waitNewMail은 수신자별 ID 집합으로 판정한다. tests/api-bulk.spec.ts 앞쪽 네 대기·집계에는 t0가 남고 뒤쪽 무알림 시험에는 이미 ID 비교가 쓰인다. docs/qa.md:104~107은 WSL 벽시계 역행과 동일 유형 오탐을 기록한다.
- src/worker/routes/requests.ts의 상태 전이·notifyRequesters 호출 블록(491~554행), services/request-notify.ts:notifyRequesters, services/outbox.ts:enqueue/processOutbox를 읽었다. planned→in_progress는 알림을 만들며, 특정 ids 발송은 nextAttemptAt 시각 필터를 적용하지 않는다. 실제 실패 시점 outbox 상태와 mock 메일 원본은 미확인; 프로덕션 결함이 없다고 단정하지 않는다.
- 정찰에서 npm test --silent를 그대로 1회 실행: exit 1, ERR_MODULE_NOT_FOUND: @playwright/test. 현재 node_modules/dist가 없고 상위 /home/hkjang/node_modules/playwright가 선택된다. 이는 직전 118개 성공 이후 실패와 다른 준비 문제다. 정찰의 파일 쓰기 제한 때문에 설치·빌드·수정 후 전체 통과는 수행하지 않았다.
- 추가 진단: Node VM으로 실제 tests/api.ts를 TypeScript transform하여 실행하고 실제 tests/mock-resend.mjs를 8799에 잠시 기동했다. 상위 설치의 실제 playwright/test.mjs를 링크하고 POST /emails에 at=since-5000인 시험 메일을 기록하자 waitMail은 실패, waitNewMail은 성공, 이미 본 ID 집합으로 다시 대기하면 실패했다(exit 0). 시각 필터의 취약성만 증명한 격리 진단이며 업무 API 배선 또는 당시 시계 역행 증명이 아니다. 서버는 종료했고 파일 변경 없음. 첫 strip-only 시도는 parameter property 미지원으로 실패하여 transform 방식으로 다시 실행했다.

구현 순서와 검증 지점(사람 확인 불필요)
1. [대기] 환경 준비·원인 구분(7~10분). 위 설치/Node 확인 후 build를 수행하고 api-bulk 전체 파일을 실행한다. 실패하면 해당 수신자·요청 제목의 실제 mock 기록을 확인해 시각 탈락인지 미발송인지 구분한다. 민감 본문·주소를 영구 진단 로그에 덤프하지 말고 mail/outbox ID와 시각·상태만 남긴다. 우연히 통과하더라도 시각 취약성의 결정적 진단과 실제 업무 흐름 통과를 구분해 기록한다.
2. [대기] tests/api-bulk.spec.ts 수정(7~10분). 요청 생성 전에 전체 mails()의 ID 집합을 잡아 네 통 집계에 사용하고, 각 알림 동작 직전에는 해당 수신자의 mailIds(p)를 별도로 저장한다. 기존 고유 프로젝트 제목 필터·수신자·본문·네 통·무알림 단정을 보존한다. 검증: npm run check 및 npm test -- --project=api tests/api-bulk.spec.ts. 실패 시 다음 단계로 진행하지 않고 원인을 기록한다.
3. [대기] 전체 검증 및 기록(6~10분). 최신 빌드로 시험 서버를 다시 준비한 뒤 npm test --silent를 실행한다. docs/qa.md와 이 회차 ledger-entry.md에 실제 결과를 기록하고 git diff --check를 확인한다. 소스/설정 검사를 동작 증명의 대체물로 삼지 않는다.

대안 비교와 추정 근거
- 선택: 기존 ID 도우미 재사용(위 2파일). 이미 로그인·같은 시험 뒷부분에서 쓰는 계약을 앞부분에도 적용하며 프로덕션 기능을 바꾸지 않는다.
- 조건부 차선: 실제 발송 누락을 고친다. 미발송 증거가 있을 때 올바른 선택이며 현재는 실패 시점 상태가 없어 정답이라고 주장할 수 없다.
- 대규모 모든 waitMail 호출 전환은 후속 범위가 크고 이번 한 건에 불필요. 환경만 다시 실행하는 무변경 대안은 성공 기록 뒤 재실패했다는 사실을 해결하지 않는다. timeout/2초 여유 확대는 문제를 남기므로 기각.
- 가장 큰 가정: 이번 실패도 수신 메일의 시각 필터 탈락이라는 점. 미확인이며 첫 검증 지점에서 수정해야 한다.
- bottom-up 기준 20~30분 + 알려진 환경 변동 예비 5~10분 = 25~40분, 보통 환경에서 가능하다는 중간 확신(통계적 신뢰구간 아님). 전체 시험 과거 235초/4.2분을 유사 사례로 교차 확인했다. 의존성/Chromium 다운로드와 Node 경로 문제가 예비 사용 조건이며 범위 밖 프로덕션 결함에 쓸 관리 예비는 배정하지 않는다. 45분 안에 원인 분리가 안 되면 성공으로 기록하지 말고 과제서의 가정과 상태를 갱신한다.
- 적용한 스킬: pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration의 로컬 SKILL.md를 직접 읽음. 이 세션에 Skill 도구는 없음. estimating의 references/sources.md도 확인했으나 외부 추정 표준의 수치/보장을 인용하지 않고 현 저장소 관찰로만 추정했다.
