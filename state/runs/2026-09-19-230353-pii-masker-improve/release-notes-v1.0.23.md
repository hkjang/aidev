## 상태 저장 도중 프로세스가 죽으면 완료된 job이 결과 파일째 사라지던 문제 수정

`persistLocked`는 `job.json`을 `os.WriteFile`로 제자리에서 덮어썼습니다. 이 방식은 파일을 먼저 비우고(O_TRUNC) 그 뒤에 내용을 쓰므로, 그 사이에 프로세스가 죽으면(OOM kill, SIGKILL, 디스크 가득 참) **빈 파일이나 잘린 JSON**이 남습니다. v1.0.22부터 `load()`는 그런 기록을 고아로 판정해 디렉터리째 삭제하기 때문에, 마스킹 결과를 이미 써 두고 마지막 `Save(completed)` 도중에 죽은 job은 다음 기동에서 결과 파일까지 함께 지워져 클라이언트에게 `404`로 사라졌습니다.

- 기록을 같은 디렉터리의 `job.json.tmp`(상수 `recordTempName`)에 쓴 뒤 `os.Rename`으로 `job.json`을 교체합니다. rename은 파일을 통째로 바꾸거나 이전 기록을 그대로 두거나 둘 중 하나이므로, 상태 변경 도중 죽어도 직전 유효 기록(`running`)이 남아 기동 시 `job_interrupted`(재시도 가능)로 정직하게 표시됩니다
- rename이 실패하면 오류를 돌려주고 tmp 파일을 best-effort로 지웁니다. 디스크의 `job.json`은 바이트 그대로 유지됩니다
- 크래시 잔재 `job.json.tmp`는 기록으로도 문서로도 오인되지 않습니다 — `readJobRecord`는 `job.json`만 열고, `firstExistingFile`은 `input_`/`output_` 접두어만 봅니다. 다음 저장이 덮어씁니다
- fsync는 하지 않습니다. 프로세스 크래시에 대한 원자성만 보장하며 전원 차단은 범위 밖입니다(코드 주석과 README에 명시)

검증: `internal/jobs` 테스트 3개 추가 — `Save`를 반복해도 디렉터리에 `job.json`·`input_*`·`output_*`만 남음 / `job.json.tmp` 자리에 디렉터리를 두어 `Save(completed)`를 실패시키면 오류가 반환되고 디스크의 `job.json`이 `running` 그대로임 / 정상 job 옆에 잘린 `job.json.tmp`를 심고 재로드하면 job이 올라오고 디렉터리가 유지되며 이어진 `Save` 후 tmp가 사라짐. 이전 구현으로 되돌리면 뒤의 두 테스트가 실제로 실패하는 것을 확인했습니다. `gofmt`·`go vet`·`go build`·`go test -count=1 ./...`·`go test -race -count=3` 전부 통과. README 보존 정책 문단에 한 문장 추가.
