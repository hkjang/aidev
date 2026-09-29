# 검증 기록

격리 컨테이너: orbit-ai-regression-20260929, postgres:16-alpine, 127.0.0.1:49761. 검증 후 제거.

- DB DSN 설정 후 `go test -race -count=1 -v ./internal/server -run 'TestStreamAIPersonLookup|TestOrbit'`: 수정 전 세 시험 FAIL (ai-red.log), 수정 후 전부 PASS (ai-green.log).
- ai.go만 HEAD 내용으로 일시 복원 후 `go test -race -count=1 -v ./internal/server -run TestStreamAIPersonLookup`: 동일 세 시험 FAIL (ai-revert-red.log); finally에서 수정본 복구.
- DB DSN 설정 후 `go test -count=1 -v ./internal/server -run 'TestStreamAI|TestExtractDelta'`: PASS, SKIP 없음 (ai-focused.log).
- `gofmt -l internal/server`: 출력 없음. `go vet ./...`, `go build ./...`: exit 0 (ai-static-build.log).
- DB DSN 설정 후 `go test -race -count=1 ./...`: 모든 패키지 통과 (ai-full-race.log).
- 설정이 원래 없는 경우 시험 후 행 부재를 SQL assertion으로 확인. 기존 value/encrypted_value/updated_by/updated_at 행을 넣고 전체 race 실행 후 모든 컬럼 일치 SQL assertion 통과 (ai-setting-restore.log).
- `git diff --check`: exit 0.

웹·외부 AI·릴리즈 실행은 이번 범위에서 제외.
