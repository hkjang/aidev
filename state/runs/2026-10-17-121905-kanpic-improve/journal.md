# 회차 노트 2026-10-17-121905-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:19] base pinned — main@b0481a9
- [러너 12:19] autonomy release — 

## 정찰 노트
- 고른 이유: `splitSheetRange` 의 `''` 미해제는 저장소 **자신의** 계약 위반이라 증거가 코드 안에 다 있다 — `reference.go:58,161` 이 `'`→`''` 로 쓰고 `structure.go:180-183`·`evaluator.go:250-252`·`reference.go:116-117` 이 풀어 읽는데 TS 한 함수만 빠졌다. 계약 재결정이 필요한 후보(LazyQuotes, boolean/TRUE 키, 1e3)와 Postgres 가 있어야 증명되는 후보(SendNow, 별칭 정규화)를 모두 제쳤다.
- 확신 없는 곳: **실행 재현을 하지 않았다**(`web/node_modules` 없음 — 코드 읽기만으로 판단). `replaceAll` 이 tsconfig 타깃에서 쓸 수 있는지도 미확인 — 과제서에 `.split("''").join("'")` 대안을 적어 두었다.
- 구조 메모: 따옴표 없는 이름 경로(`O'Brien!A1:B2`)는 지금도 바르게 동작한다. 고칠 곳은 바깥 따옴표가 있을 때뿐이고, 없을 때 풀면 Go `quotedQualifier` 와 다시 갈린다.
- 구현자가 조심할 것: 같은 파일의 `compareKey`/`looksLikeIdentifier` 에 손대지 말 것(별개 보류 과제 2건이 거기 걸려 있다), `internal/formula/*.go` 통합 금지, `EditorPage.tsx` 는 읽기만.
- 기각 기록: cron step 상한 `*/8` 은 `rejected` 로 내렸다 — 2026-10-16 회차 테스트가 그 동작을 Next 시각까지 고정해 두어 좁히는 비용이 가치를 넘는다.
- [러너 12:22] scout done — 대사 범위 파서 `splitSheetRange` 가 `'Bob''s Data'!A1:B2` 의 이중 작은따옴표를 풀지 않는다 — 저장소의 나머지 �

## 구현 노트
- `compareLists.ts` 의 `splitSheetRange` 한 자리를 세 줄로 나눠, 바깥 따옴표가 **벗겨진 경우에만** `''`→`'` 를 치환한다. Go 의 `quotedQualifier` 와 같은 계약으로 맞춘 것이고, 따옴표 없는 이름(`A''B`)은 글자 그대로 둔다.
- 확신 없는 곳: 실사용 경로를 브라우저에서 돌려 보지는 않았다 — `EditorPage.parseSheetRange` 가 고쳐진 이름으로 시트를 찾는다는 것은 코드 읽기로만 확인했다(`sheets.find(name.toLowerCase()===…)`). E2E/Playwright 미실행.
- 일부러 하지 않은 것: `internal/formula/*.go` 통합(운영자 지시 — 거울은 각각 맞춘다), `EditorPage.tsx` 수정(특히 `openCompare` 의 따옴표 없는 프리필 — 지금도 동작하므로 범위 밖, ideas.json 에 후보로 남겼다), 같은 파일의 `compareKey`/`looksLikeIdentifier`(별개 계약 결정), docs/PDF.
- 다음 역할이 조심할 것: 새 테스트 3개 중 **빨강이던 것은 1개**다(`reads a doubled quote inside the quotes as one quote`). 나머지 2개(따옴표 없는 경로, 빈 이름)는 처음부터 초록인 회귀 방지용이고 노트에 그렇게 적었다 — "바뀐 경로를 지나지 않는다" 로 읽지 말 것. `web/node_modules` 는 이번에 설치했으니 그대로 쓸 수 있다(`npm ci` 재실행 불필요).
- [러너 12:26] brief accepted — 채택 — 과제서의 코드 분석이 지금 코드와 정확히 맞았고(바깥 따옴표만 벗김, Go 쪽은 `''` 를 풂), 건드릴 파일 2개(프로�
- [러너 12:26] verify passed — 검증 7개 통과 (auto)
- [러너 12:37] resume closed — PR 이전 단계에서 중단 — 에이전트 단계는 재실행하지 않음
