v0.9.66 로컬 릴리즈 완료

- marketing:product-launch: Tier 3, 한국어 릴리즈 노트. 외부 발표 없음.
- technology:release-and-deployment: 기존 품질 게이트 및 동일 Docker 산출물 스모크 검증. 롤백·배포 중단 조건은 릴리즈 노트에 기록.
- Skill 도구 미제공으로 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins 아래 지정 SKILL.md를 읽음.
- 최근 3개 주석 태그와 릴리즈 커밋 양식, changelog, release.sh, gh_release.ps1, RELEASE_GUIDE, ci.yml 확인. CI는 태그 릴리즈/자산 자동 생성 없음.
- AppVersion·문서·Compose·fixture: v0.9.66. web package.json/package-lock.json 0.1.0은 기존 독립 내부 버전 유지. PDF·스크린샷은 최근 릴리즈 관례대로 기존 자료 유지.
- 최초 검사는 버전 갱신 도중 실행되어 불일치로 실패. 완성된 파일 상태에서 전체 검사 재실행 및 통과.
- 커밋: 98c508154d1068c9c38a8034f15de9ab159753ec
- 원격 전송 없음. PostgreSQL·Keycloak·외부 Golden 모델 회귀·브라우저 E2E·외부 사용자 검증은 외부 환경 부재로 미실시. SQLite Docker 스모크 통과.
