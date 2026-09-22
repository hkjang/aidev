# 릴리즈 조사 — v0.77.15

- 기준 HEAD: 0afe6ba, 기능 수정: 1dd6363. detached HEAD, 시작 작업 트리 깨끗함.
- 최근 태그 v0.77.14 / v0.77.13 / v0.77.12 모두 주석 태그, 주석 `git-ctx vX.Y.Z`, 커밋 `release: vX.Y.Z`; 패치 증가.
- 버전 원본 internal/version/version.go 및 OpenAPI, Kubernetes 이미지, 두 소개 페이지, 오프라인 배포·테스트 문서 동기화. 감사 기록은 새 항목 추가, 과거 기록 보존.
- 한국어 노트: docs/release-notes-vX.Y.Z.md. 제공된 GitHub Release 제목은 빈 문자열, 자산은 tar.gz와 sha256.
- .github/workflows/release.yml: v* 태그 푸시 → 검증 → linux/amd64 Docker 빌드 → scripts/package-offline-image.sh 및 verify-offline-image.sh → GitHub Release 생성·업로드·재다운로드 검증·공개. 따라서 assets=[], github_release=false.
- scripts/release.sh는 버전 검사, go build ./..., go test ./..., 커밋·태그·푸시를 수행. 원격 전송 금지에 따라 검사를 직접 실행하고 커밋·주석 태그만 로컬 생성.
- 추가 검증: FTS5 전체·race·MCP 비캐시, vet, gofmt, 버전 계약, Kustomize, 콘솔 계약, 빌드 모드 교차·DB 업그레이드, govulncheck v1.7.0.
- Skill 도구 및 marketing:product-launch / technology:release-and-deployment는 제공 도구·스킬 목록과 로컬 스킬 경로 검색에서 발견되지 않음. 해당 절차·반환 형식은 적용하지 못함.
- PostgreSQL/pgvector/Vault 통합은 로컬 전용 시험 서비스 연결 정보 없이 실행하지 않음. Docker 산출물 생성·검증 및 원격 게이트는 태그 푸시 후 CI 담당.
