- 과제: 공통 자원 폼의 JSON 필드가 비워졌을 때 선언된 배열 자리에 `{}` 를 보내는 것을 제출 전에 막기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `resourceSubmitBody`(web/src/resource-form-state.ts:83)의 `JSON.parse(values[f.key] || "{}")` 는 **필드가 비었을 때 선언된 default 를 무시하고 언제나 객체 `{}`** 를 만든다. 서비스의 `targets`(추가 공격 표면, `default: []`, 설명이 "type, value 필드의 배열") 는 배열이어야 하는데 운영자가 JsonInput 을 전부 지우고 저장하면 `{}` 가 저장되고(서버에 targets 타입 검사가 없다), `internal/app/domain.go:436` 의 변경 비교가 걸려 **진단 대상 승인이 조용히 해제**되며, `internal/app/domain.go:792` 의 `s["targets"].([]any)` 단정이 실패해 그 서비스의 공격 표면이 자산 그래프에서 조용히 사라진다. 고치면 "보내는 값 = 선언한 형태" 가 폼 단계에서 보장되고, 운영자는 서버 오류도 조용한 손실도 보지 않는다.
- 수용 기준:
  1) JSON 필드를 완전히 비운 상태로 제출하면, 그 필드의 선언 `default` 가 배열이면 `[]`, 객체면 `{}` 가 wire 값이 된다(`{}` 로 뭉개지지 않는다).
  2) 파싱된 JSON 의 컨테이너 종류가 선언 `default` 와 다르면(예: `targets` 에 `{}` 나 `5` 나 `null`, `config` 에 `[]`) 필드 라벨이 붙은 한국어 오류를 던지고 요청을 보내지 않는다. 기존 파싱 실패 문구(`올바른 JSON 형식을 입력해 주세요.`)는 그대로 둔다.
  3) 유효한 입력의 wire 값은 한 글자도 바뀌지 않는다 — 기존 `web/tests/resource-form-state.test.mjs` 의 JSON/datetime/required 단정이 모두 그대로 통과해야 한다.
  4) 테스트가 증명할 것: (a) 수정 전 `targets` 빈 문자열이 `{}` 를 만들었고 수정 후 `[]` 를 만든다, (b) 형태 불일치 입력이 **던진다**, (c) `config`(객체 default)는 기존대로 `{}`.
- 건드릴 파일 (프로덕션 1파일 + 테스트 1파일):
  - `web/src/resource-form-state.ts`:
    - `SubmitField` 타입에 `default?: unknown` 추가(호출부 `resources.tsx:1456` 은 `cfg.fields` 를 그대로 넘기고 `Field.default?: any`(resources.tsx:116)가 이미 있으므로 **호출부 무변경**).
    - `f.type === "json"` 분기를 순수 헬퍼(예: `resourceJSONWire(raw, field)`)로 빼서: 빈/공백 입력 → 선언 default 의 컨테이너를 복제(배열이면 `[]`, 그 외 `{}`), 파싱 성공 시 `Array.isArray(parsed) !== Array.isArray(declaredDefault)` 또는 parsed 가 객체/배열이 아니면 거절. `datetime` 분기의 `resourceDateTimeWire` 주석 스타일을 따라 "서버 판독기가 무엇을 기대하는가" 를 한 문단으로 남길 것.
  - `web/tests/resource-form-state.test.mjs`: 기존 파일 끝에 사례 추가(이 파일은 `../src/resource-form-state.ts` 를 직접 import 한다 — 번들 없음).
- 검증 명령 (저장소 루트에서 실제로 도는 것):
  - `npm --prefix web ci`
  - `npm --prefix web test` — **기준선 pass 수를 먼저 실측하고 기록**(v1.21.0 시점 108 로 추정, 미확인). 0건 실행을 통과로 읽지 말 것.
  - `npm --prefix web run typecheck` (tsc --noEmit), `npm --prefix web run build`
  - `npx --prefix web prettier --check web/src/resource-form-state.ts web/tests/resource-form-state.test.mjs`
  - `git diff --check`
  - Go 프로덕션 무변경이면 `internal/webassets/dist` 재복사와 `go test -race ./...`(DSN 필요)는 **불필요**. Go 쪽을 건드렸다면 `go vet ./internal/app` + `cp -a web/dist/. internal/webassets/dist/` + `go build ./cmd/hunter`.
  - 인과 확정: 새 검사를 `if (false && …)` 로 무력화하면 새 테스트 하나만 실패하는지 확인(과거 4회차에서 요구된 절차).
- 위험과 피할 것:
  - **서버 계약을 손대지 말 것.** `internal/app/domain.go` 의 services 분기에 targets 타입 검사를 새로 넣으면 이미 `{}`/null 이 저장된 기존 행의 수정이 전부 막힌다(마이그레이션 없는 계약 강화). 이번 회차는 폼에서만 막는다.
  - `initialValues`(resources.tsx:801)의 `f.type === "json" ? {}` 폴백은 `f.default` 가 먼저 평가되므로 targets 에서는 도달하지 않는다 — 건드리지 말 것(수정하면 기존 렌더 스냅샷이 바뀐다).
  - `.tsx` 는 Node 타입 스트리핑으로 import 되지 않는다. 판정 로직은 반드시 `.ts` 안에 둘 것.
  - 보호 경로(auth/OIDC/migrations/.github/workflows/Dockerfile/release.sh)는 건드리지 않는다.
  - **테스트만 있는 PR 은 이 저장소에서 닫힌 이력이 있다**(PR #15). 프로덕션 수정과 함께 낼 것.
  - 과거 교훈: 폼에서 거절 문구를 추가할 때 기존 유효 입력의 wire 문자열이 바뀌면 안 된다(10-03·10-04 회차의 핵심 수용 기준).
- 차선 후보: web 테스트 러너의 Node 하한을 `web/package.json` 의 `engines`(>=22.18)로 선언해 툴체인 드리프트를 조용한 실패가 아니라 경고로 만들기 (3/1/S). CI `node-version: '26'`·Dockerfile `node:26` 과의 단일 출처 충돌을 먼저 정하고 `engine-strict` 는 켜지 말 것. 릴리즈 경로 인접이라 신중.
