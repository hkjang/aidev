v0.3.1 파일 대조: 구분자 감지가 제목 줄에 끌려가지 않게

- CSV 구분자 감지(`detectDelimiter`, packages/tools/src/table/csv.ts)가 점수를 첫 줄의
  구분자 개수에 고정해 매겨서, 표 위에 제목 줄이 있고 그 줄에 쉼표가 하나라도 들어간
  파일에서는 쉼표가 본문의 세미콜론·탭을 이겼다. 제목 줄 위 표는 v0.3.0 직전 커밋에서
  공식 지원하기로 한 파일 모양이라 정면으로 어긋났다.
- 점수 기준을 "줄들 사이에서 가장 많이 반복되는 0 이 아닌 개수(최빈값)" 로 바꿨다.
  빈도가 같으면 더 큰 개수를 쓰고, 후보 순서(`,` `\t` `;` `|`)는 그대로 둔다.
- 감지 함수만 고쳤다. `parseCsv` 와 다른 경로의 파서(xlsx-read, compare.worker)는
  손대지 않았다.
- 검증: `npm run check`(lint, 타입 4개 패키지, 단위 10파일 122항목), `npm run build`,
  VERSION 과 wrangler.toml 의 버전 일치. 브라우저 흐름(tools-flows·e2e)은 이 기계에
  Playwright·PostgreSQL 이 없어 돌리지 못했다 — CI(check.yml)가 같은 게이트를 본다.

버전 파일: VERSION, package.json 4개와 루트, package-lock.json, docs/openapi.yaml,
wrangler.toml 의 WORKKIT_VERSION, packages/tools 의 TOOLS_VERSION.
