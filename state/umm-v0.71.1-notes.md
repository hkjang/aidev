# v0.71.1 — 이름이 길다는 이유로 거절당한 사진

v0.71.0은 생각에 그림을 붙일 수 있게 했습니다. 파일 이름은 이 기능에서 아무것도
결정하지 않습니다 — 종류는 바이트를 읽어서 정하고, 이름은 그저 나중에 사람이 알아볼
**딱지**입니다.

그런데 그 딱지 하나가 사진을 통째로 떨어뜨리고 있었습니다.

```
    파일: 4x4 PNG, 82바이트 — 규칙을 하나도 어기지 않음
    이름: "2026 화이트보드화이트보드…화이트보드.png" (144바이트)

            전                                  후
    ━━━━━━━━━━━━━━━━━━━━━━━━      ━━━━━━━━━━━━━━━━━━━━━━━━
    이름을 120바이트에서 자름        이름을 120바이트 안에서
      → 글자 한복판에서 끊김           글자 경계까지만 자름
    ↓                               ↓
    PostgreSQL:                     붙음 ✔
      invalid byte sequence          딱지는 118바이트로 남음
      for encoding "UTF8"
    ↓
    500 "그림을 저장하지 못했습니다" ✘
```

## 왜 한글에서만 터지는가

`safeFilename`은 이름을 120바이트로 잘랐습니다. `cleaned[:120]` — 바이트 자르기입니다.

ASCII 이름은 한 글자가 1바이트라 이 자르기가 언제나 글자 경계에 떨어집니다. 한글은 한
글자가 3바이트입니다. 120번째 바이트가 글자의 두 번째 바이트인 순간, 잘린 결과는 더
이상 **글자가 아닙니다.** PostgreSQL의 `text`는 UTF-8이 아닌 바이트열을 받지 않으므로
`INSERT` 전체가 거절되고, 화면에는 그림에 문제가 있다는 말만 남습니다.

40글자가 넘는 한글 파일 이름은 드물지 않습니다. 회의 사진에 날짜와 안건을 그대로 적어
두는 사람에게는 그것이 보통의 이름입니다.

## 이미 있던 규칙을 쓰게 했습니다

이 저장소에는 `textutil.LimitUTF8Bytes`가 있고, 주석에 용도가 그대로 적혀 있습니다 —
"PostgreSQL이나 외부 시스템 경계를 넘는 문자열을 위한 것". Ptium 응답, 웹훅 메시지,
User-Agent가 이미 이것을 지납니다. 첨부 파일 이름만 지나지 않았습니다.

```go
return textutil.LimitUTF8Bytes(cleaned, 120)
```

길이 상한(120바이트)은 그대로입니다. 바뀐 것은 **어디서 끊느냐**뿐이라 ASCII 이름의
동작은 한 글자도 달라지지 않습니다.

## 빈 파일은 큰 파일이 아닙니다

같은 함수의 첫 줄에서 `len(data) == 0`을 "너무 큽니다"로 취급하고 있었습니다. 0바이트
파일을 올린 사람은 413과 함께 `한 장에 5MB까지 붙일 수 있습니다`를 받고, 근처에도 가지
않은 한계를 찾아보게 됩니다. 이제 빈 업로드는 종류 판별로 내려가 `PNG · JPEG · GIF ·
WebP만 붙일 수 있습니다`라고 답합니다 — 빈 것은 그림이 아니고, 그 문장이 사실입니다.

## 검증

시험 3개를 새로 넣었습니다. 옛 동작으로 되돌리면 셋 다 실패합니다.

- `TestSafeFilenameCutsBetweenCharactersNotBytes` — 자른 결과가 여전히 글자인지
  (`a cut name is not text any more: "…화이트\xeb"`)
- `TestAttachmentAcceptsALongNonASCIIFilenameIntegration` — 진짜 PostgreSQL에 실제로
  붙는지 (`invalid byte sequence for encoding "UTF8"`)
- `TestAttachmentEmptyUploadIsNotAnImageIntegration` — 빈 업로드가 무엇이라고 답하는지

`go vet ./...` · `go test ./...`(실제 PostgreSQL에 대한 통합 시험 포함) · gofmt · tsc ·
oxlint/Prettier · i18n 992키 · vitest 156개 · `scripts/check-version.sh` 통과.
