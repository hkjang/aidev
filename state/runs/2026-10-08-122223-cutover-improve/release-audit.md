# v1.18.0 릴리즈 관례 확인 및 실행 기록

## 스킬
- Skill/skills.list/skills.read 도구가 노출되지 않아 로컬 정본을 직접 읽음.
- marketing:product-launch: /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/marketing/skills/product-launch/SKILL.md
- technology:release-and-deployment: /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/release-and-deployment/SKILL.md
- 각 references/sources.md도 확인. 이번 작업에 앱스토어 정책, 광고 주장, 라이선스 판정은 해당하지 않아 외부 출처를 인용하지 않음.
- Tier 3: 개발·유지보수 담당자에게 검증 계약을 알리는 릴리즈 노트만 작성. 사용자 기능·가격·패키징 변경 없음. 채택 지표나 캠페인은 해당하지 않으며 회귀 테스트 유지가 목표.
- 사용자의 무인 실행 지시에 따라 외부 사용자를 섭외하지 않음. 기존 독립 비평 승인 기록과 브라우저 E2E를 검증 근거로 사용.
- 배포 노출/운영 반영은 외부 러너 이후 운영 담당자 책임. 이 세션은 로컬 릴리즈와 자산 준비만 수행.
- 중단 기준: 필수 검증 실패가 환경 보정으로 해결되지 않으면 릴리즈 담당 에이전트가 태그를 만들지 않고 failed 기록. 실제 운영 롤아웃은 수행하지 않음.

## 실제 이력 확인
- git tag --sort=-creatordate | head -10 및 최근 3개 git show --stat / 객체 형식 확인.
- v1.17.0 → aec0adc, v1.16.0 → e5fe006, v1.15.0 → 51ffd03. 모두 머지 커밋에 직접 붙은 주석 태그이며 주석은 vX.Y.Z - 한국어 제목.
- git log --oneline -60의 release/릴리즈/버전/버전번호 검색은 결과 없음. 전체 이력에도 별도 릴리즈 커밋 없음.
- package.json과 package-lock.json(최상위 및 packages[""]) 버전은 0.1.0. 최근 세 태그에서도 유지. 가이드 문서 버전은 v1.3.0, README 예시는 v1.2.0, 관리자 가이드 업그레이드 예시는 v1.4.0. 모두 릴리즈마다 갱신하는 버전 파일이 아님.
- CHANGELOG, docs/RELEASE*, VERSION, pyproject.toml, Cargo.toml, Chart.yaml, plugin.json, version.go, Makefile 및 release/offline 빌드 스크립트 없음.
- .github/workflows 없음. amplify.yml은 npm ci → npm run build의 앱 배포 설정이며 태그 릴리즈/자산 생성 절차가 아님. 실행하지 않음.
- GitHub Release는 외부에만 한국어 본문으로 보관. 이전 실행 기록 release-notes-v1.17.0.md, release.json, agent-release.txt 확인.
- v1.17.0 자산은 cutover-v1.17.0.tar.gz 하나. README 및 관리자 가이드의 docker build → docker save | gzip 절차 사용. SHA256/PDF/매니페스트를 별도 첨부하는 관례 없음.
- 기존 버전 증가 방식은 마이너 증가이므로 1.18.0 선택.
- 버전 수정·릴리즈 커밋을 새로 만들지 않고 현재 detached HEAD 머지 커밋에 주석 태그를 붙이는 관례 유지.

## 범위
- v1.17.0..HEAD: lib/activityData.test.ts 1파일 +61줄, 경계 테스트 7건. 프로덕션 코드/설정/의존성/데이터 변경 없음.
- Next 설치 후 로컬 배포/standalone output/Playwright 가이드를 읽음. 앱 코드는 작성하지 않음.
- 운영 클러스터 배포/관측 및 원격 푸시/업로드는 실행하지 않음. 사용자 명시 지시로 외부 러너에 인계.

## 초기 E2E 환경 실패
- 기본 실행은 격리 HOME 아래 Chromium 부재로 브라우저 실행 실패(18 pass, 23 fail).
- 설치된 동일 Playwright revision 1234를 사용하도록 PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright 지정 후 전체 재실행. HOME/소스/테스트 설정은 변경하지 않음.
- 최초 로그 release-e2e.log, 보정 후 로그 release-e2e-final.log를 함께 보존.

## 최종 검증
- npm ci 성공. lint, 타입 검사, Next build, git diff --check 통과.
- 단위 테스트 108 pass / 0 fail / 0 skipped. 환경 보정 후 전체 E2E 41 pass.
- Docker build 성공. 같은 이미지의 /, /pc, /admin, /api/activities HTTP 200 및 API activities 배열 확인. 임시 컨테이너 종료·제거 완료.
- docker save | gzip으로 필수 자산 생성. gzip 무결성 및 manifest.json의 cutover:v1.18.0 태그와 모든 레이어 존재 확인. 체크섬은 release-asset-verification.json에 보존(별도 배포 자산 아님).
- audit, 운영 환경 배포·관측은 수행하지 않음. 기존 모듈 타입/색상 경고 유지.
