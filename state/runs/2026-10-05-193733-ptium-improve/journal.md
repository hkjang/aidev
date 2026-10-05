# 회차 노트 2026-10-05-193733-ptium-improve — ptium
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:37] base pinned — main@b3b4138
- [러너 19:37] autonomy release — 

## 정찰 노트
- 인용(`> 문장`)을 골랐다: 지난 두 회차가 1순위로 예고해 둔 자리이고, 수평선·머리말이 떨어진 뒤 마크다운 리더에 남은 유일한 "사용자에게 보이는 백슬래시"다. setext h2 는 `isThematicBreak` case 와 앞보기 순서를 함께 설계해야 해 S 가 아니고, USER_GUIDE 칸 분리(3/1/S)는 코드 결함이 아니라 차선으로 뒀다.
- 확신 없는 곳: 재현 문자열 `- \> 인용입니다.` 는 prose.go `default:` → writer.go:210 → docs.go:131 을 읽어 도출한 것이고 **실행하지 않았다**. 구현자는 red 테스트로 실제 출력을 먼저 확인할 것. 기준선 `go test ./internal/docs` 는 2.919s ok 로 실행 확인.
- 설계 권고의 핵심은 `handle` 을 `var handle func(...)` 로 바꿔 벗긴 줄을 재귀로 다시 읽는 것이다(인용 안의 `#`·`-`·`|` 가 공짜로 따라온다). `>` 를 실제로 떼었을 때만 재귀할 것 — 아니면 무한 루프다.
- 조심할 것: `escapeLine` 의 `>` 보호는 덱 DSL 표지 부제 때문에 **지우면 안 된다**(고치는 자리는 리더다). `- > 인용`·인용 안 setext 는 범위 밖으로 명시했다. 프로덕션 1파일 + 테스트 1파일 + 가이드 1줄.
- [러너 19:42] scout done — 마크다운/텍스트 업로드에서 인용(`> 문장`)의 `>` 를 슬라이드 글자에서 떼기 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- 인용 마커를 리더에서 뗐다: `handle` 을 `var handle func(line, next string) bool` 로 바꾸고, 새 `withoutQuoteMarker` 로 `>` 를 전부 벗긴 뒤 **같은 `handle` 을 재귀 호출**해 벗겨진 줄을 다시 읽는다 — 인용 안의 `#`·`-`·`|` 가 각자의 블록으로 따라온다. `escapeLine` 은 손대지 않았다.
- 과제서와 다른 한 곳: 분기를 `switch` 의 `case strings.HasPrefix(line, ">")` 로 두면 헬퍼의 `bool` 반환값을 아무도 읽지 않게 되므로, 같은 함수에서 `atxHeading` 이 이미 서 있는 자리(`switch` 앞의 `if rest, quoted := …; quoted`)로 올려 그 `bool` 이 재귀 가드가 되게 했다. `""` 는 인용이 아니고 `>` 선두는 다른 case 와 겹치지 않아 동작은 같다(전체 테스트 ok).
- `>` 뒤의 공백·탭은 CommonMark 의 "한 칸" 이 아니라 **전부** 떼었다 — 이 리더는 prose.go:281 에서 모든 줄을 TrimSpace 하므로 한 칸만 남기면 인용 줄에서만 들여쓰기가 뜻을 갖는다. 이유를 헬퍼 주석에 적었다. 비평가가 볼 곳은 여기다.
- 확신 없는 곳·검증 못 한 것: 실제 업로드·렌더링(PPTX/PDF)으로는 확인하지 않았고 `Read`→`document.Source` 문자열까지만 봤다. PTIUM_TEST_DSN 이 없어 DB 테스트는 Skip(정상), 웹 단계는 변경이 없어 건너뜀. `>` 가 섞인 docx·pdf 실파일 회귀는 돌리지 않았다(그 리더 코드와 `escapeLine` 은 한 글자도 안 바뀜).
- 일부러 하지 않은 것: `- > 인용`(목록 안 인용)은 여전히 `- \> 인용` — `withoutListMarker` 계약을 건드려야 해서 뺐다. 인용 안 setext(`> 제목`/`> ===`)도 뺐다(`next` 를 벗기면 앞보기와 `skip` 의 뜻이 어긋난다). 여러 줄 인용을 한 요점으로 합치지 않았다(`maximumPoints` 와의 상호작용). 전부 커밋 메시지와 주석에 적었다.
- 다음 역할이 조심할 것: 새 테스트 8개 중 2개(`TestAQuoteMarkerInsideASentenceIsNotAMarker`·`TestAQuoteInsideACodeBlockIsStillCode`)는 **고치기 전후 모두 green** 인 계약 고정용이다 — red 가 아니었던 것이 정상이다. 나머지 6개가 red 였다(출력은 원장에 있다). 커밋 5bf8a38, 프로덕션 1파일.
- [러너 19:48] brief accepted — 채택 — 재현 문자열 `- \> 인용입니다.`·수용 기준 7개·건드릴 파일 세 자리가 현재 코드와 정확히 맞았고, 권장한 `without
- [러너 19:49] verify passed — 검증 9개 통과 (auto)
