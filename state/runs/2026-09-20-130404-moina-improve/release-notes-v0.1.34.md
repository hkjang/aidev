## MOINA 오프라인 이미지

- 파일: `moina-v0.1.34.tar.gz`
- 이미지: `moina:v0.1.34`
- 플랫폼: `linux/amd64`
- SHA256: 태그 푸시 후 `.github/workflows/release.yml`이 이미지를 빌드·패키징해 릴리스 본문에 기록합니다.

폐쇄망 반입 전에 위 해시와 파일 해시를 대조하고 `docker load` 하세요. 릴리스 asset은 요청 범위에 따라 서비스 이미지 tar.gz 하나만 포함합니다.

## 이번 변경

`v0.1.34`은 업로드 미디어(`POST /api/v1/media`)의 저장 파일 이름 확장자를 서버가 판정한 MIME에 맞춥니다.

- 지금까지는 브라우저가 보낸 이름에서 경로·제어 문자만 걷어 내고 그대로 저장해 `photo.png`라고 보낸 JPEG가 그 이름으로 남고 `GET /api/v1/media/{mediaID}`의 `Content-Disposition`으로도 그대로 내려갔으며, 빈 이름은 `.`·`..`이 되어 기존 `image` 기본값에 닿지 못했습니다.
- 이제 허용하는 6개 MIME(JPEG·PNG·GIF·WebP·MP4·WebM)에 맞춰 확장자가 같으면(대소문자 무시, `jpeg`·`jpg` 모두 JPEG) 원래대로 두고, 다르거나 없으면 떼고 붙이며(`photo.png`→`photo.jpg`, `clip.mov`→`clip.mp4`), 비거나 `.`·`..`이면 `image.jpg`·`video.mp4`처럼 종류에 맞는 기본 이름을 줍니다.
- 200자 절단은 확장자를 붙이기 전 줄기에만 적용하고, 기존에 저장된 행의 이름은 바꾸지 않아 새 업로드부터 적용됩니다.
- `api/openapi.yaml`의 `Media.filename`에 이 규칙을 적었고(route 계약 120개 그대로), 실제 PostgreSQL을 지나는 multipart 업로드 integration 테스트로 고정했습니다.

검증: `make fmt`, `make check`, `go build`, `go vet`, `go test -race`(임시 PostgreSQL 17, integration 38개 PASS·SKIP 0), staticcheck 2025.1.1 clean, 프런트엔드 vitest 250개 통과·ESLint 0 errors·`v0.1.34` production build.
