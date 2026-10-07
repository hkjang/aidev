# Data Works v0.9.69 릴리즈 실행 기록

- 적용 스킬: marketing:product-launch, technology:release-and-deployment. Skill 도구 미제공으로 로컬 marketplace의 SKILL.md 및 references/sources.md를 직접 읽었다. 이번 작업은 외부 출처가 관할하는 앱스토어·광고·라이선스 판단을 포함하지 않는다.
- 범위: Tier 3 운영 문서 개선. 대상은 GitHub Pages 배포 장애를 판별하는 저장소 운영자. 신규 기능·스키마·요금제 변경 없음. 외부 공지는 릴리즈 노트만 준비했으며 전송하지 않았다.
- 버전 근거: 최근 주석 태그 v0.9.67/66/65는 패치 증가, 주석 Release vVERSION, 커밋 chore: release vVERSION. HEAD에는 v0.9.68 릴리즈 커밋 016b281이 이미 존재하므로 v0.9.69 선택. web/package.json 및 lock의 0.1.0은 기존 독립 패키지 버전으로 유지. 기존 15개 릴리즈 파일 정렬 관례 준수.
- 최근 게시된 GitHub Release는 제공된 정보상 v0.9.67이다. 따라서 gh_release.ps1의 이전 버전 기준을 v0.9.67로 두는 방식으로 v0.9.68의 누적 변경도 이번 노트에 포함했다.
- 자동화: ci.yml은 main push/PR/수동 검증만 수행하고 태그로 Release나 자산을 만들지 않는다. 기존 scripts/release.sh로 linux/amd64 이미지 빌드·docker save·gzip 수행. custom asset은 tar.gz 하나. gh나 원격 쓰기 명령 미사용.
- 품질 기준: npm ci, 루트 npm run lint, npm test --silent(11파일/36사례), npm run build 통과. go build ./..., go vet ./..., go test ./... -count=1, go build ./cmd/dataworks 통과. API audit 550 routes / 612 OpenAPI paths, 모든 gap 배열 비어 있음. 버전·changelog 구조 검사 포함.
- npm ci에서 high 취약점 3건 보고. 설치와 기존 필수 게이트는 모두 성공. lock 변경이나 게이트 완화 없음. 별도 의존성 개선 대상으로 남김.
- Pages: CI 이미지 v1.0.13로 변경된 문서 렌더링 성공. 결과 index.html의 v0.9.69와 OPERATIONS.html의 진단 절 확인. 토큰 생성 없이 GitHub metadata 로컬 스텁 사용. 실제 GitHub API metadata 및 호스티드 러너 배정은 검증 범위 밖.
- 자산: gzip 무결성, manifest RepoTags, linux/amd64·OCI 버전, docker load 전후 image ID 동일 확인. 적재한 동일 이미지로 임시 SQLite 기동; health·Workbench·번들 JS HTTP 200, /auth/me 버전 v0.9.69 확인. 컨테이너와 임시 볼륨 제거 완료.
- 환경 한계: 운영 PostgreSQL 접속 정보와 기존 자격증명이 없어 운영 DB 스모크·클러스터 검증은 생략했다. 비밀값 생성·변경 없음. 제3자의 최초 사용 검증은 무인 세션에서 확보할 수 없어 수행했다고 주장하지 않으며, 로컬 초기 기동 검증 결과만 기록했다.
- 중단 기준: 필수 검사 또는 아카이브/스모크 실패 시 릴리즈 중단. 게시·배포 후 최신 Pages 실패나 버전 불일치 시 확대 중단, 잡 API로 후속 실행의 concurrency 취소 여부와 러너 미배정을 판별. 운영 판단 담당은 저장소 소유자 hkjang; 이번 세션의 원격 조치는 없음.
- 게시 후 확인(외부 러너/운영자 인계, 미실행): 최종 main push 뒤 마지막 Pages 실행 하나의 성공 및 사이트 v0.9.69 확인, GitHub Release custom asset이 dataworks-v0.9.69.tar.gz 하나인지 확인. 실패 시 docs/OPERATIONS.md 절차로 재실행 판단. 런타임 회귀 시 직전 검증 이미지를 재적재하여 롤백하며 기존 데이터와 키는 유지.
- 관찰 목표: 게시 후 24시간 내 운영자 1명이 진단 절차로 최신 Pages 실행을 분류하고 사이트 버전 일치를 확인한 기록 1건. 문서 사용 중 첫 문의·실패가 생기면 해당 절차를 점검한다. 이 목표는 인계용이며 달성으로 보고하지 않는다.
