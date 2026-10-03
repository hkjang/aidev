# 회차 노트 2026-10-04-070309-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:03] base pinned — main@a9ac801
- [러너 07:03] autonomy release — 

## 정찰 노트
- 고른 이유: 기존 테스트(`internal/api/tracking_test.go:297`)가 `carried=13*1024` 로 헤더 예산을 선언하고 주석으로 "허용 목록이 헤더가 되는 유일한 관리자 작성 값" 이라 적었는데 `PolicySources` 의 ProviderCustom 분기가 그 주장을 깬다 — 저장소가 스스로 적어 둔 불변을 코드가 지키지 않는 자리라 근거가 짧고 증명이 기존 하니스로 된다. 차선(guide-shots 요약)과 콘솔 안내를 제친 이유는 둘 다 CI 가 부르지 않는 테스트에만 검증이 남거나 API 계약 결정이 과제의 절반이기 때문.
- 추측으로 적은 것: 헤더 ~24KB 는 **산술 추정이고 측정값이 아니다**(680개 × 36바이트). 구현자가 `pagePolicy()` 로 실제로 재야 한다. `AdminSettings.tsx` 는 열어 보지 않았다.
- 조심할 것: 검사를 `switch s.Provider` 안쪽에 넣지 말 것(`tracking_test.go:126` 이 provider=none 에서도 스니펫 상한이 돈다고 고정). 개수 상한만으로는 "출처 1개 × 8000룬" 이 막히지 않으므로 합계 룬 상한이 필수. 스니펫 예산은 허용 목록 예산에 **더해지므로** `carried` 를 함께 올리고 그 숫자의 이유를 적을 것.
- 한 번 클릭 허용 핸들러는 저장된 설정 전체에 `Validate()` 를 돌리므로, 상한을 넘는 스니펫이 이미 저장된 배포에서는 클릭이 400 이 된다 — 의도된 성질이지만 커밋 메시지에서 밝힐 것.
- 프로필은 2026-09-30 판(3일 전)이 구조·위험 구역·검증 함정 모두 현재 코드와 맞아 다시 쓰지 않았다. HEAD/VERSION 줄만 낡았다(지금 a9ac801 / 0.257.0).
- [러너 07:08] scout done — 붙여 넣은 추적 코드에서 읽어 낸 출처의 개수·합계 길이에 상한을 두어, 8KiB 스니펫 하나가 정책 헤더를 

## 구현 노트
- `internal/tracking/tracking.go` 하나만 바꿨다: `MaxSnippetOriginEntries=32`·`MaxSnippetOriginsTotalRunes=1024` 를 더하고 `Validate()` 의 `MaxSnippetBytes` 검사 직후(provider switch **바깥**)에 `SnippetOrigins()` 의 개수·합계 룬 검사를 배선했다. 실제 `pagePolicy()` 로 측정: 8180 바이트 스니펫 → 출처 592 개 → 헤더 **24813 바이트**. 과제서의 추정(~24.5KB)과 거의 같았다.
- 확신 없는 곳: (1) `carried` 를 13KiB→16KiB 로 올린 것이 기존 테스트의 목록-전용 검사를 느슨하게 만든다 — 정확한 검사(`grown == 3*(total+entries)`)는 그대로 두었고 예산은 새 결합 테스트가 15921 바이트로 고정하지만, 16KiB 가 "8KiB 의 두 배" 라는 근거 자체는 여전히 관례이고 측정이 아니다. 최악의 경우 여유가 463 바이트뿐이라 다음에 헤더로 가는 값이 하나 더 생기면 다시 올려야 한다. (2) live 를 **돌리지 않았다** — DSN 이 없고 이번 변경은 순수 검증 로직이라 Docker 로 DB 를 띄우지 않았다. 설정 PUT 이 실제 라우터에서 400 을 내는 것은 보지 못했고, `Validate()` 가 그 경로의 공통 지점이라는 사실(routes.go:1728)로만 추론했다. (3) 콘솔(`AdminSettings.tsx`)은 열어 보지 않았다 — 새 오류가 화면에 어떻게 보이는지 모른다.
- 일부러 하지 않은 것: `SnippetOrigins`·`PolicySources`·`pagePolicy` 를 건드리지 않았다(헤더 쪽에서 자르면 저장값과 헤더가 갈라진다). 중복 출처 제거(`Sources.all` 비대칭)와 프런트 안내는 범위 밖으로 두고 ideas.json 에 적었다.
- 다음 역할이 조심할 것: 새 테스트 둘은 DB 없이 돈다(`go test ./internal/tracking ./internal/api`). `internal/api` 쪽 측정 테스트는 `pagePolicy` 가 비공개라 반드시 그 패키지에 있어야 한다. 상한을 넘는 스니펫이 **이미 저장된** 배포에서는 한 번 클릭 허용이 스니펫 때문에 400 이 된다 — 의도된 성질이고 커밋 메시지에 밝혔다.
- [러너 07:16] brief accepted — 채택 — ProviderCustom 분기의 무제한 출처, `MaxSnippetBytes` 가 유일한 상한인 점, 두 쓰기 경로의 공통 지점, `tracking_test.go:126` 
- [러너 07:16] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 확인한 것: 검사 블록만 떼어 내 돌려 다섯 건 실패를 재현했고(테스트는 진짜로 새 경로를 지난다), 8180B→592출처→24813B 와 결합 15921B 를 직접 측정했다. 허용 목록 테스트의 정확한 단언은 carried 상향에도 남아 있다. gofmt·vet·두 패키지 테스트 통과.
- 거절 사유(수리가 가장 먼저 볼 파일): internal/api/tracking_test.go:292-301 와 internal/tracking/tracking.go:95-98 의 "설정 문서가 만들 수 있는 최악은 16KiB 아래" 주장이 거짓이다 — Provider=matomo + 8000룬 MatomoURL(또는 MomentoProxy=false 의 MomentoURL)은 Validate 를 통과하고 24339 바이트 헤더를 만든다. PolicySources:403·412 가 스니펫 출처와 똑같이 everywhere() 로 넣는데 두 URL 에는 길이 상한이 없다. 문장을 측정한 두 설정으로 한정하든 두 URL 에 상한을 두든 한쪽을 고르면 된다. 코드 자체의 검사 배선·상수 위치·룬 계산은 옳다.
- 남는 우려: carried 까지 여유 463바이트뿐. 이미 저장된 과대 스니펫이 있는 배포에서는 한 번 클릭 허용이 400 이 된다(릴리즈 노트). docs/ADMIN_GUIDE.md:374 는 아직 8KB 만 말한다.
- 못 본 것: AdminSettings.tsx, DSN 없는 환경이라 설정 PUT 의 live 경로, 브라우저 e2e.
- [러너 07:21] review rejected — 리뷰 거절: internal/api/tracking_test.go:292-301 이번 변경이 새로 적은 주석이 `carried` 를 "the whole budget for the policy header of a page … whatever the administrator stored" 라

## 수리 노트
- 지적은 **전부 맞았다**. 임시 probe 로 재현: matomo + 8000룬 MatomoURL(그리고 proxy 끈 MomentoURL)은 Validate 를 통과하고 24339 바이트 헤더를 만든다. 틀린 지적은 없었다.
- 고친 방법: 비평가가 준 두 선택지 중 **주석 한정**을 골랐다(두 URL 상한은 별개 결함이자 새 규칙 — 범위 밖). tracking.go 상수 주석과 tracking_test.go 의 `carried` 주석이 이제 "이 두 설정만 15921B 로 묶였고 헤더 전체는 아직 아니다" 를 말하고, 반례 설정과 24339 를 적어 둔다. 같은 거짓이 있던 두 문장(테스트 함수 주석, `both` 블록)도 같이 고쳤다. 코드·상수·단언은 손대지 않았다.
- 검증: gofmt·vet 깨끗, `go test -race ./cmd/... ./internal/...` 전부 ok, 측정 로그 재확인(24813 / 15921).
- 확신 없는 곳: (1) 중재자가 "거짓 주석 대신 실제 상한" 을 원했다면 이 선택이 반쪽이다 — 다만 구멍은 주석에 이름으로 남겨 다음 회차가 집을 수 있다. (2) 여전히 live PUT 경로와 AdminSettings.tsx 는 보지 못했다(이번 변경은 주석뿐이라 영향 없음). (3) docs/ADMIN_GUIDE.md:374 의 8KB 설명은 그대로 두었다.
- [러너 07:24] repair done — # 수리 요약 (시도 2)  - 비평가의 지적이 맞았다. 임시 probe 테스트로 직접 측정: `Settings{Enabled:true, Provider:matomo, MatomoURL:"https://"+8000×'a'+".corp.example", Matomo

## 비평 노트
- 확인한 것: Validate() 의 새 블록만 떼어 낸 사본에서 새 테스트 5건 실패를 독립 재현했다(테스트는 진짜로 새 경로를 지난다). ASCII 최악 15921 과 8180B→592출처→24813B 를 재측정해 구현자 수치와 일치시켰다. gofmt·vet 깨끗, 두 패키지 테스트 통과. pagePolicy 를 그대로 복제해 멀티바이트도 측정했다.
- 거절 사유(수리가 가장 먼저 볼 파일): internal/tracking/tracking.go:94-105 와 internal/api/tracking_test.go:292-299·358-417. 수리가 좁힌 문장도 여전히 거짓이다 — 상한은 룬으로 세고 예산은 바이트인데, 허용 목록 64×64룬 + 스니펫 32×32룬을 **한국어**로 채우면 Validate 가 통과하고 헤더가 **33513 바이트**가 된다(측정). 그 다음 문장이 "막아야 할 같은 크기" 로 지목한 24339 보다 크다. 상수 1024 의 근거("tripled at a little over three kilobytes")도 틀렸다 — 한국어 스니펫 단독으로 헤더 5229B(≈4.9KiB). 새 테스트는 ASCII 만 재므로 선언한 불변을 고정하지 못한다. 길은 셋: 예산을 룬으로 말하기, 테스트가 멀티바이트 최악을 단언하기, 합계를 바이트로 두기.
- 남는 우려: 같은 룬/바이트 혼동이 diff 밖에도 있다 — tracking.go:66-68 의 12.5KB, api/tracking_test.go:335 의 grown(바이트)==3*(4096+64)(룬). 이미 저장된 과대 스니펫이 있는 배포에서는 한 번 클릭 허용이 400(릴리즈 노트). docs/ADMIN_GUIDE.md:374 는 아직 8KB 만 말한다.
- 차단 없음: 입력 검증을 좁히기만 하고 새 경로·인가·비밀값·개인정보가 없다. 남은 과대 헤더는 관리자가 자기 콘솔을 넘어뜨리는 경로여서 권한 상승이 아니다.
- 못 본 것: AdminSettings.tsx, DSN 없어 설정 PUT 의 live 경로, 브라우저 e2e.
- [러너 07:28] review rejected — 리뷰 거절: internal/tracking/tracking.go:94-105 주석이 "이 두 상한이 묶는 최악의 경우" 를 15921 바이트라고 단언하지만 코드는 그것을 강제하지 않는다. 상한�
- [러너 07:28] pr created — https://github.com/hkjang/AgentHub/pull/41
