# v0.313.0 릴리즈 기록

- 기준 커밋: 700caba (0473e95 포함), detached HEAD 유지.
- 최근 태그 v0.312.0, v0.311.0, v0.310.0 은 주석 태그. 기존 minor 증가 패턴 적용.
- VERSION, frontend/package.json, OpenAPI, 배포 파일 3개, README, 안내서 2개를 갱신. package-lock.json 의 0.93.0 은 이전 태그에서도 동일하여 기존 관례대로 유지. Go 기본 버전 dev 는 빌드 시 주입.
- 한국어 노트: .github/release-notes/v0.313.0.md. 로드맵 기록·HTML·PDF, 안내서 HTML 갱신.
- release.yaml 이 태그 푸시에서 검증, linux/amd64 Docker 이미지 빌드, weekly-v0.313.0.tar.gz 생성 및 GitHub Release 게시를 담당. 따라서 로컬 assets=[] 및 github_release=false.
- 실제 DB 전체 Go 시험, vet/build, 가드 341개, 버전/OpenAPI/쪽넘김/모달, 프런트엔드 두 시간대 173개씩, 타입 검사/빌드, scripts/build.sh 통과. 세부 로그는 이 디렉터리의 release-*.log.
- 원격 advisory release-check.sh 는 gh 사용 불가 지시로 생략. 원격 전송·실제 배포 없음.
- marketing:product-launch 및 technology:release-and-deployment 는 Skill 도구가 없어 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins 아래 로컬 SKILL.md 원본으로 적용. Tier three 개선이며 릴리즈 노트만 준비. 이전 비평자의 실제 DB 회귀 검증을 인계받았으며 새 독립 사용자 체험/실제 운영 관측은 수행하지 않음. 배포 중단·롤백 및 다음 날 관측 기준은 노트에 명시.
