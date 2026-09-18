AppStore v2.11.2

관리 콘솔 AI 채팅이 서버가 마지막 SSE event 뒤에 빈 줄 없이 연결을 닫으면 마지막 텍스트 조각이나 `[DONE]` 을 잃어 대화가 끝나지 않은 것처럼 보이던 문제를 고친 patch 릴리스입니다.

- `web/src/lib/api.ts` 의 `streamAiChat` 은 `\n\n` 으로 끝난 블록만 파싱하고 `reader.read()` 가 `done` 이면 `TextDecoder` flush 도, 남은 buffer 파싱도 하지 않았습니다. Go 쪽 upstream 파서는 이미 같은 경우를 허용하므로 브라우저 쪽만 뒤처져 있던 비대칭이었습니다.
- 블록 파서를 `emitSseBlock` 하나로 모아 루프 안과 스트림 종료 뒤 flush 가 같은 코드를 쓰게 했고, 종료 뒤 남은 블록은 payload 가 완전할 때(유효한 JSON 또는 `[DONE]`)만 전달합니다. event 한가운데서 잘린 조각은 채팅 텍스트로 보여 주지 않고 버립니다.
- 서버 쪽 SSE 작성기와 그 밖의 화면은 v2.11.1 과 똑같으며, 두 가이드 PDF 는 버전 표기만 바뀌어 다시 구웠습니다.
- Schema 변경이 없어 기존 설치는 image만 교체하면 됩니다.
