# 수리 요약 (시도 1 → 커밋 e43fe34)

- 비평 3건 모두 맞았고 코드를 읽어 직접 확인했다: `assignee` 기록자는 **네 곳**(domain.go:468, finding_bulk.go:60, automation_workflows_tickets.go:50-62, imports.go:102→ingestFindings 가 imports.go:297/304 에서 validateResource 없이 직접 INSERT/UPDATE)이고, `notification_automation_recipients.go:78` 은 raw 문자열을 u.ID/u.Username 과 비교하므로 공백만 담당자는 **미할당**처럼 동작한다(큐 SQL 과 같은 방향). 어긋나는 쪽은 `web/src/triage.tsx:425` 상세 뷰 하나뿐이다.
- 고친 것: `finding_ops.go:83-110` 주석을 "네 경로 / 이번에 세 곳 적용"으로 다시 쓰고 수입 경로를 파일·행으로 명시하며 **고의 제외 + 후속 과제**임과 그 결과(수입이 계속 규칙 밖 담당자를 만들고, "같으면 skip" 때문에 이후 PUT 마다 영구 면제)를 기록했다. recipient-match 절은 삭제하고 "세 번째 해석이 아니다"로 바꿨다. 같은 과장이 있던 테스트 주석 2곳과 `openapi.json` 의 "이 규칙 전에 저장된 값" 문구도 실제 범위로 맞췄다.
- 1e2cd22 의 "every path" 과장은 amend 금지라 **교정 커밋 본문**에 실제 전달 범위(네 곳 중 세 곳)와 수입 공백을 적었다.
- 검증: 주석·문서만 바꿨으므로 동작 변화 없음. `gofmt -l`(무출력), openapi.json JSON 파싱, `go vet ./internal/app`, `go build ./cmd/hunter`, `go test -count=1 -v ./internal/app` = **47 PASS / 128 SKIP / 0 FAIL**(SKIP 은 전부 `HUNTER_TEST_DSN` 게이트), `node scripts/verify-pentagi.mjs` 312파일 통과. `sameJSONValue`→`==` 로 임시 되돌려 `comparing uncomparable type map[string]interface {}` panic 으로 FAIL 하는 것을 재확인하고 복원했다.
- 미검증: 실 HTTP·DB(DSN 없음), `go test -race ./...`, 웹 테스트(웹 변경 0), 원격 CI·Docker. 수입 공백 자체는 코드 판독으로만 증명했다(DB 없이 INSERT 경로를 돌릴 수 없음).
