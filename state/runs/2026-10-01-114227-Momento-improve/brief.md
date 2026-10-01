# 과제서 — 2026-10-01 Momento (base main@12ba629, v0.34.52)

- 과제: 두 화면의 서로 다른 표가 같은 「숨긴 열」 설정을 공유하는 것을 고친다 — `tableStorageKey` 가 열 구성을 키에 넣지 않는다 (가치 3 / 위험 1 / 작업량 S)

- 왜: 어제 머지된 97e5ad5 가 표마다 숨긴 열을 localStorage 에 기억하게 했는데, 키를 `exportFilename || title || columnKeys.join(",")` 하나로만 정한다(`web/src/components/tablePrefs.ts:10-17`). 실제로 **서로 다른 두 표가 같은 `exportFilename="momento-metric-goals"` 를 쓰고 있어**(`web/src/pages/EnterpriseAnalyticsPage.tsx:894`, `web/src/pages/EnterpriseAdminPage.tsx:468`) 한쪽에서 열을 숨기면 다른 화면의 다른 표에서도 그 열이 사라진다. 키에 열 구성을 함께 넣으면 구성이 다른 표는 다른 설정을 갖게 되고, 같은 표는 다음에도 같은 키를 얻어 기억이 유지된다.

- 수용 기준:
  1) `tableStorageKey` 가 같은 `exportFilename`(또는 같은 `title`)을 쓰더라도 **열 키 목록이 다르면 서로 다른 키**를 돌려준다. 위 두 표의 실제 인자로 호출해 키가 다름을 확인한다.
  2) 같은 입력(같은 제목·같은 내보내기 이름·같은 열 키 순서)에는 **항상 같은 키**를 돌려준다 — 호출할 때마다 달라지거나 세션마다 달라지면 설정이 매번 초기화된다. 난수·시각·`crypto` 비동기 API 를 쓰지 않는 순수 계산이어야 한다.
  3) 키는 길이가 유계여야 한다(열이 20개인 표도 짧은 키). 지금은 제목·내보내기 이름이 둘 다 없으면 `columnKeys.join(",")` 가 통째로 키가 된다.
  4) 테스트가 증명할 것: 위 1)·2)·3) 과, 기존 동작 보존 — `parseHidden` 이 모르는 열을 버리는 규칙, `toggleHidden` 의 마지막 열 보호는 그대로다. 그리고 **키 생성 함수를 "항상 같은 상수를 돌려주는" 스텁으로 바꾸면 1) 이 실패**해야 한다(테스트가 결함만 짚는지 확인).

- 건드릴 파일 (프로덕션 1개):
  - `web/src/components/tablePrefs.ts:tableStorageKey` — 식별자(`exportFilename || title`)와 **열 키 목록의 결정적 요약**을 함께 키에 넣는다. 예: FNV-1a 같은 짧은 순수 해시 함수를 같은 파일에 두고 `momento:table-columns:${id}:${digest(columnKeys)}` 형태로. 식별자가 전혀 없을 때도 `columnKeys.join(",")` 대신 요약을 쓴다(기준 3).
  - `web/test/tablePrefs.test.mjs:12` — **첫 번째 테스트가 현재 키 문자열을 글자 그대로 단언한다**(`"momento:table-columns:momento-pages"` 등). 반드시 함께 고쳐야 하며, 새 형식을 하드코딩하기보다 "두 호출이 같다/다르다" 로 바꾸는 편이 낫다. 여기에 위 두 표의 실제 열 키로 충돌 회귀 테스트를 더한다.
  - `web/src/components/DataTable.tsx` — **고칠 필요 없다**(144-146행이 이미 `tableStorageKey(title, exportFilename, columnKeys)` 로 열 키를 넘기고 있다). 함수 시그니처를 바꾸지 말 것.

- 검증 명령 (worktree 에 `web/node_modules` 가 없으므로 `npm ci` 가 선행):
  ```
  cd web && npm ci && npm run lint && npm test && npm run build
  ```
  (`npm test` = `node --test test/*.test.mjs`. `npm run build` = `tsc -b && vite build`.)
  순수 함수라 브라우저 확인은 필수가 아니지만, 할 수 있으면 `npm run build` 의 dist 를 띄워 `/enterprise-admin` 쪽 표에서 「Goal」 열을 숨긴 뒤 `/enterprise-analytics` 쪽 표에서 그 열이 그대로 보이는지 실제 DOM 으로 확인할 것(이 저장소는 최근 4회차 모두 headless Chrome 실제 배선 확인을 했다; `puppeteer-core` 는 저장소에 없고 `/tmp` 에 설치해 왔다).

- 위험과 피할 것:
  - 키 형식이 바뀌면 **어제 설정을 저장한 사용자의 숨긴 열이 한 번 초기화**된다. 기능이 하루 전(v0.34.52)에 나왔으므로 허용되지만, 마이그레이션을 하려 들지 말 것 — 옛 키를 읽어 옮기는 코드는 범위를 키우고 되돌리기 어렵다. 커밋 메시지에 초기화 사실을 적는다.
  - `DENSITY_STORAGE_KEY`(모든 표가 공유하는 밀도)는 **의도된 공유**다. 같이 바꾸지 말 것.
  - `parseHidden` 이 모르는 열을 버리기 때문에 현재 피해는 **두 표가 공유하는 열 키에 한정**된다 — 확인한 겹치는 키는 `name`, `target_value`, `period` 세 개다(`metric_name` vs `metric_label`, `value`, `comparator` 는 겹치지 않는다). 회귀 테스트와 커밋 메시지를 이 사실에 맞춰 쓸 것, 과장하지 말 것.
  - 보호 경로(`internal/auth`, `internal/httpapi`, `internal/database/migrations`, `.github/workflows`, `docs/openapi.yaml`)는 이 과제에서 전혀 닿지 않는다. 열지 말 것.
  - `npx prettier --check` 를 게이트로 쓰지 말 것 — 저장소에 prettier 의존성·스크립트·CI 단계가 없고 main 의 여러 파일이 이미 실패한다.
  - 식별자가 전혀 없는 두 표가 **열 구성까지 같으면** 여전히 키가 겹친다. 이것은 이번 범위 밖이다(화면 경로를 키에 넣으려면 `DataTable` 에 라우터 의존이 생긴다). 과제서에 남은 한계로 적고 고치려 들지 말 것.

- 차선 후보: **표 검색이 숨긴 열의 값까지 뒤져, 보이는 곳 어디에도 일치가 없는 행을 결과로 내놓는다** (가치 2 / 위험 2 / S). `DataTable.tsx:172-180` 의 `filtered` 가 `columns` 전체를 훑는데, 화면에 그리는 칸·강조(`Highlighted`, 405-429행)·CSV(`329행`의 `visibleColumns`)는 보이는 열만 쓴다. 열을 숨긴 뒤 그 열에만 있는 낱말로 검색하면 일치 표시가 하나도 없는 행이 나오고 캡션의 「N개 일치」도 보이는 것과 어긋난다. 고치려면 필터 술어를 순수 모듈로 빼고 `visibleColumns` 를 넘긴다. 위험 2 인 이유: "숨긴 열도 검색 대상" 이 의도일 수 있어 결함이 아니라 취향 변경으로 판정될 여지가 있다 — 1순위가 성립할 때는 고르지 말 것.

## 이번 정찰에서 실제로 연 파일
`web/src/components/DataTable.tsx`(전체), `web/src/components/tablePrefs.ts`(전체), `web/src/components/preference.ts`(전체), `web/src/components/csvExport.ts`(전체), `web/src/components/shortcuts.ts`(전체), `web/src/components/navPrefs.ts`(전체), `web/src/components/AppShell.tsx`(1070-1119), `web/src/pages/EnterpriseAnalyticsPage.tsx`(880-900), `web/src/pages/EnterpriseAdminPage.tsx`(455-475), `web/test/tablePrefs.test.mjs`(1-30), `git log -20`, `git show --stat` × 3.

## 미확인
- 테스트·린트·빌드를 **돌리지 않았다**(worktree 에 `web/node_modules` 가 없고 설치 비용이 정찰 예산을 넘는다). 구현자가 먼저 `npm ci` 로 기준선이 녹색인지 확인할 것.
- 두 「metric goals」 표가 같은 사용자에게 실제로 둘 다 보이는지(둘 다 `enterprise` 계열 화면이고 라우팅·권한은 열지 않았다)는 미확인. 키 충돌 자체는 코드로 확정이다.
- (해소됨) 두 표의 열 키 전체를 확인했다. 분석 쪽 `name metric_name value target_value period elapsed_percent progress_percent projected_value`, 관리 쪽 `name metric_label target_value comparator period environment active` — 겹치는 것은 정확히 `name`, `target_value`, `period` 셋이다.
