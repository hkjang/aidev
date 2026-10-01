# 회차 노트 2026-10-01-114227-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:42] base pinned — main@12ba629
- [러너 11:42] autonomy release — 

## 정찰 노트
- 고른 이유: 어제 머지된 97e5ad5(열 숨김)가 남긴 **확정 가능한** 결함이다 — 두 화면의 다른 표가 같은 `exportFilename="momento-metric-goals"` 를 써 숨긴 열 설정이 섞인다(겹치는 키 `name`·`target_value`·`period` 를 열 목록 전체로 확인). 프로덕션 1파일(tablePrefs.ts)로 끝나고 서버·보호 경로를 전혀 건드리지 않는다.
- 제친 후보: 「숨긴 열도 검색 대상」(차선)은 의도일 여지가 있어 취향 변경으로 판정될 위험이 있고, CSV format·rowKey 중복은 3회차째 관측 증거가 없다. DataTable.tsx 본문을 또 고치는 과제는 피했다(최근 3회차 연속 손댄 파일).
- 추측으로 적은 것: FNV-1a 해시는 제안일 뿐 구현 방식은 구현자 재량. 두 「metric goals」 표가 한 사용자에게 동시에 보이는지(라우팅·권한)는 열지 않았다 — 키 충돌 자체는 코드로 확정이다.
- 구현자가 조심할 것: `web/test/tablePrefs.test.mjs:12` 가 저장 키 문자열을 글자 그대로 단언하므로 반드시 함께 고친다. 옛 키 마이그레이션은 하지 말 것(기능이 하루 전 것이라 한 번 초기화되는 편이 싸다). `DENSITY_STORAGE_KEY` 공유는 의도이니 건드리지 말 것.
- 실행하지 않음: `web/node_modules` 가 없어 lint·test·build 를 돌리지 않았다. 구현자가 `npm ci` 로 기준선 녹색을 먼저 확인할 것.
- [러너 11:48] scout done — 두 화면의 서로 다른 표가 같은 「숨긴 열」 설정을 공유하는 것을 고친다 — `tableStorageKey` 가 열 구성을 �

## 구현 노트
- `tablePrefs.ts` 의 `tableStorageKey` 가 `exportFilename || title || columnKeys.join(",")` 하나로 키를 정해 내보내기 이름이 같은 서로 다른 두 표(두 화면의 「metric goals」)가 숨긴 열을 나눠 썼다. 식별자 + 열 키 목록의 FNV-1a 32비트 요약을 키에 넣어 갈랐다. 프로덕션 1파일, `DataTable` 시그니처·`DENSITY_STORAGE_KEY` 불변.
- **확신 없는 곳**: (1) `npm test` 가 **기준선 main@12ba629 부터** 이 환경의 Node v22.23.1 에서 38개 중 30개가 `ERR_UNKNOWN_FILE_EXTENSION ".ts"` 로 깨진다 — 내 변경과 무관함은 기준선 재현으로 확인했고 `--test-concurrency=1` 로 178/178 녹색이지만, **저장소의 기본 테스트 명령 그대로는 이 머신에서 빨갛다**. CI(Node 24)에서는 통과해 왔다고 프로필에 적혀 있으나 이번에 CI 를 돌려보지는 못했다. (2) 요약이 32비트라 이론상 충돌이 가능하다 — 표 수가 62곳 규모라 실질 위험은 낮다고 판단했을 뿐 계산해 보지는 않았다.
- **일부러 하지 않은 것**: 옛 키 마이그레이션(과제서 지시 — 기능이 하루 전 것이라 한 번 초기화되는 편이 싸다; 커밋 메시지에 적었다). `DENSITY_STORAGE_KEY` 공유(의도된 것). 식별자가 없고 열 구성까지 같은 두 표의 잔존 충돌(화면 경로를 넣으려면 `DataTable` 에 라우터 의존이 생긴다). `npm test` 스크립트 수정(빌드·CI 경로라 범위 밖 — ideas.json 에 후보로 남겼다).
- **다음 역할이 조심할 것**: 테스트를 돌릴 때 `node --test --test-concurrency=1 test/*.test.mjs` 를 쓸 것 — 기본 `npm test` 는 선재 환경 문제로 빨갛고, 그 빨강을 내 변경 탓으로 읽지 말 것. 브라우저 확인에 쓴 `web/verify-tmp` 와 `dist` 는 지웠고 커밋에는 2파일만 들어 있다. Go 쪽은 이번 회차에 전혀 돌리지 않았다(변경도 없다).
- [러너 11:56] brief accepted — 채택 — 인용한 행 번호(tablePrefs.ts:10-17, EnterpriseAnalyticsPage.tsx:894, EnterpriseAdminPage.tsx:468, DataTable.tsx:144-146, tablePrefs.test.mjs:12
- [러너 11:56] verify failed — 실패한 검증: cd web && npm test --silent (exit 1)
