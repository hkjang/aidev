- 과제: `catalog_entry` 의 jq 재호출을 메모이즈해 `check-versions` 를 빠르게 한다 — Go 테스트 스위트의 87% 를 차지하는 한 테스트를 줄인다 (가치 3 / 위험 2 / 작업량 S)

- 왜: 이번 정찰이 실제로 재서 보니 `go test ./cmd/... ./internal/...` 전체 20초 중 **17.44초**가 `internal/buildinfo` 한 패키지이고, 그 전부가 `TestTheImageVersionCheckCatchesTheReleaseItWasWrittenFor`(workflows_test.go:187) 하나다 — 이 테스트는 `scripts/release-catalog-images.sh check-versions` 를 두 과거 커밋(9a35494, b742060)에 대해 각각 실행하므로 한 번이 약 8.7초다. 나머지 모든 패키지는 합쳐 2초 안쪽(`internal/api` 1.563s, `internal/tracking` 0.012s)이다. 이 회차가 'agent produced no result (TIMEOUT)' 로 끝났고 이 저장소의 성공 회차들은 검증 루프에서 전체 스위트를 3~5회 돌리므로, 가장 느린 한 자리를 줄이는 것이 자율 회차가 직접 되돌릴 수 있는 유일한 비용이다. CI 의 독립 `check-versions` 스텝과 릴리즈 워크플로의 `plan` 도 같은 헬퍼를 쓰므로 같이 빨라진다.

- 수용 기준:
  1) `go test ./internal/buildinfo -count=1` 의 보고 시간이 수정 전(17.44s, 이번 정찰 측정값)보다 **눈에 띄게** 줄어든다. 구현자가 수정 전/후를 같은 기계에서 `-count=1` 로 각각 재서 두 숫자를 요약에 적는다. (목표를 숫자로 못 박지 말 것 — 아래 '추정' 참고. 줄지 않으면 가설이 틀린 것이고, 그때는 차선 후보로 옮긴다.)
  2) `bash scripts/release-catalog-images.sh check-versions` 의 **출력이 수정 전과 한 바이트도 다르지 않다**(현재 출력: `every image whose inputs changed since v0.260.0 already has a new version`). `bash scripts/release-catalog-images.sh validate` 도 그대로 통과한다.
  3) `TestTheImageVersionCheckCatchesTheReleaseItWasWrittenFor` 가 그대로 통과한다 — 즉 9a35494 는 **여전히 거절되고** 그 거절문에 `BASE_VERSION` 이 들어가며 b742060 은 **여전히 통과한다**. 이 테스트가 바로 "빨라졌지만 느슨해졌다" 를 잡는 자리이므로 테스트의 두 커밋·단정·메시지는 손대지 않는다.
  4) 알 수 없는 id 로 `catalog_entry` 를 부르면 수정 전과 같이 **빈 출력 + 비영(非零) 종료**다. `scripts/release-catalog-images.sh:41` 의 미지 의존성 검사와 `:172` 의 `catalog_entry base | jq -e …` 가 이 계약에 기대고 있다.

- 건드릴 파일 (프로덕션 1개):
  - `scripts/release-catalog-images.sh:22` `catalog_entry()` — 같은 id 에 대해 jq 를 매번 새로 띄우지 않도록 `declare -Ag catalog_entry_cache=()` 를 하나 두고 캐시 적중이면 `printf '%s\n'` 로 돌려준다. **반드시 `jq -ce` 의 현재 성질을 유지할 것**: 한 줄 compact 출력(호출자들이 `<<<"$entry"` 로 다시 jq 에 먹인다)과 미지 id 의 비영 종료. 캐시 키는 id, 캐시 값은 jq 가 낸 한 줄. 미지 id 는 캐시하지 말고 매번 원래 경로로 보내는 편이 종료 코드 계약을 지키기 쉽다(또는 '없음' 을 센티넬로 저장하고 종료 코드를 재현할 것 — 어느 쪽이든 기준 4 를 테스트로 고정할 것).
  - 같은 파일의 주석 — 왜 캐시가 안전한지(한 프로세스 안에서 `$catalog` 파일은 읽기만 하고 바뀌지 않는다)를 이 저장소 관례대로 산문으로 적을 것. `plan_entry`(`:27`)는 `$plan` 을 읽고 `plan` 은 스텝 중간에 **생성되므로** 같은 캐시를 적용하지 말 것.
  - (선택) `internal/buildinfo/workflows_test.go` — 두 커밋 실행을 `t.Run`+`t.Parallel()` 서브테스트로 나누면 벽시계가 추가로 반 줄어든다. 단정과 커밋 해시는 그대로. 기준 1 이 캐시만으로 충족되면 **하지 말 것**(파일 수를 늘리지 않는다).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `go test ./internal/buildinfo -count=1` (수정 전·후 각각, 숫자를 적을 것)
  - `go test ./internal/buildinfo -count=1 -v` — 느린 테스트의 개별 시간 확인
  - `bash scripts/release-catalog-images.sh check-versions` / `bash scripts/release-catalog-images.sh validate`
  - `go test ./cmd/... ./internal/...` 전체(이번 정찰에서 통과, 총 ~20s)
  - `go test -race -p 1 ./cmd/... ./internal/...` (DSN 없으면 live 는 skip; 과거 회차처럼 `docker run --rm -d -p 55447:5432 postgres:16-alpine` 로 일회용 DB 를 띄워 live 까지 돌리는 것이 바람직하나, **이 과제는 DB 를 지나지 않으므로 필수는 아니다**)
  - `bash -n scripts/release-catalog-images.sh` (문법), 가능하면 `shellcheck scripts/release-catalog-images.sh`(이 환경에 있는지 미확인)

- 위험과 피할 것:
  - **워크플로를 느슨하게 만들어 통과시키지 말 것.** `check_versions`(`:621`)의 판정 논리(`image_inputs_changed` 와 `[ "$version" = "$previous_version" ]`), `image_inputs_changed`(`:272`)의 `git diff`·catalog 메타데이터 비교, `base_module_inputs_changed` 는 **한 줄도 바꾸지 말 것**. 이번 과제는 "같은 답을 더 적은 프로세스로" 뿐이다.
  - `.github/workflows/*` 는 건드리지 말 것(보호 경로). 이번 과제는 워크플로 파일을 수정하지 않고도 CI 의 `check-versions` 스텝을 빠르게 한다.
  - `runtime-images.json` 과 `*_VERSION` 파일은 건드리지 말 것 — 바꾸면 `check-versions` 자체가 다른 답을 내고 기준 2 가 무의미해진다. 이 과제는 `scripts/` 만 바꾸므로 `runtime-images.json` 의 어떤 이미지 sourcePaths 에도 들어가지 않는다(이전 회차들이 매니페스트 14개 항목을 직접 읽어 확인했고, 이번 정찰도 이미지 수 14를 재확인했다) → **BASE_VERSION 상향 불필요**. 다만 수정 후 `check-versions` 를 실제로 돌려 그 말이 여전히 맞는지 확인할 것.
  - bash 연관배열은 `set -euo pipefail` 아래에서 미설정 키 참조가 터진다 — 반드시 `${cache[$id]:-}` 형태로 읽을 것.
  - `catalog_entry` 는 서브셸(`$( )`)과 파이프 안에서 불리는 자리가 있다(`:46` `< <(catalog_entry "$id" | jq -r …)`, `:280`). **서브셸에서 채운 캐시는 부모에 남지 않는다** — 그래서 캐시가 실제로 적중하는 횟수가 기대보다 적을 수 있다. 이것이 기준 1 이 숫자 목표가 아니라 '측정해서 적는다' 인 이유다. 적중이 없으면 접근을 바꿔라: 시작 시 `jq -r '.images[] | @base64'` **한 번**으로 전부 읽어 `catalog_entry_cache` 를 채워 두면(=`resolve_image_order` 직전) 서브셸도 상속된 값을 읽는다. 이쪽이 더 확실하므로 먼저 이 형태를 시도할 것을 권한다.
  - **추정(미확인)**: 8.7초가 jq/git **프로세스 생성** 비용이라는 것은 측정이 아니라 추론이다(이미지 14개 × 호출당 10여 개 프로세스 ≈ 300 프로세스/회, WSL2 의 느린 spawn). 실제 분해는 못 쟀다 — 권한 제약으로 `time`·반복 spawn 벤치를 못 돌렸다. 구현자는 먼저 `bash -x` 또는 간단한 spawn 벤치로 **jq 가 지배적인지 확인한 뒤** 고칠 것. git 쪽(`git show`·`git diff`·`git cat-file`, 이미지당 5~6회)이 지배적이면 캐시는 효과가 작고, 그때는 차선 후보로 옮기는 것이 옳다.
  - 과거 교훈: 소스 문자열 검사는 증거가 아니다. "빨라졌다" 는 `-count=1` 실측 두 숫자로, "안 느슨해졌다" 는 기준 2·3 의 실제 실행으로 증명할 것.

- 차선 후보: **콘솔 설정 화면이 추적 설정의 상한을 미리 알려 준다** (가치 2 / 위험 1 / 작업량 S) — `web/src/.../AdminSettings.tsx:167` 의 measurement id 입력에는 `maxLength` 도 안내도 없고, TrackingForm 은 스니펫 8KiB 바이트 카운터만 보여 준다. 지난 다섯 회차가 더한 상한(허용 목록 64개/4096룬, 스니펫 출처 32개/1024룬, 제공자 출처 300룬, 제공자 id 200룬, 제공자 URL 1024룬)은 관리자가 저장을 눌러 거절당할 때만 알 수 있다. 상한 계열 작업이 일단락됐으니 지금이 적기다. 약점은 그대로 — Go 상수를 프런트로 전달하는 계약이 없어 숫자가 두 곳에 하드코딩된다(그 중복을 주석으로 명시하고 `internal/tracking/tracking.go` 의 상수 이름을 가리킬 것). 프런트 파일 1~2개로 끝난다.
