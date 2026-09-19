# 회차 노트 2026-09-20-064421-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:44] base pinned — main@3ecbced
- [러너 06:44] autonomy release — 

## 정찰 노트
- 고른 이유: 미머지 MCP OAuth 브랜치(auto/2026-09-18-1143)가 아직 main 에 없어 그에 얽힌 항목(준비도 카드·부정 캐시)은 막혀 있고, 워크플로·문서 PDF 는 운영자 승인 성격이라 뺐다. 방문자 검색 빈 결과는 화면(UserExplorerPage 165-176행)과 순수 모듈(visitorTrace.ts)만 만지고 node:test 로 증명되는 가장 싼 실사용 결함이라 골랐다(DataTable 기본 Empty 가 "아직 데이터가 없습니다"로 수집 문제처럼 읽힘).
- 확신 없는 곳: 1143 브랜치가 UserExplorerPage·visitorTrace.ts 를 건드리지 않는다는 것은 미확인(diff 안 봄); queryError.ts ↔ visitorTrace.ts import 에 순환이 없는지도 미확인(현재 서로를 import 하지 않는 것으로 보임).
- 조심할 것: 요청 기간(69행 policyRange(90,…))과 문구가 같은 상수·같은 함수에서 나오게 할 것(운영자 지침: 같은 값을 읽는 경로 전부 일치). DataTable.tsx 의 기본 Empty 문구와 policyRange 는 다른 화면이 쓰므로 바꾸지 말 것. 정찰 예산이 적어 이번 회차에 테스트를 돌리지 못했다 — 구현자는 npm ci 뒤 lint·prettier·test·build 를 모두 돌릴 것.
- [러너 06:48] scout done — 방문자 검색(UserExplorerPage)의 빈 결과가 "아직 데이터가 없습니다" 대신 실제 조회 기간과 함께 "검색 결과�

## 구현 노트
- 무엇·왜: 방문자 검색 0건이 DataTable 기본 Empty("아직 데이터가 없습니다")로 수집 장애처럼 읽혀서, visitorTrace.ts 에 SEARCH_DAYS·searchWindowLabel·searchEmptyDescription(policyRange 재사용)을 두고 UserExplorerPage 가 빈 results → Empty("검색 결과가 없습니다"+실제 기간·검색어), 결과 있음 → description 앞에 기간 문구. 커밋 eaa214e, web 4파일만.
- 확신 없는 곳·검증 못 한 것: (1) tsconfig.app.json 에 allowImportingTsExtensions 를 켜고 visitorTrace.ts 가 `../components/queryError.ts` 로 import — node:test 가 확장자 없는 ESM import 를 못 찾아서다. tsc -b·vite build·eslint 통과했지만 이 플래그가 앱 tsconfig 에 처음 들어간 것이라 비평가가 먼저 볼 곳. (2) 문구 "최근 N일 안에 … 일치하는 방문자가 없습니다" — 서버는 ID·부서·조직 일치는 기간 없이, 페이지·이벤트 일치만 from/to 로 찾는다. 빈 결과에서 거짓은 아니지만 과장 가능(과제서 문구 그대로 둠, 보류 항목 기록).
- 일부러 하지 않은 것: 서버 응답에 window 추가(계약 테스트로 번짐), DataTable 기본 Empty·policyRange 변경(다른 화면이 씀), prettier 포맷 정리(저장소에 prettier 없음 — npx 3.9.8 기본 설정으로 main 이 이미 23파일 실패, CI 게이트 아님).
- 다음 역할이 조심할 것: web 게이트는 `npm audit && npm run lint && npm test && npm run build`(typecheck 스크립트는 sdk/ 것). 실제 화면 확인은 8080 이 호스트에서 사용 중이라 바이너리를 debian 컨테이너에서 띄워(포트 하드코딩) puppeteer-core(scripts/guide/node_modules)로 했고 스크린샷은 /tmp 에만 있다. 화면의 사이트 선택은 localStorage `momento:selected-site` 라 사이트가 여럿이면 다른 사이트를 보고 있을 수 있다(첫 시도에서 그렇게 헛봤음).
- [러너 06:56] brief accepted — 채택 — 과제서의 근거(69행 policyRange, 165-176행 분기, DataTable 기본 Empty, 서버 응답에 기간 없음)가 코드와 모두 맞았고 수�
- [러너 06:56] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 3ecbced..HEAD 는 web 4파일뿐(worktree 의 로컬 main 은 뒤처져 있어 pinned base 로 diff). 새 분기가 닿는 `results` 는 서버가 항상 non-nil 배열로 씀(visitor_trace.go:676,710) → 크래시 없음. 새 테스트는 문구와 policyRange 일치를 실제로 단언(옛 코드에서 실패). allowImportingTsExtensions 는 noEmit 과 유효, attention.ts 의 교차 import 는 type-only 라 이번이 첫 값 import. lint·test(100)·build·audit(0) 직접 통과.
- 못 본 것: 브라우저 렌더 확인, 구현자 스크린샷(/tmp).
- 남는 우려(승인): UserExplorerPage.tsx:185 결과 표 캡션 "최근 N일 안의 활동"은 과장 — ID·부서·조직·visitor_id 일치는 서버가 기간 없이 찾음. 빈 결과 문구는 상위집합이라 거짓 아님. 다음 회차: 캡션을 "페이지·이벤트 일치는 최근 N일 안에서" 로 정확히 하거나 서버 응답에 window 추가. 관례: pages/*.ts 순수 모듈끼리 값 import 는 '.ts' 확장자 필수.
- 판정 approve / risk low / blocking 없음.
- [러너 06:58] review approved — 리뷰 승인 (risk=low)
- [러너 06:58] pr created — https://github.com/hkjang/Momento/pull/16
- [러너 07:04] ci passed — 검사 1개 모두 success
- [러너 07:05] merge done — eaa214e
- [러너 07:14] release published — v0.34.42
- [러너 07:15] assets verified — v0.34.42 자산 2개 (이전 v0.34.41: 2)
