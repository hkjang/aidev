## 2026-10-08
- 선택: 마크다운/텍스트 목록 안 인용의 `>`를 요점에서 제거하기 (가치 2 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 목록 분기가 목록 마커만 제거해 인용 마커와 빈 요점을 남기는 원인을 새 Read 회귀 테스트로 확인하고, prose.go에서 withoutQuoteMarker를 한 번 적용해 해결했다(프로덕션 1파일·테스트 1파일·가이드 1파일, 커밋 dd7f719). 새 테스트 10개는 세 확장자의 전체 Source/경고, 모든 기존 목록 마커와 탭·중첩 인용, 빈 항목과 다섯 요점 경계, 표 두 개 분리, 제목·표·두 번째 목록 마커·수평선의 요점 유지, 리터럴 보존, 닫힌/미닫힘 펜스를 검증하며 수정 전 8개 실패·수정 후 전부 통과했고 수정 제거 시 대표 2개가 다시 실패했다. 검증은 baseline docs 3.195s → 최종 `go test -count=1 ./internal/docs` 3.933s, `go test -count=1 ./internal/docs -run TestEveryFormatReadIsAFormatSaid` 0.004s, `go test -race ./...` exit 0(테스트 보유 25패키지, docs 42.433s·일부 패키지 캐시), `go vet ./...` exit 0, 지정 두 파일 `gofmt -l` 및 `git diff --check` 출력 없음이며, PTIUM_TEST_DSN 미설정으로 DB 연결 통합 검증은 미확인이고 웹은 변경이 없어 검증 대상에서 제외했다.
- 실패 재현: markdownquote_test.go:187: read "# 제목\n\n- > 인용입니다.\n" as "# 제목\n@cover\n> 월간 보고서.md\n\n# 제목\n- \\> 인용입니다.\n!source 월간 보고서.md | 제목\n\n", want "# 제목\n@cover\n> 월간 보고서.md\n\n# 제목\n- 인용입니다.\n!source 월간 보고서.md | 제목\n\n"
  markdownquote_test.go:222: empty quoted items started a continuation: "# 제목\n@cover\n> 월간 보고서.md\n\n# 제목\n- 하나\n- 둘\n- 셋\n- 넷\n- 다섯\n!source 월간 보고서.md | 제목\n\n# 제목 (계속)\n- \\>\n- \\>>\n!source 월간 보고서.md | 제목 (계속)\n\n"
- 보류 아이디어:
  - 제품 /guide 가져오기 설명에 TSV·일반 텍스트 명시 (가치 2 / 위험 1 / 작업량 S) — 차선, 이번에 선택하지 않음.
  - 외곽 파이프 없는 GFM 표 인식 (가치 3 / 위험 3 / 작업량 M) — 앞보기·문장 오탐 계약이 필요해 보류.
  - setext h2 지원 (가치 2 / 위험 3 / 작업량 S) — 수평선·목록 우선순위가 별도 과제.
  - vitest.config.ts TypeScript 검사 포함 (가치 3 / 위험 3 / 작업량 M) — 의존성 타입 충돌 여부 진단이 먼저이며 웹 범위 제외.
- 과제서: 채택 — 현 코드의 목록 분기와 결함이 과제서와 일치하여 지정 세 파일만 수정했고 flush·공유 목록 헬퍼·escapeLine을 보존했다; `- ---` 자체는 기존 수평선이므로 `- > ---`의 비재귀 요점 계약만은 Read 쌍 대신 기대 Source를 직접 검증했다.
