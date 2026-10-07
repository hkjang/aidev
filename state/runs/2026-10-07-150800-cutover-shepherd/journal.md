# PR 처리기 노트 2026-10-07-150800-cutover-shepherd — cutover PR #15
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-07-132814-cutover-improve)
# 회차 노트 2026-10-07-132814-cutover-improve — cutover
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:28] base pinned — main@51ffd03
- [러너 13:28] autonomy release — 

## 정찰 노트
- 120자 정상 부모의 하위 추가가 134자 ID로 검증 거절됨을 실제 함수 호출로 재현해 선택했다. 프로덕션 1파일로 고칠 수 있어 전파 통합·리비전 변경보다 작고 오류 문구 정리보다 사용자 가치가 높다.
- brief 초안을 먼저 저장한 뒤 재현·95건 단위 통과로 보완했다. 기존 12아이디어 유지/재평가(2건 rejected), 신규 2건 추가; 선택 과제는 pending.
- HTTP 400 재현·build·E2E는 node_modules 부재로 미실행, 저장소 밖 ID 형식 의존은 미확인이다. 구현자는 실제 API 회귀와 클라이언트 번들을 확인할 것.
- treeUtils에 node:crypto 정적 import 금지; auth/저장 I/O/상태 전파/반려된 ESLint ignore 변경을 섞지 말 것. 명시 경로 단위 95 pass이며 npm run test:unit은 현재 exit 1이다.
- [러너 13:33] scout done — 유효한 긴 ID를 가진 부모에도 하위 작업을 추가할 수 있도록 신규 ID 길이 제한 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- 16c3384: addActivity 내부 UUID로 120자 부모 아래 추가 거절·같은 시각 ID 중복 해결. 프로덕션 1파일 + 테스트 2파일, 새 의존성 없음.
- 실패 먼저 확인: 집중 단위 26 pass/3 fail(134자 ID와 고정 시각 중복), 실제 로그인 API 400 INVALID_DATA. 코드 원복 시 같은 3건 재실패 후 수정 재적용.
- 최종 검증: 집중 단위 29 pass, 전체 단위 101 pass/0 fail/0 skip, tsc·lint·build·diff 검사 exit 0, 집중 E2E 3 pass·전체 E2E 39 pass(재시도 없음).
- 새 테스트는 실제 addActivity→validateActivityImport, 자식→손자, 입력 불변성·기본값·루트, 자동 복구되는 고정 Date.now와 PUT 200→GET 저장 결과를 검증한다.
- 확신 없는 곳·검증 못 한 것: 저장소 밖 ID 형식 의존 미확인. npm ci audit 10건(9 high/1 critical)의 영향 분석은 미실시.
- 일부러 제외: UUID 충돌 재시도·level 50 추가 정책·저장/인증/메일/배포·단위 스크립트/ESLint 변경. 기존 제한값과 ID는 유지.
- 다음 역할 주의: E2E는 /usr/bin/google-chrome·격리 ACTIVITY_DATA_FILE·실제 dev 서버 사용. 단위와 동시 실행 금지, lint는 E2E 이전. 생성 보고서는 정리했고 로그·원장·아이디어는 이 회차 디렉터리에 저장했다.
- [러너 13:39] brief accepted — 채택 — 부모 접두사+Date.now 생성, 120자 검증 상한, 실제 PUT 경로 및 ID 형식 무관 문서 계약이 현재 코드와 일치했고 결정�
- [러너 13:39] verify passed — 검증 1개 통과 (policy)
- [러너 13:39] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 13:39] pr created — https://github.com/hkjang/cutover/pull/15

## 심사 노트
- 확인: addActivity 1줄 변경(parentId+Date.now → globalThis.crypto.randomUUID). 양방향 검증 — 수정본 단위 101 pass·E2E 39 pass, 옛 구현으로 되돌리면 같은 단위 3 fail 과 E2E 가 실제 PUT 응답 400 INVALID_DATA(id 120자 초과)로 실패. 되돌린 코드는 복원해 tree 깨끗.
- 확인: ID 형식 의존 없음(호출부는 route.ts:91 뿐, buildTree·move·collect·admin·ActivityTree·mail 모두 키 조회만, docs/ADMIN_GUIDE.md:190 이 이미 형식 무관 명시). 번들 실측으로 .next/static 에 randomUUID 없음 → 클라이언트 secure-context 문제 해당 없음. tsc·lint·build 모두 exit 0.
- 확인: 보호 파일·공개 경로·권한·의존성·마이그레이션 변화 없음, PUT 은 isAdminRequest 뒤 유지. 저장된 UUID 는 옛 코드에서도 유효해 revert 안전. 차단 소견 없음.
- 못 본 것: test:unit 은 기존 glob 문제로 exit 1(관문 미사용), npm audit 10건 영향 분석(의존성 미변경), 저장소 밖에서 `부모-번호` ID 형식을 가정하는 소비자 유무.
- 권고 근거: 결함 없음 + 프로덕션 배선 실행으로 end-to-end 증명됨 + 되돌리기 쉬움 → approve/merge. risk 는 지속 저장되는 식별자를 정하는 변경이라 medium.
