# 회차 노트 2026-09-27-071157-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:12] base pinned — main@4f55998
- [러너 07:12] autonomy release — 

## 정찰 노트
- DataTable clamp 를 골랐다: 보류 목록 최상단이었고 이번에 코드로 재확인해 보니 "한 프레임 깜빡임"보다 나쁜 경우(rows.length 동일 + 검색어 유지 → useEffect 미발화 → 빠져나올 수 없는 빈 표)가 있어 가치를 2→3 으로 올렸다. 차선인 SELF_ROLE 은 지난 회차가 막 손댄 화면이라 파급이 좁고, 표는 <DataTable 사용처 65곳에 걸린다.
- 확실한 것: DataTable.tsx 231행 전체를 읽었고 87(raw slice)·103(rowKey fallback)·194(Empty 조건)·205(>10)·209-212(표시용 clamp)·86(useEffect 의존성)은 인용한 그대로다. 컴포넌트용 순수 모듈 선례(web/src/components/csvExport.ts + web/test/csvExport.test.mjs 가 ".ts" 확장자까지 적어 import)도 직접 확인했다.
- 추측으로 적은 것: 이 worktree 에서 puppeteer-core/헤드리스 Chrome 이 실제로 도는지 확인하지 않았다(web/node_modules 없음). 테스트·빌드는 한 번도 돌리지 않았으므로 main 기준 115 통과는 2026-09-26 기록에 의존한다. 차선 후보의 AdminPage.tsx 행 번호도 미확인이다.
- 구현자가 조심할 것: useEffect 의존성 배열을 "겸사겸사" 고치지 말 것(clamp 만으로 충분하고, rows 를 의존성에 넣으면 불필요한 페이지 리셋을 부른다). rowKey 중복 문제는 별건이니 103행은 page→safePage 치환만. prettier --check 는 게이트로 쓰지 말 것(저장소에 prettier 가 없고 main 이 이미 실패한다).
- [러너 07:14] scout done — DataTable 이 clamp 하지 않은 page 로 slice 해 행도 Empty 도 없는 빈 표를 보여주는 것을 고친다 (가치 3 / 위험 1 /

## 구현 노트
- `DataTable` 이 clamp 하지 않은 `page` 로 `filtered.slice` 해서 행도 Empty 안내문도 없는 빈 표가 나오던 것을 고쳤다. 새 순수 모듈 `web/src/components/tablePaging.ts` 의 `clampPage` 하나를 두고 slice·`TablePagination.page`·기본 rowKey fallback 이 모두 같은 `safePage` 를 읽게 했다(프로덕션 파일 2개).
- 확신 있는 것: 순수 함수 6건이 clamp 를 raw page 로 되돌리면 5건 실패한다(순회가 `page=1 pageSize=1 total=1 → 0행` 으로 잡는다). 배선도 대역 없이 확인했다 — vite dev 로 진짜 `DataTable` 을 real React/MUI 로 마운트하고 headless Chrome 으로 몰아, 수정 전 `{"rows":0,"empty":false,"pager":null}` / 수정 후 `{"rows":5,"first":"alpha-0"}`.
- 확신 없는 것: 브라우저로 확인한 것은 수용 기준 2(rows.length 동일·내용 변경)뿐이다. 기준 1의 "검색 직후 한 프레임" 깜빡임은 렌더 사이를 잡아야 해서 DOM 으로 재현하지 않았고 순수 함수와 코드 읽기로만 논증했다. `<DataTable` 사용처 65곳을 하나씩 열어 보지는 않았다(기본 rowKey 를 쓰는 표는 `${page}-…` → `${safePage}-…` 로 key 문자열이 바뀌지만 같은 렌더 안에서 일관되고, 페이지 이동 시 어차피 재마운트된다).
- 일부러 안 한 것: `86행 useEffect` 의존성 배열, `205행 filtered.length > 10`, `194행` Empty 분기, CSV(`149행`), rowKey fallback 우선순위(동명이인 중복은 별건)는 손대지 않았다. 과제서대로 clamp 한 조각만 담았다. Go 쪽·openapi 변경 없음.
- 다음 역할이 조심할 것: `npx prettier --check` 를 게이트로 쓰지 말 것(저장소에 prettier 가 없고 main 이 이미 실패한다). 검증은 `cd web && npm ci && npm run lint && npm test && npm run build` 이고 121 통과(main 115 + 6)다. 브라우저 확인에 쓴 임시 하네스 `web/verify-tmp` 와 `web/dist` 는 지웠고 커밋에는 3파일만 들어 있다(puppeteer-core 는 /tmp/dtverify 에만 설치, 저장소 의존성 변화 없음).
- [러너 07:19] brief accepted — 채택 — 인용한 행 번호(87·103·194·205·209-212·86)와 csvExport 선례(`.ts` 확장자까지 적는 node:test import)가 모두 현재 코드와 �
- [러너 07:20] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 대상은 42d76f1 한 커밋(3파일 +99/-6)뿐이고 Go·openapi·마이그레이션·의존성·워크플로 변경은 없다. DataTable.tsx 전체를 읽어 safePage 가 slice·TablePagination.page·기본 rowKey 세 곳에서 일관되게 쓰이는 것과 effect(86)·Empty(194)·`>10`(205)·CSV 가 그대로인 것을 확인했다. web/ 에서 npm test 121/121, eslint 무경고, tsc -b, npm run build 모두 통과.
- 테스트 강도: /tmp 사본에서 clampPage 본문을 `return page;` 로 되돌려 6건 중 5건 실패(`page=1 pageSize=1 total=1 → 0행`)를 재현했다 — 원장의 실패 재현 출력과 일치한다.
- 못 본 것: 브라우저 시나리오(구현자가 headless Chrome 으로 확인한 기준 2)와 기준 1의 "한 프레임 깜빡임" 은 직접 재현하지 않고 코드 읽기로만 논증했다. <DataTable 사용처 65곳도 개별 확인하지 않았으나 initialPageSize·getRowKey 를 넘기는 호출처는 grep 으로 0건임을 확인했다.
- 승인 후 남는 우려(다음 회차용): 배선을 지키는 테스트가 없어 DataTable 을 raw page 로 되돌려도 121건이 통과한다. state page 는 의도적으로 범위 밖에 남으므로 filtered 가 다시 커지면 예전 페이지로 되돌아간다(수정 전과 동일 동작, 회귀 아님). pageSize<=0 분기는 도달 불가능한 방어 코드.
- 판정: approve / risk low / blocking 없음(보안·법무 차단 사유 없음 — 인가·비밀값·개인정보 경로 무관, revert 는 3파일 되돌리기).
- [러너 07:23] review approved — 리뷰 승인 (risk=low)
- [러너 07:23] pr created — https://github.com/hkjang/Momento/pull/18
- [러너 07:29] ci passed — 검사 1개 모두 success
- [러너 07:29] merge done — 42d76f1
- [러너 07:38] release published — v0.34.46
- [러너 07:40] assets verified — v0.34.46 자산 2개 (이전 v0.34.45: 2)
