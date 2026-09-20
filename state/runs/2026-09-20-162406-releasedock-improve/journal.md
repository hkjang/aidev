# 회차 노트 2026-09-20-162406-releasedock-improve — releasedock
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:24] base pinned — main@ed71528
- [러너 16:24] autonomy release — 

## 정찰 노트
- 선택: SSE 로그 중복 제거 O(n²) → 마지막 id 비교(순수 헬퍼). 세 회차 연속 "다음 1순위" 로 미뤄진 항목이고, 코드(`SimpleRunDetailPage.tsx:191`, `simple.go:1188/1202`)로 근거를 다시 확인했으며 백엔드·문서·보호 경로를 전혀 건드리지 않는 순수 웹 S 과제라 골랐다. 차선(DSN 미설정 경고)은 가치는 같지만 Makefile 만 바뀌고 테스트로 증명할 게 없어 뒤로 뒀다.
- 확신 없는 곳: 기존 `some` 도 재연결 중복을 이미 막고 있어 "실패 → 성공" 식 회귀 증명이 안 된다 — 과제서에 그대로 적었고, 변화의 증거는 같은 참조 반환(`toBe`)과 `some` 제거다. 이 워크트리에는 `node_modules` 가 없어 vitest 를 이번 정찰에서 돌리지 못했다(`npm ci` 필요). 실제 렌더 비용은 프레임당 `setLogs` 재렌더가 더 크며 별도 아이디어(rAF 배치, M)로 남겼다.
- 조심할 것: `lastIdRef.current` 갱신과 `useEffect` deps 를 옮기지 말 것(재연결 안전망). `make vet`(auto/2026-09-17-0853)·MCP OAuth(auto/2026-09-18-1213) 두 브랜치는 여전히 main 미머지 — 그 전제의 아이디어는 착수 불가. 프로필은 0일 전 것이고 v0.5.15→v0.5.16 외에 어긋난 곳이 없어 다시 쓰지 않았다.
- [러너 16:27] scout done — 진행 중 실행의 SSE 로그 중복 제거를 전체 배열 탐색에서 마지막 id 비교로 바꾸기 (가치 2 / 위험 1 / 작업�

## 구현 노트
- 바꾼 것: `SimpleRunDetailPage.tsx` 의 SSE `receive` 가 프레임마다 `current.some(...)` 으로 배열 전체를 훑던 것을 순수 헬퍼 `appendStreamedLine(current, line)` ("마지막 id 이하면 같은 참조 반환, 아니면 덧붙임, 빈 배열엔 항상 덧붙임") 로 교체(커밋 9c1929f, 소스 2개 파일만). 서버가 `ORDER BY id` 로만 보내고 재연결이 `Last-Event-ID`/`?after=` 로 이어지므로 이 한 줄 규칙으로 기존 `some` 과 같은 안전망을 O(1) 로 얻음.
- 검증: 새 테스트 4건을 먼저 넣어 `is not a function` 으로 실패(기존 12건 통과)를 본 뒤 구현 → `npm ci`, `npm test -- --run` 94건 통과, `npx tsc -b --noEmit` 통과, `grep current.some` 없음. 백엔드 `go test` 는 손대지 않아 돌리지 않았고 `npm run build`(vite 번들) 도 돌리지 않음(tsc 만).
- 확신 없는 곳: 과제서가 미리 밝혔듯 옛 `some` 도 재연결 중복을 막고 있어 "고치기 전 코드가 새 테스트에 실패" 하는 회귀 증명은 없음 — 변화의 증거는 `toBe` 같은 참조와 `some` 제거뿐. 실제 브라우저에서 진행 중 실행을 열어 스트림을 보진 못했음(이 환경에 DB·서버 없음). 순수 헬퍼 규칙만 검사했고 `receive` 배선 자체는 vitest 로 통과시키지 않음(EventSource 대역이 필요해 운영자 규칙상 피함) — 배선은 한 줄 교체라 눈으로 대조.
- 일부러 하지 않은 것: `lastIdRef.current = parsed.id` 갱신·`end` 처리·`useEffect` deps·자동 스크롤은 그대로(재연결 안전망). 프레임마다 `setLogs` 재렌더(rAF 배치)는 별도 M 아이디어로 ideas.json 에 둠. 문서·백엔드·VERSION 미변경.
- 다음 역할이 조심할 것: `appendStreamedLine` 이 같은 참조를 돌려주면 React 가 상태 갱신을 건너뛰어 자동 스크롤 effect 도 안 돎 — 의도된 동작(바뀐 게 없으니). 서버가 id 순서를 깨는 변경을 하면 이 규칙이 줄을 버리므로, `simple.go:1188` 의 `ORDER BY id` 를 바꾸는 사람은 이 헬퍼를 같이 봐야 함.
- [러너 16:30] brief accepted — 채택 — 과제서의 근거(`current.some` 의 O(n), 서버 `ORDER BY id`·프레임별 `id:`·`Last-Event-ID` 이어받기)가 코드와 그대로 일치했
- [러너 16:30] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 헬퍼의 전제(id 순서 전송·`max(Last-Event-ID, ?after)` 재개)를 `simple.go:1171-1188` 로 대조 — 한 연결 안 단조 증가, 재연결은 마지막 든 id 이상에서 시작하므로 "마지막 id 이하 = 중복" 규칙은 옛 `some` 과 같은 집합을 거른다. 이 워크트리에서 vitest 16건·`tsc -b --noEmit` 직접 통과 확인. 인증·백엔드·문서·VERSION 미변경.
- 못 본 것: 실제 브라우저 스트림(DB·서버 없음). 옛/새 동작이 갈리는 유일한 입력(마지막보다 작은 id 인데 배열에 없는 줄)은 테스트가 고정하지 않았으나 서버가 그런 프레임을 보낼 경로가 없어 거절 사유 아님.
- 남는 우려: 구현 노트대로 "수정 전 실패" 회귀 증명은 없다(옛 로직이 상위 집합 dedupe). 릴리즈 노트엔 perf 항목으로만 적으면 됨. 다음 회차가 `simple.go:1188` 의 `ORDER BY id`/커서를 손대면 이 헬퍼가 줄을 버리므로 함께 봐야 한다.
- 판정: approve, risk low, blocking 없음.
- [러너 16:32] review approved — 리뷰 승인 (risk=low)
- [러너 16:32] pr created — https://github.com/hkjang/releasedock/pull/20
- [러너 16:35] ci passed — 검사 1개 모두 success
- [러너 16:35] merge done — 9c1929f
- [러너 16:40] release published — v0.5.17
- [러너 16:42] assets verified — v0.5.17 자산 2개 (이전 v0.5.16: 2)
