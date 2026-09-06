## 2026-09-07
- 선택: 업로드 도중 종료된 프로세스가 대상 디렉터리에 남긴 임시 업로드 파일 정리 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 0.5.5 에서 도입한 스테이징(`<패키지>.partial-<토큰>` 으로 먼저 쓰고 실행 기록이 대상을 확보한 뒤 확정)은 `createSimpleRun` 이 반환하는 모든 경로에서 `discard` 로 정리되지만, 프로세스가 업로드 도중 죽는 경우만은 예외라 임시 파일이 그대로 남습니다. 그 파일은 어떤 실행 기록도 가리키지 않고 화면에도 보이지 않으면서 패키지 크기(기본 상한 10 GiB)만큼 디스크를 차지하고, 아무도 지우지 않아 재시작마다 쌓입니다 — 이미 같은 이유(자식 프로세스가 함께 사라짐)로 `RecoverSimpleRuns` 가 `PENDING`/`RUNNING` 행을 마감하고 있는데 파일 쪽만 비어 있었습니다. 부팅 시 `simple_targets.upload_dir` 을 훑어 스테이징 이름 규칙에 **정확히** 맞는 일반 파일만 지우는 `RemoveStagedSimpleUploads` 를 추가하고 main.go 에서 `RecoverSimpleRuns` 바로 뒤에 호출했습니다. 이름 판정은 `isStagedUploadName` 으로 분리해 마커 뒤가 `RandomToken(16)` 의 base64url 22자와 정확히 일치할 때만 참이 되게 했고(운영자가 같은 디렉터리에 둔 파일·디렉터리·심볼릭 링크는 제외), 하드코딩한 접미사 두 곳을 상수로 묶어 스테이징 쪽과 어긋날 수 없게 했습니다. 파일시스템 단위 테스트 3건을 `simple_artifact_test.go` 에 추가했고(실제 `stageSimpleArtifact` 가 만든 이름이 인식되는지, 비슷한 이름들이 거부되는지, 스윕이 패키지·디렉터리·없는 경로를 건드리지 않는지), 로컬 도커 PostgreSQL 16 으로 `TEST_POSTGRES_DSN` 을 채워 backend/runner `go vet`·`go test ./...`(통합 테스트 포함), `npm ci`, `npm test -- --run`(78건), `npm run build` 를 모두 통과했습니다. docs/simple-mode.md 동시성 절에 한 줄 추가하고 VERSION 을 0.5.9 로 올렸습니다(저장소 관례). **주의: 아직 병합되지 않은 브랜치 `auto/2026-09-06-1820` 도 0.5.9 를 사용하므로 VERSION 충돌이 나며 둘 중 하나만 그 번호로 릴리즈해야 합니다.**
- 보류 아이디어: CI 와 Makefile 에 `go vet`(또는 golangci-lint) 단계가 없어 정적 검사가 매 세션 수동입니다 (가치 3 / 위험 1 / S).
- 보류 아이디어: `simpleRunLogger.append` 가 빈 payload 를 저장하지 않아 스크립트 출력의 빈 줄(문단 구분)이 로그에서 사라집니다 (가치 2 / 위험 1 / S).
- 보류 아이디어: `downloadSimpleRunLog` 이 조회한 `original_filename`·`status` 를 쓰지 않아 다운로드 파일명이 run id 뿐입니다 (가치 2 / 위험 1 / S).
- 보류 아이디어: 실행 상세 화면에 같은 묶음의 다른 실행 목록·링크를 표시하면 어느 패키지가 실패했는지 바로 찾을 수 있습니다 (가치 3 / 위험 1 / M).
- 보류 아이디어: `web/dist/assets/vendor` 청크가 617KB 로 커서 폐쇄망 초기 로딩 최적화 여지가 있습니다 (가치 2 / 위험 3 / M).
