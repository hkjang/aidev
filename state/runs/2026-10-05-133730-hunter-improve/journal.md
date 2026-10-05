# 회차 노트 2026-10-05-133730-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:37] base pinned — main@605ed53
- [러너 13:37] autonomy release — 
- [러너 13:46] scout timeout — 단계 제한 시간 초과
- [러너 13:46] scout done — 공통 자원 폼의 JSON 필드가 비워졌을 때 선언된 배열 자리에 `{}` 를 보내는 것을 제출 전에 막기 (가치 3 / �

## 구현 노트
- 공통 폼의 `json` 분기를 `resourceJSONWire` 로 빼서, 빈/공백 입력은 선언 `default` 의 컨테이너(`targets`→`[]`, `config`→`{}`)를 보내고 컨테이너 불일치·스칼라·`null` 은 라벨 붙은 한국어 오류로 막는다. 전에는 모든 JSON 필드가 똑같이 `{}` 를 보내 `targets` 의 승인 해제·자산 그래프 누락이 조용히 일어났다(`domain.go:436`·793 코드로 확인). 프로덕션 1파일 + 테스트 1파일, 커밋 c4bcfb1.
- **확신 없는 곳**: (1) 저장된 `{}` 가 승인을 실제로 해제하는 end-to-end 경로는 코드 독해로만 확인했다 — DSN 이 없어 서버 왕복으로 재현하지 못했다. (2) 공백만 남은 입력(`"   "`)은 수정 전 파싱 오류였는데 이제 선언 컨테이너를 보낸다. 과제서가 지시한 동작이지만 기존 거절 하나를 수락으로 바꾼 유일한 지점이니 여기를 먼저 봐 달라. (3) 실제 브라우저에서 JsonInput 을 지웠을 때의 화면은 캡처하지 않았다.
- **일부러 하지 않은 것**: `domain.go` 에 targets 타입 검사 추가(마이그레이션 없는 계약 강화 — 이미 `{}`/null 이 저장된 행의 수정이 전부 막힌다), `initialValues` 의 `json ? {}` 폴백 수정(`f.default` 가 먼저 평가돼 도달하지 않고, 바꾸면 렌더 스냅샷이 변한다), `settings.tsx` 쪽 제출 경로(다른 함수·다른 서버 검증기 — ideas.json 에 후보로 남겼다).
- **다음 역할 주의**: 새 테스트는 DB 없이 돈다(`npm --prefix web test`, 기준선 108 → 110). Go 프로덕션 무변경이라 `internal/webassets/dist` 재복사와 `go test -race ./...` 는 하지 않았다 — Go 를 건드리면 둘 다 필요하다. 인과 확정 때 `false &&` 를 `||` 체인 앞에 붙이면 우선순위 때문에 검사가 살아남으니 괄호로 감쌀 것.
- [러너 13:51] brief accepted — 채택 — 지목한 파일·행(`resource-form-state.ts:83`, `domain.go:436`·792)과 결함 메커니즘·수정 방식(`SubmitField.default?: unknown` 추가
- [러너 13:52] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인: 새 테스트 2개를 main 의 옛 모듈로 격리 재현 → `+ {} / - []` 와 `Missing expected exception` 으로 둘만 실패(6/8), HEAD 에서 110/110. `typecheck`·`build`·`verify-pentagi`(312) 통과, 트리 clean. 커밋 메시지의 `domain.go:436`(재승인 비교)·793(comma-ok 라 panic 아닌 조용한 누락) 주장과 `cfg.fields` 가 `default` 를 그대로 넘기는 호출부(resources.tsx:1456), json 필드 2개(208·541)의 라벨까지 코드로 확인했다. 판정 approve / risk low / blocking 없음.
- 못 본 것: DSN 이 없어 서버 왕복·`approved=false` end-to-end 는 재현 못 했고(구현자와 동일 한계), 실제 브라우저 화면도 캡처하지 않았다.
- 남는 우려(릴리즈 노트에 쓸 것): 이번 버그로 이미 `targets`=`{}` 가 저장된 행은 공통 폼 저장이 막히고, 비워 `[]` 로 고치면 실제 targets 변경이라 서버가 메시지 없이 `approved=false` 로 되돌린다(domain.go:440). 계약상 맞지만 "설명만 고쳤는데 승인이 풀렸다" 로 보이므로 안내 문구가 필요하다.
- 다음 회차 후보: `JsonInput validationError`(resources.tsx:1051)는 구문만 검사해 모양 오류가 제출 때만 보인다 → 인라인으로 당기기. `SubmitField.default?` 가 optional 이라 `default` 없는 미래 json 필드는 조용히 객체 모드.
- [러너 13:56] review approved — 리뷰 승인 (risk=low)
- [러너 13:57] pr created — https://github.com/hkjang/hunter/pull/19
