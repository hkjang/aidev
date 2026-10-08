# 릴리즈 검증 기록 — v0.262.0

- 시작 커밋: 4a78148eeeae4462de245fa4e1550cd9fbac5aa0 (de09ba8 포함)
- 릴리즈 커밋: 938fb1daf79041592a71780452151441314ed442
- 릴리즈 메시지: chore(release): 0.262.0
- 작성자/커미터: 환경의 hkjang 설정 유지. 트레일러 없음.
- 최근 태그 v0.261.0, v0.260.0, v0.259.0은 모두 commit을 직접 가리키는 경량 태그이며 별도 주석 없음.
- 최근 커밋 양식과 0.x 마이너 증가를 따라 v0.262.0 경량 태그 작성. detached HEAD 유지.
- 최근 릴리즈와 같은 11개 파일, 15개 표기의 버전만 수정. 기능 변경 없음.
- 저장소에 CHANGELOG 또는 정적 릴리즈 노트 없음. 기존 한국어 본문은 scripts/release-catalog-images.sh notes에서 생성되며 release.yaml이 자동 생성한 변경 내역과 함께 게시함.
- 한국어 본문 및 이번 수정 설명은 같은 run 디렉터리의 release-notes.md에 보존. 런타임 자산의 실제 원본 릴리즈는 정보가 부족하므로 추정하지 않고 태그 CI의 검증으로 남김.

## 스킬 적용

전용 Skill 도구가 없어서 아래 로컬 SKILL.md와 각각의 references/sources.md를 읽음.
- /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/marketing/skills/product-launch/SKILL.md
- /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/release-and-deployment/SKILL.md

출시 등급은 tier three: 가이드 스크린샷 생성 담당자를 위한 개선으로 릴리즈 노트만 준비. 별도 마케팅 공지·가격·패키징 변경 없음. 기존 timeout 환경변수의 기본값과 효과, 실패 시 설정 복원 시도 및 검증 한계를 노트에 명시함.
출시 기준은 기존 CI/릴리즈의 로컬 검사와 새 회귀 통과. 실패하면 릴리즈 담당 에이전트가 커밋/태그 진행을 중단하고 이 세션 변경만 복구한다. DB 스키마 변경이나 새 feature flag 없음.
실제 배포는 사용자 지시로 수행하지 않음. 외부 사용자의 최초 실행 확인도 수행하지 못함. 배포 후 후속 검증 기준은 첫 폐기 가능한 환경의 가이드 촬영 1회에서 요청 중단 후 네 설정 복원 시도를 확인하는 것; 이번 세션에서는 달성했다고 주장하지 않음. 복원 시도가 누락되면 배포 운영자가 확대를 중단하고 이전 v0.261.0 아티팩트로 복귀하는 판단을 담당함.

## 검사 결과

- node --check web/scripts/guide-shots.mjs: 통과
- npm ci, npm run lint, npm run test:sso: 통과(SSO 12건)
- node --test scripts/*.test.mjs: 154 pass, fail/skip 0 (가이드 관련 80건 포함)
- npm run build: 통과
- go test -race ./cmd/... ./internal/...: 통과
- go build ./cmd/...: 통과
- scripts/release-catalog-images.sh validate 및 check-versions: 통과
- make catalog-preflight: 통과
- kubectl kustomize deploy/kubernetes: 통과
- 개발 docker compose config --quiet: 통과
- 오프라인 Compose의 외부 PostgreSQL DSN 누락 거부: 통과
- 오프라인 Compose 두 서비스와 agenthub:v0.262.0 이미지 일치: 통과
- git diff --check, 11개 파일의 정확한 버전 치환 확인: 통과

검증 로그는 같은 run 디렉터리의 release-*-checks.log, release-go-build.log, release-commit-check.log에 보존.
기존 npm high 경고 2건 유지. 실제 Chromium·관리자 HTTP/DB·실시간 에이전트·운영 클러스터 검증은 실행하지 않음. DB live 테스트용 AGENTHUB_TEST_DSN 미설정. 테스트 통과를 해당 환경 검증으로 해석하지 않음.

## 자산 및 원격 동작

이전 GitHub Release의 다음 자산은 모두 .github/workflows/release.yaml이 태그 push에서 자동 생성/첨부한다.
- agenthub-v0.262.0.tar.gz
- agenthub-v0.262.0.spdx.json
- agenthub-v0.262.0.provenance.sigstore.json
- SHA256SUMS
- offline-bundle.json
- agenthub-offline-linux-amd64
- agenthub-offline-compose.yaml
- agenthub-offline.env.example

따라서 로컬 업로드 자산은 없으며 assets=[] 및 github_release=false. make release-archives도 확인했으나 워크플로 생성 자산을 중복 제작하지 않음. 태그 CI는 이미지 계획·빌드·상태 검사·SBOM·아카이브·manifest·체크섬·Sigstore 서명·한국어 본문·GitHub Release 게시를 수행함. 원격 Actions/OIDC와 실제 게시 자산 검증은 이 세션에서 실행할 수 없으며 외부 러너가 태그를 푸시한 후 진행됨. git push, gh, 레지스트리 publish/push, 클러스터 변경을 실행하지 않음.

## 버전 목록

- AgentHub: 0.261.0 → 0.262.0 (VERSION 및 기존 배포/문서 표기 전체)
- web/package.json, web/package-lock.json의 루트 패키지: 0.1.0 유지(독립 비공개 패키지의 기존 관례)
- internal/buildinfo/buildinfo.go의 개발 기본값과 Dockerfile의 ARG 기본값: 기존 그대로; 릴리즈 빌드가 VERSION/각 런타임 버전을 주입
- BASE_VERSION: 0.26.0 유지
- BROWSERCODE_VERSION: 0.2.0 유지
- GOOSE_VERSION: 0.1.0 유지
- HOLMES_VERSION: 0.2.0 유지
- JUPYTER_VERSION: 0.1.0 유지
- LANGFLOW_VERSION: 0.2.0 유지
- N8N_VERSION: 0.2.0 유지
- NODERED_VERSION: 0.1.0 유지
- OPENCODEREVIEW_VERSION: 0.1.0 유지
- OPENHANDS_VERSION: 1.43.1 유지
- ORCA_VERSION: 0.5.0 유지
- PI_VERSION: 0.1.0 유지
- PRIMEAGENT_VERSION: 0.1.0 유지
- QWENCODE_VERSION: 0.2.0 유지
