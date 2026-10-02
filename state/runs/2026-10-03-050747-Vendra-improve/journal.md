# 회차 노트 2026-10-03-050747-Vendra-improve — Vendra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:07] base pinned — main@bfef041
- [러너 05:07] autonomy release — 

## 구현 노트
- 무엇을 왜: `compare_suppliers` 의 공개 설명이 「비용, 평가, 위험, **계약 및 이슈**를 비교」인데 응답은 `supplierSummaryRows` 의 여덟 필드뿐이라, 모델이 이슈·계약 질문에 이 도구를 골라 답할 데이터가 없는 행을 받는다. 설명을 실제 답하는 세 차원으로 좁히고(integrations.go:415 + 표 위 주석) 같은 약속을 들고 있던 docs/USER_GUIDE.md 4.6 표의 한 행도 함께 좁혔다. 가드 둘: 실제 세션으로 POST /mcp 를 지나 응답 키와 설명을 대조(DB 필요), 가이드 행과 설명을 양방향 대조(DB 불필요).
- 확신 없는 곳·검증 못 한 것: ① `docs/USER_GUIDE.pdf` 는 **재생성하지 않았다** — 이 저장소 scripts 에 md2pdf 가 없고(공용 스크립트) 과거에도 md 만 바꾼 커밋(aaf2627)이 있다. 표 한 칸이 PDF 와 어긋나므로 릴리즈가 가이드를 다시 굽는다면 이 행이 반영되어야 한다. ② 차원 어휘 표의 `계약`/`이슈` 키 이름(`contractCount` 등)은 아직 존재하지 않는 키의 **예정 이름**이다 — 나중에 집계를 넣을 때 다른 이름을 쓰면 가드가 그 자리에서 실패하며 이름을 추가하라고 말한다(의도된 동작이지만 「없는 키를 표에 적었다」는 지적은 가능하다). ③ 웹 스위트는 돌리지 않았다(web/ 파일 무변경). ④ 번역·문구가 아닌 모든 검증은 실행했다: 전체 go test(세 DSN, httpapi 28.227s, SKIP 0 확인은 MCP·가이드 선택 실행 22건에서), vet, gofmt, gate.py secrets clean.
- 일부러 하지 않은 것: 계약·이슈 집계를 **구현**하지 않았다 — `supplier.read` 만 가진 호출자에게 계약·이슈 데이터를 답하게 되고(권한 게이트는 도구 단위다) 집계 쿼리가 `business_objects` 에 같은 `orgInScope` 를 적용하지 않으면 범위 밖 건수가 샌다. ideas.json 에 M/위험4 로 남겼다. 미병합 브랜치 `auto/2026-10-01-1412`(fcce3d7)가 고치는 `:657`·`supplierIDArg` 는 건드리지 않았으므로 그 PR 과 충돌하지 않는다.
- 다음 역할이 조심할 것: `TestCompareSuppliersAnswersEveryDimensionItsDescriptionClaims` 는 **DB 가 있어야 돈다**(VENDRA_TEST_DSN 없으면 skip). `TestUserGuideClaimsTheComparisonTheToolDescriptionDoes` 는 DB 불필요이고 저장소 루트 기준으로 docs/USER_GUIDE.md 를 읽는다(repoFile). 검증에 쓴 컨테이너는 vendra-1003-improve-pg(127.0.0.1:55461, trust, DB 3개) — 다른 프로젝트 컨테이너는 쓰지 않았다.
- [러너 05:19] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 실패 재현을 직접 했다: integrations.go:428 과 USER_GUIDE.md:289 만 수정 전 문구로 되돌리니 `TestCompareSuppliersAnswersEveryDimensionItsDescriptionClaims` 가 계약·이슈 두 줄로 FAIL 하고, 메시지가 실제 응답 키 8개를 그대로 출력한다 — 커밋이 말하는 증상과 일치. 내 컨테이너(vendra-review-pg:55471, 사후 삭제)로 세 DSN 전부 걸고 `go test ./internal/... ./cmd/...` 전부 ok(httpapi 28.174s), vet·gofmt clean. 판정: **approve**(risk low, 차단 없음).
- 다만 두 번째 가드 `TestUserGuideClaimsTheComparisonTheToolDescriptionDoes` 는 수정 **전에도 PASS** 한다(양쪽이 똑같이 넓었으므로). 회귀 테스트가 아니라 한쪽만 바뀌는 드리프트 불변식이다 — 「가드 둘」을 동등한 증거로 읽지 말 것.
- 릴리즈가 알아야 할 것: `docs/USER_GUIDE.pdf` 는 안 구워졌고 4.6 표 한 칸이 md 와 어긋나 있다(md 만 고친 선례 aaf2627 있으므로 차단 아님). 가이드를 다시 굽는 회차는 이 행을 반영할 것. docs·README 전수 grep 결과 같은 약속이 남은 다른 자리는 없다.
- 다음 회차 거리(이번 diff 밖, 같은 계열): compare_suppliers·search_suppliers 의 `CASE WHEN ... ELSE 0` 때문에 spend.read 없는 호출자에게 annualSpend 가 0 으로 가 「지출 0」과 「못 봤다」가 구별되지 않는다. recommend_suppliers 는 같은 자리에서 NULL 을 준다. 새 가드는 키 존재만 보므로 이것을 잡지 못한다.
- 못 본 것: 웹 스위트(web/ 무변경이라 생략), PDF 내부 텍스트 대조, 원격 PR 상태, 미병합 auto/2026-10-01-1412 과의 실제 머지(코드상 접점은 없음 — :657·supplierIDArg 무변경 확인).
- [러너 05:23] review approved — 리뷰 승인 (risk=low)
- [러너 05:23] pr created — https://github.com/hkjang/Vendra/pull/138
- [러너 05:25] ci passed — 검사 2개 모두 success
- [러너 05:25] merge done — c17c3da

## 릴리즈 노트
- **v0.7.67 — 태그 `v0.7.67` (lightweight) → 8887dc2** (#138 머지 커밋). 이 저장소의 릴리즈 관례를 추측하지 않고 확인했다: 최근 8개 태그 전부 **주석 없는 bare 태그**이고 각자 PR 머지 커밋을 가리킨다(v0.7.66→bfef041, v0.7.65→887bd84 …). 릴리즈 커밋도, 버전 범프도, CHANGELOG 갱신도 없다 — `git log --oneline -60` 에 릴리즈/버전 커밋 양식이 하나도 없고, `CHANGELOG.md` 는 v0.6.45(2026-08-26)에서 멈춰 있으며 `web/package.json` 의 `0.6.21`·`Makefile` 의 `VERSION ?= 0.6.21` 도 태그와 무관하게 고정된 기본값이다(release.yml 이 태그 이름에서 버전을 파생해 인자로 넘긴다). 그래서 **버전 파일과 CHANGELOG 는 손대지 않았다.**
- 버전 증가: v0.7.59→v0.7.66 전부 패치 단위였고 이번 변경은 1파일 1줄 + 가이드 1행이라 **패치**(0.7.66 → 0.7.67).
- 자산: `release.yml` 이 `v*` 태그 푸시에 반응해 `scripts/offline-release.sh` 로 `dist/vendra-v<ver>.tar.gz` 를 만들고 `softprops/action-gh-release` 로 **Release 생성과 자산 업로드를 모두 자동으로** 한다(`generate_release_notes: true`, `name: Vendra v<ver>`). 따라서 `release.json` 의 `assets` 는 빈 배열, `github_release` 는 false, `notes_file` 은 빈 문자열(본문은 GitHub 이 PR 목록에서 생성 — 이전 릴리즈 본문 양식과 동일).
- 검증(태그 전, 빠른 것부터): `gofmt -l internal cmd` 무출력 → `go vet ./internal/... ./cmd/...` 통과 → 전용 `postgres:16-alpine`(vendra-rel-0767-pg, trust, 127.0.0.1:55471, DB 3개)·세 DSN 으로 `go test ./internal/... ./cmd/... -count=1` **전부 ok**(httpapi 28.371s — 구현 28.227s·비평 28.174s 와 일치), 이번 회차 가드 선택 실행 `-run 'TestCompareSuppliers|TestUserGuide|TestMCP|TestEveryPicture|TestASupplierTool|TestExpiring'` **22 PASS / SKIP 0**(구현 주장과 일치, DB 있으므로 skip 없음 확인). 로그인 잠금 테스트의 알려진 IPv6 플레이크는 이번 실행에서 나오지 않았다.
- 그리고 **태그 푸시가 실제로 실행할 단계를 미리 돌렸다** — 이번 회차에서 아무도 돌리지 않은 유일한 게이트다: `sh scripts/offline-release.sh 0.7.67` 성공(exit 0), 워크플로의 검증 3개를 그대로 재현 — 아카이브 non-empty(8.9M), `gzip -t` OK, `docker image inspect vendra:v0.7.67` OK. 추가로 바이너리에 버전이 `0.7.67` 로 박힌 것까지 확인했다. 이미지 안에서 `vite build` 가 통과했으므로 PATH 오염 없는 인터프리터에서의 웹 빌드도 함께 증명됐다(web/ 무변경이라 vitest 는 생략). `dist/` 는 gitignore 대상이 아니라서 빌드 후 삭제했고 worktree 는 clean, 푸시·업로드는 하지 않았다.
- **PDF 는 다시 굽지 않았다** — 구현·비평 둘 다 넘긴 항목이라 명시해 둔다. 이유 셋: ① `docs/USER_GUIDE.pdf` 는 릴리즈 자산이 아니다(이전 릴리즈 자산은 tar.gz 하나뿐), ② 이 저장소 `scripts/` 에 md2pdf 가 없고 PDF 는 기능 커밋에서만 갱신돼 왔다(md 만 고친 선례 aaf2627 있음), ③ 릴리즈는 커밋을 하나도 만들지 않는 절차이므로 PDF 를 다시 구우면 커밋이 필요해지고 이는 릴리즈에 변경을 끼워 넣는 것이 된다. **4.6 표 한 칸이 md 와 PDF 사이에서 어긋난 상태로 남아 있다** — 다음에 가이드를 굽는 회차가 반영할 것.
- 런치 등급: **tier 3**(개선). 공개 설명 1줄과 가이드 1행이고 새 기능도, 새 세그먼트도, 사용자 행동 변경도 없다 — 릴리즈 노트 외의 어떤 고지도 하지 않는 것이 맞다. 다만 이 변경은 모델의 **도구 라우팅**을 바꾸므로(이슈·계약 질문에서 compare_suppliers 가 더는 후보로 뽑히지 않는다) 채택 지표는 공지 도달이 아니라 「이슈·계약 질문이 `get_supplier_issues`·`search_contracts` 로 가는 비율」이다. 그 계측은 이 저장소에 없다 — 다음 회차 거리.
- [러너 05:32] release published — v0.7.67
- [러너 05:33] assets verified — v0.7.67 자산 1개 (이전 v0.7.66: 1)
