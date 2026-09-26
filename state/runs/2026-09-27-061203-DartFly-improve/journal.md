# 회차 노트 2026-09-27-061203-DartFly-improve — DartFly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:12] base pinned — main@d3c7a9f
- [러너 06:12] autonomy release — 

## 구현 노트
- 무엇을/왜: 내려받은 CSV 와 공개 API 의 CSV 응답이 값 셀만 수식 방지를 받고 헤더 행(`column.Name`)은 그대로 나갔습니다. 컬럼 이름은 사용자의 SELECT 절이 정하므로 `SELECT @@version` → `@@version`, `SELECT -amount` → `-amount` 가 악의 없이 흔히 나옵니다. 같은 CSV 를 두 곳이 따로 만들고 있어 `writeResultCSV`(csvsafe.go) 하나로 합치고 헤더에 기존 `csvSafeCell` 을 적용했습니다. 커밋 ac365d7, 프로덕션 파일 3개(csvsafe.go·download.go·apiformat.go).
- 확신 없는 곳: (1) 헤더에 `'` 가 붙으면 CSV 를 스크립트로 읽는 쪽의 **컬럼 이름 매칭이 바뀝니다**. 평범한 이름은 그대로고 `=+-@` 로 시작하는 이름만 바뀌지만, 그런 이름을 키로 쓰는 소비자가 있다면 깨집니다 — 값 셀에서 이미 받아들인 트레이드오프이고 `DARTFLY_CSV_FORMULA_GUARD=off` 로 함께 꺼집니다. 이 판단이 운영자 뜻과 다르면 여기가 되돌릴 지점입니다. (2) `writeResultCSV` 추출로 download 의 스트리밍 경로가 `io.Writer` 를 거치게 됐습니다 — 같은 `http.ResponseWriter` 에 그대로 쓰고 flush 시점도 그대로라 동작은 같다고 보지만, 큰 결과의 메모리/스트리밍을 따로 측정하지는 않았습니다.
- 검증 못 한 것: `bash test/smoke/run.sh`(브라우저 포함) 전체는 돌리지 않았습니다. 화면 코드를 건드리지 않았고, 대신 실제 바이너리+실제 MariaDB 로 바뀐 경로 자체를 before/after 확인했습니다. XML·HTML_TABLE·JSON·마크다운 출력은 손대지 않았습니다(HTML·마크다운은 이미 컬럼명을 이스케이프합니다).
- 일부러 안 한 것: cancelQueryHandler(http.go:1659)의 '모든 실패 404' 는 남겼습니다 — 남의 실행 ID 존재 여부를 숨기려는 의도일 수 있어 403 으로 바꾸는 것이 개선인지 불확실합니다(ideas.json 에 근거 적음). history.go 의 limit/offset 무검증도 범위를 좁히려고 뺐습니다.
- 다음 역할이 조심할 것: 재현 스크립트는 `/tmp/df-csvheader-probe.sh`(Docker 필요, `bash ... down` 으로 정리). 컨테이너를 재사용하면 API 엔드포인트 경로가 이미 공개돼 publish 가 400 이 되므로 경로에 `$$` 를 붙여 씁니다. 새 테스트는 `internal/server/csvheader_test.go` 로 DB 없이 돕니다.
- [러너 06:26] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인함: diff 전체(프로덕션 3파일·테스트 1파일·docs 2파일), csvsafe.go 전문, apiformat/download 호출부, JS TSV 경로. `go test -count=1 ./internal/server/` green, `gofmt -l`·`go vet` 무출력. 새 테스트는 수정 전 `header[i]=c.Name` 에서 반드시 실패하므로 빈 테스트가 아닙니다 — 원장의 red 출력과 증상도 일치합니다.
- 구현자가 의심한 두 자리를 따로 봤습니다. (1) 헤더 `'` 접두: 값 셀이 이미 받던 규칙이고 같은 스위치로 꺼지며, JS 복사 경로(saved.js:138·app.js:1961)는 이미 헤더에 tsvCell 을 적용 중이라 오히려 정합이 맞습니다. (2) writeResultCSV 추출: download 는 전과 같이 같은 http.ResponseWriter 에 csv.Writer 를 감싸고 flush 시점도 동일 — 스트리밍/메모리 동작 변화 없음.
- 못 본 것: 브라우저 포함 smoke 미실행(화면 코드 무변경이라 필요성 낮음), download 핸들러 헤더를 직접 태우는 Go 테스트 부재(구조적으로 같은 함수 + 구현자의 실바이너리 probe 로 갈음).
- 승인이어도 남는 우려: 공개 API CSV 소비자에게 `=+-@` 로 시작하는 컬럼명 헤더가 바뀌는 호환성 변경 — 릴리스 노트에 한 줄 필요. writeResultCSV 는 여전히 csv.Writer 오류를 버려 중간에 끊긴 다운로드가 200 으로 보입니다(추출 전과 동일, 다음 회차 후보).
- 차단 부서 소견 없음: 신규 경로·인가 변경·비밀값·의존성·개인정보 처리 변화 없음. 공격면을 줄이는 하드닝입니다.
- [러너 06:28] review approved — 리뷰 승인 (risk=low)
- [러너 06:28] pr created — https://github.com/hkjang/DartFly/pull/15
- [러너 06:34] ci passed — 검사 3개 모두 success
- [러너 06:34] merge done — ac365d7
- [러너 06:41] release published — v2.76.0
- [러너 06:41] gh-release created — GitHub Release v2.76.0
- [러너 06:41] manifest ok — dartfly-v2.76.0.tar.gz dartfly-v2.76.0.tar.gz.sha256 
- [러너 06:41] assets uploaded — 2개
- [러너 06:41] assets verified — v2.76.0 자산 2개 (이전 v2.75.0: 2)
