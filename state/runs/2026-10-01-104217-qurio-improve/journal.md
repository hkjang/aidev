# 회차 노트 2026-10-01-104217-qurio-improve — qurio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:42] base pinned — main@3f7cf65
- [러너 10:42] autonomy release — 
- [러너 10:42] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- 왜: 보고된 CI 실패는 재현 불가(로그 blob 404 → 단계 '' + BlobNotFound XML). CI 전 단계를 로컬 실행해 전부 통과. 대신 그 단계(ci.yml 의 이름 없는 통합 `- run:`)에 실재한 결함을 고쳤다 — setup-go 의 `cache: true` 가 GOCACHE(테스트 결과 캐시)를 실행 간 캐시하는데 전체 통합 실행만 `-count=1` 이 없어, **public 테이블 0개인 빈 DB 에서 20여 패키지가 `ok (cached)`** 로 보고됐다. ci.yml·release.yml 에 `-count=1` + `scripts/ci_workflow_test.go` 회귀 가드. 프로덕션 Go 코드 무변경. 커밋 12223b0.
- 확신 없는 곳: **이 수정이 보고된 실패의 원인이라는 증거는 없다.** 캐시 적중이 부분적이면 실제 실행 패키지 집합이 매 실행 달라지므로 "코드 변경 없이 red 로 바뀜" 을 설명할 수는 있지만, 실제 로그를 못 봤으므로 추정이다. 비공개 저장소라 GitHub API 조회도 404.
- 검증 못 한 것: `--with-deps` 의 apt 경로, ubuntu-latest 런너 이미지, GitHub Actions 캐시/로그 서비스 동작, release.yml 의 Oracle 실서비스 경로(수정은 ci.yml 과 동일한 한 줄).
- 일부러 안 한 것: `cache: true` 해제(모듈/빌드 캐시는 유익하고 문제는 결과 캐시뿐), `continue-on-error`·재시도 추가(실패를 가리는 변경), `timeout-minutes` 추가(ci.yml 에 없음 — ideas.json 에 다음 회차로 남김).
- 다음 역할 주의: `scripts/ci_workflow_test.go` 는 DB 불필요(워크플로 YAML 문자열 검사, `go test ./...` 와 CI 통합 실행 둘 다 포함). 통합 스위트는 이제 CI 에서 캐시 없이 매번 완주하므로 잡 시간이 늘어난다(로컬 ~1분 40초, 31 패키지). 검증용 도커 컨테이너는 제거했다.
- [러너 11:08] verify passed — 검증 9개 통과 (auto)
- [러너 11:08] pr created — https://github.com/hkjang/qurio/pull/30
- [러너 11:08] guard held — .github/workflows/ci.yml .github/workflows/release.yml 
