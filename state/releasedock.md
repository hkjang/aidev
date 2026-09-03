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

## 2026-09-03
- 선택: 배포 후 단계 설정을 읽지 못한 실행이 SUCCESS 로 남는 문제 수정 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `executeSimpleRun` 은 긴 배포 중의 설정 변경을 반영하려고 명령이 끝난 뒤 `loadSimpleSettings` 를 호출하는데, 이 읽기가 실패하면 복제·앱 배포를 통째로 건너뛴 채 실행이 SUCCESS 로 기록됐습니다. 두 단계가 켜져 있었는지 여부 자체가 그 설정에 들어 있으므로, 미러링되지 않은 이미지와 교체되지 않은 앱이 초록색으로 보이는 것은 단계 순서가 막으려던 바로 그 상태입니다. 순수 함수 `outcomeWithoutStageSettings` 를 추가해 명령이 성공한 경우에만 FAILED 로 뒤집고(이미 실패한 실행은 자기 사유 유지) 로그에 `[post-deploy]` 줄을 남기도록 했습니다. 단위 테스트 1건을 추가했고 backend/runner `go vet`·`go test ./...`, `npm test -- --run`(72건), `npm run build` 를 모두 통과했습니다. docs/simple-mode.md 에 절을 추가하고 VERSION 을 0.5.3 으로 올렸습니다(저장소 관례).
- 보류 아이디어:
  - CI 와 Makefile 에 `go vet` (또는 golangci-lint) 단계가 없어 정적 검사가 수동입니다 (가치 3 / 위험 1 / S).
  - `simpleRunLogger` 는 로그 한도 도달 후 `system()` 출력까지 버려서 `exit=... status=...` 마지막 줄과 복제/앱 배포·설정 오류 안내가 사라집니다 (가치 3 / 위험 2 / S).
  - `readUploadBatch` 의 batchId/batchLast 파싱에 단위 테스트가 없습니다 (가치 2 / 위험 1 / S).
  - `web/dist/assets/vendor` 청크가 617KB 로 커서 폐쇄망 초기 로딩 최적화 여지가 있습니다 (가치 2 / 위험 3 / M).
  - 실행 상세 화면에서 `건너뜀`/`없음` 단계 상태 표기가 사용자에게 구분되는지 UI 문구 점검 (가치 2 / 위험 1 / S).
- 릴리즈: v0.5.3 (2026-09-03)
- 릴리즈: v0.5.3 (2026-09-03)

## 2026-09-03
- 선택: 로그 저장 한도를 명령 출력과 서버 진행 줄로 분리 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 실행 로그의 8 MiB 예산을 배포 명령의 stdout/stderr 와 서버가 직접 남기는 줄이 함께 쓰고 있어서, 이미지 로드·빌드 로그처럼 출력이 많은 스크립트가 한도를 채우면 그 뒤의 `exit=... status=...`, `[replication]`, `[app-deploy]`, `[post-deploy]` 줄이 전부 사라졌습니다. 운영자가 로그를 여는 이유가 바로 그 결과 줄인데 잘린 명령 출력만 남는 상태였습니다. 순수 타입 `logBudget` 을 추가해 명령 출력(8 MiB)과 서버 진행 줄(64 KiB 예비분)을 분리하고, 한도 도달 안내는 명령 예산을 소진한 그 payload 에서 한 번만 나오도록 유지했습니다. 단위 테스트 4건(`simple_log_test.go`)을 추가했고 backend/runner `go vet`·`go test ./...`, `npm test -- --run`(72건), `npm run build` 를 모두 통과했습니다. docs/simple-mode.md 동시성 절에 한 줄 추가하고 VERSION 을 0.5.4 로 올렸습니다(저장소 관례).
- 보류 아이디어:
  - CI 와 Makefile 에 `go vet` (또는 golangci-lint) 단계가 없어 정적 검사가 수동입니다 (가치 3 / 위험 1 / S).
  - `readUploadBatch` 의 batchId/batchLast 파싱에 단위 테스트가 없습니다 (가치 2 / 위험 1 / S).
  - 실행 상세 화면에서 `건너뜀`/`없음` 단계 상태 표기가 사용자에게 구분되는지 UI 문구 점검 (가치 2 / 위험 1 / S).
  - `web/dist/assets/vendor` 청크가 617KB 로 커서 폐쇄망 초기 로딩 최적화 여지가 있습니다 (가치 2 / 위험 3 / M).
  - `web/package-lock.json` 의 version 필드가 0.5.1 에 멈춰 있어 릴리즈 버전 bump 대상에서 빠져 있습니다 (가치 1 / 위험 1 / S).
- 릴리즈: v0.5.4 (2026-09-03)

## 2026-09-03
- 선택: 거부된 업로드가 실행 중인 명령의 패키지를 덮어쓰는 문제 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `createSimpleRun` 이 업로드를 대상 디렉터리에 **원래 이름으로 먼저 확정**한 뒤 실행 기록을 INSERT 해서, 대상당 1건 유니크 인덱스에 걸려 409 로 거부된 업로드도 이미 그 경로의 파일을 교체한 상태였습니다. 같은 이름 재업로드는 정상적인 재배포이므로 rename 이 기존 파일을 덮는 것이 의도지만, 그 파일이 지금 실행 중인 명령이 `$ARTIFACT` 로 받은 패키지일 수 있어 스크립트가 뒤늦게 다른 이미지를 로드할 수 있었습니다(앞선 세션들이 막아 온 "요청하지 않은 것이 배포되는" 상태와 같은 계열). `storeSimpleArtifact` 를 `stagedArtifact` 타입과 `stageSimpleArtifact`/`commit`/`discard` 로 분리해, 고유 임시 이름까지만 쓰고 INSERT 가 대상을 확보한 뒤에야 원래 이름으로 확정하도록 했습니다. 확정에 실패하면 `failSimpleRun` 으로 PENDING 행을 FAILED 로 마감해 대상이 잠기지 않게 했습니다. 파일시스템 단위 테스트 3건(`simple_artifact_test.go`)을 추가했고 backend/runner `go vet`·`go test ./...`, `npm ci`, `npm test -- --run`(72건), `npm run build` 를 모두 통과했습니다. docs/simple-mode.md 동시성 절에 한 줄 추가하고 VERSION 을 0.5.5 로 올렸으며, 그동안 0.5.1 에 멈춰 있던 `web/package-lock.json` 의 version 필드도 함께 맞췄습니다(`npm ci` 로 검증).
- 보류 아이디어:
  - CI 와 Makefile 에 `go vet` (또는 golangci-lint) 단계가 없어 정적 검사가 수동입니다 (가치 3 / 위험 1 / S).
  - `downloadSimpleRunLog` 이 조회한 `original_filename`·`status` 를 쓰지 않아 다운로드 파일명이 run id 뿐입니다 (가치 2 / 위험 1 / S).
  - 심플 배포 화면의 `waitForTerminal` 에 상한이 없어 종료 상태에 도달하지 못하는 실행이 화면을 영구히 잠급니다 (가치 2 / 위험 3 / M).
  - `web/dist/assets/vendor` 청크가 617KB 로 커서 폐쇄망 초기 로딩 최적화 여지가 있습니다 (가치 2 / 위험 3 / M).
  - (완료 확인) `readUploadBatch` 단위 테스트는 `http_helpers_test.go` 에 이미 있어 무효입니다.
- 릴리즈: v0.5.5 (2026-09-03)
- 릴리즈: v0.5.5 (2026-09-03)
