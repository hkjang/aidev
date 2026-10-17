# PR 처리기 노트 2026-10-17-093720-bamcp-shepherd — bamcp PR #2
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-17-090251-bamcp-improve)
# 회차 노트 2026-10-17-090251-bamcp-improve — bamcp
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:02] base pinned — main@70ee0f6
- [러너 09:02] autonomy release — 

## 정찰 노트
- 과제는 자동 배정(CI 실패)이라 고르지 않았다. 다만 범위를 좁혔다: 0659 회차 러너 기록에 **4개 잡이 전부 failure** 로 찍혀 있는데 PR #1 디프는 `internal/tools/` 3개 파일뿐이라 웹·플러그인·도커가 깨질 코드 이유가 없다 → 공통(계정/러너/워크플로) 원인을 1순위 가설로 적었다.
- **가장 큰 미확인**: 이 세션에서 `gh`·`go`·`npm`·`docker`·`mvn` 이 전부 권한 거부되어(허용된 것은 `git`/`ls`/`cat`/`grep` 뿐) **실패 로그를 한 줄도 못 봤다**. 과제서의 잡별 후보는 전부 정적 읽기 기준 추측이다. 구현자는 `gh run view --log-failed` 부터 하고, 내 가설과 다르면 내 가설을 버려라.
- 정적으로 확인한 것만: `args.go` 의 import 는 전부 사용 중이라 컴파일 에러는 아니다. TS 는 락파일 5.9.3 이라 `tsc -b --noEmit` 조합 자체는 합법이다. `plugin/pom.xml`·`Dockerfile`·`web/package-lock.json` 모두 존재한다. 즉 "파일이 없어서" 는 아니다.
- 구현자가 조심할 것: 워크플로를 느슨하게 해 통과시키는 것은 반려 사유다. 그리고 4개 잡을 한 번에 고치려 들면 파일 수가 바로 10개를 넘는다 — **한 잡만** 고치고 나머지는 ideas.json 에 남겨라.
- 프로필(`2026-10-17-065908-bamcp-improve/profile.md`)은 0일 전이고 이번에 읽은 범위(go.mod indirect 오표기, 커밋된 dist/JAR, TEST_DATABASE_URL 함정)와 어긋나지 않아 새로 쓰지 않았다.
- [러너 09:07] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- **배정된 CI 실패는 저장소 밖 원인으로 확정했다.** 결정적 증거는 두 개다. (1) 이 환경 저장소 28개의 GitHub 공개 여부(`api.github.com/repos/hkjang/<repo>` 200/404)와 CI 결과가 **28/28 일치** — 비공개 10개는 전부 2~3초 startup 실패, 공개 18개는 전부 스텝을 실행했다. (2) 쌍둥이 저장소 `bbmcp`(공개)가 같은 분 `22:14:39Z` 에 **동일한 액션 버전(checkout@v4·setup-go@v5·setup-node@v4)** 과 거의 동일한 ci.yml 로 통과했다 → 앞 회차 수리의 "deprecated 액션 강제 실패" 가설은 **반증**이다. 액션 버전을 올리는 변경은 하지 말 것(효과 없고 회귀 위험만 있다). 남은 조치는 결제 한도·공개 전환·self-hosted 러너로 전부 저장소 밖이다.
- 그래서 차선으로 앞 회차 비평이 지목하고 미수정으로 남긴 실제 결함을 고쳤다: `analysis.go:257` 이 `splitResultKey` 해석 실패를 권한 제외와 같은 분기에서 `c.Filtered++` 로 집계해, 배포 전 점검에서 원천 빌드가 조용히 빠지고 원인이 "권한 부족" 으로 오보고됐다. 프로덕션 파일 1개, 해석 실패만 경고로 분리하고 제외는 유지(fail-closed).
- **확신 없는 곳**: 실패 주석 본문은 끝까지 못 읽었다(비공개 저장소 + `gh` 미인증). 과금 소진이라는 결론은 28/28 상관과 공개/비공개가 Actions 과금에 유일하게 영향 주는 변수라는 사실에서 나온 추론이고, "분 소진" 과 "지출 한도 미설정" 중 어느 쪽인지는 구분하지 못했다. 또 실제 Bamboo 6.10.4 가 어떤 형태의 `planResultKey` 를 주는지는 확인 못 했다 — 테스트는 모의 서버에 잡 단위 키(`CORE-API-JOB1`)를 심어 재현했다.
- **일부러 하지 않은 것**: 워크플로 파일은 한 줄도 건드리지 않았다(원인이 아니고, 느슨화는 금지). 보류 아이디어 (1) `read.go` raw start/limit 클램프는 **기각**했다 — `client.max()`·`pageQuery()` 가 프로덕션 배선에서 이미 클램프해 실패하는 테스트를 만들 수 없다.
- **다음 역할이 조심할 것**: `TestUnparsableReleaseSourceIsNotCountedAsPermissionFilter` 는 **DB 가 있어야 돈다**. `TEST_DATABASE_URL` 없이는 skip 되고 통과처럼 보인다 — `internal/tools` 가 7.7s 쯤 나와야 진짜 돈 것이다(0.00Xs 면 skip). `env` 에 `mockURL` 필드를 추가했다(테스트 전용). 그리고 이 PR 도 CI 는 같은 이유로 실패할 것이다 — 그건 이 변경 탓이 아니다.
- [러너 09:22] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 확인한 것: postgres 17 컨테이너를 띄워 `TEST_DATABASE_URL` 로 전 패키지 통과(internal/tools 4.3s, skip 아님), analysis.go 만 main 으로 되돌려 원장이 적은 것과 똑같은 두 줄로 FAIL — 인과 확인. 프로덕션 11줄은 건전하고 fail-closed 유지, 신규 정보 노출도 없다(`release.planResultKeys` 가 이미 키 전체를 돌려준다).
- 거절 사유 1건, 수리가 먼저 볼 파일: `internal/tools/integration_test.go:637`. `dataOf(res)["sources"]` 가 존재하지 않는 필드다 — 핸들러 이름은 `sourceBuilds`(analysis.go:307). 항상 nil 이라 영원히 실패할 수 없고, 그게 바로 '해석 못한 키는 여전히 제외된다'(fail-closed 절반)를 고정하는 유일한 단언이다. 원장의 `- 실패 재현:` 이 642·652 만 적고 637 이 없는 게 이미 증거. `"sourceBuilds"` 로 한 단어 바꿔도 통과 확인(0.09s) — 가드가 틀린 게 아니라 없다.
- 승인이어도 남을 우려(릴리즈 노트에 반영): 적용 범위가 설명보다 좁다. 실 Bamboo 잡 결과 키는 `PROJ-PLAN-JOB-5` 처럼 `-<번호>` 로 끝나 `splitResultKey` 가 **파싱에 성공**하고 기존 `Filtered++` 경로로 간다. 구현자가 근거로 든 실세계 증상은 안 고쳐졌다 — 노트는 '해석 불가 키' 로만 좁혀 쓰고 잡 단위 키는 다음 회차 과제로.
- 다음 회차 참고: `planResultKey` 는 `release.create` 권한만 있으면 임의 문자열이라 이제 게이트웨이가 쓴 문장 안에 외부 문자열이 끼어 모델에게 간다(trust:untrusted 라 증분은 작음). 경고에 들어가는 외부 문자열 길이 제한을 한 번 볼 가치가 있다.
- 못 본 것: CI 실패가 저장소 밖 원인이라는 28/28 논증은 읽기만 했다(GitHub 권한 없음). 작업 트리의 `internal/webui/dist/index.html` 미커밋 1줄은 세션 시작부터 있던 것으로 이 PR 밖이다.
- [러너 09:27] review rejected — 리뷰 거절: internal/tools/integration_test.go:637 `sources, _ := dataOf(res)["sources"].([]any)` 는 존재하지 않는 필드를 읽는다. 핸들러가 내보내는 이름은 `sourceBuilds` 다
- [러너 09:27] pr created — https://github.com/hkjang/bamcp/pull/2
