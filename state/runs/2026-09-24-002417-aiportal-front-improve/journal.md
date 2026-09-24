# 회차 노트 2026-09-24-002417-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:24] base pinned — main@5d948b7
- [러너 00:24] autonomy release — 

## 정찰 노트
- 릴리즈(A)는 16회째 진입 조건 미충족 — 태그 0·version 0.0.0·CHANGELOG/VERSION/scripts/Makefile/.github 전무·.gitlab-ci.yml 12 job 모두 브랜치 조건(태그 규칙 0줄)을 직접 재확인했다. 저장소 안에 합법적 수단이 없어 무변경으로 두고 실행 과제 B 를 따로 적었다.
- B 로 「PopSimpleBot.vue:599 onCreate 의 'COM' 가드 누락」을 골랐다. 같은 파일의 다른 catch 3곳(261,295,365)과 형제 PopSimpleBotUpdate.vue:679 는 모두 가드를 가졌고 src 전체 93곳이 쓰는 관례라 비대칭이 명확하고 위험이 가장 낮다. tryUpdateAppIcon 의 빈 catch 는 기대 계약 미확인 + 기존 스펙이 현 동작을 고정해 제쳤고, loading/globalLoading 계열은 선행 조건 미확정이라 제쳤다.
- 추측이 아니라 확인한 것: 'COM' 대역 응답은 `{status:204, code:'BZ01'}`(simpleBotUpdateClose.spec.js:202-204 에서 실제로 도는 형태). 반면 400 대는 에러 핸들러로 빠져 `/app/simple` 제외 조건과 얽히므로 쓰지 말라고 과제서에 적었다.
- 미확인으로 남긴 것: 기준선 테스트 수(29파일 497테스트)는 직전 회차 기록이며 node_modules 가 비어 있어 이번에 직접 돌려 확인하지 못했다 — 구현자는 `npm ci` 후 기준선을 먼저 재확인할 것.
- 구현자가 조심할 것: `validateCreateApp()` 이 첨부 1개 이상과 LLM·임베딩 모델까지 요구해 '생성' 클릭이 onCreate 본문에 도달하지 못하면 테스트가 조용히 통과한다. **수정 전에 Red 가 실제로 나는지 먼저 확인할 것.** 소스 문자열 검사는 증거가 아니다.
- [러너 00:28] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 무엇을 왜: `PopSimpleBot.vue:599` onCreate 의 catch 에 `if (e == 'COM') return` 을 넣었다. 인터셉터가 이미 전역 Alert 을 띄우고 'COM' 으로 거절한 오류에 토스트가 겹쳐 뜨던 자리로, 같은 파일의 다른 catch 3곳과 형제 PopSimpleBotUpdate 는 이미 가드를 가진 비대칭이었다. 신규 스펙 6건(수정 전 1건 Red) + 변이 1건 6회 반복으로 인과 증명.
- **확신 없는 곳**: ①`npm ci` 직후 첫 `npx vitest run` 한 번이 497개 중 1개 실패했는데 어느 테스트인지 잡지 못했고 이후 15회 이상 재현되지 않았다(vite 캐시 삭제 후에도). 콜드 스타트 타이밍으로 추정하나 **미확인**이다 — 비평가는 CI 에서 재발하는지 봐 둘 것. ②내 스펙 변이 실행 중 딱 한 번 'HTTP 200 업무 실패' 케이스가 함께 실패했다. 토스트 대역이 `toasts` 변수를 참조로 잡아 늦은 토스트가 다음 케이스로 새는 구조라고 보고 `settle()` 을 50 마이크로태스크 × 2라운드로 늘렸으며, 이후 13회 연속 재현되지 않았다. 근본 해법(프록시 해제 순서)은 아이디어로만 남겼다.
- 일부러 하지 않은 것: `Sidemenu.vue:499` onDelete 의 같은 가드 누락 — 같은 결함 클래스이고 `pInf` 도 같은 인터셉터를 쓴다는 것까지 확인했으나, layout 컴포넌트 마운트 하네스가 없어 한 세션 범위를 넘는다. ideas.json 에 다음 1순위로 근거와 함께 남겼다. `tryUpdateAppIcon`/`tryUpdateLog` 의 빈 catch 는 기대 계약 미확인이라 제외.
- 릴리즈: 17회째 진입 조건 미충족으로 무변경. 버전 파일·태그·CHANGELOG·릴리즈 노트·원격 전송 일체 없음, `docs/RELEASE.md` 미수정.
- 다음 역할이 조심할 것: `tests/unit/simpleBotCreateCommonError.spec.js` 는 `/policy/extension`(accept 문자열)과 `/app/model`(LLM·임베딩) 대역이 있어야 폼이 유효해진다. 첫 케이스가 `/app/create/doc` + `/app/create` 요청이 실제로 나갔음을 단정하니, 폼 검증이 바뀌어 클릭이 onCreate 에 도달하지 못하면 그 케이스부터 빨갛게 된다 — 조용히 통과하지 않는다. `/app/create/doc` 이 `/app/create` 를 포함하므로 adapter 의 URL 분기 순서를 바꾸지 말 것.
- [러너 00:37] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 판정 approve. Red 를 말이 아니라 실행으로 확인했다 — 저장소를 /tmp/redcheck 로 복제(node_modules 하드링크)해 `PopSimpleBot.vue` 의 가드 2줄만 지우고 신규 스펙을 돌리니 'BZ01 + status!=200' 케이스가 `toasts` 에 '앱생성에 실패하였습니다.' 가 들어와 실패, 가드가 있으면 통과. 테스트가 대상 코드를 실제로 실행한다.
- 구현자의 두 '확신 없는 곳' 은 재현 실패: 전체 스위트 4회 전부 30파일 503테스트 통과, 신규 스펙 단독 8회 연속 통과. 콜드 스타트 1건은 미확인으로 남고 CI 에 테스트 단계가 없어 자동 재확인 경로도 없다.
- 못 본 것: `npm run build:dev` 는 돌리지 않았다(catch 블록 내부 2줄이라 SFC 컴파일은 테스트 마운트로 이미 통과). 인터셉터 전체 재검토도 하지 않고 'COM' 이 나가는 8개 지점만 읽었다.
- 승인이어도 남는 우려: interceptors.js:~305(error 핸들러 refresh 실패)의 `clearRefreshSubscribers()` 는 openAlert 없이 대기열을 'COM' 으로 거절하므로, 그 경로에서는 가드가 안내를 통째로 삼킨다. src 93곳 공통의 관례적 구멍이라 이 PR 의 결함은 아니지만 고치려면 인터셉터에서 끊어야 한다. `Sidemenu.vue:498-500` 의 같은 가드 누락도 그대로 남아 있음을 확인했다.
- 릴리즈: 무변경이 맞다(태그 0·version 0.0.0·CHANGELOG/VERSION/.github 전무). 릴리즈 노트에 쓸 것은 "심플봇 생성 실패 안내 중복 제거" 한 줄뿐.
- [러너 00:40] review approved — 리뷰 승인 (risk=low)
- [러너 00:40] pr created — https://github.com/hkjang/aiportal-front/pull/30
- [러너 00:40] ci passed — 검사 없음 — 정책으로 허용
- [러너 00:40] merge done — 6e77958
- [러너 00:43] release failed — 릴리즈 안 함: 릴리즈 진입 조건 미충족(18회째). 워크트리(HEAD 1a8cb85, git status --porcelain 출력 없음)에서 직접 재조회한 근거: (1) `git tag --sort=-creatordate` 0�
