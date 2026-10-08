# v0.319.0 릴리즈 검증

- 관례: 최근 태그 v0.318.0/v0.317.0/v0.316.0 모두 hkjang의 주석 태그이며 커밋 제목은 `chore: vX.Y.Z 을 냅니다`. 최근 0.x 마이너 증가 패턴을 따름.
- 스킬: Skill 호출 도구가 없어 headcount의 marketing/product-launch 및 technology/release-and-deployment SKILL.md와 references/sources.md를 파일로 읽음. Tier three: 릴리즈 노트 중심. 별도 외부 사용자 체험은 수행하지 않음. 관찰 기준·중단 조건·롤백은 릴리즈 본문에 기록.
- 자산: `.github/workflows/release.yaml`이 태그 푸시 후 `weekly-v0.319.0.tar.gz` 단일 Docker 자산과 `Weekly v0.319.0` GitHub Release를 생성. assets=[] 및 github_release=false. 로컬·원격 Docker 게시 없음.
- 변경: 버전 아홉 곳, package-lock 루트 버전 두 곳, 릴리즈 본문, 로드맵과 생성 문서. 기능·의존성·비밀값 무변경.
- Go: 별도 PostgreSQL DB의 전체 시험 PASS, internal/app 178.567초. JSON pass 이벤트 656개(하위 시험 포함), 참조 PPTX 부재로 시험 1개 SKIP, DB 미설정 SKIP 없음.
- guard-check --changed f291f8c: 14개 도달 PASS; 새 uploadAttachments 시험 55%/53%.
- go vet ./..., go build ./..., 변경 Go 파일 gofmt, git diff --check PASS.
- 버전/잠금 파일/생성 문서 일치, OpenAPI 119경로, paging 10목록, modal 검사 PASS.
- npm ci, lint, 두 시간대 test 각각 173개, 프로덕션 빌드, scripts/build.sh PASS. 기존 500 kB 번들 크기 경고는 있음.
- render-docs.py ROADMAP_PLAN USER_GUIDE ADMIN_GUIDE PASS: HTML 3개와 로드맵 PDF. 가이드 PDF는 스크립트의 기존 별도 생성 관례대로 미생성.
- backup-check 최초 호스트 실행은 DSN 미설정으로 중단. 기존 PostgreSQL 시험 컨테이너에서 같은 스크립트를 실행해 백업/검증/복구/누락 파일 거부 PASS. 시험 자료와 임시 DB 정리 완료.
- 미실행: UI 클릭, 새 오류 시나리오 PPTX 출력, 오버플로 외 장애 주입, mutation/authz, 원격 gh 기반 advisory release-check, Docker install-check. 운영 배포·사후 관찰은 수행하지 않음.
