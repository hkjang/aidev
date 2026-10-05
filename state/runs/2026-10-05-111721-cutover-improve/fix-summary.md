# 수리 요약 (시도 1)

- 지적 3건 모두 맞았다. `submitActivityChange` 가 200 응답 본문(새 `lastUpdated` 포함)을 버리고 `void mutate()` 만 해서, GET 이 500 인 동안 SWR 캐시의 `data.lastUpdated` 가 첫 스냅샷에 멈췄다 → 혼자 저장해도 2번째부터 409.
- 재현: 신규 e2e `읽기가 죽어 있어도 혼자 하는 연속 저장은 409 가 되지 않는다`(GET 500 주입, PUT 은 `route.fallback()` 으로 실서버) — 수정 전 저장2 가 `409`, 수정 후 `200`·값 저장 확인(라우팅 푼 뒤 실제 GET 으로 판정).
- 고친 것: ① `app/admin/page.tsx:95` → `const result = await res.json()` 후 `void mutate(result, { revalidate: false })`(243행 `onImported` 와 같은 패턴, 파싱 실패 시에만 기존 `mutate()`). ② `route.ts:70` 문구를 '다른 곳에서 상황판이 먼저 바뀌어 저장하지 않았습니다…'로 바꿔 남을 지목하지 않고 복구 경로(새로 고침)를 제시. ③ ADMIN_GUIDE 4.5 를 '내 저장은 스스로 막지 않는다 / 읽기 실패 중에는 목록이 자동 갱신되지 않으니 새로 고침' 으로 정정하고, 거의 동시 저장은 걸러지지 않는다는 한계를 명시.
- 검증(전부 이 트리에서 실행): e2e `33 passed`, `node --test lib/*.test.ts lib/mail/*.test.ts lib/tracking/*.test.ts` `# pass 95 / fail 0`, `npx tsc --noEmit` 통과, `npm run lint` 무출력, `npm run build` 성공.
- 남은 한계(변경 범위 밖, 비평 노트와 동일): read→validate→write 창이 겹치는 진짜 동시 쓰기는 여전히 `200, 200` 으로 뒤 쓰기가 앞을 덮는다 — 가이드 4.5 마지막 항목에 적어 두었다.
