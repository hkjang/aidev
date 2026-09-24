# 회차 노트 2026-09-23-202418-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:24] base pinned — main@c179034
- [러너 20:24] autonomy release — 

## 구현 노트
- `createSharedApp`(src/utils/shareAppService.js)이 모든 예외를 잡아 `{ok:false}` 로 바꾸던 것을 제거해, 호출부 4곳이 이미 가지고 있던 `catch (e) { if (e == 'COM') return; ... }` 분기를 살렸다. 그 전에는 세션 만료·BZ01·통신 오류가 전부 '권한이 없어 설치가 불가합니다' 로 안내됐다.
- 확신 없는 곳: 마운트로 실제 검증한 호출부는 `HeaderShare.vue` 하나다. `SupportList.vue:374`·`InfoSearchList.vue:277`·`MyAgentList.vue:409` 는 **읽어서** 동일한 try/catch(+`finally`로 `shareAppLoading` 해제) 패턴임을 확인했을 뿐 실행 검증은 하지 않았다 — 비평가는 여기를 먼저 보라.
- 확신 없는 곳2: 새 스펙은 `vi.stubEnv` 로 `VITE_BACKEND_API_TARGET` 을 넣고 `axios.defaults.adapter` 를 교체한다. 전역 axios 기본값을 건드리므로 `afterEach` 에서 지우지만, 이 파일과 병렬로 도는 다른 스펙이 실제 axios 를 쓰게 되면 간섭 가능성이 있다(현재는 없다).
- 일부러 하지 않은 것: `HeaderShare.getList` 의 `if(!isSuccess())`(인자 없는 호출) 오타와 `onShareAppSubmit` 이 `isLoading` 을 true 로 켜지 않는 문제는 관측 가능한 동작 변화가 없거나 별개 범위라 손대지 않았다. 실패 응답의 서버 메시지(`result.message`)를 화면이 쓰도록 바꾸는 것도 제품 문구 결정이라 제외했다.
- 다음 역할 주의: 새 스펙은 앱 라우터 대신 빈 메모리 라우터를 쓴다. 실제 `@/router` 를 설치하면 전역 가드의 SSO 재조회가 `writeUser({})` 로 사용자 정보를 지워 테스트가 간헐 실패한다(실제로 겪었다).
- 릴리즈 과제(13회째)는 무변경. 워크플로 완화 금지와 "릴리즈는 이 세션의 일이 아니다" 가 동시에 걸려 저장소 안에 합법적 수단이 없다. 상세는 ledger-entry.md.
- [러너 20:34] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 확인: 구현자가 미검증으로 남긴 호출부 3곳(SupportList.vue:374, InfoSearchList.vue:277, MyAgentList.vue:409)을 열어 try/catch(`e=='COM'` 조기 반환)+`finally` 해제가 모두 있음을 확인 — 예외 전파로 인한 스피너 누수·미처리 거절 없음.
- 확인: `shareAppService.js` 를 main 판으로 되돌려 새 스펙을 실행 → 6개 중 3개 실패. 테스트가 변경을 실제로 잡는다. 전체 27파일 477테스트 통과(main 26/471 + 신규 6).
- 못 본 것: 실제 백엔드 응답 형태는 검증하지 않았다(adapter 대역 기준). 프로필의 기준선 "23파일 452테스트" 는 낡았으니 다음 회차가 갱신하라.
- 승인이어도 남는 우려(릴리즈 노트): 호출부 4곳이 `result.message` 를 버리므로 '필수 값 누락'·서버 실패 메시지는 아직 '권한이 없어 설치가 불가합니다' 로 표시된다. 공통오류/통신예외 구간만 고쳐졌다.
- 테스트 위생: spec afterEach 가 `axios.defaults.adapter` 를 restore 없이 `delete` 한다 — 현재 파일별 격리라 무해하나 원래 값 보관이 안전. 판정 approve / risk low / blocking 없음.
- [러너 20:37] review approved — 리뷰 승인 (risk=low)
- [러너 20:37] pr created — https://github.com/hkjang/aiportal-front/pull/27
- [러너 20:37] ci passed — 검사 없음 — 정책으로 허용
- [러너 20:38] merge done — fbca18c
- [러너 20:39] release failed — 릴리즈 안 함: 버전 증가 관례를 결정할 근거가 저장소에 없어 릴리즈를 수행할 수 없음(14회째 동일 사유, 이번 회차 직접 재확인). 근거: git tag 0개; git 
