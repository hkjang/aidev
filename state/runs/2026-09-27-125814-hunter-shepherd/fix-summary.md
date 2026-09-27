# 수리 요약 — hunter PR #13 (커밋 201f9f3)

- 재현: `list-tools.tsx` 만 `origin/main` 으로 되돌리고 `npm --prefix web test` → **97 통과 / 0 실패**. 비평가 지적대로 이 PR 의 동작 변경에 회귀 보호가 0 이었다.
- 수정: `save()` 의 다섯 가드(이름 없음·8개 초과·이름 중복·값 500자 초과·snapshot 8192 초과)를 `saved-list-views.ts` 의 순수 `savedViewSaveError(prefs, title, params, filters, snapshot)` 로 뽑고, `save()` 는 그 반환값만 보고 `setError`·`focusName` 처리. 문구·판정 순서·포커스 동작은 무변경.
- 테스트: 가짜 view 객체 없이 프로덕션 함수를 직접 호출하는 6개 테스트(각 가드 + 판정 순서 + 수락 시 `readListPreferences` 왕복). **가드를 하나씩 지우면 매번 실패**(name/count/dup 각 2건, 500자 1건, 8192 2건)로 공허하지 않음을 확인.
- 주석·커밋 제목: 8192 를 "브라우저 저장소 한도" 라 한 설명을 **`readListPreferences` 의 읽기 검증 한도**로 고쳤다(localStorage 는 값당 8192자 제한이 없고 쿼터는 origin 단위). rebase·amend 금지라 이전 커밋 제목은 그대로이며, 새 커밋 제목이 정정된 표현을 담는다 — 릴리즈 노트는 새 커밋 제목을 쓸 것.
- 검증(실행 결과): `npm --prefix web test` **103 통과 / 0 실패 / 0 skip**, `npm run typecheck`(tsc --noEmit) 통과, `npm run build` 성공, `git diff --check` 깨끗, prettier 적용. Go 무변경이라 Go 스위트·`verify-pentagi` 는 돌리지 않았다. Node 22.23.1 에서 검증(프로젝트 요구 Node 26) — CI 에서 재확인 필요. 브라우저에서 실제 저장 버튼 클릭은 하지 않았다(web/tests 에 DOM 하네스 없음).
