# PR 처리기 노트 2026-10-07-090652-Momento-shepherd — Momento PR #27
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-07-075834-Momento-improve)
# 회차 노트 2026-10-07-075834-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:58] base pinned — main@1ef81d6
- [러너 07:58] autonomy release — 

## 정찰 노트
- 보류 1순위였던 「보존 정책 저장 실패 한국어 안내」를 그대로 골랐다 — v0.34.54~57 이 네 번 통과시킨 패턴이고, 이번에 처음 열어 본 서버 핸들러 `putRetentionPolicy`(internal/httpapi/advanced_analytics.go:99-122)가 코드 네 개뿐이라 S 로 끝난다. 차선이던 「JSON Schema helperText」는 다섯 줄 자유 입력의 onBlur/디바운스 설계가 본체라 S 가 아니고, cidr 500 건은 Postgres 가 없어 또 재현 불가, 설정 저장(2413)은 9개 그룹을 덮어 M 이다.
- 운이 좋은 자리: `INVALID_RETENTION` 의 영문 다섯 문장(82·85·88·91·94행)이 화면의 다섯 라벨(2902·2911·2920·2935·2948)과 1:1 로 대응해서, 코드 하나를 메시지로 가르는 판정이 `describeUserError` 의 `PASSWORD_PROBLEM`(adminErrors.ts:37) 선례와 똑같은 모양이 된다.
- 추측으로 적은 것(과제서에 미확인으로 표시했다): `UNKNOWN_SITE`(404)가 화면에서 실제로 도달 가능한지, `aggregation_months` 빈 칸 → null 경로를 브라우저로 확인하지 않은 것, 현재 web 테스트 총 개수(`npm ci` 를 돌리지 않았다 — node_modules 없음).
- 구현자가 조심할 것: `RETENTION_SAVE_FAILED` 문구를 중립으로 둘 것(`ON CONFLICT(site_id) DO UPDATE` 라 '이미 등록' 은 거짓, DB 로 재현 불가). 입력 차단(min/max·disabled)은 이번 과제가 아니다 — 별 아이디어로 남겼다. 2957 Alert 한 줄만 바꾸고 2413·2510·3112 는 두기.
- 기각 하나 기록: 「Realtime (시간)」 칸을 없애거나 안 보내게 하는 것 — AdminPage.tsx:2941-2945 가 'Kept for API compatibility' 로 의도를 명시해 뒀고 서버가 계속 검증·저장한다. 의도된 설계를 되돌리는 변경이라 집지 않는다.
- [러너 08:02] scout done — 보존 정책 저장 실패를 서버의 영문 문장·pgx 원문 대신 한국어로 안내한다 (가치 2 / 위험 1 / 작업량 S)

## 구현 노트
- 바꾼 것: `adminErrors.ts` 에 `describeRetentionError` 를 새로 두고 `AdminPage.tsx` 의 보존 정책 Alert **한 줄만** `RetentionErrorAlert` 로 교체(+테스트 1파일, 프로덕션 2파일). `INVALID_RETENTION` 은 서버 문장에서 컬럼명을 읽어 다섯으로 가르고 각 안내가 그 칸의 화면 라벨을 「」 로 부른다.
- **확신 없는 곳**: (1) `RETENTION_FIELDS` 의 라벨 문자열과 `AdminPage.tsx` 의 `TextField label` 이 **글자 그대로 같아야 하는데 구조로 보장되지 않는다** — 테스트도 양쪽을 손으로 옮겨 적은 표라 둘이 동시에 바뀌면 못 잡는다(describeSiteError 의 「IANA 시간대」 가 이미 같은 방식이라 관례를 따랐다). 라벨을 고치는 회차는 이 상수도 함께 고칠 것. (2) `UNKNOWN_SITE`(404)가 화면에서 실제로 도달 가능한지는 **확인하지 않았다** — `RetentionAdmin` 은 `useSite()` 의 사이트로만 요청하므로 사이트가 지워진 직후에만 난다. 과제서 지시대로 코드만 받아 두고 문구는 원인을 단정하지 않는 쪽으로 뒀다. (3) 안내가 품은 범위 숫자는 `validateRetention` 과 지금 일치하지만 서버가 범위를 바꾸면 갈라진다 — 잡는 테스트가 없다(서버 문장을 테스트가 글자로 품고 있을 뿐).
- **일부러 하지 않은 것**: 서버(`advanced_analytics.go`)와 `validateRetention` 범위는 손대지 않았다(통합 테스트와 메시지 매칭이 동시에 흔들린다). 입력 차단(`min`/`max`/`disabled`)도 넣지 않았다 — 과제서가 금지했고 이번 과제는 '실패를 읽히게 한다' 하나다(helperText 검증은 ideas.json 에 신규 항목으로 남겼다). `RETENTION_SAVE_FAILED` 의 원인은 특정하지 않았다 — Postgres 가 없어 재현 불가이고 `ON CONFLICT(site_id) DO UPDATE` 라 '이미 등록' 은 거짓. 네 `describe*` 와 합치지 않았다.
- **다음 역할이 조심할 것**: `web/test/adminErrors.test.mjs` 는 DB 없이 돌지만 `npm ci` 가 선행해야 한다(worktree 에 node_modules 없음, 수 분). 브라우저 하네스는 `/tmp/pptr/verify-retention.mjs` 에만 있고 커밋하지 않았다 — 되돌림 확인에 쓰려면 `node verify-retention.mjs <dist경로>`. **그 화면에 브라우저 하네스를 쓸 때 함정 하나**: 칸을 채우면 `useUnsavedWarning`(components/useUnsavedWarning.ts:7)이 `beforeunload` 를 걸어 같은 탭의 재이동이 30초 타임아웃으로 죽는다 — 사례마다 새 탭을 열어야 한다(앱은 건드리지 않았다).
- [러너 08:13] brief accepted — 채택 — 인용한 행 번호(AdminPage.tsx 2957 Alert·79-84 import·2902·2911·2920·2935·2948 라벨 다섯·3359-3370 네 `*ErrorAlert`, advanced_analyt
- [러너 08:13] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 원장에 `- 실패 재현:` 줄이 없어 직접 세웠다 — `describeRetentionError` 를 변경 전 거동(`error.message` 항등 스텁)으로 바꿔 돌리니 새 테스트 10개 중 **6개가 떨어졌다**(라벨 지시·한국어 범위·영문 컬럼명 누출·pgx 원문·남은 코드 한국어화·UNKNOWN_SITE). 테스트는 바뀐 경로를 실제로 지난다. 검증 패치는 되돌렸고 트리는 깨끗하다.
- 변경 후 `npm test` 234개 통과, `eslint` 무경고, `tsc -b && vite build` 통과(청크 크기 경고는 main 과 동일). 구현자의 확신 없는 곳 세 개 중 (1) 라벨 다섯이 AdminPage.tsx 의 `TextField label` 과 글자 그대로 같음과 (3) 범위 숫자가 `validateRetention`(advanced_analytics.go:81-94)과 일치함을 대조로 확인했다. 컬럼명 부분문자열 충돌도 검사 — 다섯 문장이 각각 한 컬럼에만 걸려 `find()` 순서가 무관하다.
- **못 본 것**: 브라우저로 실제 DOM 을 띄우지 않았다(Alert 배선은 `tsc` 와 한 줄 diff 로만 확인). Postgres 가 없어 500 두 꼴은 합성 메시지로만 봤고, (2) `UNKNOWN_SITE` 의 화면 도달성은 구현자와 똑같이 미확인으로 남는다 — 문구가 원인을 단정하지 않으므로 도달하지 않아도 해롭지 않다.
- **승인이어도 남는 우려**: 라벨↔상수 결합이 구조로 보장되지 않고 테스트도 양쪽을 손으로 옮긴 표다 — 라벨만 고치는 회차는 `adminErrors.ts:389-418` 을 함께 고쳐야 하고, 빠뜨려도 아무 테스트가 잡지 못한다. 서버 범위가 바뀌어도 같은 방식으로 조용히 갈라진다.
- **릴리즈가 알아야 할 것**: `validateRetention` 이 첫 위반에서 멈추므로 두 칸이 동시에 틀리면 안내도 한 칸만 가리킨다(서버 거동 그대로, 결함 아님). `detail` caption 에 pgx 원문이 남는 것은 변경 전 본문에 떴던 것을 좁힌 것이고 admin 전용 화면 — 근본 원인인 `err.Error()` 노출(advanced_analytics.go:117)은 별 과제다. 검토 부서 차단 없음(security·legal 모두 공격 경로·신규 개인정보 처리 없음).
- [러너 08:17] review approved — 리뷰 승인 (risk=low)
- [러너 08:17] pr created — https://github.com/hkjang/Momento/pull/27
- [러너 08:21] ci failed — 성공이 아닌 검사: test=failure · 실패한 검사: ? 잡: test 

## 수리 노트
- 맞았던 지적: source-map-js@1.2.1 high 취약점이 npm audit을 막는다는 CI 보고를 재현했다. 틀렸다고 확인된 지적은 없다.
- 고친 방법: web/package-lock.json의 source-map-js만 1.2.2로 갱신(97e3063); 원래 잠금 파일 복원 시 같은 실패도 재확인했다. 기존 audit을 회귀 검증으로 사용했다.
- 검증: npm ci && npm audit && npm run lint && npm test && npm run build 종료 0; 취약점 0, 234개 통과·skip 0. 기존 청크 크기 경고만 남는다.
- 확신 없는 곳: 로컬 Node 22.23.1/npm 10.9.8로 검증했으며 CI Node 24와 Go·DB·SDK·Docker 단계는 재실행하지 않았다. 기존 구현의 라벨·서버 범위 결합 및 UNKNOWN_SITE 도달성은 이번 수리 범위 밖이다.

## 심사 노트
- 확인: 세 headcount 스킬 원문 적용, HEAD 97e3063의 4개 변경 파일·라벨·서버 범위·권한 경계·잠금 갱신 대조; 웹 ci/audit/lint/test/build 통과(취약점 0, 234개·skip 0).
- 실행: 실제 빌드·APIError·저장 버튼·Alert DOM 11개 통과; 같은 하네스를 임시 origin/main 빌드에 실행하니 변경 대상 9개 실패. 빈 Aggregation의 null 전송 확인.
- 못 본 것: HTTP 응답은 합성 서버로 제공; 실제 PostgreSQL 장애·사이트 삭제와 CI Node 24·Go·SDK·Docker는 재실행하지 않음. 라벨/범위 수동 동기화와 기존 pgx 원문 caption 노출은 비차단 참고.
- 권고: approve/merge, risk=low, blocking=[] — 출력 개선을 실행으로 확인했고 새 개인정보 처리·권한 확대·마이그레이션 없이 되돌릴 수 있음. 코드 수정 없음.
