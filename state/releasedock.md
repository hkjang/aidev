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

## 2026-09-02
- 선택: 일괄 업로드가 마지막 파일에 도달하지 못했을 때 미뤄진 단계 누락 경고 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: "이 묶음의 마지막" 표시는 파일 업로드 시점에 고정되므로, `남은 파일 중단`·마지막 업로드 거부·마지막 파일의 배포 명령 실패 중 어느 경우든 `업로드당 한 번` 으로 미뤄 둔 복제/앱 배포를 실행할 실행이 사라지고, 앞선 파일들은 성공으로 남은 채 아무것도 미러링되지 않는 상태가 됩니다(직전 세션에서 서버 쪽으로 막은 것과 같은 불변식이 클라이언트 쪽에서 뚫려 있었음). `SimpleDeployPage` 에 순수 헬퍼 `stagesDeferred`/`stagesReached` 를 추가해, 앞선 실행이 남긴 `SKIPPED` 와 표시된 마지막 실행이 그 단계에 도달했는지를 비교하고 도달하지 못했으면 경고 Alert 를 띄웁니다(단계가 실행됐다가 실패한 경우는 그 실행 자체가 이미 실패로 보고되므로 경고하지 않음). 순수 함수 단위 테스트 3건을 추가했고 `npm test -- --run`(72건), `npm run build`, backend/runner `go vet`·`go test ./...` 을 모두 통과했습니다. docs/simple-mode.md 에 절을 추가하고 VERSION 을 0.5.2 로 올렸습니다(저장소 관례).
- 보류 아이디어:
  - CI 와 Makefile 에 `go vet` (또는 golangci-lint) 단계가 없어 정적 검사가 수동입니다 (가치 3 / 위험 1 / S).
  - `readUploadBatch` 의 batchId/batchLast 파싱에 단위 테스트가 없습니다 (가치 2 / 위험 1 / S).
  - `simpleRunLogger` 는 로그 한도 도달 후 `system()` 출력까지 버려서 `exit=... status=...` 마지막 줄과 복제/앱 배포 안내가 사라집니다 (가치 2 / 위험 2 / S).
  - `executeSimpleRun` 에서 `loadSimpleSettings` 가 실패하면 배포 후 단계 전체가 조용히 생략되고 실행은 SUCCESS 로 남습니다 (가치 3 / 위험 2 / S).
  - `web/dist/assets/vendor` 청크가 617KB 로 커서 폐쇄망 초기 로딩 최적화 여지가 있습니다 (가치 2 / 위험 3 / M).
  - (기각) 직전 기록의 `listSimpleRuns` actor UUID 검증 아이디어는 무효입니다. `simple_runs.actor_id` 는 TEXT 이므로 잘못된 값이 와도 500 이 나지 않습니다.
- 릴리즈: v0.5.2 (2026-09-02)
- 릴리즈: v0.5.2 (2026-09-02)
