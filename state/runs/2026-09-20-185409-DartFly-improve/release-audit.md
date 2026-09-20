# v2.72.0 릴리즈 근거

- 최근 태그 v2.69.0, v2.70.0, v2.71.0 모두 머지 커밋을 가리키는 경량 태그이며 주석·별도 릴리즈 커밋이 없다. 같은 방식으로 현재 detached HEAD에 v2.72.0 태그를 붙인다.
- 최근 60개 커밋에 별도 릴리즈 커밋 양식이 없고, 저장소에 CHANGELOG/릴리즈 노트 파일도 없다. 한국어 GitHub Release 본문을 외부 release-notes.md에 준비한다.
- internal/server/http.go의 개발 기본 Version은 v1.4.2로 이전 태그와 동일하다. Dockerfile ARG VERSION=dev를 deploy/build-release.sh가 실제 태그로 바꾸고 ldflags로 주입한다. 과거 시연 output 자료의 v1.2.0, 문서의 과거 버전 사례, 의존성·프로토콜 버전은 제품 릴리즈 버전 파일이 아니다.
- .github/workflows/ci.yml은 main 푸시·PR·수동 검사만 수행한다. 태그 트리거, GitHub Release 생성, 자산 업로드는 없다. 따라서 github_release=true.
- 기존 자산은 dartfly-v2.71.0.tar.gz 및 .sha256. README가 지정한 deploy/build-release.sh로 동일 이미지 빌드·기동 검사·docker save/gzip·체크섬 생성을 실행한다. 원격 전송 명령은 없다.
- 요청한 marketing:product-launch 및 technology:release-and-deployment는 제공 도구와 로컬 스킬 검색에서 찾지 못해 적용하지 못했다.
- 최초 브라우저 스모크는 삭제된 회차 HOME 캐시 경로로 실패. PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright로 설치된 Chromium을 지정하여 재검증한다.

- 최종 검사: Go race/vet/gofmt/build, JS 전체 테스트, Chromium 31페이지·편집기, 이미지 기동·로그인·질의 실행 모두 통과. gzip 무결성·SHA-256 및 배포 바이너리 v2.72.0 문자열 확인.
- 기존 관례에 따라 소스 변경·빈 릴리즈 커밋 없이 f09cfc0에 경량 태그 v2.72.0 생성. 원격 전송 없음. 업로드 자산은 기존과 같은 tar.gz 및 .sha256 두 개이며 빌더의 LOAD_OFFLINE.txt는 회차 자료로 보존.
