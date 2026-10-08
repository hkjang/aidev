# jikim v0.2.32 릴리즈 준비 결과

- 로컬 릴리즈 커밋: `096aa393259d6f66dc5956623469fafc77c6679e`
- 주석 태그: `v0.2.32`, 주석 `jikim v0.2.32`, 작성자·커미터·태거 `hkjang`
- 현재 detached HEAD에서 작성했고 작업 트리는 clean입니다. 원격 푸시·게시·업로드는 실행하지 않았습니다.

## 관례와 버전 결정 근거

실제 최근 태그 v0.2.30·v0.2.29·v0.2.28의 git show --stat, 커밋 메시지와 주석을 확인했습니다. 모두 패치 증가와 주석 태그, `fix: release ... for v0.2.x` 메시지를 사용합니다. 제공된 GitHub Release 제목은 `jikim v0.2.x`, 본문은 영어 자동 생성 `What's Changed`·PR 링크·`Full Changelog`, 자산은 tar.gz와 sha256 두 개입니다.

현재 HEAD에는 태그 목록에 없는 v0.2.31 릴리즈 커밋 aac28af가 포함되어 있고, 버전 파일·문서·CHANGELOG도 0.2.31입니다. 이를 재사용하지 않고 패치 하나를 올려 0.2.32로 정했습니다. GitHub 공개 이력의 기준은 제공된 최신 v0.2.30이므로 별도 release-notes.md는 #53과 #54를 포함합니다. 실제 게시 노트는 release.yml의 --generate-notes로 생성됩니다.

기존 릴리즈와 같은 20개 파일만 갱신했습니다. scripts/version.sh, internal/version/version.go, web/package.json, web/package-lock.json 두 루트 값, web/src/lib/format.ts, Compose 이미지, release.yml 예시·기본값, README·CONTRIBUTING·문서 프로파일을 갱신했습니다. 한국어 CHANGELOG에 같은 구조로 수정·변경·문서 항목을 추가하고 과거 항목은 보존했습니다. Dockerfile의 기존 v0.2.0 기본값은 이전 릴리즈처럼 보존하며 CI는 scripts/version.sh의 명시적 빌드 인수를 사용합니다. 화면 캡처 v0.2.9와 PDF v0.2.18 출처는 유지합니다.

## 검증과 자동화 경계

- `bash scripts/verify.sh`: exit 0. Go test·vet·gofmt, 프런트 59/59·lint·build, 문서·Compose 검증 통과.
- `go test ./internal/httpapi/ -run 'CSPReportBodySizeBoundary|PolicyReports|Tracking' -count=1 -v`: exit 0. CSP 경계 하위 테스트 10개 포함.
- 버전 필드 일치, 이전과 같은 릴리즈 파일 범위, 과거 CHANGELOG 보존, git diff --check 통과.
- 주석 태그·소스 버전·HEAD 일치, 작성자, detached HEAD와 clean 상태 확인.
- 로컬 PostgreSQL 연동은 DSN 미설정으로 실행되지 않았습니다. 이미지·브라우저 E2E·패키징은 이번 세션에서 실행하지 않았습니다. 러너가 제공한 PR #54 CI에는 소스 검사와 이미지·브라우저 E2E 두 검사가 success이나, 이는 새 릴리즈 커밋의 CI 결과는 아닙니다.
- 이전 릴리즈 커밋과 같은 로컬 검증을 수행했습니다. Makefile release-check의 이미지·스모크·E2E·package·verify-bundle 단계는 태그 푸시 뒤 release.yml이 동일한 빌드 이미지로 수행하고 성공한 경우에만 GitHub Release를 게시합니다.
- release.yml은 태그 푸시로 linux/amd64 이미지 빌드·레이블 검사·PostgreSQL 스모크·Playwright E2E·package-offline.sh·verify-offline-bundle.sh·GitHub Release 생성을 모두 수행합니다. 이전 자산 두 개를 자동 생성하므로 release.json은 github_release=false, assets=[]입니다.
- 기존 500kB 초과 청크 경고는 남습니다. 저장소에 기존 PDF 변환기가 없어 PDF를 재생성하지 않았고 CHANGELOG에 명시했습니다.

## 적용한 스킬과 배포 인계

Skill 도구가 없어 다음 로컬 원본과 references/sources.md를 읽었습니다. 외부 권위 자료의 대상(앱스토어 심사·광고 주장·라이선스 검토)은 이번 작업 범위에 해당하지 않습니다.

- /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/marketing/skills/product-launch/SKILL.md
- /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/release-and-deployment/SKILL.md

Tier 3 개선으로 릴리즈 노트만 준비했습니다. 대상은 추적 기능을 켜고 CSP 리포트를 조회하는 운영자입니다. 8KiB를 초과한 보고서의 기록 방지가 효과이며 속도 제한·동일 출처 통제나 DoS 전반의 해결을 주장하지 않습니다. 기존 204·빈 응답과 설정 gate를 유지하고 스키마 변경은 없습니다. 별도의 가격·패키징·캠페인 변경은 없습니다. 외부 사용자 첫 사용 검증과 실제 배포 후 관찰은 이번 로컬 릴리즈 세션에서 수행하지 않았습니다.

향후 배포 담당자는 hkjang으로 인계합니다. 첫 설치에서 정상·8192바이트 리포트의 수용과 초과 리포트 미기록, 204·빈 응답을 확인하는 것을 성공 기준으로 제안합니다. 단 한 건이라도 이 계약을 어기거나 readiness 검사가 실패하면 배포 확대를 중단하고 기존에 검증된 배포 이미지로 복귀합니다. 게시 자체는 CI의 어느 게이트라도 실패하면 중단됩니다. 데이터 마이그레이션은 없으며 실제 롤아웃이나 롤백은 이 세션에서 수행하지 않았습니다. 배포 후 첫 24시간의 CSP 기록 관련 운영 문의 확인도 배포 담당자에게 인계할 제안이며 예약·실행한 작업은 아닙니다.
