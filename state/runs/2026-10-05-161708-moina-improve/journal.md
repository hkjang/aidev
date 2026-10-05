# 회차 노트 2026-10-05-161708-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:17] base pinned — main@bd74c83
- [러너 16:17] autonomy release — 

## 정찰 노트
- 지난 회차가 남긴 1순위(storage_error 출구 이관)를 그대로 골랐다. 다른 pending 후보는 전부 선행 조건이 막혀 있다 — Scan 후보·전용 스키마 헬퍼는 SELECT 실패 재현 설계가 선행, Axe/e2e 후보는 `make image` + Playwright 실제 실패 재현이 선행(45분에 위험), API 키 후보는 제품 결정이 선행, CI 요약 후보는 workflow 파일이라 보호 경로에 가깝다.
- 파일 선택은 `grep -c 'storage_error' *.go`로 정했다: posts.go 41이 가장 많고 message 중복이 가장 심해(같은 문구 5~6곳) 로그 추가의 가치가 가장 크며, **프로덕션 파일 1개**로 끝나 재작업 위험 구간(파일 10개 이상)에서 가장 멀다.
- 과제서에 적은 것은 전부 실제로 열어 확인했다: 41개 출구 줄번호, `grep -n '^func '`로 뽑은 함수 경계와 출구→handler 매핑, `writePostError`의 `r` 없는 서명과 호출 3곳, `cursor_error` 500 둘(:547 :656), `posts_update_postgres_integration_test.go:153`의 BEFORE UPDATE 트리거 저장 실패 케이스. **추측(미실행)**: 그 트리거의 SQLSTATE가 `P0001`이라는 것(지난 회차 admin 테스트에서 `P0001`이었던 것에서 유추 — 다르면 단언 값만 고치면 된다), 그리고 41곳이 한 세션에 들어간다는 작업량 판단.
- 구현자가 조심할 것: 줄번호는 이관하면서 밀리므로 message 문자열과 감싼 함수로 식별할 것. `handler` 오기는 컴파일로 잡히지 않는 유일한 실질 결함이다. message 한 글자라도 바뀌면 프런트 `readableError`를 타고 사용자에게 보인다.
- 프로필은 1일 전 것이고 지금 코드와 맞아 다시 쓰지 않았다. 유일한 어긋남은 "posts.go ~35"라는 추정치이며 실제는 41이다(ideas.json에 기록).
- [러너 16:21] scout done — `posts.go`의 `storage_error` 500 출구 41곳을 `writeStorageError`로 이관한다 (가치 3 / 위험 1 / 작업량 M)

## 구현 노트
- `posts.go`의 `storage_error` 500 출구 41곳을 `writeStorageError`로 이관해 `handler`·`cause_type`·`pg_code`가 `request_id`와 함께 남게 했다(76010d8, 프로덕션 파일 1개 + 테스트 1개). 응답 불변은 한글 리터럴 multiset 비교로 증명했다 — HEAD에만 있는 리터럴 0개.
- `handler` 인자는 손으로 적지 않고 `^func ` 경계에서 감싼 함수 이름을 유도해 치환한 뒤, 같은 유도로 43개 호출 전부 mismatch 0을 재검증했다(정찰의 수기 매핑과 일치). 과제서가 지목한 "유일한 실질 위험"은 이것으로 닫혔다고 본다.
- **과제서 범위를 넘은 곳(비평가가 먼저 볼 곳)**: `deletePost`·`deleteRemoin`의 `Exec실패 || tx.Commit() != nil` 한 조건을 두 분기로 나눴다. 합친 채로 이관하면 Commit 실패 시 `err`가 nil이라 `cause_type:"<nil>"`을 찍어 이 회차의 목적을 배반한다. 단락 평가 순서·응답·message는 그대로이고 같은 message가 한 번 더 쓰인 것이 늘어난 리터럴 2개의 정체다. **이 두 경로의 Commit 실패는 테스트로 재현하지 않았다** — 등가성은 코드 읽기로만 확인했다.
- **확신 없는 것**: red→green을 실제 핸들러로 관측한 것은 `updatePost`의 UPDATE 실패 한 경로뿐이다. 나머지 40곳의 `handler` 문자열은 기계적 유도로만 보증되고 실행으로 확인하지 않았다(컴파일로도 안 잡힌다).
- 일부러 안 한 것: `cursor_error` 500 둘·404/409 분기·`storage_error.go` 헬퍼·OpenAPI는 손대지 않았다. 응답이 불변이라 프런트 vitest와 e2e는 돌리지 않았다(지난 두 회차와 같은 판단).
- 다음 역할이 조심할 것: `posts_update_postgres_integration_test.go`는 DB가 있어야 돈다(`MOINA_TEST_POSTGRES_DSN`, 없으면 `t.Skip`). 검증은 `postgres:16-alpine` 컨테이너로 했고 포트 55432는 다른 프로젝트가 점유 중이라 **55461**을 썼다. 전체 테스트는 2회만 돌렸다(로그인 5분/IP 5회 제한).
- [러너 16:27] brief accepted — 채택 — 근거가 현재 코드와 전부 일치했습니다(41곳의 줄번호, `grep -n '^func '` 함수 경계와 출구→handler 매핑 전부, `writePo
- [러너 16:27] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: red→green을 직접 재현(postgres:16-alpine @55487 — HEAD 4/4 PASS, posts.go만 base로 되돌리면 error_code/handler/cause_type/pg_code 4건 FAIL, ledger 출력과 동일). 전체 `go test -race -count=1 -v ./...` 13패키지 ok·`--- SKIP` 0줄·TestPostgreSQL* PASS 44건(v0.1.41 기준과 일치), build·vet·gofmt 통과. 응답 불변은 한글 리터럴 multiset을 제가 다시 세어 BASE-only 0 / HEAD-only 2(분기 분할이 만든 중복뿐)로 확인. handler 인자 43곳 전부 감싸는 func과 일치, 패키지 밖 호출자 없음.
- 못 본 것: deletePost·deleteRemoin의 **Commit 실패 경로**는 여전히 실행으로 확인되지 않습니다(단락 순서·defer Rollback·404 순서 보존은 읽기로만). frontend vitest·e2e는 응답 불변이 증명됐으므로 돌리지 않았고, staticcheck는 이 환경에 없어 생략했습니다.
- 승인이어도 남는 우려: posts.go:612/:631은 handler와 message가 모두 같아 스냅샷 생성/재사용 분기를 로그로 아직 못 가릅니다. posts.go:585는 json.Marshal 실패를 "저장 실패"로 남겨 pg 장애로 오해될 수 있습니다. 통합 테스트의 `slog.SetDefault`는 프로세스 전역이라 httpapi에 t.Parallel()이 하나 들어오는 순간 이 테스트와 admin_report_resolve…:125가 조용히 오염됩니다.
- 보안·법무: 차단 사유 없음. 로그에 새로 찍히는 것은 상수 handler 이름·Go 타입명·SQLSTATE 5자리뿐이고 pg message 부재를 테스트가 단언합니다(제 실행에서 통과). 새 의존성·새 개인정보·마이그레이션 없음, revert 한 번으로 돌아옵니다.
- 다음 회차: 남은 약 117개 출구(admin.go ~30·social.go ~25·settings.go ~8 …)에 같은 기계적 검사(감싸는 func 이름 유도 + 한글 리터럴 multiset diff)를 그대로 재사용하면 됩니다.
- [러너 16:34] review approved — 리뷰 승인 (risk=low)
- [러너 16:34] pr created — https://github.com/hkjang/moina/pull/41
- [러너 16:42] ci passed — 검사 2개 모두 success
- [러너 16:42] merge done — 76010d8

## 릴리즈 노트
- `v0.1.43`으로 릴리즈했습니다(커밋 `c9bfca2`, 주석 태그 `v0.1.43` = "moina v0.1.43"). 최근 세 릴리즈가 전부 패치만 올렸고 이번 변경도 관측성 한 파일이라 패치로 올렸습니다. 버전이 적힌 파일은 `0.1.42`→`0.1.43` 기계 치환으로 **24개**(`VERSION`·`Dockerfile`·`api/openapi.yaml`·docs 13개·`frontend`/`e2e` 6개·`frontend/src/config.ts`), v0.1.40~v0.1.42와 동일한 파일 집합이고 diff 모양도 같습니다(`24 files changed, 99 insertions(+), 99 deletions(-)`). 치환 전에 `0.1.42` 전체 출현을 눈으로 확인해 의존성 버전 등 무관한 자리가 없음을 확인했습니다. 한국어 조사는 이전 릴리즈들이 한 번도 손대지 않았으므로(예: v0.1.32→v0.1.33에서도 "`v0.1.33`는" 그대로) 숫자만 바꿨습니다.
- `README.md`의 릴리즈 단락은 양식대로 다시 썼습니다 — v0.1.43 설명을 새로 쓰고, 기존 유지 기능 사슬 앞에 "`v0.1.42`의 … `writeStorageError` 도입"을 끼워 넣었습니다. CHANGELOG나 `docs/RELEASE*`는 이 저장소에 없습니다(릴리즈 노트 본문은 태그 워크플로가 생성).
- **ledger·비평이 남긴 "남은 출구 수"가 둘 다 틀렸습니다.** ledger는 약 88곳(social 28·admin 20…), 비평 노트는 약 117곳(admin ~30·social ~25…)이라 했는데, 직접 세어 보니 `httpapi`에 남은 `writeError(..., "storage_error", ...)`는 **95곳**입니다 — social 29·admin 25·workflow 9·settings 9·auth 7·preferences 6·analytics 4·smtp 3·outbox 2·oidc 1. (grep은 96건이지만 1건은 `storage_error.go` 안 헬퍼 자신의 호출이라 이관 대상이 아닙니다. `posts.go`의 `writeStorageError` 호출은 43개로 구현 노트와 일치.) README와 커밋 메시지에는 확인한 95를 적었습니다. 다음 회차는 88/117 둘 다 쓰지 말고 95(파일별 위 숫자)를 기준으로 삼으십시오.
- 검증은 이전 릴리즈가 밟던 것을 그대로 돌려 전부 통과했습니다: `make fmt`, `make check`(런타임 계약 4개·앱 화면 31개·OpenAPI route **120개**·브랜드 대비·Pages QA), `MOINA_REQUIRE_SCREENSHOTS=1` 엄격 Pages QA, `go build ./...`, `go vet ./...`, `staticcheck@2025.1.1`(무지적), throwaway `postgres:16-alpine`(포트 **55447** — 55432·55433 등은 다른 프로젝트가 점유 중) + `MOINA_TEST_POSTGRES_DSN`으로 `go test -race -count=1 -v ./...` exit 0(`--- FAIL` 0줄·`--- SKIP` **0줄** = DSN 실제 적용·최상위 `TestPostgreSQL*` PASS **44건**·`TestStorageError*` PASS 3건), 프런트 vitest **250건/36파일** 통과, ESLint 0 error·**39 warning**(40 예산 내), `VITE_MOINA_VERSION=v0.1.43` 프로덕션 빌드 성공.
- **원격에는 아무것도 보내지 않았습니다.** 커밋·태그는 detached HEAD에 로컬로만 만들었고 브랜치는 만들거나 옮기지 않았습니다. `assets`는 빈 배열이고 `github_release`는 `false`입니다 — `.github/workflows/release.yml`이 `v*.*.*` 태그 push에 반응해 이미지 빌드·runtime/browser smoke·`package-image.sh`·`verify-image-package.sh`를 하고 `gh release create`로 Release와 `moina-v0.1.43.tar.gz` 하나를 직접 만듭니다. 그래서 이 기계에서 자산을 만들지 않았고 `notes_file`도 비웠습니다(본문의 SHA256은 그 워크플로가 빌드 후에 계산해 써 넣습니다).
- 태그 push 후 그 워크플로가 요구하는 선행 조건 둘은 이미 충족합니다: `VERSION` == 태그 이름(`v0.1.43`), 태그가 주석 태그 객체. 남은 하나인 "release commit의 성공한 CI"는 push 이후에만 판정됩니다.
