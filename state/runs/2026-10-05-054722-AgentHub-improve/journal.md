# 회차 노트 2026-10-05-054722-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:47] base pinned — main@68d4f7c
- [러너 05:47] autonomy release — 

## 정찰 노트
- 제공자 출처 길이 제한을 선택: 이전 회차가 닫은 목록/스니펫과 별개로 originOf→PolicySources에 무제한 경로가 남고 프로덕션 1파일로 끝난다. 가이드 요약/안내보다 사용자 영향이 크며 인증·이미지·CI 변경을 피한다.
- brief 초안 선작성 후 경계·비활성 필드·긴 URL 경로 계약을 보완했다. 세 요청 스킬은 Skill 도구 부재로 로컬 SKILL.md를 읽어 적용했다.
- 확인: Go tracking/api 통과, Node 회귀 130건 통과, guide-shots 구문 검사 통과. DB live/전체 CI 미검증, 300룬의 운영 적합성과 기존 주석의 24339바이트 수치는 이번 새 측정 아님.
- 주의: 룬 상한을 전체 헤더 바이트 보장으로 쓰지 말 것. 과대 기존 저장값의 읽기/마이그레이션은 범위 밖이며 두 파일의 제공자 무제한 주석만 현행화할 것.
- [러너 05:52] scout done — Momento·Matomo 제공자 주소에서 CSP로 전달하는 출처 길이를 제한한다 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- 6a7c0f1: 정찰 과제 채택. tracking.go 한 파일에 제공자 출처 300룬 검사 추가, 테스트 두 파일과 무제한 설명 주석 갱신.
- 요청한 세 technology 스킬은 Skill 도구 부재로 로컬 SKILL.md를 읽어 적용. 수정 전 실패 → 수정 후 통과 → 검사 제거 재실패 → 복구 확인.
- 두 제공자의 8000자 호스트가 24339바이트 CSP를 만들고 Validate/API 검증에서 거절됨; 최대 300룬 출처는 같은 nonce의 none 기준 정확히 903룬 증가.
- go test ./internal/tracking ./internal/api로 시작·마감(마지막 tracking cached / api 1.697s), go test -race ./cmd/... ./internal/... 및 diff·포맷 검사 통과.
- 확신 없는 곳·미검증: 300룬의 운영 적합성은 운영 데이터로 확인 못 함. DSN 없어 DB live 미실행; API 검증 함수 테스트를 HTTP/DB 저장 증명으로 취급하지 말 것.
- 일부러 제외: 기존 저장값 재검증·정리, 전체 CSP 바이트 상한, 프런트·문서·릴리즈. 기존 멀티바이트 33513/42309바이트 허용 계약 유지.
- 다음 역할 주의: 기존 과대 제공자 필드가 있으면 다른 설정 쓰기도 새 오류로 거절된다. 경로·쿼리·포트 처리와 originOf/Normalized/PolicySources/Snippet은 변경하지 않았다.
- [러너 05:58] brief accepted — 채택 — 현재 Validate의 제공자 길이 검사 누락과 originOf→PolicySources→pagePolicy의 세 지시문 증폭이 일치했고, 지정 과대 입
- [러너 05:58] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- approve / low, security·legal 차단 없음. 세 요청 스킬을 로컬 SKILL.md로 적용하고 main...HEAD 세 파일·커밋·원장의 수정 전 실패 출력을 확인했다.
- 경계·모드·긴 경로·API 검증/CSP 생성 및 관리자 쓰기 경계를 검토했다. go test -count=1 ./internal/tracking ./internal/api와 diff --check 통과; 저장소 코드 수정 없음.
- DB/live 환경 부재로 실제 HTTP/DB 저장·사용자 간 격리 미검증. 전체 race CI·브라우저·배포는 재실행하지 않았고 300룬 운영 적합성은 미확인이다.
- 릴리즈·후속 주의: 기존 과대 URL은 읽을 때 남고 추적 설정 재저장·한 번 클릭 허용을 막는다. 전체 CSP 바이트 상한과 저장값 정리는 별도 과제다.
- [러너 06:01] review approved — 리뷰 승인 (risk=low)
- [러너 06:01] pr created — https://github.com/hkjang/AgentHub/pull/42
- [러너 06:03] ci passed — 검사 1개 모두 success
- [러너 06:03] merge done — 6a7c0f1
- [러너 06:11] release published — v0.259.0
- [러너 06:17] assets verified — v0.259.0 자산 8개 (이전 v0.258.0: 8)
