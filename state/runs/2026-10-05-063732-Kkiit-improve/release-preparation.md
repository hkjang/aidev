# Kkiit v0.4.14 릴리즈 준비

## 관례 확인
- 최근 태그 v0.4.13/v0.4.12/v0.4.11의 git show --stat, 커밋 본문, 태그 객체를 확인했다. 모두 hkjang의 주석 태그이며 주석은 `Kkiit v버전`이다.
- 최근 커밋 제목은 `Kkiit v버전 — 한국어 변경 요약`이고, 패치 단위 증가를 유지한다.
- 현재 버전 0.4.13에서 0.4.14로 VERSION, compose.yaml, web/package.json, web/package-lock.json(두 항목), docs/openapi.yaml, README.md(세 예시)를 갱신한다.
- 별도 CHANGELOG나 docs/RELEASE 문서는 없다. 상세 변경 이력은 릴리즈 커밋 본문에, GitHub 본문은 `## What's Changed`와 한국어 PR 제목, Full Changelog 링크에 담겨 있다. 이 형식을 release-notes.md에 보존했다.
- docs/ADMIN_GUIDE.md의 v0.4.0은 기존 설치 예시이며 최근 릴리즈들도 갱신하지 않았다. cmd/kkiit/main.go의 0.0.0-dev와 Dockerfile의 개발 기본값은 VERSION을 빌드 인수로 주입받는다. 의존성 버전은 릴리즈 버전이 아니다.
- scripts/release.sh와 Makefile release 타깃은 docker build 및 docker save | gzip을 수행한다. README는 태그 기반 자동 릴리즈를 명시한다.
- .github/workflows/release.yml은 v* 태그의 VERSION 일치를 검사하고 Docker 이미지를 빌드한 뒤 압축, 삭제·재로드 검증, kkiit-v버전.tar.gz 업로드와 GitHub Release 생성을 모두 수행한다. 제공된 최근 GitHub Release의 단일 자산과 일치한다. 따라서 러너 계약에 따라 assets=[], github_release=false다.

## 스킬 적용 및 운영 인계
- 전용 Skill 도구가 없어 로컬 marketing/product-launch 및 technology/release-and-deployment SKILL.md와 references/sources.md를 읽었다. 고정 반환 스키마는 없으며 사용자 release.json 스키마를 따른다. 외부 출처의 앱스토어·광고·라이선스 판단은 이번 작업에 해당하지 않는다.
- Tier 3 개선: 퍼센트 쿠폰을 적용하는 구매자의 큰 양수 int64 주문에서 할인액 계산을 바로잡는다. 릴리즈 노트만 준비하며 마케팅 공지는 없다. 가격·패키지·API 사용법 변경은 없다.
- 독립 사용 검증은 앞선 비평 단계에서 실제 서버와 폐기 PostgreSQL로 수행했다. 결제액 0은 판매자 정산액이 아니다. 큰 금액의 후속 결제·납품·정산·환불 정확성을 보장하는 릴리즈로 안내하지 않는다.
- 릴리즈 기준은 기존 Go/웹 검사 및 PostgreSQL 통합 검증의 통과, 버전 표기 일치, 기능 변경 없는 릴리즈 커밋이다. 데이터베이스 스키마 변경은 없다.
- 원격 배포는 이 세션 범위 밖이다. 운영 후 판단 담당은 저장소 운영자 hkjang이며, 첫 24시간 동안 퍼센트 쿠폰 preview/order의 잘못된 할인액 또는 적용 가능한 쿠폰의 409 재현이 1건이라도 확인되면 확대를 중단하고 원인을 검토한다. 롤백이 필요하면 기존 v0.4.13 산출물을 사용하되 기존 큰 금액 할인 버그가 되살아남을 고려한다.
- 운영 성공 지표는 발표 조회수가 아니라 정상 할인 적용 및 주문 생성이며, 테스트 통과 수와 실제 운영 관측을 구분한다. 이번에는 운영 배포·관측을 실행하지 않았다.

## 최종 검증 결과
- 릴리즈 커밋: b1199a3e9a6a69bb43f169bb03224d3d40fa2155, 주석 태그: v0.4.14 (`Kkiit v0.4.14`). detached HEAD 유지, 작성자 hkjang, 트레일러 없음.
- Go 전체 테스트·서브테스트 221건 PASS, 별도 make test-integration 88건 PASS(89.636초), 테스트 SKIP 0. 실제 폐기 PostgreSQL 16 컨테이너는 검증 후 정리했다.
- gofmt, go vet, go build, git diff --check 통과. Node.js 24.18.0에서 npm ci --ignore-scripts, lint, 테스트 10건(실패·SKIP 0), build 통과. make check 구성 검사를 동일하게 실행하되 웹 부분은 선언 버전의 Node 컨테이너에서 수행했다.
- 버전 표기 6파일 일치, 태그와 HEAD 및 VERSION 일치, 작업 트리 깨끗함.
- 로그: release-go-test.jsonl, release-integration.jsonl, release-web.log, release-go-vet.log, release-go-build.log.
- 원격 push·게시·배포 없음. 이미지 압축 및 GitHub Release는 러너의 태그 푸시 이후 기존 CI가 실행하므로 로컬 자산 생성은 해당 없음.
