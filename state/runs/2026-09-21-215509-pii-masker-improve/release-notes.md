## Upstage 진단 메시지를 자를 때 한글이 깨지던 문제 수정

공통 `truncateString`이 문자열을 바이트 위치로 바로 잘라 UTF-8 문자의 중간을 끊었습니다. 긴 Upstage 응답을 연결 점검·마스킹 오류의 `detail`이나 성공 응답의 `debug.body`로 반환하면 마지막 한글 등이 대체 문자(U+FFFD)로 바뀌어 진단 내용이 손상될 수 있었습니다.

- 절단 위치를 UTF-8 문자 시작 경계까지 뒤로 옮겨, 정해진 바이트 예산 안에서 완전한 접두부만 남깁니다. 기존 공백 제거, 무제한 길이, ASCII 및 `...` 말줄임 규칙은 유지합니다.
- 단위 경계 테스트와 `app.New`에서 실제 HTTP를 거치는 통합 테스트로 연결 점검·마스킹 오류 `detail` 및 성공 `debug.body`의 정확한 접두부·바이트 길이·U+FFFD 부재를 검증합니다. 성공 응답의 `completed`, `applied_regions=0`, 원본 PNG 파일 파트도 확인합니다.

## 재시작 뒤 미완료 job의 잔여 결과 파일이 다운로드되던 문제 수정

v1.0.24 이후 함께 포함된 변경입니다. 기동 시 디스크에 `output_` 파일이 있으면 완료 여부와 무관하게 결과 경로를 복원해, 쓰기가 끝나지 않은 파일도 다운로드될 수 있었습니다.

- `completed` job만 결과 경로를 복원하며, 미완료 job의 다운로드 URL은 지웁니다.
- job 조회·이력에서도 완료된 결과만 URL을 표시하고, 미완료 결과의 직접 다운로드는 `404 job_result_not_found`로 차단합니다.

## 검증 및 배포 파일

- `go test -count=1 ./...`, `go vet ./...`, `go build ./...`, `git diff --check` 통과.
- Docker 이미지 빌드와 연결 점검·PNG 동기 마스킹·PDF 비동기 완료 및 결과 다운로드 확인. PowerShell이 없는 환경이므로 `scripts/smoke-test.ps1`의 같은 검증 항목을 Python으로 실행했습니다.
- 첨부한 `pii-masker-image.tar.gz`에는 기존과 같은 `pii-masker:latest` 이미지가 들어 있습니다. `scripts/export-image.ps1`과 동일하게 Docker 이미지 저장 후 gzip으로 압축했고, gzip 무결성 및 아카이브의 이미지 태그·구성·레이어 존재를 확인했습니다.
- 기존 실행 방식: `sh ./scripts/run-from-archive.sh ./pii-masker-image.tar.gz`
