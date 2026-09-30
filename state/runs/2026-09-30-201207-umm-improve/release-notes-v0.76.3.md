# v0.76.3 — 붙인 그림은 폴더가 아니라 자기 이름으로 남습니다

그림을 붙일 때 어떤 클라이언트는 파일 이름만 보내고, 어떤 클라이언트는 **읽어 온 경로를 통째로**
보냅니다. umm 은 이름표에서 `/` 와 `\` 를 **지우기만** 했으므로, 경로를 보낸 쪽의 그림은 폴더 이름이
앞에 눌어붙은 채로 저장됐습니다. 그리고 그 이름표는 나중에 그 그림을 내려받을 때 파일 이름으로
다시 나갑니다 — 올린 이름과 내려받은 이름이 달랐습니다.

```
    "회의.png" 을 붙이면

            전                          후
    ━━━━━━━━━━━━━━━━━━━━━━━    ━━━━━━━━━━━━━━━━━━━
    C:\사진\회의.png   →         C:\사진\회의.png   →
      C:사진회의.png               회의.png

    /tmp/화이트보드.png →        /tmp/화이트보드.png →
      tmp화이트보드.png            화이트보드.png

    a/b/c.png         →         a/b/c.png         →
      abc.png                     c.png
```

## 구분자에서 나누고, 마지막 조각을 고릅니다

`safeFilename` 은 이제 `/` 와 `\` **둘 다**에서 이름을 나눈 뒤 마지막 조각을 고릅니다.

```go
pieces := strings.FieldsFunc(strings.TrimSpace(name), func(r rune) bool {
	return r == '/' || r == '\\'
})
```

`filepath.Base` 를 쓰지 않았습니다. 서버가 도는 리눅스에서 `filepath.Base` 는 `\` 를 구분자로 보지
않으므로, 정작 이 문제를 가장 자주 만드는 윈도 경로를 그대로 지나보냅니다. 구분자는 **보내는 쪽의
것**이지 서버가 도는 운영체제의 것이 아닙니다.

## 꼬리에 구분자가 붙었으면 앞 조각으로 되짚습니다

`사진/` 이나 `a/b/` 처럼 구분자로 끝나는 이름은 마지막 조각이 비어 있습니다. 여기서 빈 이름을
받아들이면 그림은 **이름표를 통째로 잃습니다.** 그래서 정리한 뒤에도 비지 않는 마지막 조각까지
뒤로 걸어갑니다 — `사진/` 은 `사진`, `a/b/` 는 `b` 가 됩니다. 사람이 친 말이 아직 거기 남아 있는데
지울 이유가 없습니다.

## 조각을 고른 뒤에도 씻습니다

이름표는 `Content-Disposition` 헤더와 텍스트 칼럼 두 곳에 닿습니다. 조각을 고르는 일과 씻는 일은
나뉘었지만 **씻는 일은 하나도 줄지 않았습니다.**

- 제어문자와 `"` 는 떨어져 나갑니다 — 헤더에 끼어들 수 있는 것들입니다
- `/` 와 `\` 도 **한 번 더** 떨어집니다. 나눈 뒤라 남아 있을 수 없지만, 그게 요점입니다: 이 함수
  뒤의 어떤 코드도 "위에서 나눴으니 없겠지" 를 믿을 필요가 없습니다
- 긴 이름은 여전히 UTF-8 **글자 경계**에서 120바이트로 자릅니다. 한글은 한 글자가 3바이트라
  바이트로 자르면 글자 가운데가 끊기고, PostgreSQL 은 깨진 UTF-8 을 거절합니다 — 장식인 이름표
  때문에 그림을 잃는 일은 없습니다

경계 하나가 의도적으로 달라졌습니다. `../etc/pass"wd\x00.png` 는 전에 `..etcpasswd.png` 였고 이제
`passwd.png` 입니다. 위험한 문자는 전과 똑같이 없고, 남는 것이 사람이 말할 이름 하나로 줄었을
뿐입니다.

## 달라지지 않는 것

**이미 저장된 이름표는 그대로입니다.** 붙는 순간 정해지는 이름이라 옛 그림의 이름표를 소급해
고치려면 데이터를 건드려야 하고, 그것은 이 수정의 몫이 아닙니다. 새로 붙이는 그림부터 적용됩니다.

헤더를 조립하는 `attachmentDisposition` · `dispositionSafe`, 인계 파일 이름을 짓는 `handoffFilename`
은 건드리지 않았습니다. 넷은 계약이 서로 달라 — 어떤 것은 줄기와 확장자를 나눠 받고, 어떤 것은
`filename*` 까지 적습니다 — 하나로 합치면 이번 같은 한 줄 수정이 네 곳의 회귀 위험이 됩니다.
바뀐 프로덕션 함수는 `internal/store/attachments.go` 의 하나뿐입니다.

## 검증

이름표는 붙는 순간 데이터베이스로 들어가므로, 단위 시험만으로는 **저장된 값**을 본 것이 아닙니다.
그래서 실제 PostgreSQL 17 을 지나는 integration 시험이 함께 있습니다.

```
✓ TestSafeFilenameKeepsOnlyTheLastPieceOfAPath
    C:\사진\회의.png · a/b/c.png · /tmp/화이트보드.png · C:/사진/회의.png · 회의.png
✓ TestSafeFilenameWalksBackPastATrailingSeparator
    사진/ → 사진 · a/b/ → b · C:\사진\ → 사진
✓ TestSafeFilenameDropsControlsAndQuotesFromTheLastPiece
    ../etc/pass"wd\x00.png → passwd.png, 구분자 0개
✓ TestSafeFilenameCutsBetweenCharactersNotBytes   (기존 경계, 그대로 통과)
✓ TestSafeFilenameKeepsAShortNameWhole            (기존 경계, 그대로 통과)
✓ TestAttachmentStoresOnlyTheLastPieceOfAPathIntegration
    AttachToNote → DB 에 저장된 이름표가 "회의.png"
```

고치기 전에 같은 시험들이 실패하는 것을 먼저 확인했습니다 —
`safeFilename("C:\\사진\\회의.png") = "C:사진회의.png"`,
`safeFilename("a/b/") = "ab"`,
그리고 DB 에 저장된 이름표 `"C:사진회의.png"`. 프로덕션 파일만 되돌리면 같은 실패가 다시 나고,
긴 한글 이름·짧은 이름 시험은 그 실행에서도 계속 통과합니다 — 이 한 경로만 잘못 붙던 증거입니다.

---

**전체 검증**: `go vet ./...` · `go test -p 1 ./...` 15개 패키지 전부 통과(PostgreSQL 17 연결,
integration 포함) · `gofmt -l internal/store` 무출력 · `go build ./cmd/...` ·
`npx tsc --noEmit` · `npm run lint`(oxlint + Prettier) · `node scripts/check-i18n.mjs` 1060키 ·
`npm audit --audit-level=high` 취약점 0 · `npm test` 186개 · `npm run build` ·
`npm run verify:pwa` · `scripts/check-version.sh`
