# fix-summary

- 문제: CI `dependency-audit` 의 `cargo audit` 이 RUSTSEC-2026-0285 (rustls 0.23.42, 2026-09-14 공개, TLS 1.3 handshake 경계 오류, medium) 로 실패. PR 의 Go/문서 변경과 무관한, 새로 공개된 advisory.
- 로컬 재현: `cargo audit` (0.22.1, rustc 1.85) 에서 동일 advisory로 `error: 1 vulnerability found!`.
- 수정: `cargo update -p rustls --precise 0.23.45` → Cargo.lock 만 변경 (rustls 0.23.45, rustls-webpki 0.103.15). 워크플로 `.github/workflows/ci.yml` 의 주석이 정한 대로 "고치기"를 택했고 `--ignore` 는 쓰지 않음.
- 검증: `cargo audit` 통과(기존 허용된 chacha20 yanked 경고만 남음), `cargo check --locked` 로 rustc 1.85 빌드 확인, `govulncheck ./...` 0건, `go test ./internal/httpapi/` ok.
- 커밋: 86c066b `build: rustls 0.23.42 -> 0.23.45 to clear RUSTSEC-2026-0285 in the dependency audit` (PR 코드·테스트는 손대지 않음).
