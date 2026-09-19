# 회차 노트 2026-09-19-145336-jasql_semantic-improve — jasql_semantic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:53] base pinned — main@149edbe
- [러너 14:53] autonomy release — 

## 정찰 노트
- 'hold: budget' 은 러너 예산 소진이지 저장소 스크립트·테스트 실패가 아니다. `.github/workflows` 없음, `go test ./...` 는 main@149edbe 에서 전부 통과(실측 67s). 저장소에서 유일하게 예산에 닿는 요인이 단계마다 도는 60초 넘는 테스트라서, 게이트를 건드리지 않고 프로덕션 검색 핫패스(search.go:285 sampleBoostForTable, 테이블×샘플 전수 순회)를 빠르게 하는 수정 과제로 잡았다. 문서만 고치는 후보는 예산 문제에 효과가 없어 차선으로 뒀다.
- 확신 없는 곳: pprof 가 승인 문제로 막혀 핫스팟은 정적 분석으로 추정한 것(미확인). `c.Samples` 가 런타임에 갱신되는 경로 유무도 미확인 — 구현자가 grep 할 것. `jasql-eval` CLI 는 실행해 보지 않았다.
- 구현자 주의: 지표·순위가 1건이라도 바뀌면 안 된다(stable sort 동점 민감). 임계값·skip·골든셋 축소로 시간 줄이기 금지. before/after 시간을 실측해 PR 에 적고, 효과 없으면 넣지 말 것. 예산 hold 가 그래도 반복되면 러너 설정 문제로 원장에 남길 것.
- [러너 14:59] scout done — `SearchSchema` 의 테이블×샘플 전수 재계산을 없애 골든셋 테스트와 실사용 검색을 빠르게 (가치 4 / 위험 2 / 

## 구현 노트
- 바꾼 것: `SearchSchema` 핫패스 캐시화 — Load 시 few-shot 샘플 인덱스(`Catalog.samples`: 소문자 텍스트 + 테이블별 멤버십)와 컬럼 소문자 캐시(`Column.lower`)를 만들고 `newSampleScorer`/`scoreColumns` 가 이를 읽는다. 규칙·가중치·상한·정렬은 그대로. 커밋 f59ed6f, 브랜치 auto/2026-09-19-1453.
- 검증: pprof 로 핫스팟 확인(scoreColumns 59%, sampleBoostForTable 30%) → 80 골든 질문 × (SearchSchema 4종·RetrieveContext·SchemaContext·SearchSamples·두 평가 요약) 전후 JSON 덤프가 동일(map 순서로 원래 흔들리는 gated excluded 5건·"joinable with" 사유만 제외, 같은 코드로 두 번 돌려 그 필드가 흔들림을 확인). 골든 테스트 43.3s→9.7s, `go test ./...` 67s→~25s, 지표 table .94/join .94/column .90/metric 1.0 동일.
- 확신 없는 곳: (1) `c.Samples`/컬럼 필드가 Load 뒤에 바뀌는 경로는 grep 상 없음(overrides·stats·learned rules 모두 Load 안, 리로드는 catalog.Load 재호출) — 향후 런타임에 Samples 를 append 하는 코드가 생기면 `c.samples` 재빌드가 필요하다. (2) 새 동등성 테스트 `TestSampleBoostIndexMatchesLegacyScan` 이 4s 로 무겁다(옛 구현 verbatim 스캔) — 더 줄이려면 질문 수(6)를 낮추면 된다.
- 일부러 안 한 것: scoreColumns 지연(게이트 리포트가 컬럼 점수 포함값이라 출력이 바뀜), scoreTable 소문자 캐시(잔여 14%, 범위 확대 회피). 임계값·skip·골든셋 크기는 손대지 않음.
- 인프라: 회차 중 `/mnt/c/Users/USER/projects/jasql_semantic` 이 사라져 워크트리 `.git` 링크가 끊겼다(러너 로그 1678행). `projects/jasql`(149edbe 포함, remote semantic=../jasql_semantic) 을 로컬 clone 해 워크트리를 독립 저장소로 복구했으므로 커밋은 살아 있지만, 러너의 다음 단계(`git -C $repo …`, push)는 $repo 부재로 실패할 수 있다. 'hold: budget' 자체는 run.sh:1659 `budget_ok` 일일 상한 — 저장소 밖 문제.
- 다음 역할 주의: 골든셋 테스트는 `data/kcb/golden_queries.json` 없으면 skip. `go tool pprof` 는 이 환경에서 잘 돈다(`-cpuprofile` + `-o` 로 바이너리 지정).
- [러너 15:14] brief accepted — 채택 — 과제서의 핫스팟 추정은 pprof 로 확인했더니 scoreColumns(59%) 가 sampleBoostForTable(30%) 보다 컸고, 둘 다 같은 방식(로드
- [러너 15:14] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 판정 approve, risk low, blocking 없음: 변경 전체·캐시 생성 순서·런타임 갱신/리로드·점수 연산·테스트 단언 확인, 실제 결함 미발견.
- go build ./..., go vet ./..., go test -count=1 ./... 통과(catalog 24.352s, mcp 8.698s); 골든셋 파일 존재 확인.
- 전후 80문항 덤프·pprof·성능 비교·race는 재현하지 않음. 컬럼 테스트는 공통 변환 함수를 공유하며 향후 Load 후 직접 필드 변경에는 캐시 재생성이 필요.
- 원본 저장소 경로 부재가 지속되어 릴리즈 러너 복구 필요; 프로필/개발 문서의 외부 의존성 0 설명도 현 go.mod와 불일치(기존 문제).
- [러너 15:16] review approved — 리뷰 승인 (risk=low)
- [러너 15:16] pr create-failed — gh pr create 실패
