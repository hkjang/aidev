# fix-summary (수리 1회차)

- 지적은 맞았다: 재현 테스트로 확인 — 저장 A 응답 대기 중 스위치를 바꿔 B 를 누르면 쿼리가 다른 저장 요청이 2건 나가고, A 성공이 close() 를 불러 MainLayout v-if 가 팝업을 파괴하므로 실패한 최신 편집 B 가 사라졌다.
- 고침: `PopWidgetSetting` 에 `isSaving` 을 두어 `saveWidgetSetting` 재진입을 막고(`finally` 에서 해제), 대기 중에는 '설정' 버튼(`:disabled2`)·스위치(`:disabled`)·드래그(`:disabled`)를 잠가 유실될 '나중 편집' 자체가 생기지 않게 했다. 기존 저장 실패 계약(토스트·닫지 않음)은 그대로다.
- 회귀 테스트 4건 추가(연속 저장 잠금, 잠금 상태와 응답 뒤 해제, 버튼 우회 재진입, 예외 뒤 재저장). adapter 를 저장 요청마다 별도 resolve 를 쌓는 방식으로 바꿔 연속 저장을 순서대로 풀 수 있게 했다.
- 검증: 새 3건이 수리 전 실패 → 수리 후 `npm test` 38파일 583테스트 전부 통과, `npm run build:dev` 성공(dist 삭제). 추가한 줄마다 변이로 되돌려 관측 가능함을 확인했고, 관측되지 않은 `onToggle` 가드는 무효 변경이라 넣지 않았다.
- 미확인: 실제 브라우저에서의 sortablejs 드래그 잠금(jsdom 에서는 `_sortable.option('disabled')` 로만 단언), 백엔드 연동.
