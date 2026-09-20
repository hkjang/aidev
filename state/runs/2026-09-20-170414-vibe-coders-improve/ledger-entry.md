## 2026-09-20
- 선택: 언어 추론의 신뢰도 상승 시 기존 evidence 보존 (가치 2 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: addSignal의 신규 초기화와 기존 신호 갱신을 분리해 최대 confidence를 유지하면서 기존 정형 evidence를 발견 순서와 `; ` 구분으로 누적한다. 공개 InferLanguages 정순·역순·반복·동일 신뢰도 회귀, 실 JSON 추출을 거친 감사 재사용/새 추출, 실제 HTTP→NewServer.Routes→AsyncLogger→SQLite RequestDetail 테스트를 추가·보강했고 수정 전 근거 소실 실패를 확인했다. audit·proxy 선택 테스트, audit race, go vet ./..., go build ./..., go test ./... -count=1 모두 통과했으며 요청한 technology 스킬 3종은 도구/로컬 파일 미발견으로 고유 절차·반환 형식을 확인하지 못했다.
- 보류 아이디어:
  - tracking.custom_snippet 전용 textarea (가치 3 / 위험 2 / 작업량 M).
  - captureSsoFragment의 sso=error 마커 보존 (가치 2 / 위험 1 / 작업량 S).
  - APP_UI_ROADMAP의 추적 활성 페이지 no-store 설명 정정 (가치 2 / 위험 1 / 작업량 S).
  - .mjs·.cjs·.mts·.cts 언어 확장자 지원 (가치 2 / 위험 2 / 작업량 S).
- 과제서: 채택 — 높은 신뢰도에서 evidence가 덮어써지는 근거가 현재 코드와 일치했고 공개 추론·감사 두 경로·실제 비동기 저장에서 수용 기준을 검증했다.
