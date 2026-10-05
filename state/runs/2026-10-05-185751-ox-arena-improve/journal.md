# 회차 노트 2026-10-05-185751-ox-arena-improve — ox-arena
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:57] base pinned — main@f68a6fa
- [러너 18:57] autonomy release — 

## 정찰 노트
- base 가 2026-10-03 과 동일(main@f68a6fa): vitest PR·ci.yml PR 둘 다 **아직 머지 전**이라 package.json·src/lib/phase.ts·.github/workflows 는 전부 충돌 지뢰다. 그래서 인프라 계열을 모두 빼고, 충돌 없는 1파일 UX 결함(카운트다운 중 O/X 버튼이 활성처럼 보이지만 무반응)을 골랐다. 분모 불일치(가치 4)는 마이그레이션 = 보호 경로 + 로컬 SQL 검증 수단 없음이라 자율 회차에 부적합해 제쳤다.
- 확신 있는 것: canChoose/pick 가드 불일치(Play.tsx:79/88), 서버의 started_at-300ms 거부(0003:34), 3초 카운트다운(0001:15 주석 + Timer.tsx:10-12), `button:disabled{opacity:.4}`(styles.css:6) — 모두 파일을 열어 확인했다.
- **미확인(추측 아님, 미실행)**: 이 워크트리에 node_modules 가 없어 `npm ci`/`npm run build` 를 이번 세션에서 돌리지 못했다(2026-10-03 회차가 같은 base 에서 EXIT 0 실측). 구현자는 기준선 build 부터 확인하고 시작할 것.
- 구현자 주의: base 에 테스트 러너가 **없다** → 행동 증거는 build + 진리표 서술뿐이다. 러너를 새로 깔거나 phase.ts 를 새로 만들어 증거를 만들지 말 것(머지 대기 PR 과 충돌). `pick()` 가드를 푸는 방향으로 "고치면" 서버가 LOCKED 를 던져 지금보다 나빠진다.
- [러너 19:02] scout done — 3초 카운트다운 동안 O/X 버튼이 활성처럼 보이지만 눌러도 무반응인 것 고치기 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- `Play.tsx:79` canChoose 에서 `|| phase === 'countdown'` 만 제거(+주석 1줄) → `pick()` 의 `phase !== 'choosing'` 가드와 동일 조건. 커밋 0215227, 1파일 +2/-1. 카운트다운이 매 문제 실재하는 3초임을 `0001_init.sql:282`(`clock_timestamp() + interval '3 seconds'`, status='playing')로 확인했고, 실제 `derivePhase` 를 vite `ssrLoadModule` 로 불러 t+0/1.5/2.999s→countdown, t+3.0s→choosing, t+10.0s→locked 를 실측했다.
- **검증 못 한 것 — 여기부터 보세요**: 버튼 DOM 의 `disabled` 속성을 실제로 렌더해 확인하지 못했다. `canChoose` 는 export 되지 않는 `Game` 안의 지역 const 이고 `room` 이 `useRoom` 의 useEffect(Supabase)로만 채워져 `Play.tsx:75 if (!room) return <spinner/>` 에 막힌다 → SSR·jsdom 둘 다 버튼에 도달 못 하고, 도달시키려면 내 모듈을 mock 하거나 Supabase 를 손으로 흉내 내야 해서(운영자 금지) 안 했다. 전/후 진리표의 **버튼 조건 두 줄은 Play.tsx 에서 베껴 적은 것**이지 파일에서 읽은 것이 아니다 — 프로덕션 배선 증거는 `derivePhase` 까지다. 1줄 diff 가 곧 그 표현식이라 등가라고 판단했지만, 거기가 이 회차의 가장 약한 고리다.
- 일부러 안 한 것: ① "잠시 후 시작" 힌트(Timer 가 이미 3·2·1 을 72px 로 보여 줌, 불필요한 UI 추가) ② vitest/`src/lib/phase.ts` 추가(머지 대기 PR 과 경로 충돌 — 과제서 경로 b 를 따랐다) ③ `package.json`·`supabase/migrations/**`·`.github/workflows/**` 무접촉 ④ `errMsg` 의 LOCKED 매핑 — `Play.tsx:94` 분기가 매핑 부재에 의존해서 건드리면 조용히 깨진다.
- 다음 역할 주의: `Play.tsx:94` 의 `errMsg(e)==='LOCKED'` 분기는 **여전히 살아 있어야 한다** — 카운트다운은 막았지만 choosing 막판 제출이 `ends_at + 300ms` 를 넘으면 서버가 아직 LOCKED 를 던진다. 죽은 코드로 보고 지우지 말 것. 또 `Play.tsx:114` 는 그대로 countdown 에 문제 텍스트를 보여 준다(Display.tsx:34 와 일치) — 의도된 것이다.
- [러너 19:08] brief accepted — 채택 — 1순위를 과제서가 지정한 경로(b)대로 `Play.tsx` 1파일로 구현했고, 수용 기준 1~5 를 모두 충족했다(선택 사항인 "잠
- [러너 19:08] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 확인: diff 전체(1파일 +2/-1), Play.tsx 전문, useRoom.ts derivePhase, Timer.tsx, 0003_*.sql submit_answer+grant 목록, styles.css:6, git status(clean). `npm run build` EXIT 0.
- 구현자가 비워 둔 자리(버튼 DOM 배선)를 메웠다: Play.tsx:141-142 가 `disabled={!canChoose}` 임을 읽고, 번들에서 `S=!v&&d==="choosing"` + `disabled:!S` 를 확인 — countdown 이 프로덕션 산출물의 canChoose 에서 사라졌다. 기능 손실 없음(변경 전에도 pick() 가 막았다).
- 남는 우려(릴리즈 노트용): ① 주석은 카운트다운 전체를 서버가 거부한다 하지만 0003:35 는 started_at-300ms 부터 허용 — 동작 영향 없는 문구 오차. ② syncClock 실패/기기 시계 뒤처짐이면 선택 창 내내 countdown 에 갇히는 기존 결함이 이제 흐린 버튼으로 드러난다(회귀 아님, 다음 회차 후보).
- legal 참고(차단 아님): Play.tsx:54 "개인정보는 수집하지 않아요" vs 닉네임+성별 수집 — main 에 이미 있던 문구, 이 diff 무관. 새 수집 없어 notes.
- Play.tsx:95 LOCKED 분기는 살아 있다(지우지 말 것). 분모 불일치(rv.total vs counts.alive)는 미해결 상태로 유지.
- [러너 19:10] review approved — 리뷰 승인 (risk=low)
- [러너 19:10] pr created — https://github.com/hkjang/ox-arena/pull/3
- [러너 19:15] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
