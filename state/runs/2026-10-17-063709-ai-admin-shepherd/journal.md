# PR 처리기 노트 2026-10-17-063709-ai-admin-shepherd — ai-admin PR #48
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-17-032113-ai-admin-improve)
# 회차 노트 2026-10-17-032113-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:21] base pinned — main@2b2ef67
- [러너 03:21] autonomy release — 

## 정찰 노트
- CI 수리형 과제를 버렸다: 직전 3회차가 lint·go test -race·콜드 캐시·web npm·docker build 를 전부 로컬 통과시켜 재현 후보를 소진하고 3연속 verify-failed 였고, 운영자 메모가 "2026-10-15 이후 실패 = Actions 빌링 차단" 으로 결론 냈다. 대신 아직 아무 회차도 열어 보지 않은 최대 프로덕션 파일 `internal/server/resource_mutation.go`(1639줄)를 읽어, 이 저장소가 users·keys·providers 에서 이미 닫은 "입력 NUL = 400" 계약의 마지막 큰 구멍을 찾았다.
- 확인한 것: `validateLegacyMutation`(:333)의 한도 루프(:418-441)에 NUL 검사 없음, 모든 문자열 allowed 필드가 limits 에 들어 있어 한 자리 수정으로 전부 덮인다, `storableInVarchar` 가 main 의 users.go:421 에 있고 keys.go:121·providers.go:481 이 이미 호출, 500 경로 두 곳(:196 `resource_update_failed`, :169 `approval_create_failed`), 기존 fixture 헬퍼(`exerciseExpandedLegacyHTTP`·`legacyHTTPCreateExpect`·`legacyHTTPPatch`)와 부모 테스트 `TestEndToEndAPI`(integration_test.go:30).
- 추측으로 적은 것(구현자가 red 로 확정할 것): ① NUL 이 실제로 500 을 낸다는 것 — 코드 경로로는 확실하지만 DB 를 띄워 돌려 보지 않았다(이번 회차는 테스트·빌드 무실행). ② 오류 코드가 22021 인지 jsonb 의 22P05 인지. ③ URL 경로의 `%00` 이 chi 핸들러까지 도달하는지 — 안 되면 rowID 부분을 빼고 본문 필드만 고칠 것.
- 조심할 것: `workflow.go`·`users.go:updatePreferences`·`keys.go:updateKeyScope`·`Makefile` 은 미머지 브랜치가 점유 중이라 손대면 충돌한다. `make ci-test` 는 main 에 없다. 길이·trim 계약이나 기존 오류 문구를 "통일" 하지 말 것 — 추가는 NUL 거부 한 가지뿐.
- 프로필은 1일 전 것이 지금 코드와 일치해(main@2b2ef67, storableInVarchar 위치, PR #44~46 미머지) 다시 쓰지 않았다.
- [러너 03:25] scout done — 레거시 리소스 변경(`/api/v1/admin/resources/...`)의 문자열 필드·행 식별자에 든 NUL(U+0000)을 트랜잭션 시작 전�

## 구현 노트
- 무엇/왜: `validateLegacyMutation` 의 한도 루프와 문자열 rowID 검사에 `storableInVarchar` 를 추가해, 레거시 리소스 변경의 NUL(U+0000) 입력이 `tx.Begin` 전에 400 `resource_invalid` 로 떨어지게 했다(이전엔 500). 프로덕션 1파일(`resource_mutation.go`) + 기존 테스트 파일 1 + `docs/api.md` 한 문장.
- 확신 없는 곳·검증 못 한 것: ① DELETE 의 문자열 rowID NUL 과 ② `workflow.enabled`/`legacy_data_change` 정책을 켠 승인 경로(jsonb payload)는 **실행으로 확인하지 않았다** — 코드상 같은 자리(validate)에서 먼저 막히지만 서브테스트는 create·patch·PATCH rowID 세 건만 돈다. ③ `legacy-admins` 의 `menu_info` 정규화 경로가 같은 검사에 걸리는지는 코드 독해만 했고 NUL 로 찔러 보지 않았다. ④ 과제서가 예측한 create 500 코드는 틀렸다 — 실제로는 `resource_reference_unavailable`(중복 조회가 먼저 깨짐). SQLSTATE 는 text·jsonb 모두 22021(22P05 아님, pgx 프로브로 실측 후 프로브 삭제).
- 일부러 하지 않은 것: 길이·TrimSpace 계약과 기존 오류 문구 통일(회귀 위험), `applyLegacyMutation` SQL·`rejectTruncatedLegacyFields`·`validateLegacyReferences`·동의어 동기화·승인 실행 경로, `workflow.go`/`users.go`/`keys.go`/`Makefile`(미머지 브랜치 점유), `.github/workflows`·`go.mod`·`go.sum`(CI 실패는 빌링 차단으로 기록돼 있어 쫓지 않음), VERSION/CHANGELOG.
- 다음 역할 주의: 새 서브테스트 3건은 `TestEndToEndAPI` 안의 `exerciseExpandedLegacyHTTP` 끝에 있고 **`TEST_POSTGRES_DSN` 이 없으면 통째로 SKIP** 된다(`-v` 로 SKIP 여부 확인 필수). 통합 테스트는 `ai_admin`·`aiportal` 스키마를 DROP CASCADE 하므로 전용 폐기 컨테이너로만 돌릴 것. 검증에 쓴 `ai-admin-legacy-nul-pg`(포트 55621)는 제거했다. `make ci-test` 는 이 브랜치에 없다.
- [러너 03:35] brief accepted — 채택 — 결함 위치(한도 루프 :418-441·rowID switch :344-361)·헬퍼(`storableInVarchar`)·부모 테스트(`TestEndToEndAPI`)·기존 fixture 재사
- [러너 03:36] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: diff 3파일 전부, `allowed`↔`limits` 14개 resource 대조(최상위 문자열 필드 누락 없음), rowID 가드 조건이 `legacyResourceKeyColumn`·switch 분기와 일치, `mutateAdminResource:72` TrimSpace 가 NUL 을 남김, 원장 `- 실패 재현:` 의 red 출력·역검증이 증상과 일치.
- 거절 사유: `legacy-admins.menu_info` 가 구멍으로 남았다. `normalizeLegacyAdminMenuInfo`(:593)의 `json.Marshal` 이 실제 NUL 을 6글자 `\u0000` escape 으로 바꿔 `storableInVarchar`(:447)를 통과시키고, `$2::jsonb`(:1343)·`$4::jsonb`(:1389)에서 PostgreSQL 22P05 로 깨져 500 `resource_update_failed` 가 된다. 폐기 컨테이너로 22P05 확인 + 패키지 내 임시 probe 로 `validateLegacyMutation` 이 `err=nil` 을 돌려주는 것 확인(probe·컨테이너 모두 삭제, worktree clean). 따라서 `docs/api.md:371` 의 "모든 문자열 필드 … 400" 과 주석 :443-446 의 "as does the jsonb approval payload" 가 사실과 다르다.
- 수리가 먼저 볼 파일: `internal/server/resource_mutation.go`(:593 normalize 에 NUL 검사 추가) + `docs/api.md:371`(범위 축소). 새 헬퍼 이름 만들지 말 것.
- 남는 우려: `verify.json` 의 `go test ./...` 는 2초 — `TEST_POSTGRES_DSN` 없이 새 서브테스트 3건이 SKIP 된 채 게이트가 통과했다. 실제 DB 실행 기록은 원장에만 있다.
- 못 본 것: 승인 경로(workflow.enabled 켠 상태)와 DELETE 문자열 rowID NUL 의 실제 HTTP 실행. menu_info 결함은 승인 경로에서 영구 실행 불가 pending 승인을 남길 가능성이 있으나 미실측 — 수리 시 확인 권장. 보안·법무 차단 사유 없음(공격 경로 없음, 개인정보·의존성 변경 없음).
- [러너 03:41] review rejected — 리뷰 거절: docs/api.md:371 추가된 계약 문장 "모든 문자열 필드와 문자열 행 식별자는 NUL(U+0000)을 담을 수 없고 포함되면 400 resource_invalid를 반환합니다" 가

## 수리 노트
- 비평 4개 지적 모두 맞았다(틀린 것 없음). 유닛 red 로 `validateLegacyMutation("legacy-admins","create","",{menu_info:…NUL…})` 가 `err=nil` + `\u0000` escape 잔존을, HTTP red 로 POST /resources/legacy-admins 가 `500 resource_update_failed` 임을 재현했고, 폐기 컨테이너로 `'[{"menuNm":"a\u0000b"}]'::jsonb` → 22P05 를 다시 확인했다.
- 고친 방법: `normalizeLegacyAdminMenuInfo` 가 `menuId`(TrimSpace 전 원문)·`menuNm` 을 `storableInVarchar` 로 검사하고 sentinel `errLegacyMenuInfoNUL` 반환, 호출부 2곳은 새 `legacyMenuInfoError` 로 NUL 메시지만 보존(JSON 형식 오류 문구·길이 계약 불변). 주석 :443-446 과 docs 문장은 escape 경로를 명시하게 수정. 새 헬퍼 이름은 금지 목록(`jsonObjectHasNUL`·`storablePreferenceObject`)과 겹치지 않는다.
- 비평가의 "SKIP 된 채 게이트 통과" 우려는 유닛 테스트(`TestValidateLegacyAdminMenuInfoRejectsNUL`, DB 불필요)를 추가해 DSN 없이도 핵심 계약이 돌게 했다. 그 위에 HTTP 서브테스트 1건(배열·escape 문자열 create 거부 + 행 미변경 PATCH 거부 + 정상 값 왕복 + 리터럴 `\\u0000` 저장)을 더했다.
- 검증: 전용 postgres:16-alpine(포트 55723, 제거 완료)로 `go test -race -count=1 ./...` 전체 통과(SKIP 은 OIDC E2E 1건), `make lint`·`go build`·`git diff --check` 통과. 프로덕션 변경만 임시로 빼면 두 테스트가 다시 red(역검증).
- 여전히 확신 없는 곳: 승인 경로(workflow.enabled 켠 jsonb payload)와 DELETE 문자열 rowID 의 실제 HTTP 실행은 이번에도 안 했다 — validate 앞단에서 같은 자리로 막히지만 미실측. 22P05 분기는 일부러 추가하지 않았다(이 경로로 도달 불가, 범위 밖). workflow.go 의 제목/코멘트/payload NUL 은 다음 회차 몫으로 남아 있다.
- [러너 03:54] repair done — # 수리 요약  - 비평 거절은 전부 맞았다. `legacy-admins.menu_info` 가 구멍이었다: `normalizeLegacyAdminMenuInfo` 의 `json.Marshal` 이 실제 NUL 을 6글자 `\u0000` escape 으로

## 비평 노트
- 확인한 것(실행): main worktree에 HEAD 테스트만 얹어 역검증 — 유닛 5건·HTTP 4건 전부 red이고 증상이 주장과 일치(create 500 `resource_reference_unavailable`, notices PATCH 500 `resource_update_failed`, menu_info 500 `resource_update_failed`, `app-defaults/synthetic%00default` PATCH 500 '행 버전을 확인하지 못했습니다'). HEAD 에서 4건 모두 SKIP 아닌 RUN·PASS, `go test -race ./...` 전체 통과(internal/server 146.7s), gofmt·vet·build·diff --check 통과. 전용 컨테이너 `ai-admin-review-pg`(55817)·역검증 worktree 모두 제거, worktree clean.
- 확인한 것(독해): 14개 resource 의 `allowed`↔`limits` 문자열 필드 완전 일치(빠진 셋은 int 경로)이므로 docs/api.md:371 문장이 사실이고, `::jsonb` 바인드는 menu_info(:1368·:1414)뿐이며 중첩 키가 menuId·menuNm·authYn 로 제한(:622-626)되어 중첩 문자열 커버리지가 완전하다.
- 구현자의 '미실측' 두 자리는 배치로 해소: rowID 가드(:346)가 delete 조기 반환(:371)보다 앞, validate(:96)가 승인 생성(:167)보다 앞이며 `executeApproval`(workflow.go:547)이 같은 검증을 재실행 → jsonb 승인 payload 에도 NUL 불가. 단 workflow.enabled=true 의 실제 HTTP 왕복은 이번에도 미측정.
- 승인이어도 남는 우려: 릴리즈 노트에 '레거시 리소스 변경'만이 아니라 'legacy-admins menu_info 중첩 문자열'까지 적을 것. 다음 회차 몫은 workflow.go 제목/코멘트/payload·users.go updatePreferences·keys.go updateKeyScope 이며 미머지 브랜치가 점유 중이다.
- 차단 없음: security — 입력을 좁히기만 하고 오류가 요청 값을 반사하지 않으며 권한·세션·마이그레이션·비밀값 무변경. legal — 개인정보·의존성·라이선스 변경 없음.
- [러너 04:02] review approved — 리뷰 승인 (risk=low)
- [러너 04:02] pr created — https://github.com/hkjang/ai-admin/pull/48
- [러너 04:03] ci failed — 성공이 아닌 검사: test=failure, lint=failure · 실패한 검사: ? 잡: lint 

## 수리 노트
- 지적이 틀렸다 — 고칠 코드가 없어 커밋하지 않았다. `gh run view 38951938252` ANNOTATIONS 가 lint·test 두 잡 모두 "The job was not started because recent account payments have failed or your spending limit needs to be increased" 로, step 이 하나도 실행되지 않았다(3초 실패·`--log-failed` 빈 출력의 정체). 코드와 무관한 계정 빌링 차단이다.
- 맞았던 것: 이전 비평 회차의 승인 판단과 구현 내용 자체는 문제없었다. HEAD(9e8284d)에서 CI 와 동일한 검사를 전부 재현해 통과 — gofmt·go vet·verify-version(1.2.37), `go test -race -count=1 ./...` 전체 ok(internal/server 146.667s), web 18파일/81테스트 + 빌드, `docker build` exit 0.
- 비평가가 남긴 유일한 미해소 우려("DSN 없이 SKIP 된 채 게이트 통과")는 이번에 `-v -run` 으로 직접 확인해 닫았다: HTTP 4건(create·patch·menu_info 중첩 jsonb·rowID)과 유닛 5건이 SKIP 이 아니라 RUN·PASS.
- 검증 방법 메모: WSL 에서 `-p` 포트 공개가 500 으로 깨져 postgres 를 미공개로 띄우고 golang:1.26 컨테이너를 `--link` 해 테스트를 컨테이너 안에서 돌렸다(호스트는 172.17.x 로 직접 접근 불가). 쓴 컨테이너·이미지는 제거, worktree clean.
- 여전히 확신 없는 곳: 빌링 차단이 풀리기 전에는 실제 CI 가 초록인지 확인할 방법이 없다(로컬 재현이 유일한 근거). 승인 경로 `workflow.enabled=true` 의 실제 HTTP 왕복은 이번에도 미측정이며, workflow.go 제목/코멘트/payload·users.go updatePreferences·keys.go updateKeyScope 는 여전히 다음 회차 몫이다.
