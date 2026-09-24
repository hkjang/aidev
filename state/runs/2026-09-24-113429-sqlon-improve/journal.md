# 회차 노트 2026-09-24-113429-sqlon-improve — sqlon
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:34] base pinned — main@57f99b7
- [러너 11:34] autonomy release — 
- [러너 11:38] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- `internal/mcp/execguard.go`의 `cacheKey`가 binds를 무시해 `POST /api/query`가 같은 SQL·다른 `binds` 요청에 앞선 파라미터의 행을 `cached:true`로 돌려줬다. `cacheKey(profile, sql, dbconn.ExecOptions)`로 바꿔 Binds를 키에 포함(커밋 57563cf).
- 확신 없는 곳: **실제 DB로 캐시 히트를 재현하지 못했다.** 이 환경엔 PG/MySQL이 없어 스텁 드라이버뿐이라, 재현은 프로덕션 `cacheKey`로 캐시를 심고 프로덕션 `executeGuarded`를 두 번 호출하는 방식이다(캐시 히트/미스 판정과 키 계산은 전부 프로덕션 코드, 대역은 없음). 히트 경로는 진짜지만 "DB가 실제로 다른 행을 준다"는 마지막 한 칸은 코드 독해(`manager.go:395 db.QueryContext(qctx, limited, opts.Binds...)`)로만 확인했다.
- 바인드 인코딩은 `%T=%#v`다. JSON 유래 값(float64/string/bool/nil)에는 안전하지만, 앞으로 누가 `%#v`가 비결정적인 타입(포인터 등)을 binds에 넣으면 같은 값이 키를 달라지게 해 **캐시 미스**가 난다 — 안전한 방향이나 성능 회귀로 보일 수 있다.
- 일부러 안 한 것: (1) `POST /api/query/async`가 binds를 아예 전달하지 않는 별개 결함 — 요청 스키마 확장이라 범위 밖(ideas.json에 pending). (2) 프로파일 정책(DefaultMaxRows)이 실행 중 바뀌면 캐시된 행 수가 낡는 엣지 — 이번 결함과 무관. (3) 과제서의 "이번 선택"(PUT visibility 보존)은 2026-09-22에 이미 성공한 과제라 기각.
- 다음 역할이 조심할 것: `CHANGELOG.md`는 LF/CRLF 혼합(157줄 중 111줄 CRLF)이라 편집 도구로 고치면 전체 줄바꿈이 뒤집혀 99줄 diff가 난다 — 이번엔 바이트 단위 삽입으로 7줄만 바꿨다. 새 테스트 2개는 DB 없이 돌며, 미스 경로 단언은 "결과도 오류도 없으면 실패"로 완화해 실 DB가 있는 환경에서도 깨지지 않게 했다(버그 재주입 시 여전히 RED 확인).
- [러너 11:44] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인함: cacheKey 본문을 버그 버전으로 되돌리는 돌연변이 검증(신규 테스트 2개 RED→복원 후 GREEN), %#v의 map 정렬·NUL 이스케이프 결정성, JSON 바인드 타입 범위(dbapi.go:203), build/vet/test/-race 전부 통과, 워크트리 clean 복원.
- 못 본 것: 실제 PG/MySQL 통합 실행(-tags integration), 브라우저 UI, 장시간 TTL 만료 동작. 구현자가 남긴 "마지막 한 칸"(실 DB가 바인드별로 다른 행을 준다)은 나도 코드 독해(manager.go:397)로만 확인했다.
- 승인이어도 남는 우려(릴리즈 노트용): ApprovePlan이 캐시 키에서 빠져 있어 승인 실행 결과가 TTL 동안 미승인 호출자에게 히트할 수 있다(기존 동작, 플랜 게이트는 비용 가드이지 인가 아님). 캐시는 프로파일 단위이므로 향후 역할별 PII 마스킹을 넣으면 키에 역할을 추가해야 한다.
- 다음 회차: /api/query/submit가 요청의 binds를 조용히 무시하는 결함(dbapi.go:807-832)은 여전히 열려 있다. 범위 밖 기각은 타당했고 ideas.json에 보류 중.
- 판정: approve / risk=low / blocking 없음.
- [러너 11:48] review approved — 리뷰 승인 (risk=low)
- [러너 11:48] pr created — https://github.com/hkjang/sqlon/pull/12
- [러너 11:52] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
