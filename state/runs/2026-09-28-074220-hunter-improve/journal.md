# 회차 노트 2026-09-28-074220-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:42] base pinned — main@de4bfd3
- [러너 07:42] autonomy release — 
- [러너 07:52] scout timeout — 단계 제한 시간 초과
- [러너 07:52] scout done — 발견 건 일괄 변경의 조치 담당자 200바이트 검사를 서버와 같은 문자열에 대해 세고, 공유 벡터로 두 파서

## 구현 노트
- 웹 일괄 변경 폼이 담당자 200바이트 한도를 trim 후에 셌는데 서버(finding_bulk.go:63)는 trim 전 원문을 센다. 앞뒤 공백 + 200바이트 내용이 폼을 통과해 400 으로 되돌아오던 것을 웹 1파일(finding-bulk-state.ts:26, 검사만 원문 기준)로 닫았다. 서버 계약·보내는 값(`input.assignee.trim()`)·제어 문자 검사는 그대로다.
- 확신 없는 곳: 벡터에 U+FEFF·U+2000~U+200A·U+3000 을 **의도적으로 넣지 않았다**(Go strings.TrimSpace 와 JS trim 이 U+FEFF 에서 갈린다). 그 문자들이 붙은 담당자에서 양쪽 판정이 갈리는지는 **미검증**이며 이번 수정 범위도 아니다 — 벡터 파일 `note` 에 적어 뒀다. U+00A0 은 양쪽 비제어·양쪽 trim 대상으로 코드 근거가 있어 한 사례 넣었고 통과한다.
- 실제 브라우저 UI 로 200바이트 경계를 눌러 보지는 않았다. 대신 배선을 코드로 확인했다: finding-bulk.tsx:96-98 이 TextInput 원문(maxLength 없음)을 그대로 넘기고 반환 patch 가 POST 본문이다.
- 일부러 하지 않은 것: 서버 finding_bulk.go 는 손대지 않았다(원자적 일괄 변경·전체 롤백·권한 재검사 경로이고 받는 집합을 넓히는 계약 변경). 담당자 입력란 바이트 잔량 안내는 ideas.json 으로 넘겼다(컴포넌트 인라인, DOM 하네스 없음).
- 다음 역할 주의: 새 Go 테스트 TestFindingBulkAssigneeSharedVectors 는 **DB 없이 돈다**(testApp 미사용). 같은 `-run 'FindingBulk'` 에 걸리는 기존 5개는 HUNTER_TEST_DSN 없으면 SKIP 이니 "ok" 만 보고 전부 통과로 읽지 말 것 — `-v` 로 확인하면 보인다.
- Go 프로덕션 무변경이라 internal/webassets/dist 재복사와 전체 `go test -race ./...` 는 하지 않았다. 웹 테스트 기준선은 과제서의 97 이 아니라 **실측 103**(→ 104)이었다.
- [러너 07:57] brief accepted — 채택 — 지목한 두 파일·행(`finding-bulk-state.ts:26`, `finding_bulk.go:63`·66)과 trim 전/후 불일치, `validateFindingBulk` 순수 테스트 0�
- [러너 07:57] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인한 것: `finding_bulk.go:36-79` 전문, `finding-bulk-state.ts` 전문, `finding-bulk.tsx:88-140`(POST 본문 배선), 벡터 17사례, 두 새 테스트. 실행: Go `-run TestFindingBulkAssigneeSharedVectors -v` 17/17 PASS, `npm --prefix web test` 104/0/0, 노드·파이썬으로 벡터를 서버 규칙과 대조.
- **거절 사유(핵심)**: 폼이 보내는 값은 `assignee.trim()` 이므로 서버 `len(name)>200` 이 재는 문자열도 trim 된 값이다 → **수정 전 검사가 이미 서버와 동일**했고, 원문 기준 검사는 서버보다 엄격해져 오탐만 늘린다. `김보안`+공백200개가 이제 "200바이트까지" 오류로 막힌다(전에는 정상 처리).
- 수리가 가장 먼저 볼 파일: `internal/app/testdata/finding-bulk-assignee.json:29-45`(두 공백 패딩 사례 기대값이 틀렸다) → `web/src/finding-bulk-state.ts:26`. 커밋 전체 revert 가 가장 단순하다.
- 못 본 것: 실제 브라우저에서 눌러 보지 않았고 `HUNTER_TEST_DSN` 기반 Go 회귀도 돌리지 않았다(Go 프로덕션 무변경이라 영향 없음).
- 남는 우려(다음 회차): "실패 재현" 출력이 있어도 벡터 기대값이 틀리면 결함을 증명하지 못한다 — 공유 벡터는 **두 리더가 운영에서 먹는 문자열이 같은지**를 먼저 확인할 것. U+FEFF trim 불일치는 여전히 미검증.
- [러너 08:03] review rejected — 리뷰 거절: web/src/finding-bulk-state.ts:26-29 전제가 틀렸고 동작이 뒷걸음쳤다. 이 폼이 전송하는 값은 원문이 아니라 `patch.assignee = input.assignee.trim()`(web/src/fi

## 수리 노트
- 맞았던 지적: 폼이 보내는 값은 `input.assignee.trim()` 이라 서버가 재는 문자열도 trim 된 값이고, 따라서 수정 전 검사가 이미 서버와 동일했다 — 원문 기준으로 바꾼 것은 오탐 추가였다. 공유 벡터의 두 padded 사례도 실제로 서버가 수락하는 값을 `accepted:false` 로 잘못 고정했고, Go 테스트가 원문을 직접 넣어 폼 경로를 재현하지 않아 그 오류가 17 PASS 로 가려졌다는 지적도 맞다.
- 틀렸던 지적: 없음. `ledger-entry.md` 의 '실패 재현' 이 진짜 출력이면서도 증상을 증명하지 못한다는 지적까지 성립한다 — 실패는 벡터의 잘못된 기대값 때문이었다.
- 고친 방법: 길이 검사를 `.trim()` 기준으로 되돌리고, 벡터에 `wire`(= 요청 본문) 열을 추가해 TS 는 원문, Go 는 `wire` 를 먹인다. TS 가 `assignee.trim() === wire` 를, Go 가 `TrimSpace(wire) == wire` 를 단언해 두 trim 이 어긋나면 벡터가 깨진다.
- 여전히 확신 없는 곳: 앞뒤에만 붙은 탭/개행은 폼이 제어 문자로 거절하지만 서버는 trim 후 수락한다(클라이언트가 더 엄격). 의도적으로 벡터에 넣지 않았다 — 기존 동작이고 이번 범위를 넘어서므로 손대지 않았다. `U+0085`(Go 는 trim, JS 는 아님)와 `U+FEFF`(반대) 역시 벡터에서 제외한 채 note 로만 고정했다.
- [러너 08:13] repair done — # 수리 요약 (2026-09-28, 시도 1 거절 후)  비평이 맞았다. 이 폼이 POST 하는 값은 원문이 아니라 `patch.assignee = input.assignee.trim()` 이고, 서버 `finding_bulk.go:63` �
