# 회차 노트 2026-10-06-070803-jupiq-improve — jupiq
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:08] base pinned — main@92ff88c
- [러너 07:08] autonomy release — 

## 구현 노트
- 무엇·왜: provider=custom 스니펫에 `strings.ToLower`가 바이트 길이를 줄이는 글자(İ U+0130, K U+212A)가 있으면 소문자 사본의 위치가 원본과 어긋난다. `SnippetOrigins`는 그 어긋남으로 **끝나지 않고**(pagePolicy가 매 화면 요청마다 부른다 → 모든 페이지 요청이 돌아오지 않는다), `withNonce`는 태그 이름을 잘라 `<scrip nonce="…"t`를 내보낸다. ASCII만 접는 `IndexFold`/`LastIndexFold`를 두고 세 곳(+`injectSnippet`)이 같은 입력을 같은 위치로 읽게 했다. 프로덕션 2파일.
- 확신 없는 곳·검증 못 한 것: ① `injectSnippet`은 유입 경로를 **확인하지 못했다** — 입력이 우리가 만든 `web/dist/index.html`뿐이고 거기엔 줄어드는 글자가 없다. 같은 결함을 한 기능 안에 남기지 않으려는 일관성 수정이며, 테스트는 함수를 직접 부른다. ② 설정 **저장소**를 거치는 경로(`analyticsConfig → Store.GetSetting → ParseConfig`)는 타지 않았다. 재현은 실제 핸들러(`servePage → serveSPAWith → trackedPage → pagePolicy`)로 했지만 `analytics.Config`는 직접 넘겼고, "저장할 수 있는 상태"는 `config.Validate() == nil` 단언으로만 보였다. ③ 실제 운영에서 비ASCII 주석이 든 스니펫이 얼마나 흔한지는 모른다 — 심각도는 확실하지만 발생 빈도는 추정이다.
- 일부러 하지 않은 것: `Validate()`에 스니펫 글자 제한을 더하지 않았다(터키어 주석은 정상 입력이고 결함은 파서 쪽). `withNonce`의 `strings.Contains(strings.ToLower(tag), "nonce=")`는 불리언이라 위치를 쓰지 않으므로 그대로 뒀다. `SplitHosts`·`PolicySources`는 건드리지 않았다. web 트리 무변경.
- 다음 역할이 조심할 것: 되돌림 확인을 할 때 `SnippetOrigins` 수정을 빼면 테스트가 **실패가 아니라 멈춘다** — `go test`에 `-timeout 20s`를 꼭 붙일 것(기본 10분을 기다린다). npm 게이트는 web 무변경이라 돌리지 않았고, SQL 표면이 없어 `make test-integration`(DSN 필요)도 돌리지 않았다. `IndexFold`/`LastIndexFold`는 새로 exported 된 이름이라 다른 회차 산출물과 이름이 겹치지 않는지 머지 때 볼 것.
- [러너 07:18] verify failed — 실패한 검증: npm run lint   # CI에서 가져옴 (exit 1)
