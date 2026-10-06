# 회차 노트 2026-10-06-094748-ox-arena-improve — ox-arena
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:47] base pinned — main@f68a6fa
- [러너 09:47] autonomy release — 

## 정찰 노트
- 고른 이유: base 가 2026-10-05 와 동일해(vitest·ci.yml·Play.tsx 수정 셋 다 머지 전) 손댈 수 있는 파일이 좁았다. 그 안에서 Timer 의 `useNow(50)` 무조건 호출은 프로덕션 코드만 읽고도 증명되는 낭비이고, `derivePhase` 가 status!=='playing' 이면 now 를 안 읽으므로 동작 보존을 실측으로 보일 수 있어 1순위로 올렸다.
- 제친 후보: 코드 스플리팅(차선)은 이 세션에서 `npm i` 가 승인 거부로 막혀 절감량을 못 재서 가치를 3→2 로 내렸다 — 번들 대부분이 공용 의존성이고, 쪼개면 행사장 와이파이에서 청크 로드 실패라는 새 실패 경로가 생긴다. 탈락 분모 불일치(가치 4)는 SQL 보호 경로 + 로컬 검증 수단 없음으로 또 보류했다(3회차 연속 — 사람 검토가 붙는 회차가 필요하다).
- 확신 없는 곳: 기준선 빌드가 녹색인지 **미확인**(npm i 차단으로 이 세션은 빌드를 0회 돌렸다). 리렌더 절감량도 미측정 — 과제서에 수치로 주장하지 말라고 명시했다.
- 구현자가 조심할 것: 타이머 재시작 시 `setInterval` 전에 `setNow(serverNow())` 를 넣지 않으면 매 문제 시작 첫 50ms 에 엉뚱한 카운트다운 숫자가 스친다. 그리고 `Play.tsx` 는 열지 말 것(머지 대기 PR 경로) — 같은 200ms 루프가 있지만 이번 범위가 아니다.
- 프로필: 2026-10-02 판을 그대로 둠(4일 경과, 이번에 읽은 파일들과 어긋나는 내용 없음). 단 "검증 함정" 의 npm i 차단은 이번에도 재현됐다.
- [러너 09:53] scout done — 문제가 진행되지 않는 동안에도 계속 도는 Timer 의 20Hz 리렌더 루프 멈추기 (`useRoom.ts:useNow` + `Timer.tsx`) (가�

## 구현 노트
- 무엇/왜: `useNow` 를 `number | null` 로 넓혀 `null` 이면 인터벌을 안 걸고, `Timer`(50ms)·`Display`(200ms) 가 `status === 'playing'` 일 때만 간격을 요청한다. 재시작 시 `setInterval` 직전 `setNow(serverNow())` 로 낡은 `now` 를 즉시 동기화. 프로덕션 3파일 +8/-3, 커밋 8c829e5.
- **과제서가 "증명 수단 없음" 이라던 수용기준 1·2 를 실행 증거로 채웠다.** 이 샌드박스에 Python Playwright + chromium 이 이미 설치돼 있다(`PLAYWRIGHT_BROWSERS_PATH=/home/hkjang/.cache/ms-playwright` 필수 — HOME 이 임시 디렉터리로 바뀌어 있어 기본 경로는 틀린다). 실제 브라우저에서 프로덕션 `Timer` 를 `react-dom/client` 로 마운트하고 `window.setInterval` 을 감싸 살아 있는 50ms 인터벌을 직접 셌다: 수정 전 비-playing 4상태 전부 `live50=1`(렌더는 `null`) → 수정 후 전부 `0`, `playing` 만 `1`. `setNow` 한 줄만 지운 변이에서는 카운트다운 첫 프레임에 `'7'` 이 페인트됐다(되돌리면 `'3'`) — 그 줄이 실제로 일한다는 증명. 하네스는 `__probe__/` 에 두고 커밋 전 삭제했다(트리 클린 확인, 빌드 89 modules 가 기준선과 동일해 번들에 안 섞였다).
- 확신 없는 곳: ①절감량(리렌더 횟수·배터리·CPU)은 **측정하지 않았다** — "비-playing 에서 인터벌이 0개" 까지만 실측이다. ②`Display` 변경은 빌드와 `derivePhase` 실측(수용기준 4)으로만 뒷받침된다 — `Display` 자체를 브라우저에서 돌리려면 Supabase 가 필요해 `if (!room) return <spinner/>` 를 넘지 못했다. `room?.status` 옵셔널 체이닝이 그 구간을 덮는다는 것은 코드 읽기 근거다. ③마운트마다 `setNow` 로 리렌더가 1회 늘어난다(무해하다고 판단했으나 측정은 안 했다).
- 일부러 안 한 것: `Play.tsx` 는 열지 않았다(머지 대기 PR 경로) — 같은 `useNow(200)` 루프가 남아 있고 그 PR 정리 후 한 줄이면 된다. `package.json`/`package-lock.json`/`.github/workflows/**` 0줄 diff(확인). 따라서 상설 자동 테스트는 추가하지 않았다.
- 다음 역할이 조심할 것: 실측을 재현하려면 `PLAYWRIGHT_BROWSERS_PATH` 를 넘겨야 하고 `vite --port 5199` 가 떠 있어야 한다. 그리고 **기준선에도 있던 별개 결함을 발견했다**: 카운트다운 첫 프레임에 `Math.ceil` 반올림으로 한 칸 큰 숫자('4')가 스친다 — 이번 수정으로 우연히 가려졌을 뿐 `Timer.tsx:11` 의 식은 그대로다. ideas.json 에 별 항목으로 적었다.
- [러너 10:01] brief accepted — 채택 — 1순위를 지정된 3파일(선택 과제 `Display.tsx` 포함)로 그대로 구현하고 수용기준 1~6 을 모두 충족했다. 과제서가 "�
- [러너 10:02] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 확인한 것: `derivePhase`(useRoom.ts:35-45)가 non-playing 에서 `now` 를 전혀 읽지 않음 + `Status` 유니온 5개 전수 + `now` 소비자 4곳(Timer/Display/Admin/Play)이 모두 `derivePhase` 경유뿐임을 grep 으로 확인 → 게이트 술어 `status==='playing'` 이 정확한 집합이다. 시간/애니메이션에 의존하는 다른 코드가 부모 리렌더 루프에 업혀 있지 않은지도 전수 grep(`Date.now|rAF|setInterval|setTimeout`)으로 확인 — 없음. `npx tsc --noEmit` exit 0 재실행. deps/workflows/migrations 0줄 diff 확인. 트리 클린(프로브 삭제됨).
- 못 본 것: 브라우저 실측은 재현하지 않았다(프로브가 삭제돼 증거는 원장 텍스트뿐). Supabase 없이 Display/Play 의 실제 게임 흐름은 여전히 못 돌린다.
- 승인이어도 남는 우려: ①재시작 첫 렌더가 낡은 `now` 로 계산되고 `setNow` 가 페인트 전에 플러시된다는 React 스케줄링에 의존한다 — waiting 에 수 분 머문 뒤 첫 문제로 넘어가면 '600' 같은 큰 수가 스칠 여지가 남는다(미재현). 렌더 시점 `serverNow()` 바닥깔기가 근본 해결. ②`locked` 단계는 status 가 'playing' 이라 20Hz 루프가 계속 돈다 → **릴리즈 노트에는 "문제 진행 중이 아닐 때" 가 아니라 "waiting/ready/revealed/finished 구간" 으로 적을 것**.
- security/legal: 차단 없음. 클라이언트 렌더 타이밍 3파일 +8/-3, 인증·비밀값·RLS·마이그레이션·의존성·개인정보 표면 무접촉.
- [러너 10:05] review approved — 리뷰 승인 (risk=low)
- [러너 10:05] pr created — https://github.com/hkjang/ox-arena/pull/4
