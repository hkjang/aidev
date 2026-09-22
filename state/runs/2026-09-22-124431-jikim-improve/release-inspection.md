# 릴리즈 확인

- 최근 태그 v0.2.17, v0.2.16, v0.2.15 모두 주석 태그, 메시지 `jikim vX.Y.Z`.
- 최근 릴리즈 커밋은 `fix: release ... for v0.2.17`, `fix: release ... for v0.2.16`, `feat: release ... for v0.2.15`.
- 버전 파일: scripts/version.sh, internal/version/version.go(-dev), web/package.json, web/package-lock.json 루트, web/src/lib/format.ts. 현재 0.2.18로 동기화.
- README, CONTRIBUTING, Compose, release workflow 입력 예시, docs 문서 프로파일과 두 PDF를 이전과 같은 22개 파일 범위로 갱신. 의존성 버전은 원본과 동일.
- CHANGELOG.md: 한국어 Keep a Changelog, 날짜와 버전별 항목, 하단 릴리즈 링크. 기존 이력 유지.
- release.yml: v*.*.* 태그 푸시 후 버전/커밋 일치, verify.sh, linux/amd64 Docker, PostgreSQL/egress smoke, Playwright E2E, package-offline.sh 및 verify-offline-bundle.sh, GitHub Release 생성/게시.
- 이전 두 자산 tar.gz와 sha256은 위 workflow에서 생성. 로컬 assets=[] 및 github_release=false.
- 기존 릴리즈와 동일한 로컬 verify.sh 실행. Docker 이미지·스모크·브라우저 E2E·패키징은 태그 푸시 후 CI 담당.
- PDF: 기존 /mnt/c/Users/USER/projects/aidev/tools/guide/md2pdf.mjs 사용. 사용자 19쪽, 관리자 28쪽. 0.2.18 표지, 캡처 출처 0.2.9 검증.
- 요청한 marketing:product-launch 및 technology:release-and-deployment와 Skill 도구를 찾지 못해 사용자 절차 및 실제 릴리즈 이력을 적용.
