# 회차 노트 2026-10-05-222757-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:28] base pinned — main@2b26248
- [러너 22:28] autonomy release — 

## 정찰 노트
- 고른 이유: 상한 계열에서 **아직 열려 있는 유일한 비대칭**이다 — 붙여 넣은 스니펫은 8KiB로 묶였고 출처는 개수·합계·제공자까지 묶였는데, 제공자 분기가 렌더링하는 스니펫(세 id + URL 전체)에는 어떤 크기 상한도 없다. 차선 후보(guide-shots 요약, CI 포함)는 DISPOSABLE 환경 전제이거나 보호 경로(.github/workflows)라 가치가 낮고, 헤더 크기 경고 로그는 5초 TTL 반복 로그 때문에 기각했다.
- 확신 있는 것(직접 열어 확인): Validate 275-281행 뒤가 비활성에도 걸리는 자리, 세 id에 길이 검사 없음, Snippet()의 네 포맷 문자열, api/tracking.go:140의 주입, AdminSettings.tsx:167에 maxLength 없음, 기존 테스트 통과(tracking cached / api 1.767s).
- 추측으로 남긴 것: 상수 **숫자**(200/1024는 제안일 뿐 — 구현자가 기존 테스트가 통과하는 값으로 정하고 이유를 주석에 적을 것), 상한 안쪽 최대 설정의 Snippet 바이트 수(재지 않았다 — 테스트가 재게 했다), internal/api 테스트 한 건의 필요성(선택).
- 구현자가 조심할 것: 이건 **크기** 문제이지 주입 문제가 아니다(네 포맷 모두 html.EscapeString을 거친다) — MomentoEnvironment의 ContainsAny를 베껴 두 번째 계약을 만들지 말 것. 주석에서 룬/바이트를 혼동하지 말 것(c9fa51f·cbbc4fb가 그 혼동을 고쳤다). trim 하지 말고 거절할 것.
- 프로필: 0일 전 것이지만 "아직 열린 제공자 출처 길이" 줄이 6a7c0f1로 해결되어 낡았으므로 profile.md를 갱신했다.
- [러너 22:32] scout done — 제공자 설정으로 렌더링되는 추적 스니펫의 크기에 상한을 둔다 (사이트 id·measurement id·제공자 URL 전체 �

## 구현 노트
- 바꾼 것: `internal/tracking/tracking.go` 하나. `MaxProviderIDRunes=200`·`MaxProviderURLRunes=1024` 를 `Validate()` 의 제공자 출처 루프 자리(`if !s.Enabled` **앞**)에 배선 — 세 id 와 두 URL 전체가 `Snippet()` 을 통해 모든 추적 페이지 본문에 나가는데 상한이 없었다. 거절만 하고 trim 하지 않으며 룬으로 센다.
- 확신 있는 것: 수정 전 실패 → 수정 후 통과 → 검사만 되돌려 live 재실패 → 복구를 모두 눈으로 봤다. 일회용 postgres(55447)로 live 를 실제로 돌려, 상한 없을 때 관리자 PUT 이 200 을 받고 과대 measurement id 가 `/runs` 응답 **본문**에 905바이트로 실려 나가는 것을 관측했다. `runtime-images.json` 14개 이미지의 sourcePaths 를 직접 읽어 BASE_VERSION 불필요를 확인.
- 확신 없는 것: 숫자 200/1024 는 과제서 제안을 그대로 쓴 것이고 **조직의 실제 제공자 URL 길이는 미확인**이다 — 경로가 1024룬을 넘는 수집기 주소를 쓰는 배포가 있다면 그 저장은 이제 거절된다(기존 저장값은 읽기 경로에서 수리되지 않으므로 계속 나간다). GA4/GTM/Matomo id 형식은 일반 지식이고 벤더 문서로 확인하지 않았다.
- 일부러 안 한 것: `AdminSettings.tsx` 의 입력 안내·maxLength(과제서가 별도 아이디어로 분리), `MomentoEnvironment` 의 바이트/룬 혼동(과제서가 금지한 자리 — ideas.json 에 단독 과제로 남겼다), `cachedTrackingSettings` 가 이미 저장된 과대 값을 수리하지 않는 공백(같은 이유로 보류).
- 다음 역할이 조심할 것: 새 live 서브테스트(`trackingallowlist_live_test.go` 의 `an oversized provider id or address never reaches a page`)는 **DB 가 있어야 돈다** — `AGENTHUB_TEST_DSN` 없으면 상위 테스트가 skip 되고 아무 신호도 남지 않는다. 반대로 `internal/api/tracking_test.go` 에 더한 두 건은 `validateSetting` 단위 호출이라 **HTTP/DB 저장의 증명이 아니다**. `internal/tracking` 의 측정 테스트는 `MaxSnippetBytes` 와의 대칭을 단정하므로, 제공자 상한을 올리면 그것이 먼저 실패한다.
- [러너 22:47] brief accepted — 채택 — 근거가 현재 코드와 정확히 일치했다(세 id 에 길이 검사 없음, 278행이 `originOf` 결과만 봄, `Snippet()` 의 네 포맷이 
- [러너 22:47] verify passed — 검증 5개 통과 (auto)
