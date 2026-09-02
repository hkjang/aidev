# ReleaseDock 자율 개선 기록

## 2026-09-02
- 선택: 심플 모드에서 복제가 마지막 파일로 미뤄진 실행의 앱 배포 실행 차단 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `복제 실행 범위=ONCE` + `앱 배포 실행 범위=EACH` 조합에서 앞선 파일의 실행이 복제를 SKIPPED 로 건너뛴 뒤에도 앱 배포 명령을 실행해, 미러링되지 않은 이미지로 앱을 교체할 수 있었습니다(코드 주석이 명시한 불변식 위반). `appDeployStageRuns` 헬퍼를 추가해 복제가 SKIPPED 면 앱 배포도 미루도록 하고, 단계 상태 문자열을 상수화했습니다. 순수 함수 단위 테스트(`TestAppDeployWaitsForADeferredReplication`)를 추가했고 `go vet`, backend/runner `go test ./...`, `npm test -- --run`, `npm run build` 를 모두 통과했습니다. 관리 화면 도움말·docs/simple-mode.md 도 갱신하고 VERSION 을 0.5.1 로 올렸습니다(저장소 관례).
- 보류 아이디어:
  - `listSimpleRuns` 의 `actor` 쿼리 파라미터가 UUID 검증 없이 `actor_id=$1` 에 들어가 잘못된 값이면 400 대신 500 이 됩니다 (가치 2 / 위험 1 / S).
  - `readUploadBatch` 의 batchId/batchLast 파싱에 단위 테스트가 없습니다 (가치 2 / 위험 1 / S).
  - CI 와 Makefile 에 `go vet` (또는 golangci-lint) 단계가 없어 정적 검사가 수동입니다 (가치 3 / 위험 1 / S).
  - `simpleRunLogger` 의 로그 한도 도달 경계(정확히 remaining 만큼 잘릴 때의 안내 메시지) 동작이 테스트되지 않았습니다 (가치 2 / 위험 1 / S).
  - `web/dist/assets/vendor` 청크가 617KB 로 커서 폐쇄망 초기 로딩 최적화 여지가 있습니다 (가치 2 / 위험 3 / M).
- 릴리즈: v0.5.1 (2026-09-02)
