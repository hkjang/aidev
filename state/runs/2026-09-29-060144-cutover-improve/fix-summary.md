비평 지적 확인: STARTTLS 승인 후 TLS 무응답 시 timeoutMs=100이어도 1초 넘게 sendMail이 미완료였다.
upgrade 전용 타이머와 오류·종료 처리를 추가하고, TLS 소켓을 즉시 추적해 실패 시 정리한다.
실제 sendMail + 로컬 TCP 릴레이로 auto/starttls 회귀 테스트 2건 추가; 시간 초과 오류와 원격 연결 종료를 검증한다.
수정 제거 시 2건 재실패, 복원 후 npm run test:unit 97/97 및 npm run lint, npx tsc --noEmit, npm run build 통과.
SMTP 코드·테스트만 새 커밋으로 기록했으며 push하지 않았다.
