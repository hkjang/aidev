# Momento v0.34.54 — 사용자를 추가하다 실패하면 한국어로 무엇을 고칠지 말해 줍니다

콘솔의 **사용자 추가·사용자 편집** 다이얼로그가 거절당했을 때 띄우는 문장이 달라집니다. 서버·API·데이터베이스는 그대로입니다.

## 가장 흔한 실패가 드라이버 원문으로 올라왔습니다

두 다이얼로그는 실패를 서버가 보낸 문장 그대로 띄웠습니다. 그래서 제일 자주 만나는 거절 — **이미 쓰고 있는 이메일** — 이 이렇게 화면에 올랐습니다.

```
ERROR: duplicate key value violates unique constraint "users_email_key" (SQLSTATE 23505)
```

`internal/httpapi/admin.go` 의 `createUser` 가 실패한 INSERT 를 `writeError(w, 409, "USER_CREATE_FAILED", err.Error())` 로 답하므로, 제약 이름과 SQLSTATE 가 **스키마에서 브라우저까지 그대로** 흘렀습니다. 읽는 사람은 거기서 '이메일이 이미 있다' 를 스스로 추측해야 했습니다.

나머지도 전부 영문이었습니다 — `you cannot grant more authority than your own`, `password must be at least 12 characters`.

## 이제 무엇을 고칠지 말합니다

| 서버가 거절한 이유 | 화면에 뜨던 것 | 이제 뜨는 것 |
|---|---|---|
| 이메일 중복 | `… unique constraint "users_email_key" (SQLSTATE 23505)` | 이미 등록된 이메일입니다. 다른 이메일을 쓰거나, 사용자 목록에서 기존 계정을 찾아 편집하세요. |
| 비밀번호 규칙 | `password must be at least 12 characters` | 비밀번호가 규칙에 맞지 않습니다. + 폼이 적어 둔 **그 규칙 그대로** |
| 내 권한보다 높은 권한 부여 | `you cannot grant more authority than your own` | 내 권한보다 높은 권한은 줄 수 없습니다. 권한을 내 계정의 권한 이하로 고르세요. |
| 내 권한보다 높은 계정 편집 | 같은 영문 한 문장 | 내 권한보다 높은 계정은 편집할 수 없습니다. 목록이 오래되었을 수 있으니 새로 고친 뒤 다시 시도하세요. |

새 순수 모듈 `web/src/pages/adminErrors.ts` 의 `describeUserError` 가 이 판단을 혼자 합니다. `components/queryError.ts` 의 선례를 따라 **코드를 shape 으로 읽고**, 모르는 코드는 서버 메시지로 되돌아갑니다 — 나중에 서버에 거절 하나가 늘어도 삼켜지지 않고 그대로 보입니다. 전역 `onError` 가 없으므로(`main.tsx`) 이 모듈이 말하지 않는 것은 어디에서도 말해지지 않습니다.

## 한 코드에 원인이 둘 실려 오는 곳

세 자리는 코드만으로는 갈라지지 않아 **문장을 보고** 갈랐습니다.

- `INVALID_USER` — 역할 문제와 비밀번호 문제가 같은 코드로 옵니다. 역할 쪽 문장에도 "password" 라는 낱말이 있어, `PasswordProblem` 이 실제로 내는 `password must be` 로 갈랐습니다.
- `ROLE_ABOVE_CALLER` — 상위 계정 관리 거절에만 나오는 `administer` 로 갈랐습니다. 고쳐야 할 것이 다르니 안내도 다릅니다.
- `USER_CREATE_FAILED` — 유일성 위반 표지(`duplicate key` / `unique constraint` / `23505`)로 갈랐습니다.

**409 라는 것만으로 중복이라고 단정하지 않습니다.** 그 INSERT 는 무엇이 틀려도 409 로 답하므로, 상태 코드로 갈랐다면 끊긴 연결을 '이미 등록된 이메일' 이라고 설명하게 됩니다. 표지가 없으면 정직하게 "계정을 만들지 못했습니다" 라고 말하고 서버 문장을 작은 글씨로 함께 둡니다.

## 확인

`web/test/adminErrors.test.mjs` 가 순수 함수를 고정합니다(178 → **184**건).

순수 함수 테스트만으로 끝내지 않고 **실제 프로덕션 배선**도 확인했습니다. `npm run build` 의 `dist` 를 `/api/v1/me`·`/api/v1/users` 를 흉내 낸 임시 서버에 올리고, headless Chrome 으로 `/admin?section=users` 의 생성 다이얼로그를 실제로 채워 「생성」을 누른 뒤 Alert 의 DOM 을 읽었습니다. 에러 객체는 손으로 만든 대역이 아니라 `api()` 가 HTTP 응답에서 만든 진짜 `APIError` 입니다.

| 서버 응답 | 결과 |
|---|---|
| 이메일 중복 409 | 한국어 안내, 원문 표지 5개 **모두 없음** |
| 해시 실패 500 | "계정을 만들지 못했습니다" + 작은 글씨 `bcrypt: …` |
| 역할 부여 거절 / 상위 계정 거절 | 서로 다른 문구 |
| `WEAK_PASSWORD` | 폼 helperText 와 **같은** 규칙 문장 |
| 모르는 코드 | 서버 메시지 그대로 |

변경을 치우고 이전 상태로 되돌려 같은 시나리오를 돌리면 `users_email_key`·`SQLSTATE`·`23505` 가 Alert 에 그대로 떠 5건이 실패합니다 — 고친 것이 실제로 보이던 문제였음을 보이는 대조입니다.

## 범위

바꾼 것은 프로덕션 2파일(`web/src/pages/adminErrors.ts` 신규, `AdminPage.tsx` 의 Alert 두 곳)과 테스트 1파일입니다. 서버·`internal/auth`·성공 토스트는 손대지 않았고, 사이트 등 다른 섹션의 Alert 세 곳은 이번에 포함하지 않았습니다. Go·API 변경과 데이터베이스 마이그레이션은 없습니다.
