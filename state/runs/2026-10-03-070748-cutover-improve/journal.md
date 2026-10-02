# 회차 노트 2026-10-03-070748-cutover-improve — cutover
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:07] base pinned — main@9ed7ae8
- [러너 07:07] autonomy release — 

## 구현 노트
- `components/ActivityTree.tsx`: ✎(내용 편집)을 열 때 `setEditData({title: node.title, time: node.time})` 로 재시드. `editData` 가 마운트 때만 초기화되고 행은 `key={node.id}` 로 재사용되므로, 폴링으로 값이 바뀐 뒤 편집을 열면 옛 값이 보이고 그대로 저장하면 다른 관리자의 변경을 덮었다. 프로덕션 1파일 + e2e 1파일 + ADMIN_GUIDE 4.4 두 줄. 커밋 354f695.
- **확신 없는 곳**: ① 신규 e2e 는 외부 변경이 화면에 반영되는 것을 `expect(getByText(...)).toBeVisible()`(기본 5s) 로 기다린다 — 1초 SWR 폴링이 느려지면 이 대기에서 깨질 수 있다(저장 성공 판정에는 폴링을 쓰지 않았다). ② 입력란을 **열어 둔 뒤** 들어온 외부 변경은 여전히 내 입력이 덮는다(last-write-wins). 일부러 그대로 뒀고 ADMIN_GUIDE 4.4 에 그렇게 적었다.
- **일부러 하지 않은 것**: `docs/ADMIN_GUIDE.pdf` 재생성 — 생성 스크립트가 없고, PDF 는 18cfb20 이후 이미 두 회차(1ef8df3·255d9d7)의 md 변경과 어긋나 있어 관례를 따랐다. ESLint globalIgnores 추가도 하지 않았다(아래).
- **다음 역할이 조심할 것**: ① `npm run lint` 는 e2e 를 돌린 뒤 `playwright-report/trace/*.js` 때문에 257 errors 로 깨진다 — lint 전에 `rm -rf playwright-report test-results`(둘 다 .gitignore 에 있음). 09-29 의 globalIgnores 수정(0de2f1e)은 main 역사에 없어 반려된 것으로 보고 다시 올리지 않았다. ② `npm run test:unit` 은 글롭이 확장되지 않아 "Could not find …lib/**/*.test.ts" 를 찍고 **exit 0** 으로 끝난다 — 통과로 오해하지 말고 `node --test lib/*.test.ts lib/**/*.test.ts`(95 pass/0 fail)로 돌릴 것. ③ e2e 는 `PLAYWRIGHT_CHROMIUM_PATH=/usr/bin/google-chrome` 필요.
- [러너 07:16] verify passed — 검증 1개 통과 (policy)

## 비평 노트
- 확인한 것: 대상은 354f695 1커밋(프로덕션 `components/ActivityTree.tsx` +8줄, e2e 1파일, ADMIN_GUIDE 2줄)뿐이고 범위 이탈 없음. 재시드가 유일한 stale 경로를 덮는 것을 `app/admin/page.tsx:266`·`ActivityTree.tsx:54,131,169` 로 코드 확인 — 신규 e2e 2건은 수정 전 코드에서 반드시 실패하며 원장의 실패 출력도 그 증상과 일치한다. 재검증 직접 실행: tsc 0 / lint 0 / unit 95 pass / e2e **25 passed**, 끝나고 git status 깨끗.
- 못 본 것: 수정을 되돌려 실패를 내 손으로 재현하지는 않았다(읽기 전용 원칙 — 코드 읽기와 원장 출력으로 대체). PDF·캡처 이미지 내용도 열어 보지 않았다.
- 승인이어도 남는 우려: ① 입력란을 **열어 둔 뒤** 들어온 외부 변경은 여전히 내 title·time 으로 덮어쓴다(문서에 명시됨, 낙관적 잠금 없음 — 다음 회차 후보). ② 신규 e2e 는 1초 폴링 반영을 기본 5s expect 로 기다려 느린 러너에서 플래이크 여지. ③ `docs/ADMIN_GUIDE.pdf` 불일치가 세 회차째 누적 — 릴리즈 노트에 적고 재생성 수단을 별도 항목으로.
- 다음 역할 주의: `npm run test:unit` 은 글롭 미확장으로 **거짓 exit 0** 이다. `node --test lib/*.test.ts lib/**/*.test.ts` 를 쓰고, lint 전에 `rm -rf playwright-report test-results`.
- [러너 07:20] review approved — 리뷰 승인 (risk=low)
- [러너 07:20] pr created — https://github.com/hkjang/cutover/pull/11
- [러너 07:20] ci passed — 검사 없음 — 정책으로 허용
- [러너 07:20] merge done — 354f695
