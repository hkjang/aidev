# 릴리즈 관례 확인

- 요청된 marketing:product-launch, technology:release-and-deployment는 Skill 호출 도구가 없어
  /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/ 아래 해당 SKILL.md와 sources.md를 직접 읽었다.
  Tier three 개선으로 분류하고 한국어 릴리즈 노트와 로컬 검증에 범위를 맞췄다.
- git tag --sort=-creatordate 결과는 v0.1.1 하나뿐. 최근 태그 3개 중 확인 가능한 것은 1개이며,
  git show --stat, 태그 주석, 대상 커밋 메시지를 확인했다. 주석 태그 본문이 기존 릴리즈 노트 위치다.
- 최근 60개 커밋과 VERSION 이력에서 0.1.1 → 0.1.2 → 0.1.3 패치 증가를 확인했다.
  최근 메시지의 v0.1.x 한국어 형식을 따라 다음 버전을 0.1.4로 정했다.
- VERSION, 4개 package.json, package-lock.json의 애플리케이션 버전은 0.1.3이었다.
  wrangler.toml은 0.1.3, .env.example의 INSIGHT_VERSION은 0.1.2로 뒤처져 있어 함께 0.1.4로 맞췄다.
  engineering 문서의 0.1.0·0.1.2·0.1.3은 문서 갱신일 또는 과거 동작 설명이므로 유지했다.
  벤더·의존성 버전은 애플리케이션 버전이 아니므로 유지했다.
- CHANGELOG/RELEASE 문서와 release/offline/package 자산 스크립트·Makefile이 없다.
  제공된 최근 GitHub Release 목록도 비어 있어 github_release=false, assets=[]로 기록한다.
- .github/workflows/check.yml은 main push 및 PR에서 검사·빌드·실 DB e2e·브라우저 검사를 실행한다.
  태그/Release 트리거, GitHub Release 생성, 자산 업로드, 패키지 배포는 없다.
- 이전 주석 태그 릴리즈의 로컬 게이트(check, build, VERSION/wrangler 일치)를 다시 통과했다.
  deploy-pages.sh를 실행하면 비밀값 로드 및 원격 배포가 진행되므로 그 로컬 검사만 재현했다.
  최근 기능 커밋에 기록된 e2e/화면 검사와 이번 구현의 e2e 기록은 이번 세션 실행 결과와 구분했다.
- 새 스키마·기능·비밀값 변경 없음. 원격 전송 없음. 실제 배포 및 운영 검증은 수행하지 않았다.
