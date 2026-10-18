# 회차 노트 2026-10-18-153949-Vendra-improve — Vendra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:39] base pinned — main@dfbe947
- [러너 15:39] autonomy release — 

## 정찰 노트
- 다섯 회차 연속 차선이던 `loadViews` 를 1순위로 올렸다: 서버가 `productivity.go:463/:475` 에서 문장으로 말한 실패를 화면이 버리고 「보기 없음」과 똑같이 그리는 것을 직접 확인했고, 바로 옆 `load`(28572eb)가 베낄 본보기를 같은 파일에 이미 갖고 있어 S 로 끝난다. 「가려진 지출의 남은 절반」·「LIMIT 100」은 응답 모양/계약 정책 결정이 선행돼야 해 한 세션에 안 들어간다.
- 추측으로 적지 않은 것: 서버 메시지·좌표·하네스는 전부 파일을 열어 확인했고, `011c5a3` 이 HEAD 조상이 아님은 `git merge-base` 로 확인했다. **확인하지 않은 것**: 이번 정찰은 웹·Go 테스트를 한 번도 돌리지 않았다(이 워크트리에 `web/node_modules` 없음). 기준선 「25파일/116개」는 2026-10-09 기록을 인용한 값이지 이번 실측이 아니다.
- 구현자가 조심할 것: ① 기준 2 의 핵심(저장 성공 → 새로고침 실패)은 `post` 가 모듈 내부 `api` 를 쓰므로 `vi.mock("../api")` 로는 못 잡는다 — `objects-draft-submit.test.tsx` 의 fetch 경계 하네스를 쓸 것. ② 조회 실패를 기존 `viewError` 에 섞지 말 것(`saveView` 가 매번 비우고 `!viewModal` 에서만 렌더된다). ③ `NewObject` 는 미병합 011c5a3 과 겹치니 건드리지 말 것.
- 프로필은 다시 쓰지 않았다(1일 전, 내용 일치). `현재 기준` 줄만 낡았다 — base 는 이제 **dfbe947 / v0.7.73**, 작업 트리 깨끗.
- [러너 15:43] scout done — 저장된 보기 조회가 실패하면 목록을 비우지 말고 **서버가 말한 이유**를 말하고 다시 시도할 수 있게 하�

## 구현 노트
- `loadViews` 의 `.catch(() => setViews([]))` 를 새 `viewsFailure` state + `saved-views-load-failed` 배너(「다시 시도」가 `loadViews()` 재호출)로 바꿨다. 실패 시 `setViews`·`setCanShareViews` 를 건드리지 않으므로 저장 성공 뒤 거부된 새로고침이 방금 저장된 목록을 지우지 않는다. commit 76e7154.
- 확신 없는 곳: ① 배너를 `.saved-view-tools` 밖(`viewError` 배너 바로 위)에 둔 것은 과제서 지시대로지만 **실제 브라우저로 보지 않았다** — jsdom 만. `work-inbox.css` 한 줄(`.saved-views-load-failed>span{flex:1}` + 버튼 높이)이 레이아웃용 추측이다. ② 서버가 401 을 본문 없이 보낼 때의 문구는 `api()` 의 `요청 실패 (401)` 이고 그걸 그대로 보여 준다 — 세션 만료 전용 안내는 넣지 않았다(이 화면의 다른 경로도 그렇게 한다).
- 일부러 하지 않은 것: 공급업체 드롭다운의 빈 catch(:261-265, 같은 컴포넌트 — 과제서가 묶지 말라고 했다), `saveView` 의 `filters.order`, `NewObject`(미병합 011c5a3), Go 전부(`productivity.go` 는 읽기만 — 고칠 것 없음). Go 테스트는 돌리지 않았다(웹 전용 변경).
- 다음 역할이 조심할 것: 신규 테스트는 `fetch` 만 `vi.stubGlobal` 로 대체하고 실제 `api`/`post`/`del` 을 통과한다 — DB·네트워크 불필요. 이 워크트리의 웹 기준선은 **113개**이고(과제서가 인용한 116 은 HEAD 조상이 아닌 011c5a3 의 3개를 포함) 변경 후 **26파일/117개**다. 재시도 버튼의 접근성 이름은 목록 실패 배너와 겹치지 않게 `저장된 보기 다시 시도` 로 두었다 — 두 배너가 함께 뜨는 경우가 있다.
- [러너 15:50] brief accepted — 채택 — 근거와 좌표가 코드와 그대로 맞았고(`:274` 의 `.catch(() => setViews([]))`, `:315`·`:326` 의 `await loadViews()`, `viewError` 의 `:
- [러너 15:50] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 직접 실행해 확인: 신규 4개를 **원래 동작(`.catch(() => setViews([]))` + `.then` 의 초기화 제거)으로 되돌린 채** 돌려 4개 전부 빨강을 재현했다 — 원장이 적은 출력과 글자까지 같다(`expected [ '기본 보기' ] to include '지난달 만료 계약'` 1개, `expected null to be truthy` 3개). 되돌리기는 `loadViews` 안 두 블록만 건드려 파일이 계속 컴파일되게 했고 `git checkout --` 로 원복했다. 복구 후 전체 `npx vitest run` **26파일/117개 통과**, `tsc -b --noEmit`·`eslint src --max-warnings 0` exit 0.
- 구현자가 의심한 CSS 는 추측이 아니라 확인 가능했다: `styles.css:851` 의 `.form-error` 가 `display:flex;align-items:center;gap:8px` 이고 `:863` 이 `svg{flex:none}` 이라 아이콘·`<span flex:1>`·버튼 한 줄이 성립한다. 다만 버튼에 `flex:none` 이 없어 **좁은 폭에서 긴 메시지가 버튼을 눌러 찌그러질 수 있다**(미관, 실제 브라우저로는 나도 보지 않았다).
- 보안·법무 차단 없음: 인가·스코프·식별자·의존성 변화 0, 노출되는 문구는 `productivity.go:463/:475/:485` 의 고정 문장(원시 DB 오류 아님)이고 개인정보 신규 수집·전송 없음. 유사 배너 선례(28572eb)도 docs 를 고치지 않았고 USER_GUIDE 에 「저장된 보기」 언급이 없어 문서 동기화 의무도 없다.
- 승인이어도 남는 우려(다음 회차용): ① `loadViews` 에는 `load` 의 `loadSequence` 가드가 없다 — 「다시 시도」 연타나 업무 종류 전환 중 **먼저 보낸 요청의 거부가 뒤늦게 도착해 이미 성공한 화면에 배너를 되살릴 수 있다**(`setViews` 의 경합은 변경 전에도 있었고 이번에 경로가 하나 늘었다). ② 재시도 버튼에 in-flight 비활성화가 없다. ③ `removeView` 뒤 새로고침이 거부되면 **방금 삭제한 보기가 드롭다운에 남는다**(전에는 비워져 사라졌다) — 배너가 상태를 알려 주므로 수용 가능하다고 봤다.
- 못 본 것: Go·DB 통합 테스트는 돌리지 않았다(이 diff 에 Go 변경 0줄). 실제 브라우저 렌더링도 보지 않았다.
- [러너 15:54] review approved — 리뷰 승인 (risk=low)
- [러너 15:55] pr created — https://github.com/hkjang/Vendra/pull/147
- [러너 15:57] ci passed — 검사 2개 모두 success
- [러너 15:57] merge done — 76e7154
- [러너 16:00] release published — v0.7.74
- [러너 16:01] assets verified — v0.7.74 자산 1개 (이전 v0.7.73: 1)
