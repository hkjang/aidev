# 회차 노트 2026-09-22-110437-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:04] base pinned — main@cf3b598
- [러너 11:04] autonomy release — 

## 정찰 노트
- 신규 UTF-8 절단 오류를 선택: 3001·3002 예산에서 JSON U+FFFD 재현, 기존 3000 테스트는 우연히 통과. ACL 복사보다 사용자 출력 개선이 명확하고 반려 파서·이미 수행한 마스킹 작업을 피한다.
- 정찰 재현은 원본 함수 복사 실행이며 실제 read-file HTTP 경로는 미확인; 구현자는 SQLite→ServeHTTP의 캐시 미스/히트와 원문 접두부를 검사할 것.
- 예산 320B 예약·진단 추가·공용 runeSafeCut 계약·auth/migrations/workflows는 범위 밖. 예상 35분+예비 10분; 불성립 시 cacheKey 복사 차선.
- 요청한 세 스킬/Skill 도구 미발견. 관련 4패키지 테스트 exit 0, 전체·race·외부 통합 미실행; 코드/커밋 변경 없음.
- [러너 11:09] scout done — MCP 긴 한 줄 응답 절단 시 UTF-8 글자 경계 보존 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 1dd6363: cutAtBoundary fallback 한 줄만 원본 text/limit로 바꿔 UTF-8 중간 바이트 절단을 방지했다. 예산 예약·공용 runeSafeCut은 그대로다.
- 함수 전 바이트 경계(절단 없음 포함), SQLite→실제 ServeHTTP read-file→JSON의 4 문자 조합×12 연속 예산×미스/히트 회귀를 추가했다. U+FFFD·원문 접두부·캐시 감사 기록을 검사한다.
- 수정 전 실패 로그는 assets/regression-before.log. 초기 HTTP fixture에는 repository_files가 빠져 조회 오류였으며, 보완 후 원래 소스로 실제 문자 손상 실패를 다시 확인했다.
- 지정 MCP -count=1·MCP race·전체 sqlite_fts5 suite·vet·build·gofmt 모두 exit 0(assets 로그). app 101.576s 통과, 전체 suite 일부 패키지는 캐시 사용.
- 미검증: 외부 PostgreSQL/pgvector/Vault 통합, Docker·브라우저·실제 MCP 클라이언트. Skill 도구와 요청 technology 스킬 3개 미발견으로 해당 절차/형식 미확인.
- 일부러 제외: 320B 예약·finishCall 진단 후 최종 예산·ReadFile 자체 192KiB 절단은 별도 후보. 이번 HTTP 입력은 192KiB 미만이다.
- 다음 역할: 공유 in-memory SQLite fixture를 t.Parallel로 돌리지 말 것. 릴리즈/버전/원격 작업은 수행하지 않았다.
- [러너 11:15] brief accepted — 채택 — 현재 코드의 잘못된 호출과 실제 함수·HTTP 캐시 미스/히트의 문자 손상을 재현했으며 지정된 한 줄 수정으로 회�
- [러너 11:17] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- approve / low, 차단 없음: 3개 변경 파일·커밋·구현 미검증 항목과 UTF-8/JSON/HTTP 캐시·감사 단언을 확인; 운영 변경은 한 줄이고 revert로 복구 가능.
- MCP -count=1 재실행(0.832s)·diff --check 통과; 수정 전 실패 로그와 기존 코드 분석으로 회귀 검증력 확인, 코드는 수정하지 않음.
- 요청한 세 부서 스킬/Skill 도구 미발견으로 전용 절차 미적용; 전체·race·외부 통합·Docker·브라우저·실제 클라이언트는 이번 리뷰에서 재실행하지 않음.
- 다음 회차: 최종 응답 예산(320B 예약/진단 추가)·ReadFile 192KiB 절단은 기존 별도 과제; 릴리즈에는 긴 줄의 UTF-8 경계 보존 수정으로 설명할 것.
- [러너 11:18] review approved — 리뷰 승인 (risk=low)
- [러너 11:18] pr created — https://github.com/hkjang/git-ctx/pull/36
