# 회차 노트 2026-09-24-095429-releasedock-improve — releasedock
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:54] base pinned — main@31f64a9
- [러너 09:54] autonomy release — 

## 구현 노트
- 두 SSE 로그 스트림(simple.go, releases.go)이 `LIMIT 500` 페이지를 꽉 채운 뒤에도 폴링 틱 1초를 기다려, 저장이 끝난 2,600행이 화면에 닿는 데 6초가 걸렸다. 가득 찬 페이지면 곧바로 다음 질의를 하도록 바꿨고 end 조건과 keepalive·max_duration은 그대로다.
- 확신 없는 곳: 타이밍 단언(2,600행을 3초 안에)은 수정 전 6.0초/후 1.4초로 여유가 크지만 극단적으로 느린 CI에서는 흔들릴 수 있다. 빠른 경로에서는 maxDuration 타이머를 검사하지 않는다 — 배수는 런의 로그 저장 상한(8 MiB)으로 유한하고 느린 클라이언트는 Fprintf에서 막히므로 30분을 넘기지 않는다고 판단했으나 실측하지는 않았다. 스핀 방지는 조건문과 테스트 산문으로만 고정했고 질의 횟수를 세는 테스트는 없다.
- 일부러 하지 않은 것: 웹 쪽 스트림 개선(EventSource error 표시 등)은 미머지 브랜치 auto/2026-09-23-1104이 같은 파일을 고치고 있어 충돌을 피해 손대지 않았다. Makefile vet 타깃(auto/2026-09-17-0853)도 같은 이유로 건드리지 않았다.
- 다음 역할이 조심할 것: 새 테스트 3건은 모두 TEST_POSTGRES_DSN이 있어야 돌고 없으면 조용히 Skip한다(도커 PostgreSQL 16으로 검증). 테스트 픽스처는 New()로 서버를 만든다 — `&Server{}` 리터럴로 스트림 라우트를 타면 streams 맵이 nil이라 패닉한다(프로덕션 경로는 안전).
- [러너 10:08] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve (risk low, blocking 없음). 확인: 도커 PostgreSQL 16으로 수정 전/후를 나눠 돌려 신규 테스트 2건이 수정 전 6.0초로 실제 실패하고 수정 후 1.4초로 통과함을 봤다(세 번째는 수정 전에도 통과하는 불변식 가드). backend `go test ./...`·runner `go test ./...`·`go vet ./internal/server/` 모두 DSN 지정 상태로 통과. 검증 후 워크트리와 인덱스를 HEAD로 되돌려 깨끗함을 확인했다.
- 구현자가 의심한 자리 검증: 스핀 불가는 코드로 성립한다(sent++ 직전 lastID=id + ORDER BY id 단조 증가). 배수 유한성 주장도 양쪽에서 확인 — simple.go:39 `maxSimpleRunLogBytes = 8<<20`, runner/internal/pipeline/pipeline.go:106 `NewBudget(job.Profile.MaxLogBytes)`. 따라서 releases.go:1532 교차 참조 주석은 사실에 맞다.
- 못 본 것: 웹(변경 없음)과 실제 브라우저 SSE 동작, 질의 횟수를 세는 스핀 회귀 테스트(존재하지 않음), 수십만 행 규모 부하.
- 승인이어도 남는 우려 두 개. (1) 빠른 경로가 maxDuration(30분)을 검사하지 않아 가득 찬 페이지가 계속 이어지면 30분 상한이 적용되지 않는다 — ctx.Done은 매 회전 검사하고 스트림은 3/64로 제한되므로 차단 사유는 아니지만 릴리즈 노트에 적을 값. (2) 타이밍 단언 여유가 3초 대비 1.4초(약 2.2배)뿐이라 경합이 심한 CI에서 흔들릴 수 있다.
- 다음 회차: 이 브랜치는 마이그레이션·권한·워크플로·VERSION을 건드리지 않아 revert 한 번으로 완전히 되돌아간다.
- [러너 10:13] review approved — 리뷰 승인 (risk=low)
- [러너 10:13] pr created — https://github.com/hkjang/releasedock/pull/23
- [러너 10:15] ci passed — 검사 1개 모두 success
- [러너 10:15] merge done — 5981766
- [러너 10:21] release published — v0.5.19
- [러너 10:24] assets verified — v0.5.19 자산 2개 (이전 v0.5.18: 2)
