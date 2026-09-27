- 과제: DataTable 에 description 이 있으면 검색 중 일치 건수가 어디에도 표시되지 않는 것을 고친다 (가치 2 / 위험 1 / 작업량 S)
- 왜: `web/src/components/DataTable.tsx:130-132` 의 캡션이 `description || \`${filtered.length}개 항목\`` 이라, description 을 넘긴 표는 검색어를 쳐도 몇 건이 걸렸는지 아무 곳에도 나오지 않는다(예: `FunnelPage.tsx:350-353` 의 「상위 이동 상세」는 `description` + `searchable` 을 함께 주므로 검색 중 건수를 볼 수 없다). 게다가 일치가 10건 이하로 줄면 `DataTable.tsx:212` 의 `filtered.length > 10` 때문에 TablePagination 의 `1–N / N` 까지 사라져 규모를 알 길이 완전히 없어진다. 캡션 계산을 순수 함수 한 곳으로 옮겨 검색 중에는 `전체 N개 중 M개 일치` 를 항상 보이게 한다.
- 수용 기준:
  1) 검색어가 있으면 캡션이 일치 건수와 전체 건수를 보여준다 — description 이 있는 표는 `<description> · 전체 1,240개 중 3개 일치`, 없는 표는 `전체 1,240개 중 3개 일치`. 숫자는 지금과 같이 `Intl.NumberFormat("ko-KR")` 로 구분자를 넣는다.
  2) 검색어가 비어 있을 때의 출력은 **지금과 완전히 같다** — description 이 있으면 description 그대로, 없으면 `${total}개 항목`. 즉 검색하지 않는 모든 화면의 캡션은 한 글자도 바뀌지 않는다.
  3) 테스트가 증명할 것: 네 조합(description 유무 × 검색 유무)의 출력, 0건 일치(`전체 N개 중 0개 일치`), 전체가 다 일치(M===N), 그리고 1,000 이상에서 `1,240` 처럼 구분자가 붙는 것. 순수 함수를 삭제/상수 반환으로 되돌리면 이 테스트가 실패해야 한다(구현자가 돌연변이로 확인할 것).
- 건드릴 파일 (프로덕션 2개):
  - `web/src/components/tableSummary.ts` (신규) — `tableCaption({ description, total, matched, searching })` 하나만 export. `total` 은 `rows.length`, `matched` 는 `filtered.length`. 기존 `tablePaging.ts` 와 같은 모양(맨 위 한국어 주석으로 왜 이 계산이 한 곳에 있는지 설명, 함수 하나, 외부 의존 없음).
  - `web/src/components/DataTable.tsx` — 21-22행 옆에 import 를 추가하고, 130-132행의 `{description || \`...개 항목\`}` 을 `{tableCaption({ description, total: rows.length, matched: filtered.length, searching: !!query.trim() })}` 로 바꾼다. **이 한 곳만** 바꿀 것.
  - `web/test/tableSummary.test.mjs` (신규) — `import { tableCaption } from "../src/components/tableSummary.ts";` **`.ts` 확장자를 반드시 적을 것**(node:test 가 확장자 없는 import 를 못 찾는다 — 2026-09-20 회차에서 실제로 걸렸고 `tablePaging.test.mjs:3` 이 선례). `node:test` + `node:assert/strict`.
- 검증 명령: `cd web && npm ci && npm run lint && npm test && npm run build`
  - **worktree 에 `web/node_modules` 가 없다**(이번 정찰에서 확인). `npm ci` 가 반드시 선행한다. `npm test` = `node --test test/*.test.mjs`(현재 121건 통과 → 추가분만큼 증가), `npm run build` = `tsc -b && vite build`.
  - `npx prettier --check` 는 게이트로 쓰지 말 것 — 저장소에 prettier 의존성·CI 단계가 없고 main 의 20여 파일이 이미 실패한다.
  - Go 쪽은 무관하므로 돌릴 필요 없다(서버 변경 없음).
- 위험과 피할 것:
  - `DataTable.tsx` 의 `filtered` useMemo(78-86), `useEffect`(87), `clampPage`/`safePage`/`slice`(90-94), `rowKey`(100-111), Empty 분기(201-210), `filtered.length > 10` 페이저 조건(212)은 **건드리지 말 것**. 212행을 "검색 중엔 항상 페이저" 로 바꾸는 것은 별 과제다 — 2026-09-27 회차가 방금 고친 자리라 같이 손대면 그 수정의 검증이 흐려진다.
  - `searching` 판정은 `query.trim()` 기준이어야 한다(공백만 입력하면 `filtered`(79-80행)가 전체를 그대로 돌려주므로, trim 하지 않으면 "전체 N개 중 N개 일치" 가 뜬다).
  - `matched`/`total` 을 뒤집어 넘기지 말 것: 검색 중 `rows.length` 가 전체, `filtered.length` 가 일치다. 검색어가 없으면 `filtered === rows` 이므로 기준 2)가 자동으로 성립한다.
  - 보호 경로(internal/auth, migrations, .github/workflows, docs/openapi.yaml) 전부 무관 — 열지 말 것.
  - 미확인: description 이 긴 표(예: `VisitorInsightsPage.tsx:764`)에서 캡션이 한 줄을 넘겼을 때의 줄바꿈 모양은 브라우저로 확인하지 않았다. 캡션은 `minWidth: 0, flex: 1` Box 안의 `Typography variant="caption"` 이므로 감싸질 것으로 보이나, 실제 DOM 확인은 구현자가 할 것(이 저장소는 vite dev + headless Chrome 으로 실제 컴포넌트를 마운트해 확인한 선례가 있다).
- 작업량 근거(45분 세션 기준): 순수 모듈 ~25행 + 테스트 5~7건 + DataTable 한 줄 배선 = 25~35분, 여기에 `npm ci` 콜드 설치와 `vite build` 가 붙어 실측 상한 45분. 10회 중 8회는 이 범위 안에 든다고 본다. 실제 DOM 확인까지 하면 +10분(예비).
- 차선 후보: 자기 자신의 역할 변경(SELF_ROLE)·비밀번호(SELF_PASSWORD) 제약을 사용자 편집 다이얼로그에 반영 — 서버는 `internal/httpapi/admin.go` 의 updateUser 에서 400 SELF_ROLE / 400 SELF_PASSWORD 를 돌려주는데 화면은 저장 뒤에야 영문 오류를 보여준다. 2026-09-26 회차가 만든 `web/src/pages/roleScope.ts` 옆에 순수 함수로 얹는다. (이번 정찰에서 admin.go 행 번호와 AdminPage 다이얼로그 위치를 재확인하지 않았으므로 착수 시 먼저 열어 볼 것.)
