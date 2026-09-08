# v0.71.4 — uuid라는 이름으로 저장되던 사진

v0.71.0이 생각에 그림을 붙일 수 있게 했습니다. v0.71.1은 그 그림의 **딱지**를 —
사람이 붙인 파일 이름을 — 글자 한복판에서 자르지 않도록 고쳤습니다. 딱지는 그때부터
공들여 지켜졌습니다. 그런데 umm은 그 이름을 **브라우저에는 한 번도 말하지 않았습니다.**

첨부는 `/api/v1/attachments/{uuid}`에서 옵니다. 주소가 uuid입니다. 그리고 응답이
이름에 대해 한 말은 이것뿐이었습니다.

```go
w.Header().Set("Content-Disposition", "inline")
```

```
    붙인 파일:  2026-09-09 화이트보드.png

            전                                  후
    ━━━━━━━━━━━━━━━━━━━━━━━━━━      ━━━━━━━━━━━━━━━━━━━━━━━━━━
    캔버스에 잘 그려짐  ✔             캔버스에 잘 그려짐  ✔
    ↓  그림을 저장하면                ↓  그림을 저장하면
    b03d7110-43a5-4355-              2026-09-09 화이트보드.png ✔
      8858-26c385227b32
      어느 회의의 화이트보드인지
      알 방법이 없음
```

## 이름은 disposition이 아니라 매개변수입니다

`inline`은 바꿀 수 없습니다. 캔버스는 이 그림을 `<img>`로 그리고, `attachment`로
바꾸면 그림이 있어야 할 자리에 **내려받기가 옵니다.**

바꿀 필요도 없습니다. 이름은 disposition이 아니라 헤더의 **매개변수**이기 때문입니다.
보여 주는 방식은 그대로 두고 이름만 실을 수 있습니다.

```
inline; filename="2026-09-09 .png"; filename*=UTF-8''2026-09-09%20%ED%99%94...
```

v0.71.3이 내보내기 파일 이름을 위해 쓴 `attachmentDisposition`이 이미 이 일을
합니다 — RFC 6266으로 이름을 두 번 적고, `filename*`에 진짜 이름을 싣고, 따옴표 안에
ASCII 대체 이름을 남기고, 120바이트를 글자 경계에서 자릅니다. 그래서 그 함수를
`disposition`으로 나누고 `attachmentDisposition`과 `inlineDisposition`이 각각
`attachment`와 `inline`으로 부르게 했습니다. 호출자가 정하는 것은 disposition과,
적을 글자가 하나도 남지 않았을 때의 대체 이름뿐입니다(`umm-space` ↔ `umm-picture`).

## 확장자는 딱지가 정할 몫이 아닙니다

umm은 그림의 종류를 **바이트를 읽어서** 정합니다. 업로드가 뭐라고 말하든 믿지 않는
것이 이 파일 전체가 서 있는 규칙입니다. 그렇다면 그 규칙은 이름의 끝에도 적용됩니다.

```
    딱지                바이트      내려받는 이름
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    diagram.svg         PNG        diagram.svg.png
      거짓말한 끝을 잘라 내지도, 믿지도 않고 — 사람의 말은 남기고 참말을 덧붙임
    사진.JPEG           JPEG       사진.JPEG      ← 이미 맞으니 쓰인 대로
    2026.09.09 회의     PNG        2026.09.09 회의.png
      점이 들었다고 `.09 회의`가 확장자인 것은 아님
```

딱지에서 잘라 내면 사람이 쓴 말이 사라집니다. 그대로 믿으면 PNG가 `.svg`로 디스크에
앉습니다. 그래서 **말은 남기고, 바이트가 말하는 끝을 붙입니다.** 이미 그 형식의
끝으로 끝나는 딱지는 대소문자까지 쓰인 대로 둡니다 — `사진.JPEG`가 `사진.JPEG.jpg`로
돌아오지 않습니다.

## 검증

시험 3개를 새로 넣었습니다(단위 2 + 통합 1). 옛 `"inline"` 한 줄로 되돌리면 통합
시험이 `filename = "", want the label with the format umm read ("inline")`로
실패하는 것을 확인했습니다.

- `TestPictureName` — 딱지와 바이트가 어긋나는 경우들에서 이름과 끝이 어떻게
  갈라지는지
- `TestInlineDispositionNamesAPictureWithoutDownloadingIt` — 헤더가 `inline`으로
  남으면서 이름을 싣는지
- `TestAttachmentKeepsItsNameWhenSavedIntegration` — 실제로 올린 그림을 다시 받아
  클라이언트가 붙인 이름을 읽어 내는지

`go vet ./...` · `go test -p 1 ./...`(실제 PostgreSQL 17에 대한 통합 시험 포함) ·
gofmt · tsc · oxlint/Prettier · i18n 994키 · vitest 156개 ·
`scripts/check-version.sh` 통과.
