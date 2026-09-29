# PR 처리기 노트 2026-09-29-121030-moyro-shepherd — moyro PR #29
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-29-091225-moyro-improve)
# 회차 노트 2026-09-29-091225-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:12] base pinned — main@a6d5106
- [러너 09:12] autonomy release — 

## 정찰 노트
- 팀 이미지 게스트 차단을 선택: 형제 DELETE 선례와 기존 DB 배선이 있고 프로덕션 1파일로 끝남; patchPost 주입 미해결·초대 중복 라우트 미확인보다 확실하다.
- 초안을 먼저 남긴 뒤 기존 실제 DB 회귀 2개 통과로 보완. 신규 게스트 RED와 정상 UI의 역할 교집합 생성 경로는 미확인이다.
- 구현자는 upload의 guestGated 및 삭제 사용자 401 기대를 함께 맞추고, 감사 비동기 관찰을 생략하지 말 것. auth/migrations/workflows 및 다른 초대 경로는 제외.
- 기존 아이디어 12개 유지·재평가, 플러그인 키 통일/독립 gofmt는 rejected, 신규 2개 추가. 회사 스킬 3개는 현재 도구/로컬 카탈로그에 없어 로드하지 못했다.
- [러너 09:17] scout done — 팀 이미지 업로드에 형제 DELETE와 동일한 게스트 차단 적용 (가치 2 / 위험 2 / 작업량 S)

## 구현 노트
- POST 팀 이미지 첫머리에 DELETE와 같은 게스트 가드를 추가했다. 프로덕션 1파일 3줄, 기존 테스트 guestGated 및 신규 DB 회귀를 포함해 총 3파일 변경.
- RED: 게스트 POST 200, 본문 읽기 2회, 감사 1행; 삭제 사용자 403. GREEN: 게스트 POST/DELETE 403·읽기 0회·감사 미기록, 삭제 사용자 401 및 정상 관리자/권한/상한 계약 통과.
- 실제 DB 지정 회귀 3개 13.452s(스킵 없음), DB HTTP race 53.189s, build/vet/소스 크기/변경 파일 gofmt/diff 검사 통과.
- 확신 없는 곳·검증 못 한 것: 배우 ID를 주입하는 핸들러 회귀이며 인증 미들웨어와 정상 UI의 역할 교집합 생성은 미검증. 전 패키지 테스트·웹 검증은 실행하지 않았다.
- 감사 미기록은 양성 대조 뒤 3.2초 반복 SELECT 방식이며 goroutine 스케줄링의 수학적 barrier는 아니다.
- auth/session/guest_access.go·마이그레이션·이미지 저장·다른 초대 경로는 범위 밖으로 유지했다. UserByID DB 오류의 401 분류도 기존 계약 그대로다.
- 다음 역할: DB 테스트는 MOYRO_TEST_POSTGRES_DSN 없으면 skip한다. 회사 스킬 3개는 현재 callable 도구·로컬·리소스 카탈로그에서 찾지 못해 로드/반환 형식 준수를 주장하지 않는다.
- [러너 09:22] brief accepted — 채택 — POST의 가드 누락과 DELETE 선례가 현재 코드에 일치하며, 지정된 권한·본문·감사·삭제 사용자·DB 장애 회귀를 실�
- [러너 09:23] verify passed — 검증 2개 통과 (policy)
- [러너 09:23] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 09:23] pr created — https://github.com/hkjang/moyro/pull/29
