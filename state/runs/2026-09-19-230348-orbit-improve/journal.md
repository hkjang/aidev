# 회차 노트 2026-09-19-230348-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:03] base pinned — main@6a013f9
- [러너 23:03] autonomy release — 

## 정찰 노트
- 고른 이유: 보류 목록의 상위 후보 대부분(CI postgres·OIDC DB 테스트·MCP 연결 확인)이 미머지 auto/2026-09-18-1033 의 newDBServer/fakeIDP 에 기대는데 main(6a013f9)에는 없어 지금은 성립하지 않는다. main 위에서 바로 성립하고 서버·auth 를 안 건드리는 Time Travel 슬라이더(칸마다 두 요청 + 응답 역전 시 화면 어긋남)를 골랐다. orbitRange SQL 은 DB 테스트가 없어 검증이 정적 검토뿐이라 차선으로.
- 확신 없는 곳: MUI Slider 가 키보드 입력에서 onChange 와 onChangeCommitted 를 같이 부른다는 것과 jsdom 에서 mouseDown/mouseUp 드래그 테스트가 되는지는 기억에 의존(미확인 — 워크트리에 node_modules 가 없어 vitest 를 못 돌렸다). 응답 역전은 코드 구조상 가능하다고 판단했고 실측은 안 했다.
- 구현자 주의: `npm ci` 먼저. "현재로" 단추가 부모 onChange(undefined) 를 직접 부르므로 draft 초기화를 빠뜨리면 슬라이더가 옛 칸에 남는다. /rediscover 분리는 이번 범위 밖(아이디어로 남김). Go 쪽은 손대지 않는다 — `go test ./...` 는 정찰 시점에 전부 ok.
- [러너 23:07] scout done — Time Travel 슬라이더 — 드래그 중에는 화면만 바꾸고 요청은 놓을 때 한 번, 늦게 온 응답은 버린다 (가치 3 

## 구현 노트
- 바꾼 것: `TimeTravel` 을 `web/src/components/TimeTravel.tsx` 로 옮기고 끄는 동안은 `draft` 로 날짜 표시만, 손을 뗄 때(`onChangeCommitted`) 한 번만 부모 `onChange`; `web/src/timeTravel.ts` 의 `dayToTravelValue`·`createLatestGuard` 를 두고 `OrbitPage.load` 가 순번을 매겨 최신 응답만 상태에 반영. 커밋 dd7207f (fix(orbit)).
- 검증: `vitest --run` 109 통과(신규 14: 순수 9 + 컴포넌트 5, 옛 코드로 먼저 돌려 드래그 중 2회 호출로 빨갛게 됨을 확인), `tsc -b && vite build`, `go test ./...` 통과.
- 확신 없는 곳: (1) `createLatestGuard` 가 `OrbitPage.load` 에 붙은 자리는 tsc 와 눈으로만 봤다 — 페이지 수준 테스트(api 모킹) 는 없다. (2) 실제 브라우저에서 드래그·"현재로"·뒤로 가기를 눌러 보지는 못했다(서버·DB 없음). (3) 컴포넌트 테스트는 MUI 9 내부(`pointerdown` → document `pointermove(buttons:1)`/`pointerup`, `lastChangedValueRef`)에 기대므로 MUI 를 올리면 깨질 수 있다 — 스텁(`getBoundingClientRect`·`set/has/releasePointerCapture`)이 왜 필요한지 테스트 주석에 적어 두었다.
- 일부러 하지 않은 것: `/rediscover` 를 `load` 에서 분리하지 않음(로딩·오류 처리 묶음, 과제서 지시). 키보드 방향키는 MUI 가 칸마다 `onChangeCommitted` 도 불러 여전히 칸마다 요청 — 과제서대로 회귀 아님. 서버·Go 는 손대지 않음.
- 다음 역할 주의: 워크트리에 `web/node_modules` 는 `npm ci` 로 설치했고 `web/dist` 는 빌드로 생겼다 — 둘 다 .gitignore 라 커밋에 안 들어갔다. 슬라이더 커밋 값은 놓는 좌표가 아니라 마지막 `onChange` 값이다(테스트 이름도 그렇게 적음).
- [러너 23:13] brief accepted — 채택 — 근거(칸마다 `setTravelTo`→`load` 재실행으로 두 요청, 응답 역전 가능)가 코드와 일치했고 선택 항목(컴포넌트 분리
- [러너 23:14] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: diff 5파일 전부 읽음. 새 컴포넌트 테스트를 main 의 옛 TimeTravel 임시 사본에 돌려 3/5 가 빨갛게 됨(onChange 2회)을 실측 — 테스트가 변경을 진짜 잡는다. 새 코드 vitest 109/109, tsc -b 통과. 워크트리 clean 으로 되돌림.
- 로직: draft 0(`??`)·부모 값 초기화 시 draft 비움·순번 지킴이가 성공/실패 양쪽 stale 응답을 버림 — 모두 코드와 테스트로 확인. 서버·auth·마이그레이션 손대지 않음 → 보안·법무 차단 없음.
- 못 본 것: 실제 브라우저 드래그·뒤로 가기(서버 없음). OrbitPage.load 에 붙은 지킴이는 페이지 수준 테스트가 없어 정적 검토만.
- 릴리즈 노트·다음 회차: 키보드 방향키는 여전히 칸마다 요청(MUI 가 칸마다 onChangeCommitted 호출) — 회귀는 아니나 "요청 한 번" 이라는 문구는 포인터 드래그에 한정해 쓸 것. 컴포넌트 테스트는 MUI 9 내부 포인터 처리에 기대 MUI 업그레이드 시 먼저 깨질 자리.
- 판정: approve, risk low.
- [러너 23:16] review approved — 리뷰 승인 (risk=low)
- [러너 23:16] pr created — https://github.com/hkjang/orbit/pull/7
- [러너 23:18] ci passed — 검사 1개 모두 success
- [러너 23:18] merge done — dd7207f
