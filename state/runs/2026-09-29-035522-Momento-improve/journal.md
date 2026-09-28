# 회차 노트 2026-09-29-035522-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:55] base pinned — main@08a1b9f
- [러너 03:55] autonomy release — 

## 정찰 노트
- 고른 이유: 보류 목록의 최우선 후보(SELF_* 거울)를 실제 코드로 검증했더니 SELF_PASSWORD 는 이미 AdminPage.tsx:3439 에서 처리돼 있고 SELF_ROLE·SELF_DISABLE 두 개만 남아 범위가 더 작아졌다. DataTable 계열 후보(rowKey 중복·페이저·디바운스)는 3회차 연속 같은 파일이어서 뒤로 미뤘고, 관측 근거가 없거나(rowKey) 측정이 선행(디바운스)이다.
- 확인한 것: admin.go:949·977·986 분기, AdminPage.tsx:3194·3285·3416-3427·3428-3438·3439, roleScope.ts 전체, AuthContext.tsx:11-12 + api/client.ts:1-8 의 `User.id`, roleScope.test.mjs 의 `.ts` 확장자 import 관례.
- 추측으로 적은 것: MUI 에서 `disabled` + `helperText` 의 가독성(브라우저 확인 필요), headless Chrome 하네스에 `/api/v1/me` 모킹이 필요하다는 점(2026-09-28 하네스를 재사용하되 users+me 두 응답이 필요).
- 구현자가 조심할 것: 활성화 체크박스를 통째로 disabled 하지 말고 `edit.active && !canDeactivate` 로 — 서버는 끄는 것만 막는다. 서버·internal/auth 는 손대지 말 것. prettier 를 게이트로 쓰지 말 것.
- 프로필은 2026-09-26 판이 지금 코드와 어긋나지 않아(14일 미경과) 새로 쓰지 않았다. 테스트는 돌리지 않았다(worktree 에 web/node_modules 없음, Postgres 없음).
- [러너 03:59] scout done — 사용자 편집 다이얼로그가 자기 자신의 역할 변경(SELF_ROLE)·계정 중지(SELF_DISABLE)를 애초에 제안하지 않게 

## 구현 노트
- 서버 `updateUser` 의 자기 계정 400 세 개(SELF_ROLE·SELF_DISABLE·SELF_PASSWORD)를 `roleScope.ts` 의 순수 함수 `selfAccountLimits` 하나로 모으고, 편집 다이얼로그의 권한 select(disabled + caption)·활성화 체크박스(끄는 것만 차단)·비밀번호 필드(기존 인라인 id 비교 대체)가 그것을 읽게 했다. 프로덕션 2파일 + 테스트 1파일, 서버는 손대지 않았다.
- 확신 없는 곳: caption 을 `mt={-1.5}` 로 붙여 select/체크박스 바로 아래에 오게 했다 — 1280×1000 headless 에서 두 줄 겹침은 없었지만 좁은 화면·다국어 폭은 보지 않았다. `selfAccountLimits` 의 `callerId !== undefined &&` 가드는 `targetId: string` 인 한 `callerId === targetId` 와 동작이 같아 돌연변이로 구별되지 않는다(의도를 적어 둔 것이지 테스트로 고정된 것은 아니다). `limits` 를 컴포넌트 최상단에서 `edit?.id ?? ""` 로 구하므로 사용자 id 가 빈 문자열인 배포가 있다면 다이얼로그가 닫혀 있을 때도 self 로 판정된다(실제 id 는 UUID 라 현실성 없음).
- 일부러 하지 않은 것: 편집 버튼의 `canAdministerRole` 게이트·생성 다이얼로그·역할 서열 함수는 그대로 뒀다. 서버 오류 코드의 한국어화는 같은 파일을 다시 넓게 건드리므로 다음 회차 후보로 넘겼다(ideas.json).
- 다음 역할이 조심할 것: 새 테스트 5건은 `web/test/roleScope.test.mjs` 안이고 DB 없이 `cd web && npm ci && npm test`(135건) 로 돈다. 브라우저 확인은 커밋에 없다 — 하네스를 /tmp/harness 에만 두고 dist 를 지웠으므로 재현하려면 dist 빌드와 `/api/v1/me`·`/api/v1/users` 모킹을 다시 세워야 한다.
- [러너 04:07] brief accepted — 채택 — 인용한 행 번호(admin.go 949·977·986, AdminPage.tsx 3194·3285·3416-3427·3428-3438·3439)와 `User.id`·`.ts` 확장자 import 관례가 모
- [러너 04:07] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 대상은 baa6292 한 커밋(로컬 `main` ref 가 낡아 `main...HEAD` 가 릴리즈 7개를 끌고 오므로 pinned base 08a1b9f 기준으로 봤다). admin.go:949·977·986 세 분기, Principal.ID 의 `json:"id"`, client.ts `User.id`, 저장 핸들러가 disabled 와 무관하게 state 의 role·active 를 보내는 것, 비밀번호 조건이 기존 `edit.id !== user?.id` 와 논리적으로 동치인 것을 각각 파일에서 확인. 실행: npm test 135/135, eslint 무경고, tsc -b + vite build 성공.
- 원장에 `- 실패 재현:` 줄이 없어 직접 재현했다 — 커밋의 roleScope.test.mjs 를 baa6292^ 의 roleScope.ts 와 /tmp 에서 함께 돌리니 export 부재로 실패한다. 새 테스트는 바뀐 경로를 지난다.
- 못 본 것: 브라우저 렌더링(캡션 `mt={-1.5}` 의 좁은 화면·다국어 폭), Go 테스트(변경에 Go 파일 없음), 통합 테스트(Postgres 없음).
- 승인이어도 남는 우려 — ① 테스트가 순수 함수만 고정하고 AdminPage 의 세 `disabled`/렌더 조건은 지워도 135건이 통과한다(이 저장소에 컴포넌트 테스트 수단 자체가 없음). ② `자기 판정은 id 만 보고 대소문자…` 테스트의 근거 주석이 틀렸다 — 서버는 `uuid.Parse` 후 UUID 값을 비교하므로 대문자 UUID 도 self 로 본다(Go 가 소문자 정규형만 내보내 도달 불가, 권한도 넓어지지 않음). 다음에 그 주석을 서버 동등성의 근거로 인용하지 말 것.
- 보안·개인정보 차단 사유 없음: 서버·auth·마이그레이션 무변경, 새 엔드포인트·의존성·비밀값·개인정보 수집 없음, 화면 제안 범위를 좁히기만 함. 거울이 fail-open(세션 미확정 시 전부 열림)인 것은 의도이며 서버가 정본 — 이 함수를 통제로 오해해 서버 검사를 줄이지 말 것. 릴리즈 노트는 "자기 계정 편집 시 권한·중지가 애초에 제안되지 않는다" 로 좁게 쓰는 것이 정확하다.
- [러너 04:11] review approved — 리뷰 승인 (risk=low)
- [러너 04:12] pr created — https://github.com/hkjang/Momento/pull/20
- [러너 04:17] ci passed — 검사 1개 모두 success
- [러너 04:17] merge done — baa6292
