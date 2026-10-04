# PR 처리기 노트 2026-10-05-073730-Clustara-shepherd — Clustara PR #36
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-05-054727-Clustara-improve)
# 회차 노트 2026-10-05-054727-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:47] base pinned — main@386f21b
- [러너 05:47] autonomy release — 

## 정찰 노트
- 이벤트 GET의 실효 상한·포화 메타데이터를 선택: 프로덕션 2파일로 끝나며, 직전 notify/PSS 변경과 겹치거나 보안 신호를 바꾸는 후보보다 위험이 낮다.
- 기존 코드 경로 확인 및 NotifyScan 기준선 테스트 PASS(1.987s); 신규 K8sEvents HTTP 재현·외부 클라이언트의 추가 필드 허용은 미확인, 구현자가 검증한다.
- 500행 포화는 누락 확정이 아니다. 상한·타 호출자·auth/DDL/릴리즈는 유지하고 실 SQLite+Routes로 검증한다.
- 회사 스킬 3개는 전용 도구 부재로 로컬 원문을 읽었다. 초안 작성 후 확정본으로 갱신했으며 프로필·후보 15개(신규 2개, 완료 재확인 3개)를 기록했다.
- [러너 05:52] scout done — 이벤트 목록 API의 실효 조회 상한과 창 포화 여부 명시 (가치 2 / 위험 1 / 작업량 S)

## 구현 노트
- 채택·완료: events GET에 실효 limit/window_full 및 포화 시 window_notice 추가; 공유 K8sEventQueryLimit으로 스토어와 같은 기본100/상한500 적용. 프로덕션 2·테스트 1·문서 1파일, 커밋 0c79c53.
- 실패 재현: 실 SQLite+Routes의 limit=1000 요청에서 500행을 받고도 limit 누락으로 실패. 첫 인증 테스트의 기대 코드는 실제 Routes 미들웨어의 authentication_required로 정정했으며 인증 코드는 변경하지 않음.
- 인과 확인: 응답 맵에서 메타데이터만 잠시 빼 같은 테스트 재실패 후 복원. 성공 창 18케이스·오류 3케이스가 실제 DB 조회 행 수·정렬·필터·기본값·빈 배열을 확인.
- 검증: gofmt 출력 없음; 좁은 K8sEvents|NotifyScan 회귀 ok 2.314s, go build ./...·go vet ./... 종료 0, go test ./... -count=1 모두 통과(proxy 67.557s, store 16.214s).
- 확신 없는 곳·검증 못 한 것: 실제 PostgreSQL, 브라우저 실행, 추가 JSON 필드를 거부하는 외부 클라이언트 호환은 미검증.
- 일부러 하지 않은 것: count/has_more/페이지네이션·UI·다른 events/revisions 호출자·릴리즈는 범위 밖. 정확히 500행도 포화 true이며 누락을 확정하지 않음.
- 다음 역할 주의: 테스트는 임시 SQLite 파일과 실제 Routes를 사용하고 DB 오류 케이스에서 테스트 DB를 닫는다. 상한 초과+1 조회로 정확한 누락을 추정하지 말 것. 원장·아이디어·과제서 체크포인트 기록 완료.
- [러너 05:59] brief accepted — 채택 — 핸들러의 무메타데이터 응답과 스토어 500 제한을 현 코드 및 수정 전 실제 HTTP 실패로 확인했고 지정한 4파일 범�
- [러너 06:00] verify passed — 검증 3개 통과 (policy)
- [러너 06:00] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 06:00] pr created — https://github.com/hkjang/clustara/pull/36

## 심사 노트
- 확인: origin/main 프로덕션 2파일을 작업 트리에 되돌려 같은 테스트를 돌려 limit 누락으로 8케이스 FAIL → HEAD 복원 후 21케이스 PASS. 테스트는 실 SQLite+server.Routes() 실 HTTP 로만 검증하며 대역이 없다.
- 확인: gofmt 무출력, go build/go vet 0, go test ./... -count=1 19패키지 전부 ok. 문서·릴리즈 게이트(ReleaseGateDocVersion/Changelog*/AppVersion/OpenAPISwagger) 별도 재실행 PASS, 버전·CHANGELOG 미수정.
- 확인: 핸들러와 스토어가 store.K8sEventQueryLimit 하나를 공유하고 boundedLimit 이 멱등이라 이중 적용이 안전하며, 테스트가 미정규화 limit 으로 조회한 실 스토어 결과와 DeepEqual 로 두 경로를 맞춘다. 문서 1061행은 교체뿐이고 정본이 둘이 되지 않는다.
- 못 본 것: 실제 PostgreSQL·실제 K8s·브라우저 SPA 렌더, 알 수 없는 JSON 필드를 거부하는 외부 클라이언트. 앞의 셋은 이 변경이 손대지 않은 영역이고 저장소 내 유일 소비자 admin_ui.go:10693 은 events.events 만 읽어 영향 없음.
- 권고 merge/risk low 근거: 인증·권한·DDL·마이그레이션·의존성 변경이 없는 읽기 전용 GET 의 가산 응답 필드이고 커밋 1개 revert 로 완전 복구된다. 차단 소견 없음(보안·법무 공격 경로 미발견); window_full 의 정확히 limit 개 true 는 문서화된 계약이라 결함 아님.
