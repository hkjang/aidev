# confmcp v0.1.1 릴리즈 인계

- 적용 스킬: marketing:product-launch, technology:release-and-deployment. 전용 Skill 도구가 없어 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/ 아래 해당 SKILL.md와 references/sources.md를 읽었다. 이번 저장소 작업에 외부 출처의 정책·라이선스 해석은 필요하지 않았다.
- 출시 규모: Tier three, 기존 사용자 대상 버그 수정. 별도 외부 공지·영업 캠페인 없이 기존 한국어 GitHub Release 본문 형식을 따른다. 운영자 질문(무엇이 바뀌는지, 기존 승인 영향, 업그레이드·롤백)은 본문 및 관리자 가이드에 기록했다.
- 관례: 태그는 v0.1.0 하나뿐이어서 최근 3개 중 존재하는 1개를 조사했다. 주석은 confmcp v0.1.0, 대상 커밋은 afb0115(docs: GitHub Pages 소개·사용자·관리자 가이드, 화면 캡처, e2e 보강). 전체 이력에 전용 릴리즈 커밋이나 증가 패턴은 없으므로 작은 수정에 패치 버전 0.1.1과 기존의 conventional prefix + 한국어 메시지 양식을 적용했다.
- 버전 목록: VERSION, internal/version/version.go, web/package.json, web/package-lock.json, deploy/.env.example, deploy/docker-compose.yml 및 README·배포 스크립트 예제·서비스 문서·SVG의 서비스 버전을 갱신했다. plugin/pom.xml 및 플러그인 설명/모의 플러그인의 0.1.0은 별도 구성요소 버전이다. 기존 화면 캡처 설명과 v0.1.0 인수검증 이력은 역사적 값으로 보존했다.
- 기존 CHANGELOG·RELEASE 문서는 없으며 scripts/release.sh가 한국어 GitHub Release 본문을 임시 파일로 만들어 전달하는 관례다. 새 본문은 worktree 밖 release-notes-v0.1.1.md에 보존했다.
- ci.yml은 main push 및 PR의 Go 검사·웹 빌드·이미지 빌드/기동만 수행하며 태그 릴리즈·자산 자동 업로드는 없다. github_release=true로 외부 러너에 인계한다.
- 패키징: scripts/release.sh와 scripts/build.sh가 모두 git mode 100644여서 직접 실행 시 Permission denied였다. 업로드 단계 없이 bash scripts/build.sh --save로 동일 빌드 부분을 실행했다. 원래 GitHub 자산은 tar.gz 한 개이고, 기존 빌드 스크립트가 함께 만드는 sha256도 이번 assets에 모두 포함했다.
- 품질 게이트: DB 포함 Go 전체 test/build/vet, 웹 ci/check/build, 버전 일치, shell syntax, git diff --check, Docker build/save/load, 체크섬, 적재 이미지 HTTP·내장 자산·버전/커밋/헤더 확인 통과. 실제 외부 연동·운영 배포·팀 외부 사용자의 처음부터 사용해보기는 무인 로컬 릴리즈 범위에서 수행할 수 없어 미실시다.
- 운영 확대/중단 판단 담당: 배포 환경의 운영 담당자. 첫 제한 사용자 테스트에서 잘못된 위치 치환 또는 승인 우회 1건이면 쓰기 중지 및 확대 보류. 롤백 절차와 기존 승인·재시도 영향은 본문에 포함했다.
- 배포 후 확인 계획(자동 예약 아님): 운영 담당자가 첫 7일간 replace_text를 사용하는 계정의 성공 여부, 재승인/충돌 문의, 잘못된 위치 치환·승인 우회 건수를 확인한다. 기능 수용 기준은 공백 회귀 6사례 통과와 잘못된 치환·승인 우회 0건이다. 실제 사용자 채택 수치는 운영 접근이 없어 측정하지 않았다.
- 원격 전송 없이 detached HEAD에 커밋·주석 태그를 만든다. 원격 게시와 운영 배포의 완료를 주장하지 않는다.
