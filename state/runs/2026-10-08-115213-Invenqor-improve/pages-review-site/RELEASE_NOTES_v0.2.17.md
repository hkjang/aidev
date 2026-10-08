# Invenqor Server·Agent v0.2.17 릴리즈 노트

릴리즈 일자: 2026-08-25
호환 Agent: v0.2.17 (Linux·Windows)

운영 중 보고된 세 가지 결함을 고치고, Windows Agent가 조용히 죽는 원인을 특정할 수
있게 만듭니다. 세 결함의 공통점은 **실제로 일어난 일과 다른 것을 보고하고 있었다는
점**입니다.

## 1. 모든 ingest가 SQLSTATE 42883으로 실패했습니다

중복 탐지 질의가 `payload_json LIKE`를 사용했습니다. 이 컬럼은 PostgreSQL에서
JSONB이고 JSONB에는 LIKE 연산자가 없으므로, Server 로그에
`operator does not exist: jsonb ~~ unknown`이 계속 기록되고 있었습니다.

SQLite 대체 모드에서는 같은 컬럼이 TEXT라 동일한 문장이 정상 동작합니다. 이 저장소의
테스트는 모두 SQLite에서 실행되므로 **전부 통과하는 동안 운영에서는 실패**하고
있었습니다.

- `CAST(... AS TEXT)`로 두 엔진이 모두 받는 하나의 문장이 되었습니다.
- 마이그레이션에서 JSONB 컬럼 목록을 읽어, 캐스트 없이 텍스트 연산자를 적용한 코드를
  찾아내는 테스트를 추가했습니다. 결함을 되살려 실제로 잡히는 것을 확인했습니다.
- **함께 고친 것**: 이 LIKE는 payload 전체를 훑으므로 machine identifier가 디스크
  시리얼 같은 다른 필드에 우연히 들어 있어도 중복으로 판정했고, 무관한 두 호스트의
  병합을 제안했습니다. 이제 LIKE는 사전 필터일 뿐이고 식별 필드를 정확히 비교합니다.

## 2. 재기동 후 Keycloak 로그인 버튼이 아무 설명 없이 사라졌습니다

client secret은 인스턴스의 master key로 봉인되어 저장됩니다. 교체된 pod가 이를
복호화하지 못하면 인증 방식 조회가 HTTP 500을 반환했고, 로그인 화면의 요청 처리는
이 실패를 삼켰습니다. 그 결과 **버튼만 사라지고 어디에도 이유가 남지 않았습니다.**

- "저장된 secret을 복호화할 수 없음"을 "secret 없음"과 구분합니다. 전자는 로그인
  화면에 조치와 함께 표시되고, 진단 로그(`keycloak` 구성요소)에
  `KEYCLOAK_SECRET_UNREADABLE`로 기록됩니다.
- 원인은 대부분 **replica들이 `master.key`가 있는 상태 디렉터리를 공유하지 않는
  것**입니다. 상태 디렉터리를 RWX 공유 볼륨으로 마운트하거나, 모든 인스턴스에 같은
  키 파일을 주고 `INVENQOR_MASTER_KEY_FILE`에 그 경로를 지정한 뒤(각 Pod에 마운트한
  Secret이 이 방식입니다) secret을 다시 저장하십시오.

  *정정: 이 문서의 최초 판은 `INVENQOR_MASTER_KEY`에 값을 설정하라고 안내했습니다.
  그런 변수는 없습니다 — 실제 변수는 `INVENQOR_MASTER_KEY_FILE`이고 값이 아니라
  파일 경로를 받습니다.*

## 3. MCP 도구 호출이 파라미터 검증에서 계속 실패했습니다

인자를 `DisallowUnknownFields`로 Go 구조체에 직접 디코딩했습니다. 언어 모델은 스키마
컴파일러가 아니라 예측으로 JSON을 쓰므로 `"limit": "50"`(문자열 숫자),
`"true"`(문자열 불리언), 여분의 키 하나가 일상적으로 나옵니다. 그 전부가 거부되었고,
거부 메시지는 `asset_search arguments are invalid` — **어느 파라미터인지도 무엇이
문제인지도 말하지 않으므로 호출자는 고칠 방법이 없어 같은 호출을 반복**했습니다.

- 의미가 명확한 형태는 그대로 받습니다: 문자열 숫자, `"true"`/`"false"`/`1`/`0`,
  `10.0` 같은 정수형 실수.
- 거부할 때는 **파라미터 이름, 받은 값과 그 JSON 타입, 허용 범위, 오타의 근접 후보,
  전체 허용 파라미터 목록**을 한 번에 알려줍니다. 문제가 여러 개면 모두 함께 보고하므로
  한 번의 응답으로 호출을 고칠 수 있습니다.

  ```text
  asset_search: "limit" must be an integer between 1 and 100, received "many" (string);
  unknown parameter "limitt"; did you mean "limit"?.
  Accepted parameters: include_observations, limit, offset, q, status, type
  ```

- **MCP 고도화**: 검색과 Agent 목록이 `has_more`와 `next_offset`을 반환합니다.
  이전에는 꽉 찬 페이지와 마지막 페이지를 구분할 수 없어 호출하는 Agent가 일찍
  멈추거나 끝없이 페이징했습니다.

## 4. 수집기 하나가 죽어도 Agent 전체가 죽지 않습니다

서비스는 실행 중으로 보고되는데 수집 주기가 한 번도 완료되지 않고, 로그에는 기동 4줄
뒤 침묵, 그리고 28초 뒤 같은 4줄이 반복되는 사례를 확인했습니다. 프로세스가 첫 수집
도중 죽고 Service Control Manager가 재시작하는 것인데, **이유가 아무 데도 남지
않았고 등록조차 되지 않았습니다.**

이 Agent의 설계 원칙은 "점진적으로 degrade한다" — 수집기 하나가 실패해도 나머지는
수집해서 보낸다 — 입니다. 릴리즈 프로필의 `panic = "abort"`가 이 원칙을 정확히 반대로
만들고 있었습니다. 수집기 한 곳의 panic이 프로세스 전체를 즉시 종료시켜, **아무것도
수집되지 않고 전송되지 않고 등록되지 않았습니다.**

- 릴리즈 프로필을 `panic = "unwind"`로 바꿨습니다. panic한 수집기는 그 수집기의
  오류로 보고되고, 나머지 수집기의 결과는 정상적으로 전송됩니다. 크래시 원인을
  모르는 상태에서도 **호스트는 등록되고 인벤토리는 올라옵니다.**
- 원인을 특정하기 위한 계측도 함께 넣었습니다.
  - **panic hook**이 메시지와 위치를 tracing으로 기록합니다. 기본 hook은 표준
    오류에 쓰는데 Windows 서비스에는 표준 오류가 없어 아무것도 남지 않았습니다.
  - **수집기별로 시작 전에 이름을, 종료 후에 소요 시간과 레코드 수를 기록**합니다.
  - **실행 표지 파일**을 남겨, 정상 종료하지 않은 이전 실행을 다음 기동에서 경고로
    보고합니다.

panic이 아닌 원인 — Win32 호출의 access violation 같은 — 이라면 여전히 프로세스가
죽습니다. 그 경우 로그의 마지막 `collector started` 줄이 범인을 지목합니다.

문제가 있는 호스트에서 한 주기 뒤에 확인하십시오.

```powershell
Get-Content "$env:ProgramData\Invenqor\state\agent.log" -Tail 60
```

`the agent panicked` 줄, 또는 마지막 `collector started` 줄의 수집기 이름이
원인입니다.

## 5. 버전 변경 도구

버전 일괄 변경을 손으로 하다가 두 번 결함을 배포했습니다. `Cargo.lock`이 뒤처져
태그를 새로 체크아웃하면 `cargo build --locked`가 실패했고, Helm chart의 image tag가
따옴표 형태여서 검색에서 빠져 v0.2.8이 0.2.7 이미지를 가리켰습니다.

`scripts/bump-version.sh`가 이를 대신하며, 예상한 파일이 갱신되지 않으면 실패합니다.
문서의 `v0.2.15` 형태도 함께 처리합니다 — 단어 경계는 `v`와 `0` 사이에 성립하지 않아
표지·다운로드 URL·릴리즈 링크만 이전 버전에 남는 일이 있었습니다.

## 호환성

- 데이터베이스 마이그레이션이 없습니다.
- MCP 도구의 인자 처리가 관대해집니다. 이전에 거부되던 호출이 이제 성공합니다.
  거부되는 경우의 메시지 형식이 바뀌므로 메시지 문자열을 파싱하는 자동화가 있다면
  확인하십시오.
- `asset_search`와 `agents_list` 결과에 `has_more`·`next_offset`·`count`가
  추가됩니다. 기존 필드는 그대로입니다.
- Windows Agent가 수집기별 로그를 INFO 수준으로 남기므로 로그가 주기당 약 20줄
  늘어납니다. 8 MiB 회전 한도 안에서 한 달치 이상이 보존됩니다.
- 릴리즈 프로필이 unwind로 바뀌어 바이너리가 약간 커집니다. 동작은 바뀌지 않으며,
  이전에 프로세스를 종료시키던 panic이 이제 해당 수집기의 오류로 보고됩니다.
