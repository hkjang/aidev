# 회차 노트 2026-10-17-095933-sqlon-improve — sqlon
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:59] base pinned — main@2955489
- [러너 09:59] autonomy release — 
- [러너 09:59] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- 베이스에 `.github/workflows` 가 아예 없어 PR 이 전부 'CI 검사 없음' 이었다. 이미 통과하는 5개 명령(`go mod download`→`go build ./...`→`go vet ./...`→`go test ./... -count=1`→`sh scripts/build.sh`)만 `.github/workflows/ci.yml` 로 옮겼다(커밋 0cb5554, 변경 파일 2개 = ci.yml + CHANGELOG 5줄).
- 확신 없는 곳: **GitHub 러너에서의 실제 실행은 미검증**이다(이 세션에 push 권한 없음). 로컬에서 `GOTOOLCHAIN=go1.25.0` 으로 5개 명령 전부 통과 + `git archive` 깨끗한 체크아웃(/tmp/sqlon-ci-clean)에서 재확인(test 20 ok / FAIL 0, build.sh 3타깃)까지가 증거의 한계다. `actions/setup-go@v5` 의 `cache: true` 가 러너에서 캐시 키를 어떻게 잡는지, ubuntu-latest 에 go1.25.0 이 설치되는 속도는 확인하지 못했다. YAML 은 PyYAML 파싱으로 구조만 확인했다(GitHub 스키마 검증기는 돌리지 않았다).
- 일부러 하지 않은 것: `gofmt -l`(CRLF 때문에 지금 85개 파일을 뱉어 통과하지 않음), integration 태그 테스트(실 DB 컨테이너 필요), Oracle 빌드(CGO+Instant Client), `test/ui/menus.cjs`(Playwright + 메타 DB 서버 필요, 저장소에 package.json 없음). 이유는 ci.yml 상단 주석에도 적어 두었다.
- push 트리거는 `branches: [main]` 로 좁혔다 — 러너 PR 은 같은 저장소 브랜치에서 올라와 `pull_request` 로 검사가 붙으므로, 좁히지 않으면 같은 커밋이 두 번 돈다. PR 게이팅에는 영향이 없다.
- 다음 역할이 조심할 것: 이 PR 은 보호 경로(.github/workflows)라 사람 승인이 필요하다. 2026-09-26(3935e5e)·2026-10-03(b236046)에 사람이 승인해 머지한 것과 같은 모양인데 그 커밋들이 이 베이스에 통합되어 있지 않다 — 반려된 접근의 재제출이 아니라 베이스 분기 문제다. CHANGELOG 는 혼합 줄바꿈 때문에 Edit 도구 대신 바이트 삽입으로 다뤘고 `git diff --numstat` 5/0 을 확인했으니, 수리 단계에서도 같은 방식을 쓸 것.
- [러너 10:06] verify passed — 검증 3개 통과 (auto)
- [러너 10:06] pr created — https://github.com/hkjang/sqlon/pull/17
- [러너 10:06] guard held — .github/workflows/ci.yml 
