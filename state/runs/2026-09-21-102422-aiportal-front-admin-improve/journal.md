# 회차 노트 2026-09-21-102422-aiportal-front-admin-improve — aiportal-front-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:24] base pinned — main@fa19a86
- [러너 10:24] autonomy release — 

## 정찰 노트
- query 변경 중 이전 요청이 캐시를 되살리는 양쪽 화면의 결함을 선택: 낮은 변경 위험으로 사용자/조건 오인 방지, 정책 편집 병합·상세 탭 UX보다 계약이 명확함.
- 근거는 실제 applyRouteQuery/load 함수 정적 대조이며 런타임 재현은 미확인; npm run verify는 vue-tsc 부재로 exit 127.
- invalidate 후 loading·이전 결과도 초기화하고, 같은 조건 탭 왕복 캐시 및 늦은 성공/실패를 실제 API 배선으로 검증할 것.
- 요청 스킬은 도구/로컬 검색에서 찾지 못함. auth/CI/의존성은 제외하고 코드·커밋 변경 없이 문서만 작성함.
- [러너 10:29] scout done — Catalog·Content의 query 변경 시 이전 조건으로 진행 중인 조회를 무효화한다 (가치 3 / 위험 1 / 작업량 M)

## 구현 노트
- 15e032f: Catalog·Content query 적용에서 이전 ticket·loading·결과·완료 상태를 함께 초기화; 단순 탭 왕복 캐시는 유지.
- 실제 Vue/memory router/기능 API/HTTP envelope/Axios adapter 테스트 20개 추가; 수정 전 16개 실패, 수정 후 대상 48개 통과.
- npm ci 및 npm run build 통과: 타입 검사·전체 193개·Vite·runtime-config·offline·integrity 19개. dist는 기존 ignore로 커밋 제외.
- 확신 없는 곳·검증 못 한 것: 실제 서버/UAT, 네트워크 취소는 미검증(기존 ticket으로 화면 반영만 무효화).
- 요청 technology 스킬/Skill 호출 도구 미발견; 다른 플러그인의 동명 파일을 요청 스킬로 적용하지 않음.
- 보호 경로·공통 guard·의존성·상세 GET·활성 탭 불필요 재조회는 범위 밖이라 변경하지 않음.
- 다음 역할: 새 테스트는 외부 서버 없이 native Promise transport 지연을 사용하고 afterEach에서 wrapper·adapter·runtime 설정을 정리함. push/릴리즈 미수행.
- [러너 10:33] brief accepted — 채택 — 현재 코드와 근거가 일치했고, 실제 배선 테스트로 query 변경 뒤 이전 요청이 새 조회를 막거나 이전 결과를 남기
- [러너 10:33] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- approve / low / blocking 없음: main 대비 5개 파일과 guard·query·API 경로를 검토했고 실제 결함은 발견하지 못함.
- npm run verify 직접 통과: 타입 검사 및 25개 파일·193개 테스트; 신규 경합 테스트의 요청·DOM 단언 확인.
- 실제 서버/UAT·네트워크 취소·전체 build·수정 전 테스트 재실행은 미검증; 전송 취소가 아닌 화면 반영 무효화임.
- 요청된 세 부서 스킬/Skill 도구 미발견으로 부서별 절차 적용 불가; 코드 수정 없이 판정 기록.
- [러너 10:34] review approved — 리뷰 승인 (risk=low)
- [러너 10:34] pr created — https://github.com/hkjang/aiportal-front-admin/pull/21
- [러너 10:35] ci passed — 검사 없음 — 정책으로 허용
- [러너 10:35] merge done — 15e032f
- [러너 10:36] release failed — 릴리즈 안 함: 요청된 marketing:product-launch 및 technology:release-and-deployment 스킬과 Skill 호출 도구를 사용 가능한 도구 및 로컬 스킬 경로에서 찾지 못했다. �
