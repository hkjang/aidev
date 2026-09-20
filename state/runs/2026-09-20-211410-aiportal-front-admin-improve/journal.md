# 회차 노트 2026-09-20-211410-aiportal-front-admin-improve — aiportal-front-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:14] base pinned — main@d959303
- [러너 21:14] autonomy release — 

## 정찰 노트
- 선택: Catalog 상세 실패 상태 수정. 이전 본문 재등장과 목록 오류 오표시를 한 화면에서 고치므로 정책 3-way 보존·CI 변경보다 위험과 범위가 작다.
- 근거: selectLibrary/상세 aside/API 배선을 읽었으며 런타임 재현은 미확인; npm run verify는 vue-tsc 부재로 중단됐다.
- 주의: 실제 API 모듈+HTTP 경계 테스트, afterEach unmount, 최신 ticket·본문 텍스트 렌더링 유지; 상세 GET 자동 재시도 금지.
- 지정 세 스킬은 도구/로컬 검색에서 없었음. 상세 탭 전환 UX는 미정으로 제외; auth/router/CI 보호 경로와 미병합 304e89e를 건드리지 말 것.
- [러너 21:17] scout done — 자료실 상세 조회 실패를 상세 패널에서 알리고 이전 게시물 본문이 다시 나타나지 않게 한다 (가치 3 / 위

## 구현 노트
- f3016f9: 상세 오류를 목록 오류와 분리하고 새 선택 시 이전 본문을 비웠다. 공통 닫기는 결과·오류·loading 초기화 및 요청 무효화를 수행한다.
- 신규 HTTP 경계 컴포넌트 테스트 12개: 수정 전 10개 실패, 수정 후 대상 21개 및 전체 173개 통과. npm ci와 전체 build의 모든 게이트 통과.
- 검증 못 한 것: 실제 서버/UAT와 HTTP envelope 파서. api/catalog은 실제 사용하지만 api/http의 apiGet·apiGetPage는 대역이며 실제 Promise·Vue mount·메모리 router를 사용한다.
- 요청된 technology 스킬 3개는 callable 도구·로컬 경로에서 찾지 못해 적용하지 않았다. 루트에서 실수로 실행한 test는 스크립트 부재로 실패했으며 admin-v2 경로에서 정상 재실행했다.
- 일부러 하지 않은 것: 탭 왕복 상세 유지 UX, 목록 skeleton 정책, auth/router/CI·의존성 변경, 상세 GET 자동 재시도. 모두 범위 밖이다.
- 다음 역할 주의: 목록에 A 제목은 남는 것이 정상이다. 이전 상세 잔존 여부는 .detail-sheet 내부에서 확인하며 afterEach에서 신규 wrapper를 정리한다.
- [러너 21:21] brief accepted — 채택 — 현재 코드가 상세 실패 시 이전 결과를 유지하고 목록 오류에 기록하는 근거와 일치했으며 수정 전 컴포넌트 테�
- [러너 21:21] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- approve / low, 차단 없음: main...HEAD 3개 파일과 API·요청 guard·query 동기화·문서·신규 테스트를 확인했고 실제 회귀 결함은 찾지 못했다.
- verify 최초 기존 CatalogView.test.ts:151 실패(172 통과), 재실행 타입 검사·173개 통과; 신규 12개는 모두 통과. 간헐 실패 원인은 후속 확인 필요.
- 실제 서버/UAT·상세 HTTP 통합·전체 build 재실행은 미확인; 상세 GET 조회수 증가와 탭 왕복 UX는 기존 제약이다.
- 지정 부서 스킬 3개와 Skill 호출 도구는 검색에서 발견되지 않아 적용하지 못했으며 프롬프트 기준으로 검토했다.
- [러너 21:23] review approved — 리뷰 승인 (risk=low)
- [러너 21:23] pr created — https://github.com/hkjang/aiportal-front-admin/pull/20
- [러너 21:23] ci passed — 검사 없음 — 정책으로 허용
- [러너 21:23] merge done — f3016f9
- [러너 21:24] release missing — 릴리즈 결과 없음/손상: missing
