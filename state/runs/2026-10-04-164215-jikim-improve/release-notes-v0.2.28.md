# jikim v0.2.28

> 참고: 이 저장소의 GitHub Release 본문은 태그 푸시 후 `release.yml` 이 `gh release create --generate-notes`
> 로 자동 생성합니다(제목 `jikim v0.2.28`, 자산 `jikim-v0.2.28.tar.gz` 와 `.sha256`). 아래는 기록용 요약이며
> 릴리스 본문으로 업로드할 필요가 없습니다.

## 수정

- AI 채팅 입력 검증이 실패하는 모든 경우를 `secret_material_rejected` 로 보고하던 오류를 수정했습니다.
  빈 입력, 허용되지 않은 message role, 빈 message, 총 262144바이트 초과, `max_tokens` 범위 초과까지
  Secret 과 무관한 다섯 가지 거절이 "Secret 평문이 섞였다"는 code 로 나가, 클라이언트와 운영자가
  잘못된 요청과 실제 Secret 유출 시도를 구분할 수 없었습니다. 이제 Secret 평문 거절만
  `secret_material_rejected` 이고, 나머지 다섯 거절은 `invalid_ai_request` 로 각자의 원인을 알립니다.
- Secret 평문 거절의 code 와 메시지, 여섯 경우 모두 `400` 이라는 점, `ValidateAIInput` 의 시그니처,
  Secret 판정 패턴, `max_tokens` 상한과 262144바이트 한도는 종전과 같습니다.

## 변경

- Secret 평문 분기만 표시하는 패키지 수준 sentinel 을 두고, 핸들러가 `errors.Is` 로 두 code 를 가립니다.
  오류 문구로 분기하지 않습니다.
- 실제 인증·request ID 미들웨어를 거친 요청 왕복으로 여섯 가지 거절 각각의 상태 코드와 code 를 고정하는
  회귀 테스트를 추가했습니다.

## 문서

- 문서 프로파일을 v0.2.28 로 갱신했으며, 화면 캡처는 실제로 찍은 `v0.2.9` 를 그대로 가리킵니다.
- 두 가이드 PDF 는 v0.2.27 과 같은 이유로 표지가 `v0.2.18` 인 채로 남았습니다(변환기가 저장소에 없음).
  본문 Markdown 은 v0.2.28 로 갱신되어 있습니다.

**Full Changelog**: https://github.com/hkjang/jikim/compare/v0.2.27...v0.2.28
