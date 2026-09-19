# 수리 요약 — PR #29, 커밋 bd3a982

- **문제 1 (dlp.go Reported, 모르는 등급)**: Pod 가 보낸 `Class` 문자열을 Class·Label 에 그대로 실었다 (재현: `Reported(Finding{Class:"900101-1234568"})` → Class/Label 에 값). 이제 고정 이름 `dlp.UnknownClass`(`"unknown"`, 라벨 `알 수 없는 등급`) 로 샘플 없이 실리고, 모르는 `Action` 도 `audit` 로 정규화된다 — Pod 문자열은 어느 칸으로도 트레일에 들어가지 않는다.
- **문제 3 (64룬 컷)**: `secret`·`email` 패턴은 길이 상한이 없어 진짜 83룬 sk- 샘플이 64룬으로 다시 실렸다 (재현: 새 테스트가 수정 전 실패). 컷을 없앴다 — `mask` 는 자기 출력에 어떤 길이에서든 no-op 이고 샘플 길이는 요청 본문 1 MiB 한도가 이미 묶는다. `TestAReportedFindingIsFiledUnchangedHoweverLongTheValue` 가 83룬 키와 긴 이메일로 경계 너머를 단언한다.
- **문제 2 (ADMIN_GUIDE.md)**: "Pod 가 무엇을 보내든 값이 실리지 않는다" 를 지웠고, 등급·라벨·샘플·조치는 컨트롤 플레인 것으로 바꿔 저장하지만 서버·툴·방향은 Pod 가 말한 그대로라고 적었다. PDF 는 러너와 같은 `tools/guide/md2pdf.mjs` 로 다시 구웠다 (같은 제목·부제·버전).
- 검증: `go vet ./cmd/... ./internal/...` 깨끗, `go test -race ./cmd/... ./internal/...` 전부 ok (dlp·api 포함). DSN 이 필요한 `dlpreport_live_test` 는 로컬에 Postgres 가 없어 못 돌렸다 — 위조 케이스는 `rrn` 등급이라 동작이 바뀌지 않는다.
- 서버·툴·방향 문자열에 검증·길이 한도를 새로 넣지는 않았다 (지적은 문서의 과장이었고, 그 칸은 이 변경 이전부터 Pod 의 말이다).
