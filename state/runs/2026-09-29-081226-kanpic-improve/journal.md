# 회차 노트 2026-09-29-081226-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:12] base pinned — main@1d2d226
- [러너 08:12] autonomy release — 

## 정찰 노트
- SMTP 컨텍스트 시한 누락을 실제 TCP 4갈래로 재현하여 선택했다; 기존 mail race/전체 Go 테스트는 통과한다. 프로덕션 1파일 범위라 계약 미정 숫자 통합·DB 후보보다 명확하다.
- Timeout=0 및 Notify 감사 주소 공백 가설은 코드상 기각. 이전 프로필과 달리 README는 v0.257.0, parseCSV UTF-8 유효성 검사는 현재 base에 없다.
- TLS handshake·명시 cancel은 미실행이며 구현자가 보강할 것. 취소 감시를 dial 반환 때 해제하거나 호출자만 반환시키고 SMTP 고루틴을 남기지 말 것.
- 세 요청 스킬을 실제 headcount 경로에서 읽고 대안·단계 검증·25~35분 기본+5~10분 예비를 과제서에 반영했다. 코드/커밋 변경 없이 run 디렉터리에 초안 후 최종본·overlay 증거를 남겼다.
- [러너 08:17] scout done — SMTP 세션이 호출자 컨텍스트의 취소·시한을 끝까지 지킨다 (가치 4 / 위험 2 / 작업량 M)

## 구현 노트
- de77d56: mail.go·mail_test.go만 변경. SMTP greeting 이후에도 컨텍스트 취소가 실제 TCP 소켓을 닫으며 세션 종료는 취소 콜백까지 회수한다.
- implicit TLS 연결/handshake는 tls.Dialer.DialContext 사용; Timeout의 기존 연결 제한 의미 및 인증서·AUTH 정책 유지.
- 회귀 12사례(Deliver/Verify × greeting/EHLO/TLS × deadline/cancel) 모두 빨강→초록. 구현만 원복하면 다시 12실패; 실패 시에도 서버와 호출 고루틴을 bounded cleanup으로 회수.
- 지정 좁은 테스트·mail race(2.189s)·전체 Go 테스트(일부 cached)·vet·build·gofmt·diff 통과. 최초 12사례 재현 후 범위 유지로 판단했고 추가 작업량 확대 없음.
- 검증 못 한 곳: 실제 운영 TLS relay, STARTTLS 정지 및 DATA/QUIT 정지를 별도 회귀로 실행하지 않음. 동일 원본 소켓 감시가 세션 전체를 보호하도록 구현했고 웹·DB 통합·브라우저는 미실행.
- service.go의 WithoutCancel·재시도·감사 DB, 설정·문서·PDF·릴리즈는 의도적으로 제외. 전송 완료 응답 유실 시 중복 재시도 정책도 유지.
- 다음 역할: run 디렉터리 mail-red.log/mail-reverted.log/mail-green.log/mail-checks.log 참고. 테스트는 로컬 TCP를 사용하며 DB는 필요 없음.
- [러너 08:20] brief accepted — 채택 — 현재 코드와 실제 TCP 12사례가 시한·취소 누락을 확인했고 프로덕션 1개·테스트 1개 범위에서 해결했다.
- [러너 08:21] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- approve / low, security·legal 차단 없음. 세 부서 스킬·diff/log·호출자·표준 라이브러리·실패 재현 원장을 확인했다.
- 실제 TCP 회귀 12사례의 수정 전/원복 후 실패가 증상과 일치하며 mail race 재실행 통과(2.161s); 코드 변경 없음.
- STARTTLS·DATA·QUIT 취소는 정적 경로 확인만 했고 운영 TLS·DB·웹·브라우저는 미검증.
- 기존 service.go complete의 만료 컨텍스트 사용에 따른 감사 갱신 실패 가능성은 다음 회차 확인 대상으로 남긴다.
- [러너 08:22] review approved — 리뷰 승인 (risk=low)
- [러너 08:22] pr created — https://github.com/hkjang/kanpic/pull/37
