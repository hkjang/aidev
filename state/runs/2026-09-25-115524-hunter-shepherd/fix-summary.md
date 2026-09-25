# 수리 요약 — hunter PR #10 (커밋 e503f24)

- 지적은 맞았다. `normalizeTrackingOrigin`이 `xn--` 라벨 검사를 브라우저 `URL` 파서에 위임했는데, Node 22(ICU)는 `http://xn--a.internal`을 거절하고 Node 26(ada)은 그대로 통과시킨다. Node 26 v26.10.0을 직접 내려받아 재현했다(CI와 같은 `invalid-punycode` 1건 실패).
- 고친 방법: `web/src/tracking-state.ts`에 RFC 3492 디코더(`trackingPunycodeDecode`)를 추가해 `xn--` 라벨을 직접 풀고, 파서에는 두 런타임이 일치하는 유니코드→ASCII 방향만 맡긴다. x/net/idna가 보안 사유로 거절하는 "ASCII로만 디코드되는 ACE 라벨"(idna.go:403)도 같이 거절한다. 서버 `internal/app/tracking.go`는 무변경.
- 검증: 무작위·표적 3,400여 건을 Go `trackingOrigin`과 차등 실행 — Node 22와 Node 26 결과가 **완전히 동일**하고, 비평가가 승인한 Node 22 프로파일과도 불일치 건수가 정확히 같다(Node 26 전용 불일치 7~14건 제거, 신규 0건). `npm test` 95/95 두 런타임 모두 통과, `go test ./internal/app` ok, tsc·vite build·go vet·go build·verify-pentagi(312)·prettier 통과. DB 테스트는 `HUNTER_TEST_DSN` 없이 skip.
- 벡터 3건 추가(`undecodable-punycode`, `empty-punycode`, `ascii-only-punycode`). 앞 1건은 수정 전 Node 26에서 실패하고, 뒤 1건은 새 ASCII 규칙을 빼면 두 런타임 모두에서 실패하는 것을 돌연변이로 확인했다.
- 남은 한계: 비평가가 이미 적은 UTS46 CheckHyphens/bidi/카테고리 미반영 3~5건은 그대로다(main과 동일, 이번 변경으로 늘지 않음). 혼합 ACE 라벨(`xn--한글-989an41e`)은 idna가 받아들이지만 UI는 거절한다 — 기존 ICU 동작과 같고 화면에 없는 이름을 제안하지 않기 위해 유지했다.
