# 수리 요약 — 고칠 결함 없음 (커밋 없이 종료)

- 지적이 틀렸다. CI 의 lint·test 실패는 코드가 아니라 **계정 빌링 차단**이다: `gh run view 38951938252` 의 ANNOTATIONS 가 두 잡 모두 "The job was not started because recent account payments have failed or your spending limit needs to be increased" 이고, 어떤 step 도 실행되지 않았다(그래서 둘 다 3초, `--log-failed` 는 빈 출력).
- HEAD(9e8284d)에서 CI 와 같은 검사를 전부 로컬 재현해 통과: lint 잡 = `gofmt -l .` 무출력·`go vet ./...` 무출력·`verify-version.sh` "consistent: 1.2.37"; test 잡 = `go test -race -count=1 ./...` 전체 ok(internal/server 146.667s, 전용 postgres:16-alpine 링크 컨테이너), `npm ci && npm test && npm run build` 18파일/81테스트 통과·빌드 846ms, `docker build` exit 0.
- 비평가가 남긴 "DSN 없이 SKIP 된 채 게이트 통과" 우려도 해소했다: `-v -run` 으로 새 테스트가 SKIP 이 아니라 RUN·PASS 임을 확인 — HTTP 4건(legacyCreateRejectsNULInStringField·legacyPatchRejectsNULInStringField·legacyMenuInfoRejectsNULInNestedJSONB·legacyPatchRejectsNULInRowIdentifier)과 유닛 5건 전부 PASS.
- 따라서 PR #48 의 변경(레거시 리소스 문자열·행 식별자·menu_info 중첩 문자열의 NUL → 400 `resource_invalid`)에는 손댈 것이 없어 커밋하지 않았다. 워크플로·테스트를 고쳐 통과시키는 것은 금지 사항이라 하지 않았다.
- 차단 해제는 저장소 소유자만 가능하다(GitHub Billing & plans). 그때까지 이 PR 의 CI 는 내용과 무관하게 계속 빨갛다. 검증에 쓴 컨테이너·이미지는 제거했고 worktree 는 clean 이다.
