# 회차 노트 2026-09-24-030426-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:04] base pinned — main@0031fde
- [러너 03:04] autonomy release — 

## 정찰 노트
- A(릴리즈)는 HEAD 0031fde 에서 직접 재조회해 19회째 무변경으로 두었다(고칠 워크플로 파일이 저장소에 없고 실패 지점이 워크트리 밖). 실행 대상은 B.
- B 로 FavoriteList 의 빈 설정 팝업을 골랐다. 보류 목록의 다른 pending(getHistoryList bare catch, tryUpdateAppIcon, loading.vue 타이머)은 전부 "기대 계약 미확인" 이 선행이라 이번에도 제쳤는데, B 는 형제 4곳이 전부 :actions + @action 으로 통일돼 있어 기대 계약이 코드에 이미 확정돼 있다.
- 추측으로 적은 것: copyAppId 의 import 경로(미확인 — ServiceList 의 import 줄 복사 지시), jsdom 에서 SwiperList 가 문제없이 마운트되는지(미확인 — FavoriteList 마운트 하네스 전례 없음).
- 조심할 것: mode:'favorite' 정책이 share 도 돌려주므로 핸들러 없이 버튼만 렌더하면 죽은 버튼을 새로 만드는 셈이다(수용 기준 3). buttonSettingPolicy.js 의 common 을 고치면 형제 화면까지 번진다.
- [러너 03:09] scout done — 즐겨찾기 화면의 카드 설정(더보기) 팝업이 **빈 상자로 뜨고 아무 동작도 하지 않는** 문제 수정 (가치 4 / 

## 구현 노트
- 무엇/왜: `Sidemenu.getHistoryList`(:416) 의 bare catch 와 `!isSuccess` 무음 처리를 고쳐, 조회 실패가 Nodata 의 '최근 대화 기록이 없습니다' 라는 사실 진술과 섞이지 않게 했다. 문구·형태는 같은 파일 형제 `fetchAppList`(:208/:233) 를 그대로 따랐고 `'COM'` 은 전역 Alert 과 겹치지 않게 조용히 넘긴다.
- **과제서 B 는 기각했다 — 비평가가 여기를 먼저 보라**: `FavoriteList.mapAppToCard:56` 의 `showMore:false` 때문에 더보기 버튼(`button.btn-ico.ico-more`)이 **아예 렌더되지 않는다**(실제 마운트 스펙으로 0개 확인 후 그 탐색 스펙은 삭제). 과제서가 전제한 '빈 팝업' 은 재현 불가이고, `:actions` 만 붙이면 관측 가능한 변화가 0, 버튼까지 노출하면 승인 없는 기능 추가다.
- **확신 없는 곳**: (1) 사이드메뉴가 보조 UI 라 조회 실패를 토스트로 알리는 것이 옳은지는 여전히 제품 판단이다 — 다만 같은 파일 같은 트리거(목록 자동 조회)인 `fetchAppList` 가 이미 토스트를 쓰고 있어 문구를 새로 발명하진 않았다. (2) `getHistoryList` 호출부 5곳(:350 앱 펼치기, :498 삭제 후, :565 저장 후, :581/:641 세션 스크롤) 중 실제 DOM 으로 통과시킨 것은 :350 뿐이다 — 나머지는 토스트 빈도만 달라질 뿐 분기는 같다.
- 일부러 안 한 것: 실패/빈 결과를 구분하는 **Nodata 문구 변경**은 넣지 않았다(과제서가 '문구는 기존 것 재사용' 지시). `historyMap[appId] = []` 로 비우는 기존 동작도 유지했고, `fetchAppList` 의 `apiFailed` 같은 상태는 이력에 페이지네이션이 없어 불필요하다.
- 다음 역할 주의: 신규 스펙은 `vi.stubEnv` 로 `VITE_BACKEND_API_TARGET`/`VITE_PYTHON_API_TARGET` 을 둘 다 채워야 돈다(비우면 `getPApi()` 가 던져 이번에 고친 catch 로 빠진다). `npm ci` 선행 필수(node_modules 비어 있었음). 기준선은 이제 **32파일 516테스트**.
- [러너 03:16] brief fallback — 차선 — A(릴리즈)는 지시대로 무변경. B(즐겨찾기 ButtonSetting)는 **근거가 지금 코드와 맞지 않아 기각**했다: 과제서가 인�
- [러너 03:16] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 확인함: 4줄 변경이 형제 `fetchAppList`(:211,:233) 와 대칭인지 눈으로 대조, `src/` 만 main 으로 되돌려 신규 7건 중 4건 실패 → 복구 후 전건 통과(전체 32파일 516테스트). BZ01/204 케이스가 실제 전역 Alert 을 관측하므로 'COM' 가드도 진짜 경로로 고정된다. **approve, risk low, 차단 없음**(인증·인가·식별자·비밀값·개인정보 변경 0, ToastManager 에 v-html 없음).
- 못 본 것: 구현자가 적은 대로 호출부 5곳 중 :350 외에는 DOM 으로 안 통과시켰다. 다만 :569 watch 는 `bumpHistoryRefresh` 호출부가 src/ 전체 0건이라 발화 자체를 안 하고, :502 는 아래 우려로 남긴다.
- 승인이어도 남는 우려 ①: `:502`(삭제 성공 후 재조회) 는 "대화가 삭제되었습니다" 뒤에 조회 실패 토스트가 연달아 뜰 수 있다. 전역 Alert 겹침이 아니고 오안내도 아니라 거절하지 않았으나 새로 생긴 가시적 변화다.
- 우려 ②: `getAccessToken()` 빈 값 경로(`Sidemenu.vue:396-399`)는 여전히 무음으로 `[]` 라, 이번에 고친 '실패가 기록 없음으로 감춰짐' 이 한 갈래 남는다 — 다음 회차 후보.
- 기준선 정정: 프로필의 '30파일 503테스트' 는 stale. main(0031fde)=31파일 509테스트, 이번 HEAD=32파일 516테스트.
- [러너 03:19] review approved — 리뷰 승인 (risk=low)
- [러너 03:19] pr created — https://github.com/hkjang/aiportal-front/pull/32
- [러너 03:19] ci passed — 검사 없음 — 정책으로 허용
- [러너 03:19] merge done — ca204b0
- [러너 03:21] release failed — 릴리즈 안 함: 다음 버전을 결정할 입력이 저장소에 없어 릴리즈 진입 조건을 충족하지 못했다(20회째 동일 원인). 이번 회차 워크트리 HEAD fb38bdd(git statu
