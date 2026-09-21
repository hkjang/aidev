# PR 처리기 노트 2026-09-21-223915-postra-shepherd — postra PR #22
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-21-215512-postra-improve)
# 회차 노트 2026-09-21-215512-postra-improve — postra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:55] base pinned — main@5dd0099
- [러너 21:55] autonomy release — 

## 정찰 노트
- POP3 MaxMessageBytes 무시를 선택: 실제 메모리 보호 공백이라 일반 테스트 추가·README 버전 정렬보다 가치가 높고 auth/workflows를 피할 수 있음(4/2/M, 35분+예비10분).
- app.go:dialInbound → pop3 Dial/retrBody → sync.go:ingestOne 순서로 정적 확인; 과대 본문 실행 재현과 메모리 수치는 미확인, 구현자는 작은 상한의 실제 TCP 실패 테스트부터 작성.
- 긴 행을 다 읽은 뒤 검사·DotReader의 LF 변환·초과 시 무한 drain을 피할 것. 실제 application 배선과 정상 메일 해시/중복 유지 검증 필수, 개별 실패를 JobFailed로 바꾸지 말 것.
- 기존 pending 재평가 및 신규 2개 포함 ideas.json 작성, 현재 v0.23.3/lint 반영 profile 갱신. 전체 Go race·포맷·계약 검사 통과, git status 깨끗함. 부서 스킬/Skill 도구 미발견이며 원문 적용 주장 없음; SMTP timeout·notifymail/handoff 재적용 금지.
- [러너 22:00] scout done — POP3 본문 수신에 MaxMessageBytes 상한 적용 — 긴 단일 행까지 제한하고 실제 TCP·동기화 배선으로 검증 (가치 

## 구현 노트
- 04b15be: POP3 세션에 MaxMessageBytes를 보관하고 고정 버퍼 ReadByte로 RETR/TOP 본문을 제한; dot-unstuffing 후 CRLF 포함 크기, 초과 시 연결 종료·부분 본문 없음·0 무제한.
- 수정 전 TCP 재현 실패 → 수정 후 경계·긴 무개행 행·명령 deadline 통과; 실제 application 수집/미저장/failed 및 정상 원문·해시·UIDL 유무 중복 방지 검증.
- app.go 상한 전달을 0으로 바꾼 변이에서 통합 테스트가 failed=0/oversize=1로 실패함을 확인하고 원복; app.go 최종 변경 없음.
- 지정 집중 race·make lint(Issues 0)·build·vet·전체 race(application 58.748s, 일부 캐시)·계약 -check·diff --check 모두 통과.
- 검증 못 한 것: 외부 PostgreSQL·브라우저/프런트 실행·POP3 TLS 전체 행렬·메모리/성능 실측. 유한 상한의 메모리 경계는 고정 입력 버퍼와 상한 내 출력 버퍼 구조로 확인.
- 일부러 제외: 자동 재연결/재시도·JobFailed/oversize 정책 변경·SMTP timeout·auth/DB/workflows; 이번 과제 밖이며 기존 정책 유지.
- 다음 역할: TCP 테스트는 loopback listener와 임시 SQLite를 사용하며 외부 DB 불필요. 요청 technology 스킬 3종/Skill 도구는 검색에서 미발견, 해당 원문 절차 적용 주장 없음.
- [러너 22:07] brief accepted — 채택 — 현재 베이스의 Dial/retrBody가 상한을 무시하며 실제 TCP에서 작은 상한으로 결함을 재현했고 정해진 범위 안에서 �
- [러너 22:08] verify passed — 검증 9개 통과 (auto)
- [러너 22:08] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 22:08] pr created — https://github.com/hkjang/postra/pull/22
