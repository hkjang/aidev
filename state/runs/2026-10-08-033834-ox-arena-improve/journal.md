# 회차 노트 2026-10-08-033834-ox-arena-improve — ox-arena
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:38] base pinned — main@f68a6fa
- [러너 03:38] autonomy release — 

## 정찰 노트
- 과거 4회차 결과물이 전부 머지 전이어서 자유 경로가 `Admin.tsx`/`main.tsx`/`util.ts`/components 로 좁혀졌다. 그중 유일하게 아직 아무도 읽지 않은 큰 프로덕션 파일(`Admin.tsx` 305줄)을 전문 읽고, 실제 결함(제어 입력이 `Number('')===0` 으로 리셋 + DB check 제약 위반 메시지 영문 노출 + 서버 값 미반영)을 찾아 1순위로 골랐다. 반복해서 기각돼 온 유형(SQL 마이그레이션, 머지 대기 경로 재작업)은 피했다.
- 확신 없는 것: **이 샌드박스에서 `npm i` 가 승인 거부로 막혀 빌드를 한 번도 돌리지 못했다**(과제서에 미확인으로 표기). 기준선 녹색 여부는 구현자가 먼저 확인해야 한다.
- 코드에서 직접 확인한 사실만 과제서에 넣었다: `Admin.tsx:180` 의 `value={dur}`/`Number(e.target.value)`, `0001_init.sql:11` 의 `check (duration_sec between 3 and 60)`, `0003_elimination.sql:6-16` 의 clamp 없는 `coalesce`, `util.ts:13-29` 의 매핑 11건에 제약 위반 없음. 반면 "입력칸이 `015` 가 된다" 는 React 제어 입력 의미에서 도출한 것이고 브라우저로 실측하지는 않았다 — 구현자가 Playwright 하네스로 레드부터 찍을 것.
- 구현자가 조심할 것 ①: 서버 값 동기화를 `key` 리마운트로 하면 4초 폴링이 편집 중 입력을 날린다 → 원시값 의존 `useEffect` 필수. ②: `Admin.tsx:52` 의 `useNow(200)` 는 유혹적이지만 `useNow` 의 `null` 지원이 머지 대기 PR 에 있어 base 에서 컴파일되지 않는다.
- 프로필은 2026-10-02 판(5일 전)이 지금 코드와 어긋나지 않아 재작성하지 않았다.
- [러너 03:43] scout done — Admin 설정 폼(`RoomSettings`)의 "선택 시간(초)" 입력 깨짐과 서버 값 미반영 고치기 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- `src/pages/Admin.tsx` 의 `RoomSettings` 1파일(+20/-4, 커밋 7249b96). `dur` 을 문자열 state 로 바꿔 `Number('')===0` 리셋을 없애고, 이름 공백/선택시간 3~60 밖이면 저장 버튼을 비활성화해 DB check 제약 위반 값이 `admin_update_room` 으로 나갈 경로를 없앴다. 서버 값 동기화는 원시값 의존 `useEffect` 3개.
- 확신 없는 곳: ① `/^\d+$/` 가드라서 `015` 같은 선행 0 문자열은 통과한다(`Number('015')===15`, 범위 내이므로 안전하지만 입력칸에 그대로 보인다 — 실측상 지운 뒤 타이핑하면 생기지 않는다). ② `type="number"` 때문에 `'abc'` 를 키보드로 넣을 수 없어(브라우저가 `''` 로 만든다) 과제서가 말한 NaN→null 경로는 **빈 칸 경로와 같아지는 것으로 확인**했고 별도로 재현하지 못했다. 가드 자체는 둘 다 막는다. ③ 실제 Supabase 가 붙은 Admin 화면에서 끝까지 돌려 보지는 못했다(이 샌드박스에 DB 없음) — 검증은 `RoomSettings` 를 격리 마운트한 것까지다.
- 일부러 하지 않은 것: `errMsg` 매핑 추가(`Play.tsx:94` 가 매핑 부재에 의존 + 머지 대기 경로), `supabase/migrations/**` 의 clamp(보호 경로), `Admin.tsx:52` 의 `useNow(200)`(`useNow` 의 `null` 지원이 머지 대기 PR 에 있어 base 에서 컴파일 안 됨), `importFile` 검증(차선 후보로 `ideas.json` 에 남김).
- 다음 역할이 조심할 것: 상설 테스트는 **없다**(base 에 러너 없음, 과제서가 금지). 검증은 임시 Playwright 하네스였고 커밋 전 삭제했다 — 재현하려면 `__probe__/` 를 다시 만들고 `npx vite --port 5199` + `PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright python3 -I probe.py`(스크립트는 이 런 디렉터리에 `probe.py` 로 남겨 뒀다). 하네스는 `export function RoomSettings` 에 의존하므로 이 export 를 지우면 재현이 깨진다.
- [러너 03:48] brief accepted — 채택 — 1순위를 지정된 1파일로 지정된 방식(문자열 state + 저장 가드 + 원시값 의존 `useEffect`)대로 구현하고 수용 기준 1~5
- [러너 03:48] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 원장의 실패 재현을 **독립 검증**했다: `probe.py` 하네스를 직접 재구성해 HEAD 15/15 그린, base `RoomSettings` 로직은 13/15 레드(`p_duration=0` 실전송, `T3a/b/c` 미반영, `T5 '045'`) — 테스트가 바뀐 경로를 실제로 지난다. `npm run build` EXIT 0, 하네스 삭제 후 트리 clean 확인.
- 가드 경계를 12종 직접 실측: `3`/`60` 양끝 통과(DB `between` 과 일치), `015`→15·`0060`→60 정수 전송, `15.5`/`1e1`/`-15`/`00000000061`/거대수 모두 저장 비활성. 범위 밖·비정수가 `admin_update_room` 으로 나가는 경로를 찾지 못했다.
- 못 본 것: 실제 Supabase 가 붙은 Admin 전체 화면, 2기기 동시 편집 실환경, 붙여넣기 경로(합성 ClipboardEvent 가 안 먹어 불확정 — 단 정규식 가드가 문자열 형태 전부를 덮는다).
- 승인이어도 남는 우려 3개: ①상설 회귀 테스트 없음, CI 는 main push 빌드만 — 깨져도 못 잡는다(릴리즈 노트에 '일회성 하네스 검증' 명기할 것). ②`Admin.tsx:173` 의 `export` 는 저장소 내 소비자가 없다(하네스 미커밋) — 다음 회차가 쓰거나 되돌릴 것. ③`Admin.tsx:180` 주석의 "초기화" 는 부정확(`admin_reset` 은 이 3필드를 안 건드린다) — 무해.
- 보안·법무 차단 없음: 인증·인가·데이터 수집 무변경, 전송량은 오히려 감소, 새 의존성·비밀값 없음. 별건 참고로 `src/lib/supabase.ts:4-5` 하드코딩 publishable key 는 base 기존 사항이며 RLS 정책 단독 검토 회차 가치가 있다.
- [러너 03:53] review approved — 리뷰 승인 (risk=low)
- [러너 03:53] pr created — https://github.com/hkjang/ox-arena/pull/5
- [러너 03:58] ci no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)
