## 2026-09-17
- 선택: 수정 과제 — PR #22 의 CI `dependency-audit` 작업이 `cargo audit` 에서 두 번 같은 이유로 실패: reqwest 아래 `rustls 0.23.42` 의 RUSTSEC-2026-0285 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: GitHub 공개 API 로 두 실패 run(35120269448·35120276584)의 job 을 조회하니 열한 개 작업 중
  `dependency-audit` 의 "Audit the Agent dependencies" 단계만 실패했고, 로컬 `cargo audit 0.22.1` 로 같은
  결과를 재현했다 — 2026-09-14 공개된 RUSTSEC-2026-0285(TLS 1.3 핸드셰이크 메시지가 암호화 단계 경계를
  넘어 받아들여짐, 5.3 medium)가 `rustls 0.23.42` 에 걸리고, 덤으로 `chacha20 0.10.1` 이 yanked 경고.
  PR 의 Go 변경과는 무관하게 새 권고가 그 사이 나온 것이다. 워크플로는 손대지 않고 `Cargo.lock` 만
  `cargo update -p rustls --precise 0.23.45`(rustls-webpki 0.103.15 동반)·`-p chacha20 --precise 0.10.2`
  로 고정 갱신했다. 세 크레이트 모두 crates.io index 의 rust_version 이 1.85 이하라 `rust-toolchain.toml`
  의 1.85.0 그대로다. 검증: `cargo audit` 취약점 0·경고 0, `cargo fmt --all -- --check`, `cargo clippy
  --all-targets -D warnings`, `cargo test --all-targets` 105개 통과, 호스트 `--locked --release` 빌드,
  그리고 `cross` 가 이 세션의 임시 HOME 때문에 rustup 을 못 찾아 `rust:1.85-alpine` 컨테이너에서
  `x86_64-unknown-linux-musl` `--locked --release` 정적 빌드(static-pie, `--version` 실행)로 대신 확인했다.
  PR #22 의 커밋(b982008)은 이 브랜치에 없으므로, 머지·리베이스 시 이 lockfile 갱신이 함께 들어가야
  #22 도 초록이 된다. 버전 범프·릴리즈 노트는 하지 않았다.
- 보류 아이디어: cargo audit·govulncheck 를 main 에 대해 매일 cron 으로도 돌려 새 권고가 남의 PR 이 아니라 자기 이름의 run 에서 먼저 드러나게 함 (가치 3 / 위험 1 / S) · 콘솔 Query DSL 화면이 새 `total`·`offset` 을 쓰지 않아 첫 페이지만 보여줌 — `operationsPages.tsx` 는 여전히 `items`·`truncated` 만 읽음, webui/dist 재빌드 동반 (가치 3 / 위험 2 / M) · 여러 목록 핸들러가 `rows.Err()` 를 확인하지 않아 부분 결과를 200 으로 돌려줌 — listAgents·settings·자산 상세 sources/history/relations (가치 3 / 위험 1 / M) · MCP `asset_get` 이 병합된 자산에 "asset not found" 만 답함 — `merged_into` 로 primary 안내 (가치 3 / 위험 2 / M) · 콘솔·외부 REST `assetRelations` 가 상한 없이 모든 edge 를 돌려줌 — MCP 와 같은 `limit`/`offset`·`has_more` (가치 3 / 위험 2 / M)
