# Vendra v0.7.58 릴리즈 검증

- 대상: 5ed444c (PR #130, 구현 2e67eec).
- 최근 v0.7.55/56/57은 머지 커밋을 가리키는 경량 태그이며 별도 릴리즈 커밋/태그 주석 없음. 다음 버전은 패치 증가 0.7.58.
- 최근 60개 커밋에서 별도 릴리즈 커밋 양식 없음. CHANGELOG.md 최신 항목은 v0.6.45이며 최근 릴리즈는 GitHub 자동 생성 영어 노트 사용.
- Makefile VERSION, web/package.json 및 package-lock.json 루트 버전은 0.6.21. internal/httpapi/app.go Version은 dev. Dockerfile VERSION=dev/COMMIT=unknown/BUILD_TIME=unknown 기본값은 빌드 인자로 대체됨. README 예제는 0.6.21/0.7.26/0.7.20, compose 예제는 0.7.26, requirements-traceability 예제는 0.3.0. 최근 관례대로 이 값들은 변경하지 않음.
- .github/workflows/release.yml은 v* 태그 푸시에 scripts/offline-release.sh를 실행하고 vendra-v<version>.tar.gz를 검증한 다음 Vendra v<version> GitHub Release 및 자동 노트를 생성하고 자산을 첨부함. 따라서 github_release=false, assets=[]로 러너에 전달.
- 로컬에서도 같은 스크립트의 Docker 빌드/아카이브 생성, gzip 검사, 이미지 존재 검사 통과. 원격 업로드/푸시 없음. 로컬 아카이브는 검증용이며 CI가 게시 자산을 다시 생성함.
- gofmt, go vet, Go 전체 테스트(독립 PostgreSQL 16 DB 세 개), go build, git diff --check 통과. httpapi 27.019초. 수동 성능 테스트 2개, 슈퍼유저 권한으로 REVOKE가 적용되지 않는 테스트 2개, 빈 공급업체 등록부를 요구하지만 등록부가 이미 채워져 있는 테스트 1개가 SKIP. 자세한 원인은 release-go.log 참조.
- npm ci --ignore-scripts, tsc, eslint, 프런트 테스트 21개 파일/93개 테스트, npm run build 통과.
- marketing:product-launch 및 technology:release-and-deployment 스킬은 제공된 도구/리소스/로컬 스킬 경로에서 발견되지 않아 고유 절차와 반환 형식을 적용할 수 없었음. 사용자의 명시적 릴리즈 절차와 저장소 CI를 적용함.
