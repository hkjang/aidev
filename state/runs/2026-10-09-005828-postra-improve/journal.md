# 회차 노트 2026-10-09-005828-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:58] base pinned — main@c40b91d
- [러너 00:58] autonomy release — 

## 정찰 노트
- greeting 예산 오기록이 syncTiming 조기 중단 문구까지 이어짐을 확인해 선택; retrBody 반려 위험·프런트 자산 변경·대규모 취소 전파보다 범위가 작다. 초안을 먼저 작성한 뒤 앱 배선 수용 기준으로 보완했다.
- 실제 침묵 서버 신규 실패 재현은 미확인. 기존 양쪽 어댑터 race 및 관련 앱 테스트 PASS; 예상 30–40분은 픽스처 재사용 전제의 추정이다.
- IMAP failed 전역 수정·WrapInbound 이중 래핑·서버 cleanup EOF 오탐을 피한다. fastRetries 전역 변경 테스트는 병렬 금지, 원문 표식은 저장 Job에서 검사한다.
- 59faa19는 구현 완료 기록이 있지만 pinned main 조상은 아니다. 중복 구현하지 않으며 ideas의 done은 구현 완료 의미로 명시했다.
- [러너 01:05] scout done — POP3·IMAP greeting 실패 진단의 Timeout을 실제 connectTO로 맞춘다 (가치 3 / 위험 1 / 작업량 M)

## 구현 노트
- bf0b12d: POP3 greeting 1곳·IMAP 읽기/BYE 2곳의 Timeout만 connectTO로 수정(프로덕션 2파일, 테스트 포함 총 5파일).
- 실제 TCP+Dialer에서 1초/4초 예산을 분리해 timeout/rejected·peer EOF·후속 명령 4초를 검증. 관리자 설정→CreateAccount→StartSync→GetJob→incident까지 양쪽 통과, 저장 Job에 거절 표식 없음.
- 수정 전 및 원복 변이: greeting Timeout=4s, 앱 TimeoutMS=4000과 조기 중단 문구 재현. greeting-*-red.log·greeting-adapters-revert.log에 증거, 수정본 복원 완료.
- 최종 영향 3패키지 race PASS(application 88.856s/IMAP 5.563s/POP3 6.161s); build·vet·lint-format·contracts -check·diff --check 모두 exit 0.
- 검증 못 한 것: 전체 저장소 Go 테스트·외부 PostgreSQL·브라우저 e2e·프런트 빌드·보안 스캐너·실제 운영 서버는 미실행. 테스트는 로컬 SQLite와 루프백 TCP 기준.
- deadline·기본값·session.failed·재시도·STARTTLS·TLS·retrBody·ensureIndex·계약/UI/문서는 범위 밖이라 미변경. 원문 raw POP3 에러 자체의 정제도 범위 밖.
- 다음 역할: fastRetries는 전역이므로 새 앱 테스트에 t.Parallel 금지. EOF는 cleanup 전 확인하며 서버의 5초 deadline을 정상 종료 증거로 취급하지 않는다. 릴리즈·push 미실행.
- [러너 01:11] brief accepted — 채택 — 지정된 세 반환과 앱 요약까지의 원인을 수정 전 실행으로 확인했고, 지정 5파일에서 수용 기준과 최종 검증을 �
- [러너 01:12] verify passed — 검증 9개 통과 (auto)
- [러너 01:12] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 01:13] pr created — https://github.com/hkjang/postra/pull/39
