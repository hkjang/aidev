# 과제서 2026-09-20 — Vendra

- 과제: 가이드 표류 가드 — 두 가이드가 참조하는 그림 전수와 관리자 가이드의 환경 변수 표를 코드·파일 시스템에 묶는 테스트 (가치 3 / 위험 1 / 작업량 S)

- 왜: 2026-09-11·09-12·09-14 회차가 매번 「참조된 그림 30장 ↔ 파일 목록 양방향 대조 누락·미사용 0」과 「환경 변수 표 ↔ internal/config 의 os.Getenv 전수 일치」를 **손으로** 대조했고, 그 뒤로도 가이드·설정이 회차마다 바뀌고 있다(tracking·mail·mcp-oauth). 지금 이 대조를 지키는 테스트가 없어 그림 하나를 이름 바꾸거나 환경 변수를 하나 더하면 가이드가 조용히 거짓이 된다 — `deployment_docs_test.go` 가 이미 같은 방식(저장소 파일을 읽어 코드와 대조)으로 compose·콜백 경로·CI DSN 을 지키고 있으니 그 옆에 두 가드를 더하면 다음 회차부터 사람 대조가 필요 없다.

- 수용 기준:
  1) `docs/USER_GUIDE.md`·`docs/ADMIN_GUIDE.md` 의 모든 `![…](images/…)` 참조가 `docs/` 기준으로 실제 존재하는 파일을 가리키고, 반대로 `docs/images/guide/*.png` 전부가 두 가이드 중 적어도 하나에서 참조된다 — 어느 쪽이 깨져도 테스트가 **그림 파일 이름을 들어** 실패한다(고아 그림 / 깨진 참조 각각 다른 메시지).
  2) `docs/ADMIN_GUIDE.md` 의 환경 변수 표(「필수」 표, 현재 `POSTGRES_DSN`·`BOOTSTRAP_ADMIN`·`BOOTSTRAP_ADMIN_PASSWORD`·`ENCRYPTION_KEY` 네 줄)가 `internal/config/config.go` 가 `os.Getenv` 로 읽는 이름 전수와 **양방향으로** 같다 — 코드가 읽는데 표에 없으면 그 이름을, 표에 있는데 코드가 안 읽으면 그 이름을 들어 실패한다. `TZ`·`VENDRA_IMAGE`·`VENDRA_GUIDE_*`·`VENDRA_TEST_*` 는 애플리케이션이 읽는 값이 아니므로(컨테이너·compose·스크립트·테스트 몫) 비교 대상에서 제외한다 — 제외 규칙을 테스트 주석에 적을 것.
  3) 테스트가 실제로 표류를 잡는다는 증명: 구현자가 (a) `docs/images/guide` 의 그림 하나를 임시로 이름 바꾸고, (b) 가이드 표에서 한 줄을 지우고 각각 테스트를 돌려 실패 메시지를 확인한 뒤 되돌린다(그 출력을 회차 노트에 적는다). 현재 트리에서는 통과해야 한다(2026-09-14 회차가 손으로 확인한 상태: 그림 30장·환경 변수 4개 — 미확인, 구현자가 첫 실행에서 확인).

- 건드릴 파일:
  - `internal/httpapi/deployment_docs_test.go` — `repoFile` 헬퍼(../../ 기준) 를 그대로 쓰고, 그 옆에 두 함수를 더한다. 이름은 이 파일의 관례대로 행동 문장: 예 `TestEveryPictureTheGuidesShowExists`, `TestAdminGuideNamesEveryEnvironmentVariableTheServerReads`. 그림 목록은 `os.ReadDir("../../docs/images/guide")`, 참조는 `regexp` `!\[[^\]]*\]\(([^)]+)\)` 로 긁는다(두 가이드가 `images/guide/xxx.png` 상대 경로만 쓴다 — 2026-09-20 확인, `docs/images/` 에 guide 외에는 logo·favicon 이 있고 가이드가 참조하지 않으므로 대조 범위는 `images/guide` 로 한정). 환경 변수는 `internal/config/config.go` 원문에서 `os\.Getenv\("([A-Z_]+)"\)` 를 긁고(다른 패키지에는 Getenv 가 없음 — 2026-09-20 `grep` 으로 확인, `os.LookupEnv` 도 없음), 가이드는 표 행 `| \`NAME\` |` 중 「필수」 표 구간(ADMIN_GUIDE.md 125~130행 부근, 「컨테이너 이미지가 함께 정하는 값」 표 이전)에서 긁는다. 구간을 자르는 기준은 표 제목 문자열이 아니라 「필수」 열이 있는 4열 표(`|---|---|---|---|`)로 잡는 편이 안전하다 — 제목 문구는 문서 편집에서 바뀌기 쉽다.
  - 새 파일을 만들어도 된다(`guide_docs_test.go`, package httpapi) — 다만 `repoFile` 은 deployment_docs_test.go 에 있으니 같은 패키지에 두어야 한다.
  - 다른 파일은 건드리지 않는다. 가이드 본문·PDF·config.go 는 수정 대상이 아니다(테스트가 현재 상태에서 통과해야 한다; 만약 첫 실행에서 실제 표류가 발견되면 가이드 쪽을 최소로 고치고 PDF 는 공용 md2pdf 로 다시 굽는다 — `--title "관리자 가이드" --project Vendra --version <최신 태그>`, 기억 파일 참조).

- 검증 명령:
  - `cd internal/httpapi && go test -run 'TestEveryPicture|TestAdminGuideNames|TestAdminGuide|TestCompose|TestCIRuns' -count=1 .` (DSN 없이 0.05초 — 2026-09-20 확인, 기존 넷 통과)
  - `gofmt -l internal cmd` (출력 없음), `go vet ./internal/... ./cmd/...`
  - `go test ./internal/... ./cmd/... -count=1` — 통합 테스트는 DSN 셋(`VENDRA_TEST_DSN`·`VENDRA_TEST_MIGRATE_DSN`·`VENDRA_TEST_UPGRADE_DSN`, docker postgres:16-alpine, 각각 다른 DB)이 있어야 돌고 없으면 조용히 skip. 이번 과제는 DB 를 쓰지 않으므로 DSN 없이도 이 과제의 증명은 완전하다 — 다만 전체 초록은 관례대로 한 번 돌린다.
  - 러너 비밀정보 검사: 테스트 안에 12자 이상의 리터럴을 password/secret/token 옆에 두지 말 것(예: 표 행을 그대로 문자열 상수로 넣지 말 것). diff 에 `gate.py secrets` 를 먼저 돌린다.

- 위험과 피할 것:
  - auth·migrations·workflows 를 건드리지 않는다. 마이그레이션 번호는 017 이 둘(`017_role_permissions.sql`·`017_tracking.sql`) — 적용 기록이 파일명 전체를 키로 써서 지금은 무해하지만(`internal/db/db.go:91~`), 새 마이그레이션은 이 과제에 없다.
  - 「효과 없는 변경」 반려 기준: 테스트가 현재 통과만 하고 아무것도 못 잡는 모양이 되지 않게, 수용 기준 3 의 「깨뜨려 보기」를 반드시 하고 노트에 남긴다. `len(named) < N` 이면 `t.Fatalf("… proves nothing")` 하는 `TestCIRunsEveryDocumentedTestDatabase` 의 자기 검증 패턴을 따를 것(그림 0장·변수 0개를 긁었으면 비교가 무의미하다고 실패).
  - 소스 문자열 검사를 「동작의 증거」로 쓰지 말라는 운영자 지시가 있으나, 이 과제는 문서 ↔ 코드 정합 가드이지 동작 검증이 아니다 — 문서 가드는 이 저장소에서 이미 같은 방식으로 셋 존재하며(`TestAdminGuidePublishesTheRealCallbackPath` 등) 반려된 적 없다. 대신 config 쪽은 가능하면 원문 긁기 대신 `config.Load()` 를 빈 환경에서 불러 오류 메시지에 이름이 나오는지도 함께 보는 식으로 실제 코드를 통과시키면 더 좋다(선택; 필수 아님 — `Load()` 가 세 이름을 한 문장으로 답하고 `ENCRYPTION_KEY` 는 따로 답한다, config.go:38·55).
  - Vitest 는 /mnt/c 에서 돌지 않지만 이번엔 web 을 건드리지 않으므로 무관.

- 차선 후보: CI 의 go job 이 `./cmd/...` 를 빼놓아 Makefile·README 와 불일치 — `.github/workflows/ci.yml:55` 의 `go test ./internal/... -count=1` 을 `go test ./internal/... ./cmd/... -count=1` 로(엄격하게 하는 방향이라 워크플로 보호 규칙에 걸리지 않음; 단 `cmd` 에 테스트 파일이 없어 지금은 컴파일 검사 효과뿐이므로 1순위가 성립할 때는 고르지 말 것). (2/1/S)
