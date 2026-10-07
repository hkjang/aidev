- 과제: Markdown 내부 링크의 역슬래시 우회 차단 (가치 4 / 위험 1 / 작업량 S)
- 왜: `apps/web/src/components/Markdown.tsx:14 safeHref()`는 `/\example.org/path`를 내부 링크로 허용하지만 URL 해석 결과는 외부 도메인이어서, 실제 렌더링에서도 외부 링크에 필요한 `target`·`rel`이 빠진다. AI 답변과 문서 화면이 공유하는 이 검사만 좁히면 링크 문법을 늘리지 않고 기존 외부 이동 정책을 지킬 수 있다.
- 수용 기준:
  1) Markdown 입력 `[링크](/\example.org/path)` 및 슬래시·역슬래시 변형이 클릭 가능한 내부 anchor가 되지 않고, 기존 거절 동작처럼 링크 라벨을 텍스트로 표시한다. 단순 `//example.org/path`도 계속 거절한다.
  2) `/w/demo/tasks?status=todo#list` 같은 정상 내부 경로는 href를 유지하고 외부 링크 속성을 붙이지 않는다. `https://example.org/path`, `http://example.org/path`는 기존처럼 `target="_blank"`, `rel="noopener noreferrer nofollow"`를 유지한다. `javascript:`, `data:`는 계속 거절한다.
  3) 새 시험은 실제 `Markdown` export를 import하고 React `createElement` + `react-dom/server`의 `renderToStaticMarkup`으로 최종 HTML을 검사한다. 결함 입력 시험은 수정 전 실패·수정 후 통과해야 하고, 정상 링크/위험 스킴 회귀 시험도 통과한다. 구현 소스를 정규식으로 검사하거나 `safeHref` 사본으로 시험하지 않는다.
- 건드릴 파일:
  - `apps/web/src/components/Markdown.tsx:safeHref()` — 내부 경로 분기에 원문 역슬래시 우회 거절을 추가한다. 우선 내부 경로에 원문 역슬래시가 있으면 null을 반환하는 작은 변경을 검토하고, 나머지 URL 처리·inline·블록 파서는 보존한다. percent-decode나 광범위 URL 정규화는 추가하지 않는다.
  - `apps/web/src/components/Markdown.test.ts` (신규) — Vitest의 기존 `apps/**/*.test.ts` 패턴에 맞춰 실제 컴포넌트 출력 시험. 테스트 파일은 JSX 없이 `createElement`를 써도 된다. 프로덕션 1파일 + 시험 1파일로 제한한다.
- 검증 명령: 저장소 루트에서 의존성 준비 후 순서대로 실행한다.
  ```bash
  npm ci --no-audit --fund=false
  npm test -- apps/web/src/components/Markdown.test.ts
  npm run check
  npm run build
  ```
  `npm ci`는 구현자 환경 준비용이며 정찰에서는 저장소 쓰기 금지 때문에 실행하지 않았다. root package.json의 `test=vitest run`, `check=lint+typecheck+test`, `build=web+server`와 vitest.config.ts의 시험 탐색 패턴을 직접 확인했다. 현 체크아웃의 `npm test -- --reporter=dot`는 의존성 부재로 `vitest: not found`(127)였으므로 전체 시험 성공을 주장하지 않는다. DB·OAuth·실제 브라우저는 이 단위 회귀 시험에 필요하지 않다.
- 위험과 피할 것: 인증·세션·migrations·workflows·배포 설정·재무 로직·package-lock을 변경하지 않는다. 다른 Markdown 파서 통합, 지원 문법 확대, sanitizer 의존성 도입, raw HTML 렌더링을 하지 않는다. 이 결함은 외부 이동 분류/속성 우회로 확인했으며 XSS 실행이나 계정 탈취로 과장하지 않는다. 역슬래시가 있는 상대 링크를 의도적으로 쓴 저장 문서의 존재는 미확인이다. 기존 링크의 실패 표시인 span을 유지한다.
- 차선 후보: 달력 날짜 검증의 윤년·월말 회귀 시험 보강 (가치 3 / 위험 1 / 작업량 S) — 1순위 결함이 실제로 재현되지 않을 때만 채택. `packages/core/src/util/validate.ts:v.day`, `parseOne`의 실제 export를 사용해 신규 `packages/core/src/__tests__/validate.test.ts`에서 2024-02-29 허용, 2023-02-29·2026-04-31·형식 오류 거절, 원래 필드 이름 유지 여부를 검증한다. 현재 날짜 검증 자체는 왕복 비교를 이미 수행하므로 프로덕션 코드는 바꾸지 않는다. `npm test -- packages/core/src/__tests__/validate.test.ts`와 `npm run check`로 검증한다.

확인 근거와 한계
- 기준 커밋 `022da5b`. `git log -30`에서 실제 이력은 8건이며 최근 `c86e672`가 에이전트 답변 Markdown 렌더링을 추가했다.
- `apps/web/src/routes/w/ai/Agent.tsx:438`, `apps/web/src/routes/w/docs/DocumentDetail.tsx:76`에서 같은 Markdown 컴포넌트를 사용함을 확인했다(읽기 전용, 변경 대상 아님).
- Node 22.23.1에서 `new URL('/\\example.org/path', 'https://biz.irumx.app').href`는 `https://example.org/path`였다. 현재 실제 Markdown 소스를 TypeScript로 메모리에서 변환해 React로 렌더링한 결과는 `<a href="/\example.org/path">링크</a>`였고 target/rel이 없었다. 정찰의 이 실행에는 같은 기준 커밋의 주 작업 트리 `/mnt/c/Users/USER/projects/biz.irumx.app/node_modules`에 있던 TypeScript/React를 읽기 전용으로 사용했다. 검사할 컴포넌트는 이번 체크아웃에서 읽었으며 사본 구현이나 대역은 사용하지 않았다.
- 실제 브라우저 클릭 시험 및 CI 전체 통과는 미확인이다. 전체 시험 설치 장애가 나면 기준 실패로 구분하고 우회 목적으로 설정/의존성을 바꾸지 않는다.

실행 순서와 체크포인트 (각 단계 상태: 대기)
1. 위 두 소비 경로 및 컴포넌트 계약을 확인하고 렌더링 회귀 시험을 작성한다. 증명: `npm test -- apps/web/src/components/Markdown.test.ts`에서 우회 입력만 의도대로 실패하는지 확인한다. 사람 승인 체크포인트 없음; 의도하지 않은 설정 오류면 먼저 시험 구성을 바로잡는다.
2. `safeHref`의 내부 링크 허용 조건만 좁힌다. 증명: 같은 명령의 수정 후 통과 및 정상 http(s)/내부 링크 속성 비교. 사람 승인 체크포인트 없음; 의미가 예상과 다르면 과제서를 수정하고 범위를 늘리지 않는다.
3. `npm run check`, `npm run build`와 `git diff --stat`로 회귀·파일 범위를 확인한다. 사람 승인 체크포인트 없음; 구현자는 실행 결과와 미확인을 회차 노트에 남긴다.

대안 비교 (solution-exploration)
- 선택: 내부 경로의 원문 역슬래시를 거절. 변경량과 정책 변화가 가장 작으며 라벨 표시 계약을 그대로 쓴다.
- 보류: 모든 링크를 기준 URL로 해석해 외부/내부를 재분류. 상대 경로·기준 origin 계약까지 넓어져 이번 한 결함에 불필요하다.
- 보류: 범용 Markdown/링크 라이브러리 도입. 문법 호환성과 의존성 검토가 추가되어 45분 범위를 벗어난다.
- 현상 유지: 외부 링크는 원래 허용되므로 심각도는 제한적이지만, 현재 주석·속성 정책과 실제 출력이 어긋나는 재현 사례가 있어 선택하지 않는다.
- 핵심 가정: 역슬래시 경로를 정당한 내부 링크로 지원할 요구가 없다. 기존 스킬 지침처럼 파서를 통합하거나 허용 문법을 넓히지 않고 좁혀 해결한다.

작업량 근거와 예비 시간 (estimating-and-contingency)
- Bottom-up 추정: 회귀 시험/실패 확인 8~12분 + 조건 수정 3~5분 + 전체 check/build·결과 기록 10~15분 = 기본 21~32분.
- 알려진 변동에 대한 contingency: JSX 변환·시험 탐색 확인에 5분, 합계 26~37분. 비슷한 과거 구현의 시간 자료가 없어 유사/모수 추정은 하지 않았다. 이는 코드 규모에 근거한 중간 신뢰의 주관적 범위이며 통계적 성공확률은 미확인이다.
- Management reserve는 별도로 배정하지 않는다. 45분 제한에서 설치 장애·새로운 문법 요구가 발견되면 범위를 추가하지 말고 장애를 기록한다. 기본 작업에 미리 padding을 넣고 다시 예비 시간을 더하지 않았다.
- 적용 스킬: `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`, `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md`, `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md`를 읽었다. 전용 Skill 도구는 제공되지 않아 파일로 읽었다. PMO sources.md도 확인했으며 외부 기관의 추정 수치·확률을 인용하지 않았다.
