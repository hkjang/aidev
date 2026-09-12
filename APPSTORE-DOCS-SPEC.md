# AppStore — 앱마다 가이드 문서 첨부

사내 서비스 스물세 곳이 `docs/USER_GUIDE.pdf` 와 `docs/ADMIN_GUIDE.pdf` 를 갖게 됐다.
그런데 그 문서를 찾으려면 그 저장소를 알아야 한다. 사람들이 앱을 찾는 곳은 AppStore 다.

앱 설정에서 가이드 문서를 **첨부파일로 올리고**, 앱 화면에서 **내려받을 수 있게** 한다.

## 저장

`branding_assets` 와 같은 모양을 쓴다. 바이트는 데이터베이스에 담고 체크섬을 함께 둔다.

```
app_documents(
  id, app_id, kind, title, filename, content_type,
  content, bytes, checksum, uploaded_by, created_at, updated_at
)
```

- `kind` — `user_guide` · `admin_guide` · `release_notes` · `other`
- 같은 앱에서 `user_guide` 는 하나다. 새로 올리면 갈아 끼운다(이력이 필요하면 `other` 로).
- `title` 은 화면에 보일 이름이다. 비면 파일 이름을 쓴다.

## 받기

`readBrandingUpload`(`internal/httpapi/branding_handlers.go:81`)의 방식을 그대로 따른다.
그 함수의 주석이 이유를 이미 적어 두었다 — `ParseMultipartForm` 은 메모리 예산을 넘긴
것을 **제한 없이** 임시 파일로 쏟아 내므로, 본문에 상한을 걸지 않으면 크기 검사가
이미 디스크에 다 쓴 뒤에야 돈다.

- `http.MaxBytesReader` 로 본문을 먼저 묶는다.
- `io.LimitReader(file, max+1)` 로 읽고 길이를 확인한다.
- 상한은 **25MB**. 화면 캡처가 든 가이드 PDF 는 2~6MB 이고, 여유를 둔 값이다.
- 형식은 헤더의 `Content-Type` 을 보고, 미덥지 않으면 `http.DetectContentType` 으로
  다시 본다. 둘 다 목록에 없으면 거절한다.

허용: `application/pdf` · `text/markdown` · `text/plain` ·
`application/vnd.openxmlformats-officedocument.wordprocessingml.document`(docx) ·
`...presentationml.presentation`(pptx)

**받지 않는다: `text/html` · `image/svg+xml`.** 둘 다 스크립트를 실어 나른다.

## 내주기 — 여기가 가장 조심할 곳

올린 파일을 이 앱과 **같은 오리진**에서 내주는 순간, 남이 올린 파일이 이 앱의 권한으로
브라우저에서 실행될 수 있다. 그래서 내려받기 응답은 다음을 모두 지킨다.

- `Content-Disposition: attachment` — **절대 inline 으로 열지 않는다.** PDF 뷰어는
  스크립트를 돌린다.
- `X-Content-Type-Options: nosniff`
- `Content-Security-Policy: default-src 'none'; sandbox`
- 저장된 `content_type` 을 그대로 쓰되, 목록에 없는 값이면 `application/octet-stream`.
- 파일 이름은 `filename*=UTF-8''...` 로 인코딩해 내보내고, **HTML 에 그대로 넣지 않는다.**

마크다운을 화면에서 바로 보여 주고 싶더라도 이번에는 하지 않는다. 올린 마크다운을
렌더링하는 것은 또 하나의 스크립트 실행 경로이고, 안전하게 하려면 정화기가 필요하다.
내려받기부터 온전히 만든다.

## 누가

- 올리고 지우는 것은 **그 앱의 소유자와 관리자**.
- 내려받기는 **그 앱을 볼 수 있는 사람**. 앱의 `visibility` 를 그대로 따른다 —
  문서가 앱보다 넓게 보이면 안 된다.
- 올리기·지우기는 감사 로그에 남긴다(`internal/audit`). 무엇이 언제 누구에 의해
  바뀌었는지가 문서에서도 필요하다.

## 화면

- **앱 설정**: 문서 목록, 올리기, 이름 바꾸기, 지우기. 각 줄에 종류·크기·올린 날짜.
- **앱 상세**: "문서" 자리에 내려받기 링크. 없으면 그 자리를 비운다.

## 테스트

`branding_handlers_test.go` 의 세 가지를 그대로 옮겨 온다.

- 상한을 넘는 본문은 **다 읽기 전에** 거절된다.
- 상한에 딱 걸리는 파일은 받아들인다.
- 허용하지 않는 형식은 거절된다(특히 HTML 과 SVG).

그리고 이 기능의 것들.

- 내려받기 응답에 `Content-Disposition: attachment` 와 `nosniff` 가 있다.
- 앱을 볼 수 없는 사람은 문서도 받을 수 없다.
- 같은 종류를 다시 올리면 갈아 끼워지고 두 벌이 남지 않는다.
- 올리기·지우기가 감사 로그에 남는다.

## 검증

- [ ] 실제로 가이드 PDF 를 올려 받아 보고, 브라우저가 **열지 않고 내려받는지** 확인했다.
- [ ] HTML 파일을 올려 보고 거절되는지 확인했다.
- [ ] 비공개 앱의 문서가 권한 없는 사용자에게 보이지 않는지 확인했다.
- [ ] 관리자 가이드에 문서 첨부 방법과 크기·형식 제한을 적었다.
