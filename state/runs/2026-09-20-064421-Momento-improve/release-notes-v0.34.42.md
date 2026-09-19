# Momento v0.34.42 — 빈 검색 결과가 수집 장애처럼 읽혔습니다

## "아직 데이터가 없습니다"

방문자 검색은 **`policyRange(90, max_exact_days)` 기간만** 조회합니다. 그 기간에 아무도 일치하지 않으면 `DataTable`의 기본 Empty가 나왔습니다 — **"아직 데이터가 없습니다 / SDK에서 이벤트가 수집되면…"**

그 문장은 **사이트에 아무것도 도착하지 않았다**는 뜻입니다. 100일 전에만 활동한 사람을 찾는 사용자는 **몇 일 안에서 못 찾은 것인지**, **정책 상한 때문에 기간이 줄었는지** 알 수 없었고, 검색을 반복하거나 수집을 의심했습니다.

## 「검색 결과가 없습니다」

빈 결과는 이제 **실제로 본 기간과 검색어**를 말합니다.

- 기본: **최근 90일 안에 "kim"과 일치하는 방문자가 없습니다.**
- 정책이 기간을 줄였으면: **최근 60일(조회 정책 상한) 안에 "kim"과 일치하는 방문자가 없습니다.**

결과가 있으면 표 설명 앞에 같은 기간 문구가 붙습니다 — **"최근 90일 안의 활동을 최근 순으로 보여줍니다."**

`visitorTrace.ts`의 `SEARCH_DAYS`·`searchWindowLabel`·`searchEmptyDescription`이 **요청과 같은 상수, 같은 `policyRange`** 를 읽습니다. 요청이 보는 일수와 문장이 말하는 일수가 다른 곳에서 나오지 않습니다. 로딩·오류 분기, `DataTable`, `policyRange`, 서버는 그대로입니다.

## 테스트

`visitorTrace.test.mjs`가 상한 없음·0·60·180에서 문구의 일수가 `policyRange`의 결과와 같은지, 빈 결과 문구가 검색어를 담는지 고정합니다. 빌드한 번들을 실제 서버(Postgres 17)에 올려 headless Chrome으로 기본 90일 빈 결과, 정책 60일로 줄인 빈 결과, 이벤트 수집 뒤 결과 표의 문구를 DOM에서 확인했습니다.

순수 모듈이 처음으로 다른 순수 모듈을 import 하므로 `web/tsconfig.app.json`에 `allowImportingTsExtensions`를 켰습니다(`tsconfig.node.json`과 같은 설정).

## 그 밖에

Go 쪽 변경은 없습니다. Database Migration, SDK, API, 환경변수 변경 없음.
