## 대기열을 거친 job의 `created_at`이 처리 시작 시각으로 밀리던 문제 수정

`runJob`은 마스킹이 끝난 뒤 `job.Metadata = *metadata`로 기록 전체를 덮어썼습니다. `process`는 메타데이터를 새로 만들어 돌려주기 때문에, `CreateJob`이 **업로드를 접수한 시각**으로 찍어 둔 `created_at`이 이 덮어쓰기에서 살아남지 못하고 **슬롯이 비어 실제 처리가 시작된 시각**으로 바뀌었습니다.

`PII_MASKER_MAX_CONCURRENT_JOBS` 슬롯이 모두 차 있어 `queued` 상태로 오래 기다린 작업일수록 오차가 커집니다(테스트 환경에서 306ms). 그 결과 클라이언트가 `updated_at`과 `created_at`의 차이로 재는 **대기 시간은 언제나 0에 가깝게** 나왔고, 이력 목록에 남는 **접수 순서도 실행 순서로 뒤바뀌었습니다**. `POST /v1/jobs`의 `202` 응답이 알려 준 `created_at`과 나중에 `GET`으로 조회한 값이 서로 달라지는 문제이기도 합니다.

- 기록을 덮어쓰기 직전에 원래 `created_at`을 잡아 두었다가 덮어쓴 뒤 되돌려 놓습니다. `updated_at`은 종전처럼 갱신되므로, 이제 두 값의 차이가 대기 시간과 처리 시간을 합한 실제 소요 시간을 나타냅니다.
- 프로덕션 배선(`app.New` → 실제 리스너 + 게이트를 건 mock 업스트림, `MaxConcurrentJobs=1`)을 그대로 지나는 통합 테스트 `TestQueuedJobKeepsTheTimeItWasAccepted`로, 슬롯이 막혀 대기한 job의 `created_at`이 `POST` 202 응답·`GET` 단건 조회·이력 목록·디스크의 `job.json` 네 경로에서 모두 같은 접수 시각으로 읽히는지 확인합니다.
- README의 대기열 문단에 `created_at`이 접수 시각으로 고정된다는 설명을 덧붙였습니다.

이 변경은 서버 내부의 시각 기록만 바로잡으며, API 스키마·마스킹 동작·저장 형식은 그대로입니다. 기존에 저장된 job 기록을 옮기거나 고칠 필요는 없습니다.
