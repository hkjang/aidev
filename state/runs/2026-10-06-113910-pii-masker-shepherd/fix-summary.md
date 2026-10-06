# 수리 요약 — PR #34 (commit c068b40)

- 지적이 맞았다. `/MediaBox [0 0 0 0]` 1페이지 PDF를 `MaskPDFFile`에 정상 좌표로 넘겨 재현했다: `page 1 has no usable dimensions` + `errors.As(err, &*RegionPlacementError)` = true (정상 `[0 0 595 842]`는 nil). 손상된 업로드가 `masking_failed`/502/`retryable:true`로 나가고 있었다.
- `engine.go:431`의 `target.Width<=0||Height<=0` 한 줄만 `placementError` → `fmt.Errorf`로 되돌려 `processing_failed`/400/`retryable:false` 분류를 복원했다. 나머지 세 자리(NaN·Inf / 페이지 크기 미보고 좌표 초과 / 페이지 미접촉)와 `service.go:598/:618`는 분류가 맞아 손대지 않았다.
- 코드와 반대로 말하던 `engine.go:399-406` 주석과 README:85의 "업로드는 정상"·"다시 보내면 성공" 문구를 고치고, README 열거에 빠져 있던 "페이지가 크기를 알려주지 않을 때(`processing_failed`)" 항목을 더했다.
- 테스트: `engine_test.go:262` 표에서 `page without usable dimensions` 케이스를 빼고(단언 완화 아님 — 잘못된 기대였음), 대상을 실제로 실행하는 `TestMaskPDFFileBlamesTheDocumentForAPageWithoutDimensions`를 추가했다. 이 테스트는 수정 전 실패·수정 후 통과하며, 그 한 줄만 되돌리면 다시 실패함을 확인했다(인과 증명).
- 검증: `go build ./...`, `go vet ./...`, `gofmt -l ./cmd ./internal`(무출력), `git diff --check`, `go test -count=1 ./...` 8패키지 전부 ok. Docker 빌드·`GOARCH=386`은 여전히 미확인.
