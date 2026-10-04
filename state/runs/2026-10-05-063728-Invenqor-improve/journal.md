# 회차 노트 2026-10-05-063728-Invenqor-improve — Invenqor
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:37] base pinned — main@12ada2f
- [러너 06:37] autonomy release — 

## 정찰 노트
- 관계 생성 source/target·삭제 relationID 검증을 선택: 기존 helper 재사용으로 프로덕션 1파일이며, tx 교착·페이지네이션보다 범위와 증명이 명확하다. 초안 저장 후 근거를 보완했다.
- 기존 관련 SQLite(6.683초)·PostgreSQL(httpapi 11.853초) 테스트 통과; 새 입력의 오류 응답·부작용은 소스 추론이며 직접 재현은 미확인. 전체 suite는 미실행.
- 두 인증 경로를 실제 키/세션으로 검증하고 36자 대문자만 정규화한다. DELETE 부모 assetID 미사용·confidence 범위는 별도 후보로 두며 tx/auth/migrations/workflows는 건드리지 않는다.
- 세 부서 스킬은 전용 도구 대신 로컬 원문을 읽어 적용했다. Query limit는 done, split 존재 사전조회는 rejected; 프로필의 낡은 UUID 검증 설명을 갱신했다.
- [러너 06:42] scout done — 자산 관계 생성·삭제의 UUID 입력을 기존 canonicalUUID 계약으로 검증 (가치 3 / 위험 1 / 작업량 M)

## 구현 노트
- 79f1889: assets.go의 관계 생성 source/target·삭제 relationID에 기존 UUID 검증·소문자 정규화를 적용. 프로덕션 1파일+실제 인증/라우터 테스트 1파일.
- 수정 전 SQLite 409/대문자 실패, PostgreSQL 중괄호·32자의 실제 생성/삭제 및 감사 원문 저장을 재현; 수정 제거 시 SQLite도 재실패. red-*.log·revert-sqlite.log 참조.
- 양 DB 관련 테스트 및 전체 suite 통과(SQLite httpapi 30.384s, PostgreSQL 68.456s); vet·서버 빌드·gofmt 통과, 빌드 산출물 제거. 상세 명령은 verification.md.
- 확신 없는 곳·미검증: 변경 밖 Web/Rust·릴리즈·배포는 실행하지 않음. PostgreSQL 준비 확인 1회 실패 후 같은 설정 재실행 통과; 스크립트는 수정하지 않음.
- 일부러 제외: DELETE 부모 assetID 검증/소속·confidence 범위·존재 사전조회·오류 분류·tx/auth/migrations/workflows·버전/PDF는 과제 범위 밖.
- 다음 역할: 기본 go test는 SQLite. PostgreSQL 검증에는 실제 DB와 고유 컨테이너/빈 포트 필요(이번 55516, 컨테이너 정리 완료). 테스트는 Fake 없이 실제 세션/API key 사용.
- [러너 06:51] brief accepted — 채택 — 현재 핸들러·라우팅·스키마가 근거와 일치했고, 미재현이던 방언 차이와 감사 부작용을 수정 전 실제 요청으로
- [러너 06:53] verify passed — 검증 8개 통과 (auto)

## 비평 노트
- 판정 reject / high / security: 요청한 로컬 main(c9527e7)...HEAD에 포함된 SMTP LOGIN이 auto 모드에서 STARTTLS 제거 시 비밀번호를 평문 채널로 전송함을 실제 TCP로 재현; 수리는 server/internal/mail/mail.go:427과 review-smtp-evidence.md부터 확인.
- 기준 주의: origin/main·회차 base는 12ada2f이며 SMTP 결함은 그 이전 152c6ba에서 유입; 이번 79f1889의 2파일 UUID 변경에서는 실제 결함을 찾지 못함. 로컬 main 기준 판정과 혼동 금지.
- 실패 재현 원장·로그, 실제 인증/라우터 테스트, DB·감사 부작용과 권한을 확인; 관련 SQLite 10.317s·PostgreSQL 16.139s 통과. 새 개인정보 처리·마이그레이션·의존성 변경 없음.
- 전체 suite·vet·Web/Rust·PDF·실제 배포는 재실행하지 않음. DELETE 부모 소속·confidence는 기존 별도 과제; OpenAPI 400 설명 보완 여지. 저장소 코드 수정 없음.
- [러너 06:58] review rejected — 리뷰 거절: server/internal/mail/mail.go:427 [P1][security·차단] 요청한 main...HEAD에 추가된 LOGIN 인증은 TLS 없이 SMTP 비밀번호를 전송한다. 메일 활성화·자격증명 �
- [러너 06:58] review blocked — 검토 부서 차단 소견(security) — 수리·중재 없이 운영자의 위험 수용(risk-accepted) 필요
- [러너 06:58] pr created — https://github.com/hkjang/invenqor/pull/31
