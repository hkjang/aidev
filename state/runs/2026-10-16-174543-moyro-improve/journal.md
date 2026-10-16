# 회차 노트 2026-10-16-174543-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:45] base pinned — main@c0ab918
- [러너 17:45] autonomy release — 

## 정찰 노트
- 우선 과제가 배정되었으므로 새 후보를 고르지 않고 "같은 이유로 두 번 실패한 릴리즈" 의 원인 특정에 집중했다. 직전 수리 커밋 `fddc0b8` 이 **외부 사실 두 개**(Go 1.27.2 출시 여부, `golang:1.27.2-alpine` 다이제스트)를 저장소에 핀으로 박았고 그 둘은 입력이 바뀌지 않는 한 매 실행 같은 메시지로 실패하므로 "두 번 같은 이유" 와 모양이 가장 잘 맞는다. `1.27` 문자열이 `server/go.mod` 와 `Dockerfile` 두 파일에만 있어 수정 범위도 2~3파일로 끝난다.
- 확신 없는 곳(과제서에 "미확인" 으로 명시): 실패한 **단계 이름**(gh 미인증), Go 1.27.2 의 실제 출시 여부, `Dockerfile:14` 다이제스트의 유효성. 이 세션에는 `curl`·WebFetch·`go version` 실행 권한이 없어 세 가지 모두 외부 확인을 못 했다 — 전부 연역이고, 과제서 수용 기준 1)이 그 확인을 구현자의 첫 단계로 돌려놨다.
- 확인한 사실만: `server/go.sum:347-348` 에 `x/net v0.60.0` 해시 두 줄이 있어 그 모듈 버전은 존재한다. `plugin-sdk/go/go.mod` 는 `go 1.22` 로 무영향. `Dockerfile:14` 는 `golang:1.27.2-alpine@sha256:f92b6ef8…`, `server/go.mod:3` 은 `go 1.27.2` 이며 `toolchain` 줄은 없다.
- 구현자가 조심할 것: 워크플로를 느슨하게 해 통과시키는 것은 이번 회차 실패로 취급된다(다이제스트 핀 제거·`continue-on-error`·audit 레벨 완화 금지, `.github/workflows/` diff 0). a~c 진단이 전부 초록이면 추측으로 파일을 고치지 말고 2026-10-04 처럼 "깨지는 게이트 없음" 을 출력과 함께 보고하고 차선으로 넘어갈 것.
- 기각·차선이 반복된 유형(httpapi 상태코드 재분류·`decodeCappedBody` 계열)은 1순위로 올리지 않았고 차선 1건만 남겼다.
- [러너 17:51] scout done — `fddc0b8` 이 넣은 **검증되지 않은 Go 1.27.2 핀**(`server/go.mod:3` + `Dockerfile:14` 의 다이제스트)이 CI·릴리즈를 매 

## 구현 노트
- 과제서의 인과가 뒤집혀 있었다. **공개 저장소의 Actions API 는 `gh` 인증이 필요 없다** — `curl .../actions/runs` + `.../runs/<id>/jobs` 로 실패 런이 38859433565(PR #39, `verify` 잡, step 9 "Test server") 1건뿐이고, `fddc0b8` 이 그것을 **고쳐** 다음 런이 통과, PR #39 머지, v0.2.50 릴리즈 런 38865940583 **success** 임을 확인했다. 핀 두 개도 실제로 옳다(go1.27.2 실존, 다이제스트 실존, 이미지 내부 go1.27.2, `docker build` exit 0). 그래서 과제서의 "둘 다 아닐 때" 경로로 가 릴리즈 순서를 전부 로컬 재현(전부 초록)하고 **차선 후보**를 구현했다: `postsByIDsReactions` 의 `ListForPost` 장애만 기존 500 `api.reaction.list.app_error` 로 분리(프로덕션 1파일 +12/-2, 새 오류 id·상태코드 0).
- **확신 없는 곳·검증 못 한 것**: ① `verify-product-ui.sh`(Playwright 77 테스트)는 **돌리지 않았다** — 예산 때문이고, 이 변경은 웹 0파일이라 영향 경로가 없다고 판단했다. ② 과제서가 범위 밖으로 둔 `ChannelForPost`·`IsMember` 두 `continue` 는 **여전히 장애를 정상 필터로 접는다**; 보존 단언으로 현재 동작만 고정했을 뿐 고치지 않았다. ③ 응답 detail 에 `err.Error()` 를 그대로 싣는데, 이는 `listReactions`(handlers.go:2503)의 기존 계약을 그대로 따른 것이다 — 그 계약 자체가 옳은지는 판단하지 않았다. ④ CI 의 `verify` 잡에 npm/Playwright 쪽 라이브 요소가 있어 시점에 따라 결과가 달라질 수 있다(이번 시점엔 전부 초록).
- **일부러 하지 않은 것**: 워크플로 파일은 한 줄도 건드리지 않았다(`git diff --stat .github/` = 0). `fddc0b8` 의 핀을 "혹시 몰라" 내리는 것도 하지 않았다 — 핀이 옳다는 증거가 나왔으므로 내리는 쪽이 오히려 회귀였다. 같은 함수의 64 KiB 조용한 절단(late.go:311)은 같은 줄을 연달아 건드리지 않으려고 미뤘다.
- **다음 역할이 조심할 것**: 새 테스트는 **DB 가 있어야 돈다** — DSN 없으면 skip 하고, 컨테이너 `moyro-pg-improve`/55433 의 자격증명은 `moyro/moyro/moyro` 다(`postgres/postgres` 로 추측하면 httpapi 가 0.5초에 FAIL 한다). 이 테스트는 `reactions` 를 뷰로 바꿨다 되돌리므로 **전용 스키마**(`newOperationsTestDB`)가 필수다. 이 러너에서 **처음으로** `TestMattermostArchivesInstallRunAndSurviveRestart` 가 skip 아닌 `PASS (5.63s)` 로 돌았다(아카이브 4개를 실제로 받아 줬다). 끝으로 내 커밋 sha 로 이미지를 다시 빌드해 `EXPECTED_COMMIT` 를 준 `verify-release.sh` 가 통과했으므로 변경이 출하 산출물까지 도달하는 것은 end-to-end 로 확인됐다.
- [러너 18:08] brief fallback — 차선 — 과제서의 **인과가 뒤집혀 있었다**. 범인으로 지목한 `fddc0b8` 의 핀 두 개(`server/go.mod:3` 의 `go 1.27.2`, `Dockerfile:14` �
- [러너 18:09] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인함: diff 2파일 전부 정독 — 뷰의 4개 컬럼이 `reactions/service.go:47` SELECT 와 정확히 일치하고, `seedSidebarHandlerFixture`(sidebar_handlers_postgres_test.go:171-183)가 user-a 를 chan-alpha 멤버로 넣으므로 보존 단언들이 실제로 성공 경로를 지난다. 실제 DB(55433)로 테스트를 직접 돌려 **9/9 PASS, skip 0, 1.03s**(0.00s 가 아님)를 재현했다. `.github/`·migrations diff 0줄, 트리 clean, 라우트(`router.go:773`)·오류 id(`handlers.go:2503` = 500 `api.reaction.list.app_error`) 둘 다 원장 주장과 일치.
- RED 은 코드를 되돌리지 않고 **연역으로만** 확인했다(리뷰어는 편집 금지): `continue` 로 되돌리면 `{"ids":["post-boom"]}` 가 `out={}` → 200 `{}` 가 되어 원장의 RED 출력과 바이트 단위로 같다. 재현 실행은 하지 않았다.
- 남는 우려(차단 아님): ① `err.Error()` 가 응답 `message` 에 그대로 실린다 — `listReactions` 의 기존 계약과 동일하고 호출자는 이미 인증+멤버 게이트를 통과했으므로 새 노출은 아니지만, 저장소 전역의 이 관행은 언젠가 한 번에 정리할 대상이다. ② 영구적으로 읽히지 않는 반응 행 하나가 **배치 전체**를 500 으로 만든다(부분 응답 → 전무). 조용히 틀린 데이터보다 낫고 upstream Mattermost 와 같은 방향이지만 릴리즈 노트에 "읽기 실패 시 배치 전체 거부" 로 적어 둘 가치가 있다.
- 못 본 것: `ChannelForPost`·`IsMember` 두 `continue` 는 **여전히 DB 장애를 정상 필터로 접는다**(구현자가 범위 밖으로 명시, 보존 단언으로 고정만 됨) — 다음 회차의 자연스러운 후속이다. 같은 함수의 64 KiB 본문 조용한 절단(late.go:311)도 그대로 남았다. 웹 소비자는 없으므로(`ids/reactions` grep = 서버·테스트뿐) Playwright 미실행은 이번엔 무해.
- 판정: approve / risk low / blocking 없음.
- [러너 18:12] review approved — 리뷰 승인 (risk=low)
- [러너 18:12] pr created — https://github.com/hkjang/moyro/pull/40
- [러너 18:23] ci passed — 검사 3개 모두 success
- [러너 18:23] merge done — 33ff037
- [러너 18:51] release published — v0.2.51
- [러너 19:08] assets verified — v0.2.51 자산 1개 (이전 v0.2.50: 1)
