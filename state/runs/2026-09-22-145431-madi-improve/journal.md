# 회차 노트 2026-09-22-145431-madi-improve — madi
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:54] base pinned — main@fd2c3b5
- [러너 14:54] autonomy release — 

## 정찰 노트
- 선택: 실제 고정 Alpine에서 lock 충돌 exit 123을 재현하여 APK 사전 점검 도구 선정; 인증/브라우저 후보보다 작고 기존 lock 갱신 PR과 겹치지 않는다.
- 스킬 3개는 형제 headcount 저장소 원문으로 적용했다(전용 Skill 도구 없음); 지난 회차의 스킬 부재 차단을 반복하지 않았다.
- 미확인: 신규 스크립트 자체는 구현 전이며 원격 네트워크의 향후 가용성은 보장 못 함. 실제 APK 양성 실행 및 기존 deployment-contract/Node 2개 시험은 통과.
- 주의: fd2c3b5에는 과거 성공 변경 다수가 미반영. lock·grpc·dist·인증·워크플로를 함께 고치지 말고 기본 lock의 예상 실패를 테스트 실패와 구분할 것.
- [러너 14:59] scout done — 고정 runtime APK 잠금 파일을 전체 빌드 전에 검사하는 사전 점검 스크립트 (가치 3 / 위험 1 / 작업량 M)

## 구현 노트
- 2fd17bf: APK 사전 점검 CLI·실컨테이너 테스트·배포 안내 추가. Dockerfile에서 이미지 조회, lock 읽기 전용, 자동 업데이트 없음.
- 세 technology 스킬 원문 적용; CLI 부재의 RED 후 구현. 실제 네트워크 차단 시 APK가 경고+exit 0을 반환함을 발견해 진단 보존+실패 처리, guard 제거 시 회귀 실패 후 복원까지 확인.
- 최종 bash -n / bash tests/check-apk-lock.sh / go test -count=1 ./tests/deployment-contract / git diff --check 모두 통과. verification.log 및 regression-proof.log 참조.
- 기본 lock은 /tmp에서 절대 스크립트 경로 실행해 ca-certificates·tzdata 충돌 exit 2 확인. 기본 lock·fixture SHA256 및 git diff 불변.
- 검증 못 한 것: 전체 웹/Go/이미지·대응 소스 빌드, 다른 Docker context/원격 데몬, 타 아키텍처 에뮬레이션, 실제 pull 장애는 미검증. 미래 네트워크/저장소 상태는 보장 못 함.
- 의도적으로 Dockerfile·lock·의존성·CI·릴리즈·dist는 변경하지 않음. APK stderr는 종료 직후 그대로 출력하며 경고도 불확실로 비영 처리.
- 다음 역할: 테스트는 Docker/연결망 필요; 프록시 장애는 임시 DOCKER_CONFIG로만 재현하고 정리함. 기본 lock의 예상 실패와 신규 테스트 실패를 구분할 것.
- [러너 15:05] brief accepted — 채택 — Docker와 실제 고정 이미지 사용이 가능하고 lock 충돌 근거가 일치하여 지정된 세 파일만 변경했다.
- [러너 15:05] verify failed — 실패한 검증: go build ./... (exit 1)
