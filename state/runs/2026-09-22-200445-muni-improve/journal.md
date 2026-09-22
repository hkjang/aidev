# 회차 노트 2026-09-22-200445-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:04] base pinned — main@1547faa
- [러너 20:04] autonomy release — 

## 정찰 노트
- 제목 중복(md/html/docx)·ZIP 목록.md 충돌은 지난 네 회차가 이미 먹었고, e2e CI(4/2/M)는 workflows 보호 경로와 계정 시드 때문에 한 세션에 안 끝나 제쳤습니다. 남은 것 중 "조용히 글자가 망가지는" 버그는 splitTableRow 하나뿐이라 그것을 골랐습니다.
- 추측으로 적은 것: 오동작을 코드 정독으로만 확인했습니다(샌드박스가 go run 승인을 거부해 동적 재현 미실행). `go test ./internal/httpapi/ -count=1` 은 DSN 없이 ok 인 기준선만 확인했습니다.
- 구현자가 조심할 것: `\\|` 나 역슬래시 일반 이스케이프로 범위를 넓히면 내보내기(파이프만 이스케이프)와 어긋나 왕복이 깨집니다. 끝 구분자 판정 한 가지만 고치고, 빈 마지막 셀(`| a | |`)과 구분자 행이 그대로인지 같은 테스트에서 함께 증명하세요.
- base(1547faa)에는 지난 회차의 DOCX·ZIP 수정이 아직 없습니다. 같은 파일을 다시 고치지 말 것.
- [러너 20:08] scout done — 마크다운 표에서 끝 구분자 없는 행의 마지막 셀에 있는 이스케이프된 파이프(`\|`)를 보존하기 (가치 3 / �

## 구현 노트
- `splitTableRow` 가 이스케이프를 훑기 전에 끝 파이프를 지워서, 끝 구분자를 생략한 행의 마지막 셀이 `\|` 로 끝나면 파이프가 사라지던 것을 고쳤습니다. 끝 파이프는 이스케이프되지 않았을 때만 구분자로 봅니다(`endedWithDelimiter`).
- 확신 없는 곳: 없음에 가깝습니다. 고치기 전 실패를 단위·live 양쪽에서 눈으로 보고, 수정만 되돌려 같은 둘만 재실패하는 것까지 확인했습니다. 다만 프런트·브라우저는 전혀 돌리지 않았고(표 파싱은 서버 안에서 끝남), `\\|`(역슬래시 두 개 뒤 파이프)는 **전과 똑같이** "이스케이프된 파이프" 로 읽습니다 — 과제서대로 GFM 전면 준수는 범위 밖이라 손대지 않았습니다.
- 일부러 하지 않은 것: 역슬래시 일반 이스케이프 풀기(내보내기가 `|` 만 이스케이프하므로 `a\b` 왕복이 깨짐), 파서 통합, 프런트·문서 변경(사용자에게 보이는 계약이 아니라 조용한 데이터 손실 수정).
- 다음 역할이 조심할 것: `import_markdown_test.go` 셋은 DB 없이 돌지만 `import_markdown_live_test.go` 의 `TestAnUploadedMarkdownTableKeepsAnEscapedPipeInItsLastCell` 은 MUNI_TEST_DSN 이 있어야 돕니다(없으면 SKIP). 이번 검증은 127.0.0.1:55571 의 전용 postgres:16-alpine 컨테이너로 httpapi SKIP 0 을 확인했고, 그 컨테이너(`muni-tbl-pg`)는 세션 끝에 지웁니다.
- 표-주도 테스트 `TestSplitTableRowKeepsExistingSplits` 는 기대값을 **수정 전 동작**에서 뽑았고 수정 전에도 통과했습니다 — 일부러 그렇게 두어 "기존 입력은 한 글자도 안 바뀐다" 를 고정합니다. 여기 값을 바꾸는 변경은 회귀로 보세요.
- [러너 20:13] brief accepted — 채택 — 과제서의 근거(`import_markdown.go:577` 의 TrimSuffix 선행, `render.go:645` 의 pad 가 항상 끝 ` |` 를 붙임, 네 호출자가 모두 �
- [러너 20:13] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: `import_markdown.go` 만 1547faa 로 되돌려 새 테스트가 `last cell = "나\\", want "나|"` 로 실제 실패하는 것을 눈으로 봤고, 되돌리면 통과. go vet ./... · gofmt -l . · go test ./... 전부 통과.
- 확인함: 끝 파이프가 구분자든 이스케이프든 셀 개수가 N 으로 동일해 `len(alignments) != len(header)` 게이트와 `isTableDelimiterRow` 판정이 수정 전후 같습니다. 네 호출자 모두 영향 없음. render.go:648 이 `|` 만 이스케이프하고 pad 가 항상 끝 ` |` 를 붙이므로 왕복도 그대로.
- 못 본 것: MUNI_TEST_DSN 이 없어 `TestAnUploadedMarkdownTableKeepsAnEscapedPipeInItsLastCell` 은 SKIP(컴파일만 확인). 프런트·브라우저 미실행 — 표 파싱은 서버 안에서 끝나므로 공백 아님.
- 승인이어도 남는 것: 줄이 `\\|`(역슬래시 둘 + 파이프)로 끝나면 결과가 ` b\\` → ` b\|` 로 바뀝니다. 행 중간 `\\|` 는 수정 전에도 이스케이프된 파이프였으니 오히려 일관되지만, 다음 회차가 GFM 전면 준수를 건드릴 때 여기부터 보세요.
- 릴리즈 노트용: 사용자에게 보이는 계약 변화 없음, 마이그레이션 없음, revert 로 완전히 되돌아옴. "끝 `|` 를 생략한 마크다운 표 행의 마지막 셀에서 `\|` 가 사라지던 문제" 한 줄이면 충분.
- [러너 20:16] review approved — 리뷰 승인 (risk=low)
- [러너 20:16] pr created — https://github.com/hkjang/muni/pull/23
- [러너 20:21] ci passed — 검사 2개 모두 success
- [러너 20:22] merge done — 1c442d1
- [러너 20:33] release published — v0.42.0
- [러너 20:38] assets verified — v0.42.0 자산 1개 (이전 v0.41.0: 1)
