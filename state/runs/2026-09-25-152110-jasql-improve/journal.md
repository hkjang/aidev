# 회차 노트 2026-09-25-152110-jasql-improve — jasql
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:21] base pinned — main@5e7934f
- [러너 15:21] autonomy release — 

## 구현 노트
- 무엇/왜: docs/development.md 가 소스와 어긋나 있어(Go 1.24+ vs go.mod 1.25.0, "외부 의존성 0·go.sum 없음" vs godror/pgx/x-crypto, ~17초 vs 실측 ~70초, 도구 24 vs 실제 42, 코드 트리에 internal/meta·oracle 누락) 소스에서 하나씩 확인해 고치고 CI 절·goldgen 덮어쓰기 경고를 넣었다. 재드리프트 방지로 internal/mcp/docs_test.go 추가 — 문서의 숫자를 실제 ServeStdio tools/list 왕복 결과와 비교한다(커밋 2bf4e68).
- 확신 없는 곳: (1) "~70초" 는 이 머신 1 회 측정(catalog 56s + mcp 10s)이라 CI 러너에서는 다를 수 있다 — 대략치로 읽어 달라. (2) docs_test.go 의 정규식이 문서 표현에 결합돼 있다; 표현을 바꾸면 "no longer states" 로 빨개지는데, 이 실패 경로는 일부러 마커를 지워 red 를 확인했다. (3) development.md 의 확장 포인트·불변식·릴리즈 절 본문은 심볼 존재만 확인했고(registerAdmin, standalonePrevMonth, datasetLoadedCount, applyLearnedRules, personalPrepare, setCatalog, cleanIdent, dataMu, matchOne, SkippedRule 등 전부 실재) 그 절차를 처음부터 끝까지 따라 해 보지는 않았다. 릴리즈 절차는 scripts/release-image.sh 가 server.go 의 Version 을 읽는다는 것까지만 대조했고 실제 docker 이미지 빌드·적재는 미검증.
- 일부러 안 한 것: internal/oracle 의 gofmt 드리프트 2 파일(oracle_test.go, profile.go) — 이번에도 실재 확인했으나 문서 PR 의 diff 를 오염시키지 않으려 남겼다(ideas.json pending). docs/README.md 의 데이터셋 개수도 범위를 넓히지 않으려 손대지 않았다.
- 다음 역할이 조심할 것: **base 가 미머지 '성공' 커밋 2 개만큼 뒤처져 있다** — caf7a00(eval -verbose MISS), c712da1(timeparse 보고단위 순서). `git merge-base --is-ancestor <sha> HEAD` 로 확인했고 둘 다 HEAD 에 없다. 정찰 과제서는 caf7a00 과제를 현재 결함으로 제시했는데 근거는 맞지만 재구현하면 열린 PR 과 중복이라 기각했다. 이번 커밋은 이 둘과 겹치는 파일이 없다.
- 새 테스트는 DB 가 필요 없다. data/kcb 실데이터를 읽기만 하고(catalog.Load) 쓰지 않는다. goldgen 확인은 임시 디렉터리로만 돌렸고 `git status data/kcb` 가 비어 있음을 확인했다.
- [러너 15:27] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음. 새 테스트는 진짜다 — 문서의 42 를 41 로 바꾸니 `documents 41 tools, tools/list served 42` 로 red, 되돌려 green(git status 로 복원 확인). 문서 주장은 소스와 1:1 대조: go.mod 1.25.0, godror/pgx/x-crypto, `//go:build oracle` / `!oracle`, goldgen 기본 -out(main.go:33), release-image.sh:18 의 Version sed, 코드 트리·테스트 표의 파일 전부 실재, oracle_test.go 의 TestProfile* 로 '프로파일' 커버.
- 구현자가 의심한 자리를 먼저 쳤다: "~70초" 는 TZ=UTC 전체 go test ./... 로 재현(catalog 57.2s + mcp 10.0s) — CI 러너를 흉내낸 UTC 에서 timeparse 포함 전부 통과, go vet 도 클린. ci.yml timeout 15분은 여유.
- 승인이어도 남는 우려 2 개(릴리즈 노트에 쓰지 말 것, 다음 회차 참고): (a) development.md:31 "1.25.0 보다 낮은 툴체인으로는 빌드되지 않습니다" 는 GOTOOLCHAIN=auto 기본에서는 거짓(툴체인 자동 다운로드) — 오프라인/GOTOOLCHAIN=local 에서만 참. (b) :116 "docs_test.go 가 둘의 불일치를 잡습니다" 는 느슨함 — docs_test 는 문서↔served 만 비교하고 stdio_test 리터럴은 안 본다(결과적으로 둘 다 red 이긴 함).
- 못 본 것: 실제 GitHub Actions 러너 실행, docker 이미지 빌드·적재, `oracle` 태그 빌드, JASQL_TEST_PG PG 경로. 보안/법무는 diff 에 인증·권한·비밀값·개인정보·의존성 변경이 없어 차단 사유 없음(ci.yml 은 pull_request + contents:read + 시크릿 미사용이라 포크 PR 이 얻을 권한이 없다; actions 가 SHA 가 아닌 태그 핀인 점만 참고).
- 범위·되돌리기: diff 는 ci.yml(이미 머지된 b9fbec8 가 로컬 main=e9f3fb2 때문에 딸려 옴)·docs/development.md·docs_test.go 뿐. 혼입 리팩터·포맷 변경 없고 데이터/외부 상태 변경이 없어 revert 로 완전 복구된다.
- [러너 15:30] review approved — 리뷰 승인 (risk=low)
- [러너 15:31] pr created — https://github.com/hkjang/jasql/pull/7
- [러너 15:34] ci passed — 검사 1개 모두 success
- [러너 15:34] merge done — 2bf4e68
- [러너 15:43] release published — v0.31.1
- [러너 15:43] gh-release created — GitHub Release v0.31.1
- [러너 15:43] manifest failed — 누락/불량: jasql-mcp-v0.31.1.tar.gz
