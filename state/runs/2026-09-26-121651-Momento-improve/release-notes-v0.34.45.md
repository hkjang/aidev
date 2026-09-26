# Momento v0.34.45 — 누를 수 있는 편집 버튼이 저장할 수 없는 편집이었습니다

## 서버가 이미 거절하기로 한 조작을 화면이 계속 내주었습니다

사용자 관리는 **모든 행에 편집 버튼**을 주고, 「사용자 추가」와 「사용자 편집」 두 다이얼로그가 **다섯 역할을 그대로** 보여 줬습니다.

서버는 그렇지 않습니다. `createUser`·`updateUser`는 `auth.RoleAbove`로 **자기보다 높은 역할의 계정**을 `ROLE_ABOVE_CALLER`(**403**)로 막습니다. v0.34.43에서 비밀번호 재설정이 붙으면서 편집 대상 계정까지 같은 방식으로 제한되었으니, 그 규칙은 이제 이 화면의 거의 모든 조작에 걸립니다.

그래서 `organization_admin`이 `super_admin` 행의 버튼을 누르면 다이얼로그가 열리고, 값을 고치고, **저장을 누른 뒤에야** 한국어 화면 위에 영문 오류를 만났습니다. 역할 select 도 마찬가지로 `super_admin`을 고를 수 있었고 결과는 같았습니다.

막힌 조작이 **실패한 조작**처럼 보였습니다 — 권한의 문제인지, 입력의 문제인지, 서비스의 문제인지 화면이 말해 주지 않았습니다.

## 할 수 없는 것은 누를 수 없습니다

- 자기보다 **높은 역할의 행**은 편집 버튼이 **비활성**이고, Tooltip 이 이유를 말합니다 — **「내 권한보다 높은 계정은 편집할 수 없습니다」**. (비활성 버튼은 포인터 이벤트를 내지 않으므로 Tooltip 이 들을 요소로 `span` 을 감쌌습니다.)
- 두 다이얼로그의 **권한 select** 는 `assignableRoles(내 역할)`만 나열합니다. `organization_admin`에게는 네 역할, `super_admin`에게는 다섯 역할입니다. **부여할 수 없는 역할은 목록에 없습니다.**

거절을 저장 뒤가 아니라 **누르기 전에** 말합니다.

## 규칙은 여전히 서버의 것입니다

같은 서열이 화면에서도 **한 곳**에서 나옵니다 — `web/src/pages/roleScope.ts` 의 `ROLE_ORDER`·`roleRank`·`canAdministerRole`·`assignableRoles`.

- `ROLE_ORDER` 는 `internal/auth/auth.go` 의 `roleRank` 와 **같은 순서**이고, 모르는 **대상** 역할은 Go 맵의 기본값처럼 **0** 이 됩니다.
- `canAdministerRole` 은 `RoleAbove(대상, 호출자)` 의 정확한 반대이되, **호출자** 역할을 모를 때만 서버보다 엄격하게 `false` 로 닫습니다. 화면이 편집을 **내주지 않는** 쪽이 무해한 방향이기 때문입니다.

**서버 검사가 여전히 경계입니다.** `internal/auth` 도 `internal/httpapi/admin.go` 도 손대지 않았고, 화면이 가린다는 이유로 느슨해진 곳은 없습니다. 이것은 그 규칙의 **거울**이지 대체물이 아닙니다.

## 테스트

`web/test/roleScope.test.mjs` 가 서버 `roleRank` 의 복제본을 두고 **5×5 순회**로 두 규칙이 갈라지지 않음을 고정합니다. 규칙을 바꿔 보는 돌연변이 세 가지 — 항상 허용, 한 단계 위까지 허용, `ROLE_ORDER` 순서 뒤집기 — 가 각각 실제로 실패하는지 확인했습니다.

빌드한 번들을 API 를 흉내 낸 서버에 올려 headless Chrome 으로 DOM 을 확인했습니다: `organization_admin` 에게는 `super_admin` 행만 비활성 버튼과 Tooltip 이고 두 select 가 네 역할, `super_admin` 에게는 비활성 행이 없고 다섯 역할 그대로입니다.

## 그 밖에

Go 쪽 변경은 없습니다. Database Migration, SDK, API, 환경변수 변경 없음.
