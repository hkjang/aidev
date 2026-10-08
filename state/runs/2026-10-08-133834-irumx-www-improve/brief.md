- 과제: 상속된 객체 속성이 문의 유형 검증을 통과하는 오류 수정 (가치 4 / 위험 2 / 작업량 S)
- 왜: `src/lib/inquiry.ts:validate`가 `TYPE_LABELS[v.type]`의 참값 여부만 검사해 `constructor`·`__proto__` 같은 미등록 유형을 정상 문의 유형으로 인정한다. 허용 목록의 직접 소유 속성만 인정하면 브라우저와 Worker 양쪽에서 같은 오류 메시지로 이 입력을 거절하고, 잘못된 유형이 메일 처리 단계로 넘어가지 않는다.
- 수용 기준: 1) `validate`는 `constructor`, `__proto__`, `toString`, `hasOwnProperty`, 빈 문자열, 임의 미등록 문자열에 `type: '문의 유형을 골라 주세요.'`를 반환한다(나머지 입력이 유효한 상태, `needsReply=false/true` 모두). 2) `TYPE_LABELS`에 직접 등록된 여섯 유형은 모두 그대로 통과하며 이름·이메일·본문·동의 검증과 제목/본문 작성 결과는 바꾸지 않는다. 3) 단위 테스트가 수정 전 상속 속성 사례에서 실패하고 수정 후 통과하며, 기존 Playwright 문의 API 테스트에 `type='constructor', token=''` 요청을 추가해 400·`code='invalid'`·`fields.type`을 확인한다(기존 코드에서는 검증을 통과해 토큰 없음 403 분기로 진행하는 것이 코드상 예상이며 HTTP 실행은 미확인).
- 건드릴 파일: `src/lib/inquiry.ts:validate` — 유형 존재 검사만 `Object.hasOwn(TYPE_LABELS, v.type)` 또는 동등한 직접 소유 속성 검사로 변경. `tests/inquiry.test.mjs`(신규) — 실제 TS 모듈을 import해 위 허용/거절 사례와 두 모드를 Node 내장 테스트로 검증. `tests/contact-send.spec.ts: test.describe('문의 API')` — 기존 `good`, `post`, `freshIp`를 재사용하는 API 회귀 테스트 한 건. 합계 3개, 프로덕션 1개.
- 검증 명령: 저장소 루트에서 아래 순서로 실행한다. 새 단위 테스트 명령은 구현 후 사용하며, 정찰이 이미 실행한 재현 명령은 별도로 기록했다.
  1. `node --experimental-strip-types --test tests/inquiry.test.mjs` — 의존성·빌드·서버 없이 단위 검증. 이 환경 Node v22.23.1에서 같은 방식으로 실제 `inquiry.ts`를 import할 수 있음을 확인했다.
  2. `npm ci` → `npm run test:build` → `npx playwright install chromium` — 현재 저장소 README/package.json의 실행 전제. `test:build` 첫 단계에 `npm run build`(타입 검사·산출물 점검)가 포함된다.
  3. `npx playwright test tests/contact-send.spec.ts --project=desktop --grep '문의 API'` → `npm test --silent` — 국소 API 검증 후 기존 전체 검증. 새 테스트 단독 명령만으로 전체 통과를 주장하지 않는다.
- 위험과 피할 것: `worker/`, `worker/svix.ts`, `wrangler.jsonc`, 인증·메일 관문·CSP·배포·워크플로는 변경하지 않는다. `TYPE_LABELS`를 null-prototype 객체로 바꾸거나 여섯 유형을 별도 배열에 중복 선언하지 않는다. `composeSubject`, `composeText`, `mailtoHref`까지 일반화하지 않는다. 의존성·package.json·Playwright 설정·pretest·포트 자동 할당·과거 커밋 체리픽은 범위 밖이며 기존 단정·skip·timeout을 완화하지 않는다. 현 main의 서버는 8820~8823 고정·재사용 금지이고 pretest가 없으므로 먼저 테스트 빌드가 필요하다. 전체 환경 실패가 나면 원인과 실행 로그를 기록하고, 이번 수정이 해결했다고 쓰지 않는다.
- 차선 후보: 빌드된 서비스 페이지의 사이트맵 누락 자동 감지 (가치 3 / 위험 1 / 작업량 S) — 1순위가 구현 시 이미 해결된 경우에만 선택. `scripts/verify-build.mjs`의 `htmlFiles`/`walk` 결과에서 `services/*.html` 경로를 얻어 기존 수동 필수 경로 검사에 추가한다. `tests/site.spec.ts:PAGES`와 합치지 말고, 새 서비스 누락 실패·정상 포함 성공을 증명한다. 현재 17개 필수 경로에 없는 서비스 누락을 놓치는 것은 정찰의 메모리 실행으로 확인했다.

범위·근거 (HEAD 94eb01f)
- 최초 초안은 사이트맵 감지였고, 실제 함수 실행에서 더 직접적인 문의 유형 오류를 발견해 이 최종 과제서로 교체했다. 이 작업은 입력 허용 목록 검증 수정이며, 객체 프로토타입 오염이나 메일 발송 성공을 재현했다는 뜻은 아니다.
- `src/lib/inquiry.ts`의 TYPE_LABELS/validate/composeSubject/composeText/mailtoHref, `worker/index.ts:handleInquiry`, `tests/contact-send.spec.ts`의 문의 API 블록을 직접 읽었다. Worker는 `validate(inquiry, true)` 오류를 400으로 반환한 다음에야 토큰을 검사한다.
- 정찰 재현: `node --experimental-strip-types --test /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-08-133834-irumx-www-improve/assets/scout-inquiry-repro.test.mjs` → 2 통과·2 실패·exit 1. 정상 여섯 유형과 일반 미등록 유형은 기대대로이고 constructor·__proto__만 type 오류가 undefined라 실패했다. 재현 파일의 import는 현재 작업 트리 절대 경로이므로 영구 테스트에는 `../src/lib/inquiry.ts` 상대 경로를 쓴다.
- 현재 node_modules와 dist 3벌이 없다. 의존성 설치·빌드·HTTP/e2e는 정찰에서 실행하지 않았으며 소요와 통과 여부는 미확인이다. `node --check scripts/verify-build.mjs`는 통과했다.

실행 계획·체크포인트 (구현자는 진행 상태를 이 항목에 갱신)
1. [완료: 정찰 2 통과·2 실패, 영구 단위 16 통과·8 실패; assets/unit-red.log] 재현 파일을 실행하고, tests/inquiry.test.mjs에 같은 계약의 영구 테스트를 작성한다. proof: 위 단위 명령에서 상속 속성 입력이 실패하는지 확인. 체크포인트: 사람 승인 없이 구현자 확인; 예상과 다르면 과제서부터 갱신.
2. [완료: 단위 24 통과; 원래 조건 재적용 시 동일 8 실패 후 수정 복원·24 통과] validate의 한 조건을 직접 소유 속성 검사로 바꾸고 단위 검증을 통과시킨다. proof: 위 단위 명령 exit 0. 체크포인트: 기존 여섯 유형·두 모드가 보존된 후 다음 단계.
3. [완료: npm ci/test:build/Chromium 준비 exit 0; API 7 passed, 전체 78 passed·4 기존 skipped; caf9e01] 문의 API 블록에 토큰 없는 미등록 유형 회귀 한 건을 추가한다. 유효한 good() 나머지 필드와 freshIp를 유지해 외부 Turnstile 호출 없이 400 분기로 끝나게 한다. proof: 준비 명령 후 국소 Playwright 명령 및 전체 npm test. 체크포인트: 결과·선재 실패를 구분해 기록하고 비평 단계로 인계; 자동 승인/릴리즈는 정찰 범위 아님.

선택 비교·가정·견적
- 선택지 A(채택): validate의 직접 소유 속성 검사 — 실제 실패가 확인됐고 프로덕션 1개 파일에 국한된다. 선택지 B: TYPE_LABELS 자료구조/모든 label 접근을 개편 — 직접 호출되는 문장 함수까지 계약이 넓어져 이번 범위에서는 제외. 선택지 C: 테스트만 추가하고 현상 유지 — 실패를 알릴 수 있지만 미등록 유형을 계속 허용하므로 제외. 사이트맵 후보는 위험이 더 낮지만 미래의 누락 감지이며 현재 사용자 입력 결함보다 우선순위가 낮다.
- 핵심 가정: TYPE_LABELS의 직접 등록된 여섯 키가 허용 유형의 전부다. 정의와 기존 UI/API 사용을 확인했으며, 상속 속성을 일부러 허용해야 하는 요구는 발견하지 못했다.
- 상향식 추정: 재현·단위 사례 5~8분, 검증 조건 수정 2~4분, API 사례 4~6분, 준비·빌드·검증 8~17분 = 기본 19~35분. 알려진 변동(콜드 빌드·브라우저 준비)에 5~10분 예비를 따로 두어 총 24~45분을 계획한다. 신뢰는 중간 이하이며 통계적 보장 아님; 환경 다운로드/전체 검증이 길어지면 45분을 넘을 수 있으므로 첫 설치·빌드 후 재산정한다. 테스트 범위를 줄여 시간에 맞추지 않는다.
- 과거 유사 사례는 검증이 37초였던 기록과 이미지 3벌 빌드가 3~5분 이상 걸린 기록이 함께 있어 단일 속도를 적용할 수 없다. 이번 추정은 미실측이며 외부 서비스 대기·기존 환경 수리는 제외한다. 새로운 범위의 관리 예비는 별도로 배정받지 않았고, 발생하면 다음 과제로 남긴다.
- 적용 스킬: 전용 Skill 도구가 노출되지 않아 로컬 [estimating-and-contingency](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md), [implementation-planning](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md), [solution-exploration](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md)를 읽어 대안 비교·범위·단계별 증명·체크포인트·가정과 예비 분리를 적용했다. estimating 스킬의 references/sources.md도 읽었으며 위 시간은 외부 기준 인용이 아닌 이번 코드와 제공 기록에 근거한 주관적 추정이다.
