# v0.316.0 — 주소 제한으로 차단된 로그인이 실제 남은 차단 시간을 말합니다

## "1분 후에 다시 시도하세요" 가 15분 동안 되풀이됐습니다

로그인을 막는 계수기는 둘입니다 — 계정당 실패(`auth.max_login_attempts`)와 주소당 실패(`auth.max_login_attempts_per_ip`). 그런데 `loginThrottleFor`(`internal/app/loginthrottle.go`)는 남은 시간을 **계정 기준 `min(created_at)` 하나로만** 계산했습니다.

주소 제한으로 차단된 호출자는 **자기 실패 기록이 없습니다.** 주소 계수기를 채운 것은 같은 주소로 보이는 다른 사람의 오타이기 때문입니다. 그래서 그 사람에게 계정 기준 `min` 은 NULL 이고, `RetryAfter` 는 0 이 되고, 응답은 이렇게 나갔습니다:

- `Retry-After: 1`
- `로그인 시도가 너무 많습니다. 1분 후에 다시 시도하세요.`

그러나 주소 차단은 `auth.lockout_minutes`(기본 15분) 동안 유지됩니다. **안내를 그대로 따른 사용자는 1분마다 같은 거부를 다시 만나며 15분을 보냅니다.** 비밀번호는 처음부터 맞았는데도 그렇습니다.

차단 기록도 한쪽만 말했습니다. `auth.login_blocked` 감사 기록의 `detail` 에는 계정 실패 횟수만 남았으므로, 주소 계수기가 거부한 모든 건은 운영자에게 **`failures: 0` 으로 차단됨**으로 읽혔습니다 — 어느 제한이 거부했는지 기록만으로는 알 수 없었습니다.

## 실제로 차단한 쪽의 남은 시간을 말합니다

- 한 질의에서 **두 계수기의 `min(created_at)` 을 각각** 읽습니다. 차단한 쪽의 가장 오래된 시도가 창(window)을 벗어나는 시각이 실제로 기다려야 하는 시각입니다.
- 두 제한에 동시에 걸렸으면 **늦게 풀리는 쪽**을 따릅니다. 짧은 쪽을 알려 주면 그 시간 뒤에 같은 거부를 다시 만나므로, 고치려던 문제가 그대로 남습니다.
- 남은 시간 계산을 `windowRemaining` 한 곳으로 모았습니다. `nil` 과 이미 지난 시각을 모두 0 으로 돌려주므로, 두 계수기가 같은 규칙을 지납니다.
- 차단 기록에 `addressFailures` 를 `failures` 와 **함께** 남깁니다. 두 값을 비교하면 어느 제한이 거부했는지 읽힙니다.
- 응답 코드(`429 TOO_MANY_ATTEMPTS`)·문구 형식·필드와 OpenAPI 는 바꾸지 않았습니다. 바뀌는 것은 문구 안의 숫자와 `Retry-After` 값입니다. 프런트엔드는 한 줄도 건드리지 않았습니다.

주소 제한은 **기본 비활성**이므로, 그것을 켜지 않은 배치에서는 달라지는 것이 없습니다.

## 검증

구현 단계에서 새 시험 2개(`internal/app/loginlockoutwait_test.go`)를 실제 PostgreSQL 위에서 돌렸습니다. 둘 다 프로덕션 `Handler()` 를 지나는 실제 로그인 요청이며, 한 시험 안의 모든 요청이 같은 클라이언트 주소에서 오는 것이 요점입니다 — 사무실 NAT 하나, Reverse Proxy 하나가 바로 주소 계수기가 존재하는 이유입니다.

- `TestAnAddressLockoutSaysHowLongItActuallyLasts` — 남의 이름으로 주소 계수기를 3회 채운 뒤, **실패 기록이 없는 계정이 맞는 비밀번호로** 로그인합니다. 주소 계수기만이 이것을 거부할 수 있습니다. `Retry-After` 가 14분 이상이고 문구가 `15분` 을 말하는지 봅니다.
- `TestTheBlockedLoginRecordNamesTheCounterThatRefusedIt` — 같은 상황에서 `audit_logs` 의 마지막 `auth.login_blocked` 행을 읽어 `failures` 가 0, `addressFailures` 가 3 인지 봅니다.

고치기 전 코드에서 두 시험이 먼저 실패하는 것을 확인했습니다 — `Retry-After says 60s, but the address stays blocked for 15 minutes` 와 `the record reports <nil> failures for the address`. 고친 뒤 둘 다 통과합니다.

릴리즈 단계에서도 실제 PostgreSQL(pgvector/pgvector:pg16) 위에서 `go test ./... -count=1` 전체를 통과했습니다. `go build ./...`·`go vet ./...`·`gofmt`, 가드 검사, 버전·OpenAPI·쪽넘김·모달 검사, 프런트엔드 시험과 타입 검사·프로덕션 빌드, `scripts/build.sh` 를 통과했습니다. 로드맵 HTML·PDF 와 두 안내서 HTML 을 기존 스크립트로 다시 만들었습니다.

검증하지 않은 것을 밝힙니다. 새 시험은 **주소 제한을 켠 상태**만 밟습니다 — 두 제한이 동시에 걸려 "늦게 풀리는 쪽" 을 고르는 경로는 코드로는 다루지만 시험으로는 밟지 않았습니다. 창을 실제로 15분 기다려 차단이 풀리는 것을 확인한 것도 아니고(남은 시간 계산만 봅니다), 프록시 뒤에서 여러 사용자가 한 주소로 보이는 실제 배치에서의 확인도 범위 밖입니다. 제자리에서 소스를 고치는 `mutation-check`·`authz-check` 는 과거 두 회차가 시간을 넘겨 돌리지 않았습니다. `gh` 를 사용할 수 없는 세션이므로 원격 이력을 보는 advisory `release-check.sh` 는 실행하지 않았습니다.

## 업그레이드

마이그레이션·새 환경변수·설정 변경은 없습니다. 기존과 같이 PostgreSQL 을 백업한 뒤 v0.316.0 이미지를 올립니다.

`auth.max_login_attempts_per_ip` 를 **켜 두지 않은 배치(기본값)에서는 달라지는 것이 없습니다.** 켜 둔 배치에서는 차단 안내의 숫자와 `Retry-After` 가 커집니다 — 이전에 "1분" 이라고 답했던 거부가 이제 남은 실제 시간(최대 `auth.lockout_minutes`)을 말합니다. 이것은 차단이 길어진 것이 아니라 **이전부터 그만큼이던 차단을 처음으로 정확히 말하는 것**입니다. 헬프데스크가 "1분 뒤에 된다고 했는데 안 된다" 는 문의를 받고 있었다면 그 문의가 사라집니다.

이 릴리즈는 **지난 기록을 고치지 않습니다.** 결함이 있던 동안 `failures` 만 남은 `auth.login_blocked` 행에는 `addressFailures` 가 없습니다. 그 기간에 어느 제한이 거부했는지를 조사해야 한다면 같은 시각의 `login_attempts` 를 주소로 세어 보십시오.

배포 담당자는 주소 제한을 켠 환경에서 배포 후 ① 주소 계수기로 차단된 로그인의 `Retry-After` 가 1분이 아니라 `auth.lockout_minutes` 에 가까운 값인지, ② 그 문구의 분(分)이 같은 값을 말하는지, ③ `auth.login_blocked` 기록에 `addressFailures` 가 함께 남는지를 확인합니다. ①이 여전히 1분으로 나오면 확대를 중단하고 v0.315.0 이미지로 되돌립니다 — 되돌리면 잘못된 안내도 함께 돌아옵니다. 배포 다음 날까지 로그인 차단 관련 문의를 확인하며, **"안내받은 시간 뒤에도 거부된다" 문의 0건**을 확인 기준으로 삼습니다.
