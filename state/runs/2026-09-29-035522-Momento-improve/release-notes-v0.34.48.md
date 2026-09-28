# Momento v0.34.48 — 자기 계정을 편집하면 저장한 뒤에야 영문 오류를 만났습니다

## 서버가 이미 거절하기로 한 조작을 다이얼로그가 계속 제안했습니다

`updateUser`(`internal/httpapi/admin.go`)는 요청을 보낸 사람이 **자기 계정에 대해** 하는 세 가지를 **400** 으로 거절합니다.

```
SELF_DISABLE   id == p.ID && in.Active != nil && !*in.Active
SELF_ROLE      id == p.ID && current != in.Role
SELF_PASSWORD  id == p.ID && in.Password != ""
```

그런데 「사용자 편집」 다이얼로그는 **자기 행을 열어도** 권한 select 와 「계정 활성화」 체크박스를 그대로 열어 두었습니다. 자기 권한을 `viewer` 로 고르거나 자기 계정의 체크를 풀 수 있었고, 그것이 저장될 수 없는 값이라는 사실은 **저장을 누른 뒤에** 한국어 화면 위의 영문 오류로 전달됐습니다.

세 제약 중 비밀번호 재설정만 거울이 있었습니다 — 그것도 다이얼로그 본문에 `edit.id !== user?.id` 로 **인라인으로 박혀** 있어서, 나머지 둘이 빠져 있다는 것이 한눈에 보이지 않았습니다.

## 판정을 한 곳으로 모았습니다

세 분기가 모두 **"자기 계정이냐"** 하나로 갈립니다. `web/src/pages/roleScope.ts` 에 순수 함수 **`selfAccountLimits(callerId, targetId)`** 를 두고, 권한 select·활성화 체크박스·비밀번호 필드가 **같은 답**을 읽습니다.

- **권한 select** — `disabled` 로 두고 현재 역할을 그대로 보여 줍니다. 아래에 「자기 계정의 권한은 바꿀 수 없습니다.」 한 줄이 붙습니다.
- **활성화 체크박스** — `edit.active && !limits.canDeactivate`. **끄는 것만** 막습니다.
- **비밀번호 재설정** — 인라인 id 비교를 `limits.canResetPassword` 로 바꿨습니다. 동작은 이전과 같고, 판정하는 자리만 옮겼습니다.

안내 문구는 select 의 `helperText` 가 아니라 `Typography variant="caption"` 입니다 — MUI 는 `helperText` 를 disabled 된 필드와 **함께 흐리게** 만드는데, 이유를 말하는 유일한 줄이 그렇게 되어서는 안 됩니다.

## 서버보다 넓지도 좁지도 않습니다

거울이 규칙보다 넓으면 그 자체가 결함입니다. 서버가 **막지 않는** 것은 화면도 막지 않습니다.

- **역할을 그대로 두고** 다른 항목만 고쳐 저장하는 것 — 서버는 `current != in.Role` 일 때만 거절하므로, select 가 비활성이어도 폼의 나머지는 그대로 저장됩니다.
- **중지된 자기 계정을 다시 켜는 것** — `SELF_DISABLE` 은 `!*in.Active` 일 때만 걸립니다. 그래서 이미 꺼져 있는 체크박스는 **열려 있습니다.**
- **비밀번호 칸을 비워 두는 것** — 이전과 같습니다.

`callerId` 가 아직 없는 상태(세션이 정해지기 전 `useAuth().user` 가 `null`)에서는 아무것도 제한하지 않습니다. 서버는 주체 없이 이 경로에 닿지 않으므로, 여기서 추측하면 **남의 계정**을 잠글 뿐입니다.

**서버와 `internal/auth`, 역할 서열 함수(`roleRank`·`canAdministerRole`·`assignableRoles`), 사용자 생성 다이얼로그는 손대지 않았습니다.** 서버 검사가 여전히 권한의 정본입니다. 프로덕션 파일은 **두 개**입니다.

## 테스트

`web/test/roleScope.test.mjs` 가 순수 함수를 고정합니다 — 자기 계정이면 셋 다 막히는 것, 남의 계정이면 셋 다 열리는 것, `callerId` 가 없을 때 아무것도 잠기지 않는 것. 테스트 수는 **130 → 135** 입니다.

순수 함수만으로는 **배선**을 증명하지 못하므로 거기서 멈추지 않았습니다. `npm run build` 의 dist 를 `/api/v1/me`·`/api/v1/users` 를 흉내 낸 임시 서버에 올리고 headless Chrome 으로 `/admin?section=users` 를 몰아, 자기 행과 남의 행을 열어 DOM 을 읽었습니다.

- **자기 행** — `{"roleDisabled":true,"roleClickOpensMenu":false,"roleNote":true,"checkboxDisabled":true,"activeNote":true,"passwordFieldShown":false}`
- **남의 행** — 여섯 값이 모두 반대입니다.
- **수정 전**의 같은 시나리오는 자기 행에서 `{"roleDisabled":false,"roleClickOpensMenu":true,"checkboxDisabled":false,…}` 였습니다 — 서버가 400 으로 거절할 두 조작을 그대로 제안하고 있었습니다.
- **중지된 자기 계정**(`active:false`)으로도 띄워, 체크박스가 `disabled:false` 로 열려 **다시 켤 수 있는** 것을 확인했습니다.

## 그 밖에

Go 쪽 변경은 없습니다. Database Migration, SDK, API, 환경변수 변경 없음.
