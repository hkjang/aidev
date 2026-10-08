# LDAP User Federation 가이드

ReSSO의 User Federation은 Realm별로 여러 LDAP/Active Directory 공급자를 등록하고 우선순위 순서로 사용자를 인증합니다. 로컬 계정은 기존 방식으로 인증되며, LDAP에 연결된 사용자는 로컬 비밀번호로 우회하지 않습니다.

## 개발 중 디렉터리 연동 테스트

통합 테스트는 PostgreSQL과 디렉터리 두 개(평문, TLS)를 필요로 합니다. 없으면 60여 개가 건너뛰어지는데, 건너뛴 테스트도 `go test`는 `ok`로 보고하므로 로컬에서는 초록이고 CI에서 실패하는 일이 생깁니다. `make test`가 건너뛴 개수를 알려줍니다.

```bash
eval "$(scripts/test-services.sh)"   # 서비스 기동 + 환경변수 설정
go test ./internal/...
scripts/test-services.sh --stop      # 정리
```

CI도 같은 스크립트를 실행하므로, 로컬에서 통과한 것과 CI가 검증하는 것이 갈라지지 않습니다.

## 권장 구성 순서

1. 관리 → User Federation에서 Realm을 선택합니다.
2. `Other / OpenLDAP` 또는 `Active Directory` 프리셋을 선택합니다.
3. Connection URL, Bind DN/Credential, Users DN을 입력합니다.
4. 조직 CA를 PEM으로 넣고 LDAPS 또는 StartTLS를 사용합니다.
5. 연결 테스트 후 전용 시험 사용자의 인증 테스트를 수행합니다.
6. 수동 전체 동기화를 실행하고, 완료된 뒤 결과와 사용자 화면의 `LDAP` 소스를 확인합니다.
7. 필요할 때만 자동 동기화 주기와 누락 사용자 `DISABLE` 정책을 켭니다.

Bind Credential은 Data Encryption Keyring을 이용한 AES-256-GCM envelope로 PostgreSQL에 저장됩니다. 조회 API는 Secret을 반환하지 않고 설정 여부만 제공합니다.

## 주요 설정

| 설정 | 의미 |
|---|---|
| Priority | 작은 숫자의 공급자부터 JIT 로그인 사용자를 검색합니다. |
| Connection URL | `ldap://host:389` 또는 `ldaps://host:636` 형식입니다. URL 자격증명과 Query는 허용하지 않습니다. |
| StartTLS | `ldap://` 연결을 TLS로 승격합니다. `ldaps://`와 동시에 사용하지 않습니다. |
| Users DN | 사용자 검색 Base DN입니다. |
| Username/RDN/UUID attribute | 로그인 ID, DN 구성 속성, 변경되지 않는 외부 식별자입니다. AD는 보통 `sAMAccountName`, `cn`, `objectGUID`입니다. |
| User object classes/filter | 객체 클래스는 AND 조건으로, 추가 필터는 RFC 4515 필터로 결합됩니다. 로그인 입력은 Escape 후 결합됩니다. |
| Import users | 전체/주기 동기화를 허용합니다. 끄면 전체 동기화는 거부되고 JIT shadow 등록만 사용할 수 있습니다. |
| Sync registrations | 아직 가져오지 않은 LDAP 사용자를 첫 로그인 때 ReSSO shadow 사용자로 등록합니다. |
| Missing user action | `KEEP`은 유지하고 `DISABLE`은 성공한 전체 동기화에서 사라진 사용자를 비활성화하고 세션을 종료합니다. 일부 사용자 동기화가 실패하면 안전을 위해 비활성화를 실행하지 않습니다. 디렉터리가 사용자를 하나도 반환하지 않은 경우에도 실행하지 않습니다. |
| Batch size | LDAP paged search 크기입니다. 50~5000 범위입니다. |
| Sync period | 0은 끔, 그 외에는 300~604800초입니다. |

ReSSO의 shadow 사용자는 SSO Session, Role, Audit의 안정적인 관계형 참조를 위한 최소 로컬 레코드입니다. 비밀번호는 저장하지 않으며 실제 인증은 매번 원본 LDAP Bind로 수행합니다.

## Edit mode

| 모드 | 동작 |
|---|---|
| `READ_ONLY` | 프로필과 비밀번호 변경을 거부하고 원본 디렉터리 변경을 안내합니다. |
| `WRITABLE` | email/displayName 변경을 LDAP Modify로 전달합니다. OpenLDAP은 Password Modify extended operation, AD는 TLS 연결에서 `unicodePwd` 변경을 사용합니다. |
| `UNSYNCED` | 프로필의 로컬 변경을 허용하고 이후 동기화에서 덮어쓰지 않습니다. 인증은 계속 LDAP에서 수행하며 비밀번호는 원본에서 변경합니다. |

AD 비밀번호 변경은 반드시 LDAPS 또는 StartTLS가 필요합니다. 디렉터리 ACL이 서비스 계정의 사용자/비밀번호 변경을 허용해야 합니다.

## Group → Realm Role

사용자 엔트리의 `memberOf` 값을 직접 Realm Role로 매핑합니다. 한 줄에 다음 형식으로 입력합니다.

```text
CN=Admins,OU=Groups,DC=company,DC=local => realm-admin
Developers => developer
```

전체 Group DN 또는 첫 RDN의 CN으로 일치시킬 수 있습니다. 대상 Realm Role은 먼저 생성되어 있어야 합니다. ReSSO가 추가한 Federation Role만 추적해 제거하므로 관리자가 직접 할당한 동일 Role을 동기화가 임의로 제거하지 않습니다.

## 전체 동기화 실행

전체 동기화는 디렉터리 전체를 조회하고 사용자마다 갱신하므로 규모에 따라 수 분이 걸립니다. 요청은 즉시 반환되고 동기화는 백그라운드에서 계속됩니다.

- 관리 화면은 진행 중에는 `동기화 중`을 표시하고 자동으로 상태를 갱신하며, 완료되면 조회·추가·갱신·실패 건수를 보여줍니다.
- 같은 공급자에 대해 동기화가 진행 중이면 새 실행은 거부됩니다. 예약 동기화도 같은 규칙을 따르며, 겹치는 경우 실패가 아니라 건너뜀으로 기록됩니다.
- 서버가 중단되어 실행이 끊기면 30분 뒤 자동으로 해제되어 공급자가 잠긴 상태로 남지 않습니다.
- 시작과 완료(성공·실패, 건수 포함)는 모두 감사 이벤트로 기록됩니다.

REST API로 실행할 때는 `POST /api/admin/v1/realms/{realmID}/user-federations/{federationID}/sync`가 `202`와 `{status, message}`를 반환합니다. 결과는 공급자 목록의 `last_sync_status`, `last_sync_at`, `last_sync_added`, `last_sync_updated`, `last_sync_failed`에서 확인합니다. 진행 중 재호출은 `409`입니다.

## 장애 및 보안 동작

- 연결된 LDAP 사용자는 공급자 비활성화 또는 LDAP 장애 시 인증에 실패하며 로컬 비밀번호로 fallback하지 않습니다.
- 잘못된 LDAP 비밀번호도 Realm 잠금 정책의 실패 횟수에 포함됩니다.
- 연결·인증 테스트, 설정 변경, 동기화는 감사 이벤트에 기록되지만 비밀번호는 기록하지 않습니다.
- LDAP 작업은 TLS 1.2 이상, 인증서 검증, 연결/작업 제한시간을 사용합니다.
- `DISABLE`은 LDAP 조회 전체가 성공하고 사용자별 실패가 0건일 때만 적용됩니다.
- 디렉터리가 사용자를 **하나도** 반환하지 않았는데 ReSSO에는 해당 Provider의 활성 계정이 있으면 비활성화를 실행하지 않고 동기화를 실패로 기록합니다. 결과가 비어 있는 것은 실제로 전원이 디렉터리를 떠난 경우와, `Users DN` 오타·이름이 바뀐 Base·Bind 계정이 해당 하위 트리를 읽을 권한을 잃은 경우가 구분되지 않기 때문입니다. 후자에서 비활성화를 실행하면 다음 예약 동기화 한 번으로 조직 전체가 로그인할 수 없게 됩니다. `last_sync_error`와 `LDAP_FEDERATION_SYNC` 감사 이벤트에서 확인하고 검색 설정을 점검하세요.
- 공급자 삭제는 연결 사용자가 있으면 기본 거부됩니다. 명시적 연결 해제를 선택하면 사용자를 비활성화하고 세션과 Federation Role을 정리한 뒤 비활성 Local 계정으로 보존합니다.
- **연결 해제는 되돌릴 수 없습니다.** 남은 계정은 Local이므로, 같은 디렉터리를 새 공급자로 다시 연결해도 이름이 겹치는 계정마다 `username ... already belongs to a local account or another federation`으로 거부됩니다. 새 공급자는 그 사람들을 하나도 가져오지 못하고, 남은 길은 계정마다 비밀번호를 재설정해 Local로 쓰는 것뿐입니다. **Connection URL·Users DN·속성 이름을 포함한 모든 설정은 제자리에서 수정할 수 있으므로, 설정을 고치려고 삭제하지 마세요.**

현재 버전은 direct `memberOf`만 처리합니다. 중첩 Group 탐색, Kerberos/SPNEGO, Changed Users Sync, LDAP 연결 Pool은 후속 확장 범위입니다.
