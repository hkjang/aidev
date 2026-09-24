## 2026-09-23
- 선택: 알림/확인 팝업 본문의 HTML 주입 차단·개행 렌더링 정정 (vitest SFC 마운트 지원 포함) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `Alert.vue`/`Confirm.vue` 는 본문을 `v-html="text"` 로 출력하는데 호출부 다수가 `openAlert(res?.data?.message || '...')` 로 서버 응답 메시지를 그대로 넘긴다. 서버 메시지의 태그가 실제 DOM 요소로 만들어지고(마운트 테스트로 `<script>` 요소 생성 확인), 반대로 본문의 `\n` 은 `.alert-text` 에 `white-space` 지정이 없어(morpheus-common.css:117) 줄바꿈으로 보이지 않았다. 본문을 이스케이프하고 개행과 기존 호출부의 `<br>` 표기만 `<br>` 로 되살리는 `src/utils/alertText.js` 를 추가해 두 컴포넌트에 연결했고, 증명 수단이 없던 문제를 먼저 해결하려 `vitest.config.js` 에 `@vitejs/plugin-vue` 를 연결하고 `@vue/test-utils` 를 devDep 으로 추가해 두 컴포넌트를 실제로 마운트하는 spec 을 작성했다. 검증: 기준선 `npm test` 19파일 424테스트 → 새 spec 2개로 Red 4건 확인(실제 DOM 에 `<script>` 생성·`br` 0개) → 수정 후 21파일 444테스트 통과. 컴포넌트 배선만 되돌리면 4건 실패, 개행 변환만 제거하면 4건 실패로 인과를 각각 증명했고, `npm run build:dev` 통과 후 `dist/` 를 삭제했다.
- 보류 아이디어: ChatStorageDetail.vue openLawViewer 의 고아 startLoading() 제거 — pending, 이제 SFC 마운트가 가능해져 증명 수단이 생겼다(단 이 뷰는 의존이 무거워 마운트 가능 여부 미확인).
  - SupportOcr.vue 업로드 실패 경로의 stopLoading 누락 — pending, 위 건과 함께 고쳐야 참조 카운트 도입이 가능하다.
  - Base.vue 의 `@:click` 오타로 배경 클릭 닫기가 동작하지 않음 — pending(신규), 바로 위 줄이 주석 처리된 정상 코드라 의도적 비활성일 수 있어 기대 계약 확인이 선행된다.
  - Alert/Confirm 이 `showHeader` 를 넘기지 않아 `title` 이 영구히 렌더링되지 않음 — pending(신규), `openAlert(msg, error)` 처럼 title 에 Error 객체를 넘기는 호출부 2곳도 이 때문에 증상이 감춰져 있다.
- 과제서: 없음(러너 09:59 판정) — 정찰 노트의 1순위(고아 startLoading 제거)는 SFC 마운트 없이는 증명할 수 없어 pending 으로 남기고, 정찰이 "1순위가 막히면 주저 말고 그쪽으로" 라고 지정한 차선(vitest SFC 지원)을 실제 버그 수정과 함께 구현했다.
