# fix-summary (수리 시도 2)

- 비평 3건 모두 맞음을 확인: `writeSSE` 는 항상 `\n\n` 로 끝내고, `[DONE]` 은 Go(`internal/ai/stream.go`)가 소비해 브라우저에 오지 않으며, 관리자 AI 채팅은 `finish` 를 분기하지 않음 → 원 커밋 메시지의 "finish 를 못 받는다" 사용자 영향 주장은 사실이 아님(커밋은 amend 금지라 그대로 둠).
- 실제 코드 결함(3번, 퇴행)만 고침: 스트림이 빈 줄 없이 끝났을 때 남은 마지막 블록은 payload 가 완전한 JSON 이거나 `[DONE]` 일 때만 전달하고, 잘린 조각(`{"type":"token","text":"안`)은 종전처럼 버림 (`emitSseBlock(..., { completeOnly: true })`, 커밋 5d5315f).
- 재현 테스트 추가: `drops a trailing block whose JSON was cut off by the connection closing` — HEAD 에서 조각이 raw 문자열로 emit 되어 실패하는 것을 확인한 뒤 수정으로 통과.
- 검증: `npm --prefix web test` 56/56, `npm --prefix web run lint` 무경고, `npm --prefix web run build` + `check-offline-assets.sh` 통과. Go 코드는 손대지 않음.
- 솔직한 평가: 퇴행은 제거했지만 남는 변경은 이 서버로는 재현되지 않는 경로(프록시/TCP 절단으로 마지막 이벤트가 `\n\n` 직전에 잘리는 경우)에 대한 방어적 하드닝이며 사용자 영향은 없음. 이 저장소에서 검증 하드닝 PR 은 사람이 반려한 전례가 있으니 PR 자체를 접을지는 비평가/러너가 판단할 것.
