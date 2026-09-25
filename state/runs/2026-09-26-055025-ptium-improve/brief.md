- 과제: 한 시트가 슬라이드 두 장으로 이어질 때 앞장은 차트, 뒷장은 표로 갈리는 것 — 모양을 시트 전체로 한 번만 정하기 (가치 3 / 위험 2 / 작업량 S)
- 왜: `writeSheet`(server/internal/docs/tables.go:109)가 첫 슬라이드는 `allNumeric(body, 1)`(153행)로, 이어지는 슬라이드는 각자 `allNumeric(piece, 1)`(181행)로 따로 판정해서, 열이 둘인 12행 시트에 숫자가 아닌 칸이 하나만 있어도 같은 시트가 `::columns`(막대 차트) 한 장 + `::table`(표) 한 장으로 갈린다. 두 슬라이드는 제목 줄도 다르게 잡고(차트는 `rows[0][1]`, 표는 `rows[0][0]`) 표 쪽만 헤더 행을 다시 쓰므로, 읽는 사람에게는 "매출" 차트 다음에 "지역" 표가 "(계속)" 이라는 이름으로 나오는 앞뒤가 안 맞는 덱이 된다.
- 재현(이번 정찰에서 실제로 확인함): `internal/docs` 에 임시 테스트를 넣어 `writeSheet(&b, "실적.csv", "", grid, placement{})` 를 직접 불렀다. grid = 헤더 `{"지역","매출"}` + 12행, 값은 `i*100`.
  - 10번째 행만 "미정" → 슬라이드1 `::columns 매출` (지역1~8), 슬라이드2 `::table 지역` + 헤더 행 + 지역9~12.
  - 3번째 행만 "미정" → 슬라이드1 `::table 지역`, 슬라이드2 `::columns 매출` (지역9~12). 순서만 바뀌면 반대로 갈린다.
  - (임시 테스트 파일은 지웠다. 작업 트리는 clean.)
- 수용 기준:
  1) 열이 둘이고 8행을 넘는 시트에서, 실제로 슬라이드에 쓰이는 행 중 숫자가 아닌 칸이 하나라도 있으면 **모든** 슬라이드가 `::table` 이다(첫 장·이어지는 장 모두).
  2) 실제로 쓰이는 행이 전부 숫자면 **모든** 슬라이드가 `::columns` 이고, 지금처럼 이어지는 장에도 헤더 행을 쓰지 않는다.
  3) 판정에 쓰는 행은 **슬라이드에 실제로 쓰이는 행만**이다 — `maximumTableSlides` 에 걸려 잘려 나간 뒤쪽 행(예: 40행 시트의 33행 이후)은 모양을 바꾸지 못한다.
  4) 한 슬라이드로 끝나는 시트, 열이 셋 이상인 시트, 기존 경고 문구(`…열이 많아…`, `…행이 많아 앞 %d줄만…`)와 `!source` 인용 범위는 전과 같다.
  5) 테스트가 증명할 것: 위 재현의 두 경우(늦은 "미정"·이른 "미정")에서 `::columns` 와 `::table` 이 한 시트 안에 섞여 나오지 않는다는 것. 고치기 전 코드에서 red 여야 한다.
- 건드릴 파일:
  - `server/internal/docs/tables.go:writeSheet` — `body` 와 `carried` 를 정한 **뒤에** 모양을 한 번 정하고(`chart := columns == 2 && allNumeric(body, 1)` 에 더해 `carried` 의 각 조각도 `allNumeric(piece, 1)` 이어야 함), 153행 블록과 181행 블록이 그 하나의 값을 쓰게 한다. 두 블록은 본문 쓰기가 그대로 복사된 코드이므로, 조각 하나를 쓰는 작은 헬퍼(예: `writeBody(builder, chart, header []string, piece [][]string, columns int)`)로 묶어 한 군데서만 판정하게 하는 것이 가장 안전하다.
  - `server/internal/docs/longtable_test.go` — 새 테스트 2개를 `TestASheetLongerThanASlideContinues` 옆에 더한다(`writeSheet` 를 직접 부르는 기존 관례 그대로). 열이 둘·12행·한 칸만 "미정" 인 격자를 `missingAt` 로 만들어 늦은 자리와 이른 자리를 각각 덮고, `strings.Count(source, "::columns")` 와 `strings.Count(source, "::table")` 중 하나가 0 임을 본다.
- 검증 명령:
  - `cd server && go test ./internal/docs`  (이번 정찰에서 2.955s 로 통과 확인)
  - `cd server && go test -race ./...`  (`internal/golden` 포함 — 시트 모양이 바뀌면 골든이 먼저 운다)
  - `cd server && go vet ./...` 와 `gofmt -l internal/docs`
  - 웹·API·문법 문서 변경이 없으면 `make test` 의 웹 단계는 건너뛰어도 된다.
- 위험과 피할 것:
  - **숫자 파서를 건드리지 말 것.** `allNumeric`·`amountOf`·`bareFigure`·`amountSigns`(tables.go:283~431)와 `deck` 의 `parseNumber`/`parseBareNumber`/`chartFields` 는 일부러 서로 다른 계약을 지킨다. 이 저장소에서 분류기를 넓히려던 시도가 2번 반려됐다(2026-09-09 교훈 두 건). 이번 과제는 **누가 판정하느냐**만 바꾸고 **무엇이 숫자냐**는 한 글자도 바꾸지 않는다.
  - 판정에 `all`(잘려 나간 행 포함)을 쓰지 말 것 — 수용 기준 3 을 깬다. `body` + `carried` 만이다.
  - `source(last)`·`last += len(piece)` 의 행 계산과 `from.column`/`from.row` 매핑은 최근 두 회차(3c3493f, 80ebe4c)가 손본 자리다. 인용 범위는 그대로 두고, 헬퍼로 묶을 때 `last` 누적 위치가 바뀌지 않게 할 것.
  - 보호 경로(auth·httpapi·db/migrations·scripts/release.sh)는 이 과제와 무관하다. 버전·릴리즈 노트도 손대지 않는다.
  - `tables.go:180` 이 `writer.go:continued()` 대신 `" (계속)"` 을 직접 붙이는 것은 **이번 범위 밖**이다. 같이 고치지 말 것.
- 차선 후보: `scripts/build-offline.sh` 의 매니페스트 검사가 "PyYAML 없음" 을 "매니페스트가 잘못됨" 으로 둔갑시키는 것 분리 — 73~103행의 `python3 - "$manifest" <<'DECODE'` 히어독이 `import yaml` 로 시작하므로, python3 은 있고 PyYAML 이 없는 호스트에서 ImportError 가 그대로 "The Kubernetes manifest in this bundle is not valid." 가 되어 멀쩡한 매니페스트로 릴리즈가 멈춘다(코드에서 확인; **이 호스트의 PyYAML 유무는 이번에도 확인하지 못했다 — 샌드박스가 python3 실행을 막았다**). 고른다면 import 를 먼저 떼어 내 전용 종료 코드(예: 2)로 "검사하지 못했다" 와 "잘못됐다" 를 가르고, `bash -n scripts/build-offline.sh` 와 잘못된 매니페스트 조각으로 손 시뮬레이션까지 할 것. 릴리즈 경로이므로 도커 빌드 전체를 돌리지 않는다면 그 사실을 적을 것.
