## 업로드 크기 제한이 표현 범위를 넘으면 음수로 넘쳐 모든 업로드가 거절되던 문제 수정

`config.Load`는 `PII_MASKER_MAX_FILE_SIZE_MB`를 정수로 읽은 뒤 `1024*1024`를 곱해 바이트로 바꿨습니다. 곱하기 전 상한 검사가 없어 `PII_MASKER_MAX_FILE_SIZE_MB=8796093022208`처럼 큰 양수를 지정하면 `MaxFileSizeBytes`가 음수로 감싸였습니다. 제한을 올리려던 설정이 오히려 제한을 없애 버리는 것이 아니라, 그 값을 읽는 업로드 검사 세 곳이 모두 음수와 비교하게 되어 비어 있지 않은 **모든** 업로드가 `413 payload_too_large`(`request body exceeds the maximum size of -9223372036854710272 bytes`)로 거절되었습니다. `/v1/config/public`도 감싸인 음수를 그대로 노출했습니다. 이제 표현 범위를 초과한 값은 기본값 50MB로 복귀합니다.

- `PII_MASKER_MAX_FILE_SIZE_MB` 호출 지점을 새 헬퍼 `envMebibytes`로 바꿨습니다. 이 헬퍼는 `strconv.ParseInt(..., 10, 64)`로 읽고 `MaxInt64 / 1MiB` 상한을 곱셈 전에 검사하며, 이는 v1.0.31에서 시간 설정에 적용한 `envDuration`의 방식을 그대로 따른 것입니다. 바이트 변환이 가능한 상한은 8796093022207(MB)입니다.
- `envInt` 자체는 그대로 두었습니다. 나머지 호출자인 `MAX_CONCURRENT_JOBS`·`MAX_CONCURRENT_SYNC`의 동작은 바뀌지 않습니다.
- 경계값 설정 테이블 테스트와, 프로덕션 배선을 그대로 쓰는 HTTP 회귀 테스트를 함께 추가했습니다. 되돌리면 `8796093022208` 설정에서 984바이트 업로드가 다시 413으로 거절되는 것을 재현합니다.
