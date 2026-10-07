# AppStore v2.11.16 릴리즈 인계

요청한 marketing:product-launch 및 technology:release-and-deployment는 Skill 호출 도구가 없어 headcount/plugins의 해당 SKILL.md와 references/sources.md 원문을 읽어 적용했다. 별도 반환 템플릿은 없었다. 이번 범위는 기존 소유자의 결과 해석을 돕는 Tier 3 개선이며 릴리즈 노트로 전달한다. 외부 공지·지원 메시지·배포는 수행하지 않는다. 실제 사용자 첫 사용·지원 문의·운영 채택률은 이 로컬 검증으로 확인할 수 없다.

최근 v2.11.13~v2.11.15는 patch 증가, chore(release): AppStore vX.Y.Z 커밋, 한국어 상세 본문과 AppStore vX.Y.Z 주석 태그를 사용한다. 저장소에 별도 CHANGELOG/RELEASE 노트 파일은 없고 커밋/태그에 변경 설명을 남긴다. 동일 관례로 release-notes.md 내용을 태그 주석으로 사용한다.

버전 원천은 안정 SemVer 태그이다. package/lock의 제품 버전, Compose 기본값, README, Pages 가이드, Markdown/PDF 가이드와 캡처 manifest를 갱신했다. Go buildinfo와 Dockerfile/Makefile의 dev 값은 빌드시 태그를 주입하는 기본값이다. 독립 의존성 버전과 E2E fixture 버전은 변경하지 않는다. 검증 로그는 release-logs/에 보존한다.

release.yml이 v*.*.* 태그 푸시 후 test → image/archive build → load/non-root/offline smoke → SHA-256 영어 본문 생성 → GitHub Release 및 custom asset 1개 업로드를 수행한다. 최근 Release 본문/자산 형식과 일치한다. 자산은 appstore-v2.11.16.tar.gz 한 개이며 CI가 scripts/release-image.sh로 만든다. 별도 sha256/PDF/compose 자산은 없다. 이에 따라 github_release=false, assets=[]이며 로컬에서 Docker 자산을 만들지 않았다. 사용자/관리자 PDF는 저장소 문서로 커밋한다. notes_file은 한국어 태그 노트 보존본이며 GitHub Release의 최종 영어 본문과 digest는 워크플로가 생성한다.

검증: Go race/vet/gofmt, 새 UI를 embed한 Go build, React 17 files/111 passed, lint/Prettier/build, E2E 81 passed/1 skipped(재시도 없음), 캡처 90개/체크섬, PDF 표지/태그 링크/file URI 부재, 환경/오프라인/문서/버전 일치/diff 검사 통과. 캡처 변환 최초 시도는 브라우저 기본 경로 부재로 실패하여 설치 경로 지정 후 성공했고, PDF 스크립트 직접 호출은 실행 비트 부재로 실패하여 sh 실행으로 해결했다. 제품/스크립트 변경은 추가하지 않았다.

검증 한계: DB DSN 미설정으로 DB 통합 테스트 skip, 실제 DB·Keycloak·SecCheck 연동 미검증. E2E는 실제 번들/Chromium과 HTTP fixture 기반이다. Docker build/load/smoke 및 원격 CI·GitHub Release 생성·다운로드 후 검증은 외부 러너의 푸시 후 워크플로에 위임한다.

운영 인계: 릴리즈 워크플로의 검사 또는 image smoke 실패 시 게시를 중단한다. 배포 담당자는 기존 이미지 v2.11.15를 보존하고, 배포 뒤 ready/version과 소유자의 최종 결과 표시·제출 조건을 확인한다. ready 실패 또는 제출 조건 회귀가 하나라도 재현되면 v2.11.15로 복귀한다. 서버/schema 변경이 없어 이번 변경을 위한 데이터 마이그레이션은 없다. 실제 배포·관찰과 독립 사용자 검증은 이 세션에서 수행하지 않았다.
