# 회차 노트 2026-10-18-165932-bbmcp-improve — bbmcp
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:59] base pinned — main@d71db92
- [러너 16:59] autonomy release — 

## 정찰 노트
- 결함 수정 + 묵은 테스트 공백을 한 변경으로 닫을 수 있어 골랐다. 세 회차째 차선이던 "Resolver TTL 캐시 테스트" 는 공유 fixture 의 CacheTTLSec=0 을 건드려야 해 위험했는데, 같은 캐시에 실제 결함(무제한 증가)이 있어 패키지 내부 단위 테스트로 DB 없이 red 를 만들 수 있다.
- 제친 후보: approval.Create 의 Marshal 오류 삼킴(표시 필드만 영향, 가치 2) / aiproxy 테스트(제공자 계약 미확인, M) / MCP 세션 404 검증(외부 규격 미확인, 다섯 회차째 보류 — 반복 기각 유형이라 다시 고르지 않았다).
- 추측으로 적은 것: 상한값 10000 은 근거 없는 제안값이다(구현자가 바꿔도 된다). "repository 인자만 바꿔 메모리를 늘릴 수 있다" 는 코드 판독 결론이고 런타임 재현은 하지 않았다 — 구현자는 단위 테스트의 len(cache) 로만 증명하면 된다.
- 반려 확인: BranchWritable 의 cfg, _ := 는 fail-open 이 아니다(settings.load 가 오류 시 FailClosed:true 기본값 반환). 이 사실을 ideas.json 에 rejected 로 남겼다.
- 구현자 주의: integration_test.go:246 의 perm.CacheTTLSec = 0 을 바꾸지 말 것. 회수 로직은 반드시 기존 mu 락 안에서, 만료 시각을 늘리지 않는 방향으로만.
- [러너 17:04] scout done — Resolver 권한 캐시의 무제한 증가를 막고 TTL 캐시 경로에 첫 테스트를 붙인다 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- `internal/permission/resolver.go:store_` 에 항목 상한(`maxCacheEntries = 10000`)과 `evictLocked` 를 넣었다: 상한에 닿았을 때만 만료 항목을 쓸어내고, 전부 유효하면 만료가 가장 이른 항목 하나를 비운 뒤 삽입한다. 다시 조회되지 않는 키(미존재 저장소도 None 결정으로 캐시됨)가 프로세스 수명 내내 남던 경로를 닫았다. `cached()`/`Reset()`/`Repository()`/`Project()` 와 `store_` 호출 지점은 그대로다.
- 확신 없는 곳: 상한값 10000 은 정찰의 제안값을 그대로 썼고 실측 메모리 근거는 없다. 또 캐시가 전부 유효 항목으로 꽉 찬 상태에서는 새 키 삽입마다 맵 전체(1만 항목)를 1회 순회한다 — 락 안이라 그 구간이 직렬화되는데, 1만 항목 순회 비용은 측정하지 않았다(업스트림 왕복 대비 무시할 수준이라 판단).
- 검증 못 한 것: "repository 인자만 바꿔 게이트웨이 메모리를 늘릴 수 있다" 는 코드 판독 결론이며 실제 MCP 호출로 런타임 재현하지 않았다. 증거는 단위 테스트의 `len(r.cache)` 뿐이다.
- 일부러 하지 않은 것: 백그라운드 janitor 고루틴(Resolver 에 종료 경로가 없다), 캐시 키에 mode/failClosed 추가(별건), `internal/tools/integration_test.go:246` 의 `perm.CacheTTLSec = 0` 수정(공유 fixture 전제라 손대지 않았다).
- 다음 역할 주의: 새 테스트 6건은 `package permission` 내부 단위 테스트로 DB·HTTP 없이 돈다. 그중 2건이 `maxCacheEntries + 100` 개를 삽입하므로 상수를 줄이면 테스트는 따라오고 늘리면 느려진다. 만료 테스트 2건은 20~40ms sleep 에 의존한다(극단적으로 느린 러너에서는 민감할 수 있음). 전체 회귀는 `TEST_DATABASE_URL` 이 있어야 하고 `-v` 로 `--- SKIP` 0건 확인이 필요하다 — 이번 실행은 142 PASS / SKIP 0 / FAIL 0.
- [러너 17:08] brief accepted — 채택 — 과제서의 근거(`cached()` 의 `delete` 가 유일한 회수 지점, `Reset()` 외 회수 경로 없음, `restRepository` 가 미존재 저장소
- [러너 17:08] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 확인: `-overlay` 로 main 의 resolver.go 를 되돌려 red 를 독립 재현했다 — 10100/12000 실패 수치가 원장 '실패 재현' 줄과 정확히 일치하고, 나머지 4건이 수정 전에도 PASS 라는 자기신고도 사실이었다. 전체 회귀 재실행 142 PASS / 0 SKIP / 0 FAIL, `-race` 12건 PASS, build·vet 무출력.
- 상한 보장(evict 후 len<=max-1 → 삽입 후 <=max), haveLive 가드, range 중 delete 안전성, 회수가 exp 를 건드리지 않음을 코드로 확인했다. store_ 가 now 를 락 **전에** 찍어 만료가 기존보다 짧아지는 쪽이라 권한 확대 방향 변화는 없다 — 보안·법무 차단 사유 없음(신규 개인정보 수집·의존성·비밀값 없음).
- 구현자가 의심한 두 자리를 시험했고 둘 다 통과: (1) 상한 10000 → 메모리 약 2MB 수준, (2) 전수 순회 비용 → 기본 TTL 60초에서 '전부 유효한 만원' 상태는 지속 166 miss/s(각 miss 가 REST 5+ 왕복)를 요구해 업스트림이 먼저 병목이라 실무상 도달 불가. **단 상한을 크게 올리면 이 전제가 깨지므로 그때는 배치 회수로 바꿔야 한다.**
- 못 본 것 / 남는 우려: 실제 MCP 호출 런타임 재현은 나도 하지 않았다(증거는 단위 테스트 len(cache) 뿐, 원장도 그렇게만 주장하므로 과장 아님). `TestEvictionNeitherExtendsExpiryNorRevivesExpired` 의 '만료 미부활' 단언은 항상 참이라 커버리지로 세지 말 것이고, 만료 테스트 2건의 20/40ms sleep 은 과부하 CI 플레이크 여지가 있다.
- 범위 밖: 워크트리에 커밋 안 된 `internal/webui/dist/index.html` 드리프트가 남아 있다(이 PR diff 에는 없음). 커밋된 dist 가 web/ 소스보다 낡았는지 다음 회차가 확인할 만하다. integration_test.go 의 `CacheTTLSec = 0` 금지선은 지켜졌다(변경 파일 2개뿐).
- [러너 17:12] review approved — 리뷰 승인 (risk=low)
- [러너 17:12] pr created — https://github.com/hkjang/bbmcp/pull/8
- [러너 17:14] ci passed — 검사 3개 모두 success
- [러너 17:14] merge done — 291b5b4
- [러너 17:21] release published — v0.2.13
- [러너 17:21] gh-release created — GitHub Release v0.2.13
- [러너 17:22] manifest ok — bbmcp-v0.2.13.tar.gz bbmcp-v0.2.13.tar.gz.sha256 bbmcp-deploy-v0.2.13.tar.gz bbmcp-deploy-v0.2.13.tar.gz.sha256 
- [러너 17:22] assets uploaded — 4개
- [러너 17:22] assets verified — v0.2.13 자산 4개 (이전 v0.2.12: 4)
