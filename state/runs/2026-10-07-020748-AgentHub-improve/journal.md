# 회차 노트 2026-10-07-020748-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:07] base pinned — main@49c4a1f
- [러너 02:07] autonomy release — 
- [러너 02:49] improve timeout — 단계 제한 시간 초과
- [러너 02:49] improve error — error: agent produced no result (TIMEOUT )
- [러너 02:49] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 확인: 일회용 PostgreSQL 로 live 를 직접 돌려 main 의 tracking.go 에서 PUT 200 저장 + /runs CSP 의 script-src 4회(앞 셋 비어 있음)를 재현했고, 고친 뒤 통과를 확인했다. 새 테스트 셋은 수정 전 전부 실패한다 — 대상을 진짜 지난다.
- 확인: url.Parse 가 Host 에 ';'·',' 를 남기고 공백·탭·줄바꿈은 거절한다는 주석의 주장을 ASCII 전수로 검증했다. 경로·질의의 분리자는 origin 에 남지 않아 과대거절도 없다. SnippetOrigins·허용 목록까지 훑어 이 종류의 구멍이 더 없음을 확인했다. go test -race ./cmd/... ./internal/... 전부 통과, gofmt·vet 깨끗.
- 못 봄: web 단위·브라우저 e2e·클러스터·Keycloak·SMTP(diff 가 Go 2개 패키지뿐이라 생략). improve 단계가 TIMEOUT 이라 구현 원장이 비어 있어 재현·검증을 내가 직접 했다.
- 승인이나 남는 우려(릴리즈 노트): 이미 저장된 분리자 포함 주소는 수리되지 않는다 — 헤더는 관리자가 고칠 때까지 깨진 채 나가고 그 사이 설정 PUT 과 원클릭 allow 가 400 이 된다. 기존 길이 상한과 같은 동작.
- 다음 회차 거리: 비ASCII 공백(U+00A0·U+2028·U+3000·U+FEFF)이 Host 를 통과해 출처에 남는다 — CSP 는 ASCII 공백으로만 가르므로 주입이 아니라 무의미한 출처일 뿐이라 차단하지 않았다. TrackingForm 의 안내·maxLength 부재도 그대로다.
- [러너 02:54] review approved — 리뷰 승인 (risk=low)
