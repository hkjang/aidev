# PR 처리기 노트 2026-09-23-113745-relio-shepherd — relio PR #29
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-23-110438-relio-improve)
# 회차 노트 2026-09-23-110438-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:04] base pinned — main@38c88ce
- [러너 11:04] autonomy low-risk — 롤백 PR 

## 정찰 노트
- `rows.Err()` 누락을 골랐다: 값이 실제 오동작(끊긴 목록을 200 으로 내보냄)이고, 이 저장소의 절반 이상이 이미 검사하고 있어 새 관례가 아니라 빠진 자리를 메우는 일이다. ClientIP/XFF(4/3/M)는 신뢰 경계 결정이 필요해 6회차째 보류했고, 남은 S 후보(감사 부제·allowed_origins 시드)는 가치가 1~2 라 단독 회차로 약했다.
- 증명 방식이 이 과제의 핵심 판단이다. `Server.DB` 가 `*pgxpool.Pool`(server.go:41) 이라 가짜 행을 먹일 수 없어 단위 테스트로 스트림 오류를 재현할 수 없다. 대신 `internal/crm/search_test.go:43` 이 이미 `internal/` 전체를 훑는 소스 스캔 테스트라는 선례를 확인하고 그 모양을 지정했다.
- 추측으로 적은 것(미확인): 새 테스트 파일을 `internal/platform/database/` 에 두라고 했으나 그 패키지는 `database.go` 하나뿐이고 `filepath.Walk` 상대 경로 깊이(세 단계)를 실행해 보지 않았다. 고칠 자리 목록도 파일별 개수 비교 + 의심 자리 육안 확인이라 빠진 곳이 더 있을 수 있어, red 일 때 테스트가 찍는 `path:line` 을 정답으로 삼으라고 적었다.
- 구현자가 조심할 것: `intelligence/health.go:255`·`:278` 은 `return out` 이라 오류를 돌려줄 시그니처가 없다 — 여기서 호출부까지 번지면 M 이 L 이 된다. 그럴 때 조용히 화이트리스트를 늘리지 말고 스캔 범위를 정직하게 좁히라고 명시했다. `admin.go:122` 는 `rows` 변수를 재사용해 145행 Err 가 두 번째 루프만 검사하는 자리라 개수 비교로는 안 걸린다.
- 회피 이력: 최근 기각·무변경이 난 유형(가이드 캡처/Chrome 탐색, 브라우저·문서 회차)은 후보에서 뺐고 ideas.json 에 "세 번째로 내지 말 것" 으로 남겼다.
- [러너 11:08] scout done — 행 반복이 끝난 뒤 `rows.Err()` 를 검사해 스트림 오류를 조용한 부분 응답으로 삼키지 않기 (가치 4 / 위험 2 

## 구현 노트
- 무엇/왜: `for rows.Next()` 뒤 `rows.Err()` 가 없던 13자리를 고쳐 끊긴 스트림이 200+부분 목록이 아니라 기존 오류 경로(핸들러 `s.serviceError`, 서비스 `err` 반환)로 나가게 했다. 증명은 go/ast 소스 스캔 테스트 `internal/platform/database/rows_err_test.go`(red 13자리 → green, `admin.go:122` 되돌리면 그 줄만 재실패 확인). 커밋 9f713a0, 순수 추가 56줄.
- 확신 없는 곳: **실제 DB 로 끊긴 스트림을 재현하지 못했다**(`Server.DB` 가 `*pgxpool.Pool` 이라 가짜 행 주입 불가, 이번 회차에 PostgreSQL 미기동). 즉 "500 이 실제로 나간다" 는 코드 경로 독해이지 실행 증거가 아니다. 또 스캔 테스트의 판정 규칙(같은 블록 안에서 변수 재할당 전까지만 `<name>.Err()` 를 찾음)은 루프가 `if`/클로저 안에 있고 검사가 바깥 블록에 있는 형태를 false positive 로 찍을 수 있다 — 현재 트리에는 그런 자리가 없어 드러나지 않았을 뿐이다.
- 일부러 안 한 것: `internal/intelligence` 의 `contactRoles`·`stageLimits`(health.go:255·278) 는 주석이 best-effort 라고 설계 의도를 명시하고 오류를 돌려줄 시그니처가 없어 스캔 범위에서 뺐다 — 화이트리스트를 늘리는 대신 `scannedPackages` 를 명시 목록으로 두고 테스트 주석에 이유를 적었다. `pgxpool.Pool` 인터페이스화 리팩터, `auditRow`/`auditItem` 재포매팅, 프런트는 열지 않았다.
- 다음 역할이 조심할 것: 이 테스트는 DB 없이 돌지만 **저장소 트리 상대경로**(`../../../internal`)에 의존한다 — 패키지를 옮기면 깨진다. `scannedPackages` 에 없는 패키지는 검사되지 않으니, 새 패키지가 생기면 목록에 더해야 한다(현재 `mail` 은 main 에 없어 제외). 동작 변화가 있는 변경이다: 스트림 오류 시 200→500. 행이 0개인 정상 경우는 `Err()` 가 nil 이라 빈 배열 200 그대로다.
- [러너 11:12] brief accepted — 채택 — 과제서가 지목한 자리와 근거(`admin.go:122` 의 rows 재사용, `:336` 의 기존 검사, 85 대 70 셈)가 코드와 전부 일치했고,
- [러너 11:12] verify passed — 검증 9개 통과 (auto)
- [러너 11:27] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 11:27] pr created — https://github.com/hkjang/relio/pull/29

## 심사 노트
- 확인한 것: 프로덕션 훅 7파일을 `git apply -R` 로 되돌려 테스트가 13건 정확히 red(고친 자리와 1:1 일치), 복원 후 green 임을 직접 재현했다. go build/vet/gofmt/go test ./... 전부 통과. 호출부(server/crm.go:76·211, mcp/server.go 5곳, relationship/account.go:187, crm/advanced.go:269, admin_keys.go:39) 가 err 를 모두 전파함을 확인했고, 조기 반환의 rows.Close()/defer·tx.Rollback 누수 없음을 각 자리에서 읽었다. 제외된 internal/{api,config,webui,platform} 에 미검사 루프가 없음(platform/secrets/integrity.go:102·124 는 이미 검사)도 확인해 남은 구멍은 주석에 적힌 intelligence 하나뿐이다.
- 구현 노트의 "확신 없는 곳" 처리: 소스 스캔이 증거라는 점은 저장소 선례(internal/crm/search_test.go:42 의 동형 문자열 스캔)의 AST 판형이고, 변경 자체가 자리마다 기존 오류 경로를 4줄 호출하는 것이라 배선 여지가 없어 수용했다. checksErrAfter 의 false positive 형태(루프는 if 안, 검사는 바깥)는 현 트리에 해당 자리가 없음을 확인했다.
- 못 본 것: 실제 PostgreSQL 로 끊긴 스트림을 재현하지 못했다(pgxpool.Pool 주입 불가, DB 미기동) — non-nil rows.Err() 는 pgx 계약에 의존한 판단이다. 프런트는 열지 않았다.
- 정정 1건: 회차 노트의 "200→500" 은 틀렸다. serviceError(server.go:662) 는 기본 400 이고 status 는 404/403/409 로만 바뀌어 :683 의 `status>=500` 로깅 분기가 도달 불가라, 스트림 오류는 400 으로 나가고 서버 로그에 남지 않는다. 단 같은 함수의 기존 rows.Scan 오류 경로와 동일한 처리라 이 PR 이 만든 노출면이 아니고 커밋 메시지의 주장은 정확하므로 note 로만 남겼다.
- 권고 근거: 결함 없음, 순수 추가 229줄, 마이그레이션·인증·권한·의존성 변화 없어 revert 로 완전 복원. 보안·법무 차단 소견 없음 → approve / merge / risk low.
