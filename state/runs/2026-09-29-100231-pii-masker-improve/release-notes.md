## 시간 설정이 표현 범위를 넘으면 음수로 넘쳐 보존 기간 정리가 꺼지던 문제 수정

`config.Load`는 초·시간 단위 환경변수를 정수로 읽은 뒤 `time.Duration` 단위를 곱했습니다. 곱하기 전 상한 검사가 없어 `PII_MASKER_JOB_RETENTION_HOURS=2562048`처럼 큰 양수를 지정하면 음수 보존 기간으로 바뀌고, 작업 파일 정리가 비활성화되어 업로드 원본과 결과가 남을 수 있었습니다. 이제 표현 범위를 초과한 값은 해당 설정의 기본값으로 복귀합니다.

- 시간 설정 여섯 개를 `strconv.ParseInt(..., 10, 64)`로 읽고 `MaxInt64 / unit` 상한을 곱셈 전에 검사합니다. 초 단위 상한은 9223372036, 시간 단위 상한은 2562047입니다.
- 적용 대상은 `READ_HEADER_TIMEOUT_SECONDS`, `IDLE_TIMEOUT_SECONDS`, `SHUTDOWN_TIMEOUT_SECONDS`, `DEFAULT_TIMEOUT_SECONDS`, `SYNC_QUEUE_WAIT_SECONDS`, `JOB_RETENTION_HOURS`입니다(모두 `PII_MASKER_` 접두사). 기존 기본값과 일반 정수 파서의 계약은 유지합니다.
- `PII_MASKER_JOB_RETENTION_HOURS=0`의 정리 비활성화와 `PII_MASKER_SYNC_QUEUE_WAIT_SECONDS=0`의 즉시 거절 동작은 그대로 유지됩니다. 유효한 범위의 설정은 변경할 필요가 없습니다.
- Load 경계 테이블 78건과 실제 `config.Load → app.New → HTTP` 회귀 테스트로 초과 보존 기간이 기본 24시간으로 복귀하여 48시간 된 작업은 삭제/404가 되고 최근 작업의 파일은 유지/200이 되는 것을 검증했습니다. 수정 전과 프로덕션 수정 되돌림 모두에서 실패가 재현됐습니다.

검증: `go test -count=1 ./...`, `go vet ./...`, `go build ./...`, config·httpapi race 테스트, `GOARCH=386` config 테스트, gofmt 및 diff 검사가 통과했습니다. Dockerfile의 Go 1.25.0/Alpine 3.21 이미지 빌드와 해당 이미지의 mock 연결, PNG 동기 마스킹, PDF 비동기 완료·결과 다운로드도 통과했습니다. 로컬 테스트 도구는 Go 1.26.7 linux/amd64이며, 32비트 전체 HTTP 통합은 실행하지 않았습니다.

기존과 같이 `pii-masker:latest`를 저장·gzip 압축한 `pii-masker-image.tar.gz`를 제공합니다. `sh ./scripts/run-from-archive.sh ./pii-masker-image.tar.gz`로 실행할 수 있습니다. PowerShell이 없는 빌드 환경에서는 기존 export 스크립트와 동일한 Docker image save 및 gzip 방식을 셸로 수행했습니다.

배포 후에는 헬스체크·마스킹 성공 여부와 보존 기간이 지난 작업의 정리 여부를 확인하세요. 헬스체크 실패 또는 마스킹 회귀가 한 번이라도 확인되면 배포 운영자가 확대를 중단하고 보관한 v1.0.30 이미지로 복귀합니다. 삭제된 작업 파일은 이미지 롤백으로 복원되지 않습니다. 실제 운영 배포·외부 사용자 확인은 이 로컬 릴리즈 준비에 포함되지 않습니다.
