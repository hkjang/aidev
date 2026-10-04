# Momento v0.34.56 — 망 구분을 추가하다 실패하면 한국어로 무엇을 고칠지 말해 줍니다

콘솔 **관리 → 망 구분**의 「망 구분 추가」 가 거절당했을 때 띄우는 문장이 달라집니다. 지난 두 릴리스가 사용자 다이얼로그(v0.34.54)와 사이트 다이얼로그(v0.34.55)에 한 것을 이 폼에 합니다. 서버·API·데이터베이스는 그대로입니다.

## 손으로 적는 CIDR 칸의 실수가 세 낱말로 돌아왔습니다

이 Alert 은 `{create.error.message}` 로 **서버가 보낸 문장을 그대로** 띄웠습니다.

이 폼은 콘솔에서 **CIDR 을 손으로 적는 유일한 칸**을 가지고 있습니다. 그래서 가장 흔한 실패가 이렇게 돌아왔습니다.

```
CIDR is invalid
```

세 낱말입니다. 무엇이 틀렸다는 말은 있지만, **무엇을 적어야 하는지는 말하지 않습니다** — 칸 바로 위에 올바른 모양이 placeholder 로 떠 있는데도 그렇습니다.

500 으로 떨어지면 더 나빴습니다. `internal/httpapi/admin.go` 의 `createNetwork` 가 실패를 `err.Error()` 로 그대로 실어 보내므로(`:823`), pgx 의 원문이 **스키마에서 브라우저까지 그대로** 흘렀습니다.

```
ERROR: relation "network_ranges" does not exist (SQLSTATE 42P01)
ERROR: invalid cidr value: "10.0.0.5/24" (SQLSTATE 22P02)
```

## 이제 무엇을 고칠지 말합니다

| 서버가 거절한 이유 | 화면에 뜨던 것 | 이제 뜨는 것 |
|---|---|---|
| CIDR 표기 | `CIDR is invalid` | CIDR 표기가 올바르지 않습니다. 「CIDR」 칸에 `10.20.30.0/24` 처럼 주소 뒤에 「/」 와 **비트 수**(IPv4 는 0~32, IPv6 는 0~128)를 붙여 적으세요. |
| 본문을 읽지 못함 | 영문 한 문장 | 보낸 내용을 읽지 못했습니다 + 작은 글씨로 **서버 원문** |
| 추가 500 | pgx 원문 | 망 구분을 추가하지 못했습니다 + 작은 글씨로 **서버 원문** |
| 요청이 닿지 않음 | 영문 한 문장 | 요청이 서버에 닿지 못했습니다 |

안내가 드는 예시 `10.20.30.0/24` 는 **칸 자신의 placeholder 와 같은 값**입니다. 읽는 사람이 화면에서 두 곳을 맞춰 볼 수 있습니다.

### 「넷마스크 경계」 라고 쓰지 않았습니다

`net.ParseCIDR` 이 거절하는 것은 **표기 자체**입니다 — 비트 수가 없거나 범위를 벗어난 경우. 넷마스크 오른쪽에 비트가 선 값(`10.0.0.5/24`)은 **통과시킵니다.** 그러니 이 400 은 넷마스크 경계 때문에 날 수 없고, 그렇게 안내했다면 실제로 틀린 것을 가리키지 못했을 것입니다. 안내는 비트 수 표기를 말합니다.

## 500 은 원인을 말하지 않습니다

`NETWORK_CREATE_FAILED` 의 문구는 일부러 중립으로 두었습니다.

특히 **'이미 등록된 망' 이라고 설명하지 않습니다.** `network_ranges` 에는 UNIQUE 가 **아예 없습니다** — `internal/database/migrations/001_initial.sql:64-70` 은 `id PRIMARY KEY` 와 평범한 네 칸뿐입니다. 중복이라고 말했다면 그것은 그냥 거짓입니다.

`cidr` 컬럼이 `net.ParseCIDR` 을 통과한 값을 거절할 가능성은 있습니다. 하지만 **이 기계에서 데이터베이스로 재현하지 못했으므로** 그 추측에 기대어 문구를 쓰지 않았습니다. 그 꼴의 원문은 '믿을 수 없는 입력' 으로만 테스트에 넣어 **본문에 새지 않는지**만 묶었습니다.

서버 원문은 지우지 않고 `UserErrorAlert`·`SiteErrorAlert` 과 같은 모양의 작은 글씨로 함께 둡니다 — 관리자에게 전달할 유일한 단서입니다.

## 앞의 두 함수와 합치지 않았습니다

`web/src/pages/adminErrors.ts` 에 `describeNetworkError` 를 **따로** 두었습니다. `createNetwork` 는 자기 코드 셋을 답하고 고쳐야 할 칸이 다릅니다. 하나의 `switch` 가 세 계약을 대신 답하게 만들지 않았습니다.

대신 **에러에서 코드와 메시지를 읽어내는 `refusal` 하나만** 세 경로가 공유합니다. 공용 상수 `PASS_TO_ADMIN`·`UNREADABLE_PAYLOAD`·`LOST_REQUEST` 는 **재사용만 하고 문장은 그대로 두었습니다** — 기존 13건이 글자 그대로 단언하고 있어, 통과한다는 사실이 곧 바뀌지 않았다는 증명입니다. 화면 쪽도 같은 모양입니다: `AdminNoticeAlert` 이 배치를 맡고 `NetworkErrorAlert` 은 어느 `describe*` 에 물을지만 정합니다. 모르는 코드는 **서버 메시지로 되돌아갑니다** — 나중에 서버에 거절 하나가 늘어도 삼켜지지 않습니다.

## 확인

`web/test/adminErrors.test.mjs` 가 이 성질을 고정합니다(191 → **199**건).

순수 함수 테스트에 더해 **실제 프로덕션 배선**을 확인했습니다 — `npm run build` 의 dist 를 `/api/v1/me`·`/api/v1/sites`·`/api/v1/networks` 를 흉내 낸 임시 서버에 올리고 headless Chrome 으로 `/admin?section=networks` 의 추가 칸을 채워 「추가」 를 눌러 Alert DOM 을 읽었습니다. 에러 객체는 손으로 만든 대역이 아니라 `api()` 가 HTTP 응답에서 만든 진짜 `APIError` 입니다. 5개 시나리오 전부 기대대로였고, 변경을 되돌린 대조에서는 `CIDR is invalid`·`SQLSTATE`·`relation`·`invalid cidr value` 가 Alert 본문에 그대로 떠 4건이 실패했습니다.

릴리스 검증은 이전과 같습니다 — `go test`·`go vet`, sdk `npm audit`·typecheck·테스트 27건·build, 콘솔 `npm audit`(`found 0 vulnerabilities`)·`eslint`·테스트 199건·`npm run build` 모두 통과합니다.

## 범위

바꾼 것은 프로덕션 2파일(`web/src/pages/adminErrors.ts`, `AdminPage.tsx` 의 Alert 한 곳)과 테스트 1파일입니다. **서버는 손대지 않았습니다** — `admin.go` 가 검증의 정본으로 남습니다. `describeUserError`·`describeSiteError`·`AdminNoticeAlert` 과 나머지 Alert 들은 각각 다른 핸들러의 코드 집합이라 이번에 포함하지 않았습니다. Go·API 변경과 데이터베이스 마이그레이션은 없습니다.
