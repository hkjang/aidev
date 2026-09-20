# 과제서 2026-09-20 — moina

- 과제: 업로드 미디어의 저장 파일 이름 확장자를 판정한 MIME에 맞추고, 이름이 비면 종류에 맞는 기본 이름을 준다 (가치 2 / 위험 2 / 작업량 S)
- 왜: `backend/internal/httpapi/social.go:732 safeFilename`은 브라우저가 보낸 이름에서 경로·제어 문자만 걷어 내고 그대로 저장하므로, 클라이언트가 `photo.png`라고 보낸 JPEG(또는 `clip.mov`라고 보낸 MP4, `x.html`이라고 보낸 PNG)이 판정된 `mimeType`(`image/jpeg`)과 어긋난 이름으로 저장되고 `GET /api/v1/media/{id}`의 `Content-Disposition`(social.go:959)으로 그 이름 그대로 내려가 사용자가 저장하면 확장자와 내용이 다른 파일이 됩니다. 또 이름이 비면 동영상도 `"image"`가 되고 `"."`·`".."`(filepath.Base 결과)가 그대로 이름이 됩니다. 고치면 저장·다운로드 이름이 항상 실제 형식과 일치하고, 서버가 이미 판정한 MIME(`detectMediaType`)이 이름의 단일 출처가 됩니다.
- 수용 기준:
  1) `safeFilename`이 판정 MIME을 받아 확장자를 정규화한다 — 확장자 표: `image/jpeg→.jpg`, `image/png→.png`, `image/gif→.gif`, `image/webp→.webp`, `video/mp4→.mp4`, `video/webm→.webm`. 이름의 확장자(대소문자 무시)가 그 MIME의 허용 확장자(`jpeg`·`jpg` 둘 다 JPEG로 인정)와 같으면 원래 이름 그대로(`사진.JPG` 유지), 다르거나 없으면 기존 확장자를 떼고(`photo.png`→`photo.jpg`, `notes`→`notes.jpg`) 붙인다. 이름이 비거나 `.`·`..`이면 `image.jpg`/`video.mp4`처럼 종류 + 확장자.
  2) 한글·공백 이름과 200 rune 절단, 경로·제어 문자 제거는 지금 동작 그대로(절단 뒤에도 확장자가 남도록 절단은 확장자 붙이기 **전** 줄기(stem)에 적용).
  3) 테스트가 증명할 것: (a) 단위 테스트 `TestSafeFilenameMatchesDetectedMIME` — 위 표의 케이스(일치 유지·불일치 교체·확장자 없음·빈 이름·`..`·`jpeg`/`JPG` 인정·경로 제거·200자 절단 뒤 확장자 유지). (b) PostgreSQL integration 테스트(파일명 `media_upload_postgres_integration_test.go`, `MOINA_TEST_POSTGRES_DSN` 없으면 skip — `media_delete_postgres_integration_test.go:19-56`의 사용자·cleanup 패턴, `posts_update_postgres_integration_test.go:104-127`의 `New(repository, secrets, …)`·세션 cookie·`X-CSRF-Token` 패턴 그대로) — 실제 `multipart.Writer`로 1×1 PNG 바이트(`image.Encode`로 만들거나 `image/png` 고정 바이트)를 `filename="photo.jpg"`로 `POST /api/v1/media`에 올려 응답 `filename`이 `photo.png`이고 `GET /api/v1/media/{id}`의 `Content-Disposition`이 `inline; filename="photo.png"`인 것, 그리고 `filename=""`로 올리면 `image.png`인 것. 수정 전 코드에서 (a)·(b)가 실패하는 것을 먼저 보고(TDD) 수정한다.
- 건드릴 파일:
  - `backend/internal/httpapi/social.go:732 safeFilename(header)` → `safeFilename(header, mimeType)` 시그니처로 바꾸고 확장자 정규화 추가(호출 자리 social.go:655 하나뿐 — `grep -rn safeFilename backend`로 재확인). 확장자 표는 `detectMediaType`이 돌려주는 6개 값과 1:1인 package-level map으로 두고, social.go:642의 허용 목록과 같은 6개임을 주석으로 잇는다.
  - `backend/internal/httpapi/httpapi_test.go` — `TestDetectMediaTypeByMagic`(488행) 근처에 단위 테스트 추가(또는 새 `media_filename_test.go`).
  - `backend/internal/httpapi/media_upload_postgres_integration_test.go` — 새 integration 테스트.
  - `api/openapi.yaml:1364` `Media.filename` — "서버가 판정한 형식에 맞춘 확장자" 한 줄 설명 추가(route 수·응답 코드는 건드리지 않음). `frontend/API_CONTRACT.md`에 Media.filename 설명이 있으면 같은 문장으로 맞추고, 없으면 두지 않는다(미확인).
- 검증 명령 (저장소 루트 `/home/hkjang/.cache/auto-improve-wt/moina` 기준):
  - `cd backend && go test ./internal/httpapi -run 'SafeFilename|DetectMediaType|ContentDisposition' -count=1` (정찰에서 기존 두 테스트 0.01초 통과 확인)
  - integration: `docker run -d --name moina-test-pg -e POSTGRES_PASSWORD=moina -e POSTGRES_USER=moina -e POSTGRES_DB=moina -p 127.0.0.1:55432:5432 postgres:17-alpine` 뒤 `cd backend && MOINA_TEST_POSTGRES_DSN='postgres://moina:moina@127.0.0.1:55432/moina?sslmode=disable' go test -race ./... -count=1` (지난 회차들이 같은 방식으로 integration 32개 실행·skip 0을 확인). 끝나면 `docker rm -f moina-test-pg`.
  - `cd backend && go vet ./... && go run honnef.co/go/tools/cmd/staticcheck ./...` (Makefile `lint`의 STATICCHECK 변수 값을 그대로 쓰면 됨) · `make fmt` · `make check`(OpenAPI route 계약 120개 — openapi.yaml 설명만 바꾸므로 수가 변하면 안 됨).
  - 프런트·e2e는 무변경이면 생략 가능. 단 `frontend/src/components/MoinComposer.tsx:90`이 `item.filename`을 표시 이름으로 쓰므로 응답 형식(문자열)은 그대로여야 함.
- 위험과 피할 것:
  - `detectMediaType`·`mp4MajorBrands`·`unsupportedMediaMessage`(social.go:684-730)는 건드리지 않는다 — 이번 과제는 이름만이고 판정 로직은 별도 테스트로 고정돼 있다.
  - `contentDisposition`·`asciiFilename`(social.go:756-)은 그대로 — 한글 filename* 처리는 이미 `httpapi_test.go:580` 테스트가 있다.
  - 기존 저장 행의 `filename`은 마이그레이션으로 고치지 않는다(migrations는 추가만 허용이고, 과거 행을 재작성할 가치가 없음). 새 업로드부터 적용됨을 커밋 메시지에 적는다.
  - 운영자 규칙: "실제 출력이 바뀌지 않는 수정"은 반려된다 — 이 과제는 응답 `filename`·`Content-Disposition` 값이 실제로 바뀌므로 integration 테스트로 그 변화를 보인다. 대역 mediastore를 주입하지 말고 실제 `New(repository…)` 배선과 pgx를 지날 것(지난 회차 `posts_update_postgres_integration_test.go`와 같은 수준).
  - 보안 경계(`auth.go`·`oidc.go`·`mcp_oauth.go`)·migrations·`.github/workflows`는 손대지 않는다.
  - MCP 도구가 media filename을 내는지(`backend/internal/mcp` 없음 — `grep -rn Filename backend/internal --include=*.go`로 확인, 미확인 부분은 이 grep으로 정리)와 `docs/api-mcp.md`에는 filename 언급이 없음(정찰 grep 결과 0건).
- 차선 후보: Makefile `test` 타깃을 CI(`.github/workflows/ci.yml:64 go test -race ./...`)와 같게 `-race`로 맞추기 (가치 2 / 위험 1 / S) — `Makefile:34` 한 줄. 커밋 메시지에 "CI 동등성" 목적을 명시하고, `make test`가 실제로 -race로 도는 것을 출력으로 보인다(로컬 `make`가 승인 거절되면 `cd backend && go test -race ./...`로 대신 증명).
