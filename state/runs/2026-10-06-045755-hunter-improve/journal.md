# 회차 노트 2026-10-06-045755-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 04:58] base pinned — main@398cd71
- [러너 04:58] autonomy release — 

## 정찰 노트
- 골랐다: scopes·policies 폼의 숫자 상·하한이 `policy.go:79`·`96` 의 서버 테이블과 **7개 전부** 어긋나는 것. 이 저장소가 네 회차 연속 성공한 "폼이 검사한 값 ≠ 서버가 받는 값" 패턴의 가장 큰 남은 사례이고, 양쪽 리터럴을 직접 읽어 확정했다. 차선(engines 선언)은 릴리즈 경로 인접이라, 3순위(list-export RangeError)는 도달 가능성이 낮아 제쳤다.
- 기각: 보류 1순위였던 "settings.tsx 가 공통 폼 가드를 공유하지 않음" 은 settingFields 에 json·datetime 타입 필드가 하나도 없어 도달 결함이 없다(grep 전수 확인) → ideas.json 에 rejected.
- 미확인(추측으로 적은 것): ① Mantine 8 NumberInput 이 비운 입력에서 `""` 를 준다는 것과 blur 클램프 동작은 **코드 추론**이다 — `web/node_modules` 가 없어 실측하지 못했다. ② 기준선 테스트 수(110)도 실측하지 못했다. 둘 다 구현 환경에서 먼저 확인할 것.
- 조심할 것: 서버 범위를 넓히는 방향은 기각이다(진단 안전 통제). 폼 `min` 을 올리면 이미 저장된 범위 밖 행이 열릴 때 클램프로 조용히 바뀔 수 있으니, 위험하면 상한·소수만 고치는 축소 범위로 내려올 것 — 과제서에 그 분기를 적어 뒀다. 정수 잠금은 반드시 필드 선언에서 읽을 것(`resources.tsx:1019` 렌더러는 모든 kind 가 공유한다).
- [러너 05:05] scout done — 진단 범위·실행 정책 폼의 숫자 상·하한을 서버가 실제로 수락하는 범위로 맞추고 공유 벡터로 두 쪽을 �

## 구현 노트
- 바꾼 것: `scopes`·`policies` 의 숫자 7개 선언이 새 공유 테이블 `resourceNumberBounds`(`web/src/resource-form-state.ts`)를 펼쳐 쓰고, 공유 렌더러는 선언에서 읽은 `allowDecimal={f.integer !== true}` 로 정수를 잠근다. 서버(`policy.go` 두 테이블, `number()`)는 무변경, 유효 입력의 wire 값도 무변경. 프로덕션 2파일 + 테스트 3파일.
- **하한은 과제서 수용 기준 2 를 기각했다.** 테이블 2열은 `def`(폴백)이고 검사는 `n < 1 || n > b.max` 이므로 서버의 실제 바닥은 1 이다 — 일회성 탐침으로 `scopes max_requests=9`·`policies timeout_seconds=9` 수락을 직접 확인했다. 하한을 10·30 으로 올리면 폼이 서버가 받는 안전한 값을 거절하고, 이미 저장된 행을 탭으로 지나가는 순간 blur 클램프가 값을 올린다. `max_rps` 의 `min: 0.1` → 1 만 거절을 막는 변경(1 미만은 `int()` 로 0). 상한 4개는 서버 테이블과 일치시켰다.
- **확신 없는 곳**: 정수 잠금이 실제 브라우저에서 소수 타이핑·붙여넣기를 막는 것은 Mantine 8.3.18 구현(`NumberInput.mjs:379` `decimalScale: allowDecimal ? decimalScale : 0`)을 **읽어** 확인했을 뿐이다. 저장소에 DOM 하네스(jsdom/testing-library)가 없어 렌더링으로 실측하지 못했다 — 비평가가 먼저 볼 곳. TS 쪽은 `resources.tsx` 소스 텍스트로 "렌더러가 `f.integer` 를 읽는다"만 고정한다(정규식이라 문법을 크게 바꾸면 깨진다).
- 일부러 안 한 것: `resourceSubmitBody` 의 `number` 분기(비운 입력의 `""`). `contribution_points`·`interval_minutes` 동작을 바꾸지 않기로 해 제외하고 ideas.json 에 pending 으로 남겼다. `datetime` 컨트롤의 `max`/`step` 도 같은 렌더러지만 섞지 않았다.
- 다음 역할이 조심할 것: Go 테스트 `^TestResourceNumberBounds` 는 **DSN 불필요**(16서브테스트, SKIP 0). 벡터의 `min` 을 1 위로 올리면 Go 가 의도적으로 실패한다 — 그게 하한을 서버 바닥에 묶는 장치이니 "폼을 더 엄격하게" 하려고 올리지 말 것. 서버 범위를 넓히는 방향은 진단 안전 통제 위반이다.
- 미실행: 원격 CI(Node 26)·Docker 빌드·화면 캡처. Go 프로덕션 무변경이라 `internal/webassets/dist` 재복사와 `go test -race ./...`(DSN 필요)는 생략했다.
- [러너 05:14] brief accepted — 채택 — 지목한 파일·행(`resources.tsx` 의 7개 선언, `policy.go:79`·96, `number()` 의 `int(n)` 절삭, 렌더러 1019 에 `step`·`allowDecimal` 
- [러너 05:15] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인함: `policy.go:79·96` 리터럴과 `number()`(domain.go:115)로 하한이 실제로 1 임을 재확인 — 구현자가 과제서 수용 기준 2 를 기각한 것이 옳다. Mantine 기본값은 node_modules 소스로 실측(`NumberInput.mjs:56·57·379`): `integer` 없는 필드는 `allowDecimal={true}` = 기본값이라 settings.tsx 와 다른 kind 에 영향 없음. 실측 113통과/0실패/0건너뜀, `^TestResourceNumberBounds` 16/16 PASS·SKIP 0, vet·tsc·vite build·verify-pentagi(312) 모두 green.
- TS 세 테스트는 수정 전 코드에서 전부 실패한다(옛 리터럴 deepStrictEqual, 스프레드 소스 단언, `f.integer` 정규식). Go 쪽은 수정 전에도 통과하는 서버 특성화 앵커이며 그 역할로 적절하다. `policy.go` 는 v1.0.0 이후 무변경이고 쓰기 경로가 validateResource 하나뿐이라 상한 축소가 저장된 행을 클램프할 수 없다. 서버 범위 확대 없음 → 보안·법무 차단 사유 없음.
- 못 본 것: 실제 브라우저/DOM 렌더링(하네스 없음), 원격 CI(Node 26)·Docker 빌드·`go test -race`(DSN 필요)·화면 캡처.
- 승인이어도 남는 우려: ① react-number-format 의 `decimalScale:0` 은 절삭이 아니라 **반올림**이라 `2.7` 붙여넣기는 `3` 이 된다 — 릴리즈 노트에서 "서버와 같은 절삭"으로 쓰지 말 것(표시값=전송값이므로 결함은 아니다). ② `max_concurrency` min=max=1 이 되어 스테퍼가 사라지고 사실상 고정 필드가 된다(UX, 다음 회차).
- 다음 회차 1순위: 같은 패턴의 마지막 열린 사례 — `resources.tsx:282` `contribution_points` 가 **max 미선언**인데 `domain.go:500` 은 10000 초과를 거절한다. `interval_minutes` 는 이미 서버와 일치하므로 할 일 없음. 비운 입력의 `""` → 서버 `def` 폴백(`scopes.max_requests` 지우면 10 저장)도 여전히 pending.
- [러너 05:20] review approved — 리뷰 승인 (risk=low)
- [러너 05:20] pr created — https://github.com/hkjang/hunter/pull/20
