### 수정

- AI 채팅 스트림에서 청크 소비자의 예외를 JSON 파싱 오류로 오인해 같은 이벤트를 원문으로 다시 전달하거나 성공으로 처리하던 오류 수정. `streamChat`의 예외 처리 범위를 `JSON.parse`로 좁혀 `onChunk`의 `Error`와 `SyntaxError`를 동일 객체로 호출자에게 전파하고, 해당 이벤트의 중복 호출과 뒤 이벤트 전달을 막았습니다. JSON `null`은 기존처럼 원문으로 전달하며, 소비자가 정상 처리하면 뒤 정상 이벤트도 계속 전달합니다. 실제 HTTP 서버와 native fetch를 통한 회귀 테스트로 확인했습니다

### 문서

- 문서 프로파일과 두 가이드 PDF 표지를 v0.2.17로 갱신했습니다. 화면 캡처는 실제로 찍은 `v0.2.9`를 그대로 가리킵니다

**Full Changelog**: https://github.com/hkjang/jikim/compare/v0.2.16...v0.2.17
