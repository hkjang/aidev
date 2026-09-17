# Invenqor Server·Agent v0.2.36 릴리즈 노트

릴리즈 일자: 2026-09-17
호환 Agent: v0.2.36 (Linux·Windows)

이번 릴리즈는 제품 동작을 바꾸지 않습니다. 바뀐 것은 **Agent 가 HTTPS 를 말할 때
쓰는 TLS 라이브러리의 버전** 하나입니다. Agent 는 Server 로 나가는 모든 전송에
rustls 를 쓰는데, 지금까지 고정되어 있던 rustls 0.23.42 에 2026-09-14 공개된
권고 **RUSTSEC-2026-0285** 가 닿습니다. 권고가 닫히는 가장 낮은 버전인 **0.23.45**
로 올렸고, 같은 감사에서 yanked 로 경고된 crate 하나도 함께 바꿨습니다.
`Cargo.toml` 은 그대로이고 **lockfile 만 바뀝니다.**

## 1. Agent 의 TLS 라이브러리에 권고가 닿아 있었습니다

`cargo audit` 가 `reqwest` 아래의 rustls 0.23.42 에서 RUSTSEC-2026-0285 를
찾았습니다 — TLS 1.3 핸드셰이크 메시지가 암호화 단계의 경계를 넘어 받아들여지는
문제로, 심각도는 5.3(medium)입니다. 권고는 트리를 건드린 어떤 변경과도 무관하게
새로 공개된 것이었고, 그 사실은 무관한 PR 의 `dependency-audit` 작업이 같은
이유로 두 번 실패하면서 드러났습니다. 이 저장소는 그 실패를 그대로 두지 않고
**고치거나, `--ignore` 와 사유를 남기거나** 둘 중 하나를 하기로 정해 두었고,
이번에는 고쳤습니다.

| crate | 전 | 후 | 이유 |
|---|---|---|---|
| `rustls` | 0.23.42 | 0.23.45 | RUSTSEC-2026-0285 를 닫는 가장 낮은 버전 |
| `rustls-webpki` | 0.103.13 | 0.103.15 | rustls 0.23.45 가 함께 끌어옴 |
| `chacha20` | 0.10.1 | 0.10.2 | 0.10.1 이 yanked 되어 감사에 경고로 남음 |

세 crate 모두 crates.io 가 기록한 `rust_version` 이 1.85 이하이므로,
`rust-toolchain.toml` 의 **1.85.0 은 그대로**이고 오래된 배포판을 위한 빌드
하한도 움직이지 않습니다. `cargo audit` 는 이제 취약점 0, 경고 0 을 보고합니다.

## 검증

- **의존성 감사.** `cargo audit`(0.22.1) 취약점 0·경고 0.
- **Agent.** `cargo fmt --all -- --check`, `cargo clippy --all-targets -D warnings`,
  `cargo test --all-targets` 105개 통과. 호스트와 `x86_64-unknown-linux-musl`
  (`rust:1.85-alpine`) 에서 `--locked --release` 정적 빌드가 성공하고 `--version`
  이 실행됩니다.
- **Server·콘솔.** Go 와 콘솔 코드는 이번 릴리즈에서 바뀌지 않았습니다. 릴리즈
  커밋에서 `go test ./...` 가 SQLite fallback 과 실제 PostgreSQL
  (`scripts/test-postgres.sh`) 양쪽에서 전 패키지 통과하고, `go vet` · `gofmt` ·
  `go build` 와 콘솔의 vitest 가 통과합니다.

## 호환성

- **데이터베이스 마이그레이션이 없습니다.**
- **API 응답 형식 변경이 없습니다.** REST API, Query DSL, MCP 도구는 이전과
  같습니다.
- **Agent 의 동작 변경이 없습니다.** 설정 파일, 큐, 전송 프로토콜, mTLS·사설 CA
  처리는 이전과 같고, TLS 라이브러리의 패치 버전만 올라갑니다. 기존 Agent 는
  자동 업데이트 또는 패키지 재설치로 v0.2.36 으로 바꿀 수 있으며, 서두르지 않아도
  Server 와의 호환에는 영향이 없습니다.
- 소스에서 빌드하는 경우 `cargo build --locked` 가 새 lockfile 을 그대로 씁니다.
  Rust 1.85.0 이면 충분합니다.
