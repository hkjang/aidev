# Invenqor v0.2.1 릴리즈 노트

릴리즈 기준일: 2026-07-29

## 핵심 변경

- 사용자 UI, 관리자 API, Agent 수집·heartbeat·업데이트, REST API와 MCP를
  기본 TCP `7070` 하나로 통합했습니다.
- CentOS 7, Red Hat UBI 8/9, Ubuntu 22.04/24.04 LTS와 Alpine에서 정적
  musl Agent의 실제 수집·전송 E2E를 통과했습니다.
- Agent가 주기적으로 새 버전을 확인하고 SHA-256, 크기, OS/Architecture,
  고정 Ed25519 공개키를 검증한 뒤 원자 교체하는 서명 자동 업데이트 체계를
  추가했습니다.
- Server의 Kubernetes 멀티 파드 운영을 위해 PostgreSQL advisory migration
  lock, 공통 Master Key Secret, Pod별 RWO state/spool, 업데이트 RWX PVC,
  StatefulSet parallel 기동, probe와 PDB를 적용했습니다.
- Server 2개 Pod 동시 기동, DB migration, 교차 Pod 로그인과 MCP API key
  인증을 실제 PostgreSQL 환경에서 검증했습니다.
- 폐쇄망에서 `docker load` 한 번으로 Server와 PostgreSQL을 적재할 수 있는
  `linux/amd64` Docker image `invenqor-0.2.1.tar.gz`와
  `pull_policy: never` Compose 파일을 제공합니다.

## 자산 API·MCP·키 관리

- Scoped Bearer API key를 사용하는 자산 CRUD·관계·Query DSL 외부 REST API를
  추가했습니다.
- MCP `2025-11-25` Streamable HTTP의 stateless `/mcp` endpoint와
  `asset_search`, `asset_get`, `asset_relations`, `agents_list` 읽기 도구를
  제공합니다.
- MCP 도구 목록은 key scope에 따라 필터링되며 Origin 검증, request body 제한,
  key별 rate limit과 untrusted asset data 지침을 적용했습니다.
- API key 원문은 생성·회전 시 한 번만 반환하고 DB에는 SHA-256만 저장합니다.
- Scope 추가, 전체 교체, 개별 삭제, 이름 변경, 만료, 최대 7일 유예 회전,
  즉시 폐기와 소유 사용자 비활성화 연동을 지원합니다.
- Key 생성·scope 변경·회전·폐기는 Secret을 제외하고 감사 로그에 기록됩니다.
- 관리 콘솔에 API·MCP 키 발급, scope 토글, 회전, 폐기 화면을 추가했습니다.

## 안정성·보안 개선

- Agent Token 회전·차단 직후 인증 cache를 즉시 무효화합니다.
- Update download URL을 동일 Server의 상대 경로로 제한해 Bearer Token의
  다른 Origin 전송을 차단합니다.
- Update Content-Length, manifest size와 실제 크기를 강제 일치시키고 artifact
  크기를 128 MiB로 제한합니다.
- 여러 Server Pod가 동일 update version을 동시에 게시해도 기존 artifact를
  덮어쓸 수 없는 불변 저장 방식을 적용했습니다.
- 멀티 파드 최초 관리자 Token 경쟁 조건을 DB 선점 방식으로 제거했습니다.
- 공유 32-byte Master Key로 Pod 간 OIDC/TOTP 암호화 비밀을 동일하게
  해독합니다.

## 문서

Markdown과 PDF로 사용자 가이드, 관리자 가이드, 임원 보고서, Server
설치·오프라인·Kubernetes 가이드, 자산 API·MCP·키 관리 가이드를 제공합니다.
수집 원천·필드·제외 데이터, 설치, 운영, 백업·복구, 서명 업데이트와 장애 대응
절차를 포함합니다.

## 검증 증적

- Rust format, Clippy `-D warnings`, 단위 테스트 19개
- Go 전체 단위 테스트, race detector, vet
- React TypeScript production build, npm audit 취약점 0
- OpenAPI recommended lint 통과
- Helm lint/template, 온라인·오프라인 Compose config 통과
- x86_64/aarch64 musl 정적 Agent build와 SHA-256 검증
- 6종 Linux client/server 수집·전송·서명 update·API key·MCP E2E
- PostgreSQL Server 2-Pod 동시 migration·세션·MCP E2E
- Docker image tar.gz `docker load`, `pull_policy: never` 오프라인 기동
- 최종 Server image Trivy HIGH/CRITICAL 취약점 0
