# 수리 요약 (a578657)

비평 네 건 모두 맞았고, 실측으로 재현했다. `r.ParseForm()` 은 본문과 URL 쿼리 두 파스를 오류 하나로 합치고 둘 다 실패하면 본문 것을 돌려준다. 그래서 (a) 쿼리의 깨진 이스케이프와 (b) 깨진 미디어 파라미터(`mime.ParseMediaType` 이 base type 은 그대로 돌려주므로 본문은 온전히 읽힌다) 둘 다 본문이 멀쩡한데 non-nil 오류를 내고, 1152행이 올바른 `uri_not_registered` 를 `form_unreadable` 로 덮었다.

고친 방법: `readLogoutPostForm` 을 새로 두어 본문만 직접 읽고(1MiB MaxBytesReader 그대로 사용), 결과를 `r.PostForm` 에 넣은 뒤 `ParseForm` 을 부른다 — `ParseForm` 은 `r.PostForm != nil` 이면 본문 읽기를 건너뛰므로 `r.Form` 병합만 하고, `formErr` 는 본문의 오류만 남는다. 판정 규칙(`formErr != nil && redirectTo == ""`)과 이유 코드 문자열·거절 없음·상한은 그대로다. 1144행의 뒤집힌 근거 주석, "three ways a body fails", const 주석의 "폼이 아닌 Content-Type", docs/operations.md 표의 `Content-Type` 지시를 모두 사실에 맞게 고쳤고, 폼이 아닌 Content-Type 은 **아무 reason 도 남지 않는다**는 실제 공백을 docs 와 const 주석에 명시했다.

테스트 세 케이스 추가: 쿼리 이스케이프 오류 + 본문 미등록 주소, 깨진 미디어 파라미터 + 본문 미등록 주소(둘 다 `uri_not_registered` 단언, 수정 전 코드에서 `form_unreadable` 로 **실제 실패 확인**), JSON 본문(무음 손실 — reason 없음 고정). 기존 두 케이스와 쿼리 성립 케이스는 그대로 통과.

검증(실제 PostgreSQL, `RESSO_TEST_POSTGRES_DSN` 설정됨 — SKIP 아님): `go test -race ./internal/httpserver/... -count=1` → ok 111.234s. `make lint` → golangci-lint 0 issues, govulncheck 0, ESLint 통과. 되돌리기 검증도 했다(HEAD 판 oidc.go 로 교체 시 새 두 케이스가 지적된 그 문장 그대로 FAIL).
