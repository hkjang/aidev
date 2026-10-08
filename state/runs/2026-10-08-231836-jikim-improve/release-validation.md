# jikim v0.2.33 릴리즈 준비 기록

- 적용 스킬: marketing:product-launch, technology:release-and-deployment. 전용 Skill 도구가 없어 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/ 아래 각 SKILL.md와 references/sources.md를 읽었습니다. 이번 작업은 앱스토어·광고 주장·라이선스 심사를 포함하지 않아 외부 참고문헌 적용 대상이 없습니다.
- Tier 3: 기존 목록 API 이용자의 범위 초과 limit/offset 처리를 고치는 유지보수 릴리즈로 CHANGELOG와 릴리즈 노트만 준비합니다. 가격·패키징·신규 사용 대상 변화는 없습니다.
- 기준: 깨끗한 detached HEAD 54de779, 수정 fb7ecd8 / PR #55. 최근 태그 v0.2.32·v0.2.30·v0.2.29의 주석은 jikim vX.Y.Z, 커밋 제목은 fix: release ... for vX.Y.Z입니다. 패치 증가와 동일한 20개 파일 갱신 관례를 따릅니다.
- 소스 버전: scripts/version.sh=v0.2.33, Go 기본값=0.2.33-dev, web/package.json 및 lock의 루트 두 값=0.2.33, 프런트 폴백·Compose·워크플로 입력·현재 문서 프로파일=v0.2.33. 이전 CHANGELOG·v0.2.9 캡처 출처·v0.2.18 PDF는 보존합니다.
- 로컬 게이트: bash scripts/verify.sh 및 go test ./... -count=1 exit 0. 웹 59개, Go vet·포맷, lint·build, 문서, Compose 통과. 변경 파일 집합·정확한 버전 치환·diff 공백 검사 통과. 로그는 release-verify.log와 release-go-test.log입니다.
- PostgreSQL 통합은 이번 실행에서 JIKIM_TEST_POSTGRES_DSN 미설정으로 skip. 구현의 폐기 가능한 PostgreSQL 회귀 통과, 독립 비평 승인과 러너가 기록한 머지 전 CI 2개 성공을 확인했습니다. 새 사용자의 수동 첫 사용 검증은 수행하지 않았습니다.
- 기존 500kB 청크 경고 유지. PDF 변환기 부재에 따른 과거 표지 유지 사항은 CHANGELOG에 명시했습니다.
- .github/workflows/release.yml은 v*.*.* 태그 푸시 후 태그·소스·커밋 일치, 소스 검증, linux/amd64 이미지 빌드 및 레이블 확인, 내부 PostgreSQL·egress 차단 스모크, 전체 화면 브라우저 E2E, package-offline.sh와 verify-offline-bundle.sh를 실행합니다. 검증한 동일 이미지를 저장한 jikim-v0.2.33.tar.gz와 .sha256 두 자산으로 GitHub Release를 생성합니다. 따라서 runner의 github_release=false, assets=[]입니다. 로컬 이미지·브라우저·번들 게이트는 이전 릴리즈와 같이 CI 담당입니다.
- GitHub Release 제목은 jikim v0.2.33이며 본문은 이전 --generate-notes의 What's Changed / Full Changelog 형식으로 release-notes.md에 준비했습니다. 게시·원격 푸시는 외부 러너와 기존 워크플로가 수행합니다.
- 배포 후 확인 제안(실행·예약하지 않음): 운영 담당 hkjang이 제한된 검증 계정에서 기본 페이지, 정상 페이지, 범위 초과 limit/offset의 총 4개 요청을 비교하여 예상 HTTP 200·항목 순서 4/4를 확인한 후 확대합니다. 하나라도 불일치하거나 새 5xx가 발생하면 확대를 중단하고 기존 오프라인 설치 가이드에 따라 v0.2.32 이미지로 복귀합니다. 스키마 변경은 없습니다. 초기 운영 기간에는 목록 누락·건너뜀 문의의 재발 여부를 확인합니다.
