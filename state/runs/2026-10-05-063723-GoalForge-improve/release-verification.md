# v0.52.0 릴리즈 인계

- 릴리즈 커밋: 37bbc7f4e86fab3a7eacc59c2e9bd7286e3000e2; 시작 커밋: b932923 (PR #80 병합).
- 현재 detached HEAD 유지, 작성자 hkjang 유지, 트레일러 없음.
- 최근 v0.50.0·v0.51.0은 lightweight tag, v0.49.0은 annotated tag. 가장 최근 두 릴리즈 관례에 따라 v0.52.0은 lightweight tag.
- 버전 증가: 최근 0.x minor 증가 관례를 따라 0.51.0 → 0.52.0.
- 버전 파일: docs/GUIDE.md만 갱신. cmd/goalforge/version.go의 dev/빈 commit/빈 buildDate는 링크 시 주입하는 기본값으로 유지. docs/index.html 및 index_en.html의 v0.2.0은 과거 출시 이력이라 유지. 의존성·표준 팩 버전은 제품 버전과 별개.
- 저장소에 CHANGELOG/RELEASE 노트 파일 없음. 기존 한국어 GitHub Release 본문 양식(## 바뀐 것, 변경 항목, 검증)에 맞춰 release-notes.md 작성.
- 전용 version-check/release-check 및 Makefile 없음. 실제 CI와 v0.49.0 주석에 기록된 build/vet/test/gofmt/tidy/6플랫폼 교차 빌드/체크섬/버전 스모크 검사 통과.
- release-checks.json 및 release-*.log에 실행 근거 보존. 신규 세 테스트 모두 pass, 전체 32개 테스트 패키지 pass, 테스트 없는 두 패키지 및 기존 push 제한 테스트 4개 skip.
- 러너의 ci-f6d9e5c26327.json: PR 구현 SHA의 Linux/Windows/macOS 테스트, 포맷·tidy, 교차 빌드 총 5개 success. 로컬은 Linux에서 실행했으며 다른 OS 런타임 재검증은 하지 않음. 새 태그 CI 결과는 아직 없음.
- release.yml은 v* 태그 push에 반응하여 3개 OS에서 vet/test 후 6개 아카이브·SHA256SUMS 생성, 출처 증명, GitHub Release 생성·업로드. 따라서 github_release=false, assets=[]. 로컬 dist 산출물은 검증용이고 배포용 파일은 CI가 해당 태그에서 생성.
- 원격 push, GitHub Release 작성·업로드, 패키지 배포 미실행.

## 스킬 적용

Skill 도구가 제공되지 않아 다음 로컬 원문과 references/sources.md를 읽음. 반환 형식은 별도로 규정되어 있지 않으므로 사용자 지정 release.json 형식을 사용.
- /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/marketing/skills/product-launch/SKILL.md
- /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/release-and-deployment/SKILL.md

제품 출시 등급은 Tier 3(회귀 보호 테스트 추가)이며 기존 CLI/MCP 사용자·유지보수자가 대상. 새 기능·새 채택 목표가 없으므로 마케팅 캠페인·이용률 목표를 만들지 않고 릴리즈 노트로 변경과 한계를 전달. 외부 첫 사용자의 신규 체험 검증은 수행하지 않았으며 기존 독립 비평과 실제 CLI/MCP 저장 경로 테스트 결과를 인계. 기존 제품 동작·가격·스키마·기능 노출 변경 없음.

릴리즈 실행 주체는 외부 러너, 운영 책임자는 hkjang. 중단 기준은 필수 검증 중 하나라도 실패하거나 체크섬·버전·커밋 불일치가 발생하는 것. 로컬 기준은 모두 통과했으며, 태그 푸시 뒤 3개 OS verify가 실패하면 publish 작업이 실행되지 않음. 이후 워크플로 게시 결과·다운로드 체크섬·최초 사용 오류를 러너/운영자가 확인. 게시 후 문제가 확인되면 v0.51.0으로 되돌릴 수 있고 데이터 마이그레이션은 없음. 이 세션에서 외부 게시나 게시 후 검증 완료를 주장하지 않음.
