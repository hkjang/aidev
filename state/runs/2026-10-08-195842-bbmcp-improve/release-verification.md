# bbmcp v0.2.10 릴리즈 검증

- 적용 스킬: marketing:product-launch, technology:release-and-deployment. 전용 Skill 도구는 제공되지 않아 headcount 의 로컬 SKILL.md 와 sources.md 를 읽음. 외부 자료의 적용 주제(앱스토어 정책·광고 주장·라이선스 심사)는 이번 작업에 해당하지 않음.
- Tier 3 패치: PR 실행 도구 운영자를 대상으로 한국어 릴리즈 노트만 준비. 기능·가격·패키징 정책 변경 없음.
- 최신 태그 10개, 최근 3개 git show --stat 및 태그 객체, 최근 60개 커밋, 사용자가 제공한 GitHub Release 목록과 자산, scripts/build.sh·release.sh, CI 와 README 를 확인함.
- v0.2.7 / v0.2.8 / v0.2.9 는 주석 태그, 메시지 bbmcp v<버전>, 커밋 chore: v<버전> 릴리스. v0.2.10 으로 패치 증가.
- 별도 CHANGELOG/docs 릴리즈 노트 파일 및 버전 검사 스크립트 없음. 기존 GitHub Release 의 한국어 본문 및 release.sh 의 설치·업그레이드·해시 표 형식을 사용함.
- 버전 변경: VERSION, README.md, deploy/.env.example, deploy/docker-compose.yml, deploy/upgrade.sh 의 주석 예제, docs/index.html, docs/guide-admin.html. 관리자 가이드의 v0.2.9 마이그레이션 이력은 보존.
- 독립 버전 유지: web/package.json 및 package-lock.json 0.1.0, internal/version/version.go 개발 기본값 0.1.0 (이미지 빌드 시 VERSION 으로 주입), plugin/pom.xml 1.0.0 및 플러그인 descriptor 의 project.version. 이전 릴리즈도 동일하게 유지함.
- CI 는 main push / pull_request 검사만 수행하며 태그로 릴리즈나 자산을 만들지 않음. github_release=true 로 러너에 GitHub Release 생성을 위임함.
- commit: 410645f82c055294209f44b61e0a209407678dcf; annotated tag: v0.2.10. detached HEAD 유지, 작성자 hkjang, 트레일러 없음.
- Go build/vet 및 전용 DB(15532) 전체 테스트: 9개 패키지, 131 PASS(하위 테스트 포함), SKIP/FAIL 0. web npm ci/check/build 통과. 버전 일치·bash 문법·git diff 검사 통과.
- release.sh 와 build.sh 가 저장소에 100644 로 기록되어 직접 실행은 permission denied. 권한을 수정하지 않고 bash scripts/build.sh --save 를 실행하고 release.sh 의 install/tar/sha256 절차를 그대로 재현함. 도커 빌드 1800초 상한, 완료까지 대기함.
- 이미지 tar.gz 와 배포 tar.gz 및 각 sha256 파일 4개 완성. linux/amd64. 체크섬, 이미지 RepoTags, 배포 파일 내용/권한(0755/0644)/소유자(0:0) 검증 통과.
- 이미지 tar.gz 를 docker load 로 재적재한 동일 이미지로 기동 점검. 최초 host 네트워크 점검 시간 초과 후 bridge 내부 PostgreSQL 연결 및 localhost 포트 게시로 수정하여 성공. 별도 테스트 DB 와 컨테이너는 정리됨. 기존 개발/통합 테스트 fixture 만 재사용, 운영 비밀값 생성·수정·출력 없음.
- /healthz, /api/config, /api/version, OAuth 보호 리소스 메타데이터 버전 헤더, 내장 UI 확인. 실행 버전 0.2.10 및 커밋 410645f 확인.
- 제한: 운영 Bitbucket·클러스터 배포·실사용자의 독립 신규 설치 체험은 미실행. 이번 권한은 로컬 릴리즈 준비까지이므로 외부 발표/푸시/업로드 없음. 이전 독립 비평 및 CI 통과 근거는 journal.md 참조.
- 운영 인계(미실행): hkjang 이 기존 테스트 계정 1개로 업그레이드 직후 정상 PR 실행과 조회 실패 시 차단을 확인하고 첫 24시간 PERMISSION_UNKNOWN 관련 문의·감사 로그를 관찰. 정상 조회에서 실행 실패 또는 제한 우회 1건이면 확대 중단. 문제 시 영향 도구를 비활성화하고 필요 시 보관한 v0.2.9 이미지와 BBMCP_VERSION 을 사용해 되돌리되, 이전 버전에는 이번 차단 보완이 없음을 고려. 스키마 변경 없음.
