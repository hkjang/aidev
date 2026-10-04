## 선언한 형식만 보고 재인코딩을 결정해 추론 엔드포인트가 받지 않는 바이트를 그대로 보내던 문제 수정

`prepareUpstreamAttachment`는 업로드를 PNG로 다시 인코딩할지를 `attachment.MIMEType`, 즉 **클라이언트가 선언한 형식**만 보고 결정했습니다(`!EqualFold(MIMEType, "image/jpeg")`이면 바이트를 그대로 보냄). 그런데 `detectAttachmentMIMEType`은 선언한 값을 그대로 신뢰하고, 내장 Playground는 그 값을 파일 이름의 확장자에서 만듭니다. 그래서 JPEG 파일을 `.png` 확장자로 저장해 올리면 선언 형식이 `image/png`이 되어, JPEG 바이트가 `Content-Type: image/png`를 달고 추론 엔드포인트로 그대로 나갔습니다. 이 함수가 존재하는 이유 자체가 "엔드포인트는 PDF와 PNG만 받는다"는 것이고(내장 mock의 `isMockSupportedMIMEType`도 같은 계약입니다), 바로 그 불변식이 깨져 있었습니다. 이제 판단 기준이 선언한 형식이 아니라 **파일 내용**입니다.

- `prepareUpstreamAttachment`의 재인코딩 판단을 새 헬퍼 `document.DetectImageFormat`으로 바꿨습니다. 이 헬퍼는 `image.DecodeConfig`로 **헤더만** 읽어 실제 포맷을 돌려주므로, 이미 PNG인 업로드는 픽셀 버퍼를 할당하지 않습니다.
- 내용이 PNG이면 바이트를 그대로 두고 선언 타입과 파일 확장자만 PNG로 교정합니다(`asUpstreamPNG`). PNG가 아닌 이미지는 디코딩해 PNG로 다시 인코딩합니다. PDF는 이전과 똑같이 바이트 단위로 그대로 전달됩니다.
- 원본 `attachment`은 손대지 않습니다. 따라서 마스킹 결과와 응답 메타데이터(`output.mime_type`, 결과 다운로드의 `Content-Type`)는 v1.0.33에서 정한 "선언한 형식을 따른다"는 계약을 그대로 유지합니다.
- 변경 파일은 `internal/upstage/client.go`, `internal/document/image.go`와 README 한 줄입니다.
