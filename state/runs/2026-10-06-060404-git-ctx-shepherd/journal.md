# PR 처리기 노트 2026-10-06-060404-git-ctx-shepherd — git-ctx PR #47
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-06-045751-git-ctx-improve)
# 회차 노트 2026-10-06-045751-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 04:57] base pinned — main@22eb1cf
- [러너 04:57] autonomy release — 

## 정찰 노트
- 골랐다: CODEOWNERS 의 `docs/` 같은 비앵커 디렉터리 패턴이 루트에만 적용되는 비대칭(`codeowners.go:134-137` 이 `/**` 를 붙인 뒤 중첩을 판정한다). 보류 1순위였던 fencesContent 가드(4회 연속 차선, 프로덕션 변경 0)와 프로브 대기 원인(남은 벽시계가 ~56s 로 줄어 수익 하락)을 제친 이유는, 이것이 같은 세션에서 **실측으로 확정한 결함**이고 프로덕션 1파일·실질 2줄이며 기존 테스트 14줄이 무변동이기 때문이다. 내리 네 회차를 `internal/mcp` 포매터 계열에서 골랐으므로 영역을 바꾼 것도 의도적이다.
- 추측으로 적은 것: ① 실제 색인된 CODEOWNERS 중 `docs/` 형태가 몇 건인지 — 외부 GitLab·Bitbucket 데이터가 없어 **영향 규모는 미확인**(결함 자체는 18케이스 실측으로 확정). ② `find-code-owner` MCP 왕복 end-to-end 는 호출자 열거로만 확인했고 실행하지 않았다 — 수용 기준 3)이 이 구간을 메우게 해 뒀다.
- 구현자가 조심할 것: 매칭이 넓어지면 `matchCodeowners` 의 "마지막 매칭이 이긴다" 로 결정되는 소유자가 바뀔 수 있다 — `TestCodeownersParsingKeepsTheDecidingRuleLast`·`dependencies_test.go`·`ownership_test.go` 가 무수정 통과하는지 꼭 보고, 움직이면 숨기지 말 것. `globMatch`·`cutPrefixSegments` 는 건드리지 말 것(이 비대칭과 무관하고 기존 네 줄이 움직인다).
- 차선 후보의 선행 조건을 이번에 해소했다: `c.txt`·`server.log` 를 실제로 열어 비밀값 없음을 확인했다(libcurl 쿠키 파일 헤더 3줄 / 키 값이 아닌 에러 메시지 한 줄). 9회차 연속 "내용 확인 선행" 으로 미뤄진 항목이니 1순위가 성립하지 않으면 바로 집을 수 있다.
- 임시 프로브 테스트는 실행 후 삭제했고 작업 트리는 깨끗하다. 프로덕션 코드는 한 글자도 바꾸지 않았다.
- [러너 05:06] scout done — CODEOWNERS 패턴에서 **뒤에 슬래시를 붙인 디렉터리 이름이 루트에만 적용되는** 비대칭 고치기 — `internal/se

## 구현 노트
- 무엇을: `codeownersMatch` 가 `/**` 를 붙인 뒤에 중첩 여부를 판정해 `docs/` 같은 비앵커 디렉터리 이름을 루트에 묶던 것을, 접미사 **앞**에서 `nested` 를 정하도록 순서만 바꿨다(프로덕션 1파일 `internal/search/codeowners.go`, 실질 2줄 + 주석 2곳). 왜: `docs` 는 `src/docs/readme.md` 를 잡는데 `docs/` 는 놓쳐 선언이 커밋 추정으로 떨어졌다.
- 확신 없는 곳: ① **영향 규모 미확인** — 실제 색인된 CODEOWNERS 중 `docs/` 형태가 몇 건인지는 외부 GitLab·Bitbucket 데이터가 없어 이번에도 못 봤다(결함 자체는 18케이스 실측으로 확정). ② `matchCodeowners` 의 "마지막 매칭이 이긴다" 로 결정되는 소유자가 넓어진 매칭 때문에 바뀔 수 있는데, 이 저장소 픽스처에서는 움직이지 않았다(`TestCodeownersParsingKeepsTheDecidingRuleLast`·`dependencies_test.go`·`ownership_test.go` 무수정 통과) — **실제 저장소의 CODEOWNERS 에서는 비앵커 디렉터리 규칙이 이제 더 많이 매칭되므로 그 규칙이 뒤에 있으면 소유자가 바뀐다. 이것이 의도된 수정 방향이지만 사용자 눈에 보이는 변화다.**
- 일부러 안 한 것: `globMatch`·`cutPrefixSegments`·`matchCodeowners`·`declaredOwners` 무수정(이 비대칭과 무관하고 기존 표 네 줄이 움직인다). 기존 표 14줄도 한 줄 안 건드렸다(`git diff -U0` 삭제 줄 0건). 차선 후보 `c.txt`·`server.log` 는 1순위가 성립해 손대지 않았다. 릴리즈·버전 올리기 없음.
- 다음 역할이 조심할 것: 새 `TestDeclaredOwnersApplyADirectoryRuleBelowTheRoot` 는 공유 in-memory SQLite 이름(`codeowners-nested-directory`)을 쓰므로 `t.Parallel()` 금지 — 기존 `TestDeclaredOwnersAnswerWithoutTheSourceServer` 와 같은 모양이고 외부 DB 는 필요 없다. `-tags sqlite_fts5` 없이는 `./internal/search` 가 돌지 않는다. 새로 알게 된 기존 계약 하나: `docs/` 는 `docs` 라는 **파일** 에도 맞는다(앵커 형태가 전부터 그랬고 이번 표가 명시적으로 고정했다) — ideas.json 에 후보로 적었다.
- [러너 05:11] brief accepted — 채택 — 지정한 함수·줄 번호(`:130-137`)·근거·제시한 수정 모양이 현재 코드와 정확히 맞았고, 과제서가 실측으로 예고�
- [러너 05:12] verify passed — 검증 4개 통과 (auto)

## 비평 노트
- 확인한 것: 되돌림 프로브를 직접 돌렸다 — `codeowners.go` 만 main 으로 되돌리면 원장이 적은 네 줄이 같은 값으로 실패하고(`:55` 3건 + `:164` "no source connector is configured") 되돌리면 통과한다. 대조군 12줄은 수정 전후 모두 통과하므로 표가 과대적합이 아니다. 추가로 `gofmt -l`·`go vet ./...`·`go build -tags sqlite_fts5 ./...`·`verify-version-sync.sh`(v0.77.24)·`./internal/{search,mcp,app,indexer}` 테스트를 내 세션에서 다시 돌려 전부 green 확인(app 52.1s).
- 논리 검토: 변경이 영향 주는 패턴은 '앵커 없음 + 뒤 슬래시 + 내부 슬래시 없음' 뿐이고, 기존 앵커 분기의 두 `globMatch` 호출이 새 루프의 index 0 과 동일하므로 **매칭을 잃는 회귀가 구조적으로 불가능**하다(상위집합). `**/`·`*/`·`docs/api/`·`/docs/` 를 손으로 추적해 동작 불변 확인.
- 못 본 것: 실제 색인된 CODEOWNERS 중 `docs/` 형태의 분포(외부 GitLab·Bitbucket 데이터 없음) — 결함은 확정, **규모만 미지**. govulncheck·외부 DB·Vault 미실행(이 diff 와 무관).
- 승인이어도 남는 우려 → **릴리즈 세션이 볼 것**: 비앵커 디렉터리 규칙이 더 많이 매칭되므로 "마지막 매칭이 이긴다" 로 결정되는 유효 소유자가 **재색인 없이 달라질 수 있다**. v0.77.6 노트처럼 상단 경고로 적을 것(권고 통보 대상 목록도 `dependencies.go:445 declaredOwnersFor` 를 통해 영향).
- 다음 회차가 알 것: 새 표가 `{"docs/","src/docs",true}` 로 **디렉터리 전용 패턴이 파일에도 맞는 것**을 명시적으로 고정했다(앵커 형태에서 전부터 그랬던 기존 성질의 확장). ideas.json 후보를 집을 때 이번에 추가한 표 줄을 함께 손봐야 한다.
- [러너 05:16] review approved — 리뷰 승인 (risk=low)
- [러너 05:16] pr created — https://github.com/hkjang/git-ctx/pull/47
- [러너 05:32] ci cancelled — 성공이 아닌 검사: Build, vet and unit tests=cancelled · 실패한 검사: ? 

## 심사 노트
- 확인한 것: 되돌림 프로브를 내 세션에서 직접 돌렸다 — `codeowners.go` 만 origin/main 으로 되돌리면 새 단언 4건이 실패하고(`codeowners_test.go:55` 3건 + `:164` "no source connector is configured") 복구하면 통과, 대조군 14줄은 전후 모두 통과. 취소된 CI 를 메우려 `gofmt`·`go vet ./...`·`go build -tags sqlite_fts5 ./...`·`verify-version-sync.sh`(v0.77.24)·`go test -tags sqlite_fts5 -count=1 ./...` 전체를 다시 돌려 green 확인. 프로브 후 작업 트리 청결 확인.
- 확인한 것: 새 루프 index 0 이 옛 앵커 분기의 두 `globMatch` 호출과 동일 입력이라 상위집합(회귀 불가), `codeownersMatch` 프로덕션 호출자는 `codeowners.go:108` 한 곳. 테스트는 대역 없이 실제 store·Service·`FindOwners`→`FormatOwners` 왕복이라 매처·렌더러 두 경로를 함께 고정한다. ACL 은 `service.go:3237`·`dependencies.go:470` 의 `repositoryACL` 로 상류에서 걸리고 이 diff 가 건드리지 않아 권한 확대 없음 — security·legal 차단 소견 0건.
- 못 본 것: 실제 색인된 CODEOWNERS 중 `docs/` 형태의 분포(외부 GitLab·Bitbucket 데이터 없음 — 영향 규모만 미지, 결함은 확정). 외부 DB·Vault·Docker·실브라우저·govulncheck 미실행(의존성·릴리즈 경로 무변동).
- 권고 근거: `approve`/`merge`, risk=low — 프로덕션 1파일 2줄에 질의 시점 매칭 변경뿐이고 마이그레이션·외부 상태 변경이 없어 커밋 하나 revert 로 되돌아온다. 비앵커 디렉터리 규칙이 더 많이 매칭되어 유효 소유자가 재색인 없이 달라질 수 있는 점은 릴리즈 노트 상단 경고로 넘겼다(권고 텍스트일 뿐 자동 통보 발송은 없음).
- 다음 회차에: `codeowners_test.go:48` 의 `{"docs/","src/docs",true}` 는 디렉터리 전용 패턴이 같은 이름 **파일** 에도 맞는 과잉 매칭을 고정한 것이다(앵커 형태에서 전부터 그랬던 성질의 깊이 확장, base 에서도 동일함을 프로브로 확인). 매처가 파일/디렉터리 구분을 못 받으므로 지금 구조로는 못 고친다 — 반려 사유로 보지 않았다.
