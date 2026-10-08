# PR 처리기 노트 2026-10-08-165742-postra-shepherd — postra PR #38
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-08-160826-postra-improve)
# 회차 노트 2026-10-08-160826-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:08] base pinned — main@c40b91d
- [러너 16:08] autonomy release — 

## 정찰 노트
- 선택: IMAP ensureIndex의 선행 완료 표시가 runSync UIDL→List 폴백에서 실패를 빈/부분 성공으로 바꾼다. 프로덕션 1파일로 좁힐 수 있어 저가치 문서·번들 정리와 PR #22 보류 접근보다 우선했다.
- 미확인: 새 회귀 테스트 실패 출력과 운영 사례. 코드 경로는 확인했으며 어댑터 race(4.450/5.126초), 관련 앱 테스트(2.092초), 포맷·계약 검사는 PASS. 기존 테스트 성공을 새 결함 재현으로 표현하지 않았다.
- 주의: indexed만 늦추면 부분 index가 재시도에 중복된다. 로컬 누적 후 성공 커밋; 실제 TCP+Dialer로 앱 최종 failed/enumerate까지 증명. 프레이밍·POP3 폴백·auth·workflows는 변경 금지.
- brief 초안 저장 후 보강했고 기존 9개 보류를 재평가해 신규 2개와 완료 3개를 더했다. 프로필은 현재 베이스/TLS 해결 상태에 맞춰 갱신. 전용 Skill 도구 대신 로컬 요청 스킬 3개를 읽었다.
- [러너 16:15] scout done — IMAP 열거 실패 뒤 빈 목록·부분 인덱스를 정상 캐시로 반환하지 않게 한다 (가치 4 / 위험 2 / 작업량 M)

## 구현 노트
- 59faa19: ensureIndex가 로컬 슬라이스에 전 배치를 모은 뒤에만 index/indexed를 커밋하도록 수정; 프로덕션 1파일·테스트 포함 3파일.
- 체크포인트 1 완료: 수정 전 첫 거절 뒤 0건 성공·후속 거절 뒤 2000건 성공 재현, 수정 후 어댑터 race PASS; 요청 범위 기록으로 재시도·캐시 사용 확인, indexed만 늦춘 변이는 4001건 중복으로 실패 후 원복.
- 체크포인트 2 완료: 실제 TCP+imap.Dialer+CreateAccount→StartSync→GetJob. 원래 구현 복원 시 succeeded/seen:0 재현, 수정 후 failed/enumerate/FETCH/rejected·서버 표식 미저장; 관련 앱 테스트 PASS(2.393s).
- 체크포인트 3 완료: 최종 3패키지 전체 race PASS(application 88.796s / imap 5.120s / pop3 5.593s), build/vet/포맷/계약/diff 검사 exit 0; ledger-entry.md와 ideas.json 갱신.
- 확신 없는 곳·미검증: 운영 서버 사례, 전체 저장소 Go 테스트, 외부 PostgreSQL·브라우저 e2e·프런트 빌드는 미실행; 빈 메일함은 처음부터 통과하는 호환성 검사이며 결함 재현으로 세지 않음.
- 일부러 제외: POP3 폴백 정책·프레이밍·broken/abandon·IDLE·UIDVALIDITY·폴더 전환·진단 정책·PR #22 영역; 캐시 결함 외 범위 확장 없음.
- 다음 역할 주의: 테스트는 루프백 TCP와 로컬 SQLite/암호화 저장소 사용; 서버 명령은 mutex로 읽고 cleanup 전 단언. 서버 표식을 감사 details로 옮기지 말 것; push/릴리즈는 수행하지 않음.
- [러너 16:23] brief accepted — 채택 — 선행 완료 표시와 공유 인덱스 누적이 실제 UIDL→List 폴백을 빈/부분 성공으로 바꾸는 것을 수정 전 실행으로 확�
- [러너 16:25] verify passed — 검증 9개 통과 (auto)
- [러너 16:25] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 16:25] pr created — https://github.com/hkjang/postra/pull/38
