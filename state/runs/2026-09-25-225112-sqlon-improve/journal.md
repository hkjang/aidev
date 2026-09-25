# 회차 노트 2026-09-25-225112-sqlon-improve — sqlon
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:51] base pinned — main@57f99b7
- [러너 22:51] autonomy release — 
- [러너 23:00] scout timeout — 단계 제한 시간 초과
- [러너 23:00] scout done — `GET /api/metrics` 가 인증 없이 DB 프로파일 인벤토리(pools·breakers 키)를 노출하는 결함 수정 (가치 4 / 위험 2 / 

## 구현 노트
- `internal/mcp/dbapi.go:870` 의 `GET /api/metrics` 에 `requireActor` 한 줄을 앞세웠다 — 이 핸들러만 게이트 없이 `Snapshot()` 을 돌려줘 미인증 요청자가 `pools`·`breakers` 맵 키(= 비공개 포함 DB 프로파일 ID)를 받아 갔다. 형제 `GET /api/db/alerts` 와 같은 게이트이며 역할은 새로 요구하지 않는다.
- 확신 없는 곳·검증 못 한 것: 실제 브라우저에서 화면을 띄워 보지 않았다(코드 검색상 HTML/JS 어디도 `/api/metrics` 를 부르지 않아 UI 영향은 없다고 판단). 실 DB·컨테이너 통합 테스트는 이번 회차에 돌리지 않았다(변경이 HTTP 게이트 한 줄이라 무관).
- 과제서 수용 기준 3) 의 "AdminToken 이 설정된 단독 모드에서는 토큰 없는 요청이 거절된다" 는 틀렸다 — `auth.go:164` 의 `requireActor` 는 단독 모드에서 토큰과 무관하게 즉시 `nil, true` 다. 테스트는 실제 계약(단독 모드는 토큰 유무와 무관하게 200, `/api/db/alerts` 와 동일)을 단언한다. 비평가는 여기를 먼저 볼 것.
- 일부러 하지 않은 것: `GET /metrics`(Prometheus) — 스크레이퍼가 세션 쿠키를 못 보내 게이트를 걸면 관측이 끊긴다(ideas.json 에 별건 기록). `openapi.go:593` — 해당 항목에 `security` 필드가 아예 없어 과제서 지시("없으면 건드리지 말 것")대로 두었고, 그 결과 문서가 이제 과소 기술 상태임을 ideas.json 에 남겼다. 캐시 키(execguard.go:88)·async prune·submit binds 세 건은 과거 성공 재구현 금지로 손대지 않았다.
- 다음 역할이 조심할 것: `CHANGELOG.md` 는 CRLF/LF 혼합이라 편집 도구 대신 바이트 삽입으로 고쳤다 — 손댈 때 `git diff --numstat` 로 추가 줄 수를 반드시 확인할 것(이번엔 5/0).
- [러너 23:04] brief accepted — 채택 — 지정 결함을 현 HEAD 에서 실제 HTTP 로 재현하고 지정한 한 줄만 고쳤다. 다만 수용 기준 3)의 "`AdminToken` 이 설정된 
- [러너 23:04] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인함: dbapi.go 만 main 으로 되돌려 새 테스트가 실제로 실패(익명 200 + pools 스냅샷)하고 수정본에서 통과함을 재현. go vet ./... 및 go test ./... -count=1 전체 통과. requireActor 게이트가 /api/db/alerts:753·/api/query/history:760 과 동일함을 소스로 대조.
- 구현 노트의 "단독 모드에서 AdminToken 무관"주장은 auth.go:163-167 에서 사실 확인 — 과제서 수용 기준 3 이 틀렸고 테스트가 실제 계약을 고정한다. 중재는 재론하지 말 것.
- 못 본 것: 브라우저 실제 화면, 실 DB/컨테이너 통합 테스트, race 검사. 코드 검색상 /api/metrics 호출부가 없어 UI 회귀는 없다고 판단.
- 승인이어도 남는 우려(릴리즈 노트에 반영 권장): 같은 리스너의 무인증 GET /metrics 가 dbconn/manager.go:784 PrometheusText 로 동일한 pools 를 db_pool_*{profile="ID"} 라벨로 계속 노출한다. main 에 이미 있던 경로라 차단하지 않았으나 "프로파일 인벤토리 노출을 막았다"로 넓게 읽히지 않도록 별건 과제를 명시할 것.
- openapi.go:593 은 security 필드 누락으로 이제 과소 기술 상태(형제 /api/query/history 도 동일). 공격 경로 없음, 다음 회차 정리 대상.
- [러너 23:06] review approved — 리뷰 승인 (risk=low)
- [러너 23:06] pr created — https://github.com/hkjang/sqlon/pull/14
- [러너 23:11] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
