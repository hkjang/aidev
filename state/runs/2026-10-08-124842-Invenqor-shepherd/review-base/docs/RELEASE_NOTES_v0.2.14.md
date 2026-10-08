# Invenqor Server·Agent v0.2.14 릴리즈 노트

릴리즈 일자: 2026-08-21
호환 Agent: v0.2.14 (Linux x86_64·aarch64, Windows x86_64)

v0.2.14는 Windows Agent가 정상 등록·수집됐는데도 관리 화면에서 **“운영체제 확인
전”**으로 남던 계약 불일치를 수정하고, 수천 개의 프로세스 관찰을 운영자가 실제로
관리할 수 있는 **주요 소프트웨어 제품 단위**로 자동 정규화합니다. 별도 매핑 테이블을
사람이 유지하지 않아도 설치·실행 상태, 버전, 호스트와 판별 근거를 한 화면과 API에서
확인할 수 있습니다.

Kubernetes 배포용 공개 Server 이미지는
`ghcr.io/hkjang/invenqor-server:0.2.14`로 제공하며 `linux/amd64`와
`linux/arm64`를 지원합니다. 태그 릴리즈 파이프라인은 발행 직후 익명 조회와 두
플랫폼 manifest를 자동 검증합니다.

## 1. Windows 운영체제가 확인되지 않던 직접 원인

Windows Agent의 `system` 레코드는 다음과 같이 올바른 정보를 이미 전송하고
있었습니다.

```json
{
  "os_family": "windows",
  "os_name": "Windows 11 Enterprise",
  "os_version": "24H2",
  "os_build": "26100.4652",
  "architecture": "x86_64"
}
```

그러나 Server의 Agent 메타데이터 추출기는 Linux 레코드의
`os_release.pretty_name`과 `os_release.name`만 읽었습니다. 그 결과 이벤트와 자산은
정상 처리되는데 `agents.os_name`만 비어 있었고, 화면은 이를 첫 인벤토리 미수신처럼
표시했습니다. 이는 Windows 레지스트리 수집 실패나 Agent 등록 실패가 아니라
**Agent와 Server 사이의 운영체제 필드 계약 불일치**였습니다.

v0.2.14에서는 양쪽을 보완했습니다.

- Server는 최상위 `os_name`을 우선 사용하고, Linux식 `os_release`를 다음 순서의
  fallback으로 사용합니다. 이름을 읽지 못했더라도 `os_family=windows`가 확인되면
  최소한 `Windows`로 정직하게 표시합니다.
- Server는 전체 snapshot을 우선 파싱하고, 업그레이드 후 `system` 레코드가 delta의
  `changes.added` 또는 `changes.updated`로 전송되는 경우도 이어서 파싱합니다. 기존
  등록 Windows Agent를 새로 등록하지 않고도 첫 변경 inventory에서 자동 복구합니다.
- raw `system` 원천은 이미 저장됐지만 `agents.os_name`만 비어 있는 v0.2.13 장비는
  변경 inventory를 기다리지 않습니다. v0.2.14 Server가 첫 heartbeat에서 저장된
  최신 원천을 읽어 운영체제·호스트명·아키텍처를 자동 복구합니다. 필드가 이미
  완전한 Agent는 이 fallback 조회를 수행하지 않습니다.
- Agent는 Windows 전용 최상위 필드를 유지하면서 `os_release` 호환 객체도 함께
  전송합니다. 객체에는 `id`, `name`, `pretty_name`, `version_id`, `build_id`가
  포함됩니다.
- Windows 11은 레지스트리 `ProductName`이 여전히 Windows 10으로 보고되는 특성을
  고려해 build 22000 이상에서 이름을 Windows 11로 보정합니다. Windows Server
  제품명은 바꾸지 않습니다.
- `DisplayVersion`이 없는 구형 Server 에디션에서는 build를 버전 fallback으로
  사용하되 존재하지 않는 버전을 추측하지 않습니다.

이중 표현은 롤링 업그레이드의 양방향 호환을 위한 것입니다. 새 Server는 기존
Windows Agent의 최상위 필드와 저장된 원천을 이해하고, 새 Agent는 기존 Server가
이해하던 `os_release` 형태도 제공합니다. 운영 중인 기존 Agent는 첫 heartbeat 또는
다음 `system` 변경이 처리되면 별도 재등록이나 상태 디렉터리 삭제 없이 운영체제
정보가 갱신됩니다.

### 1.1 Windows 네이티브 수집·기동 안정성

릴리즈 전 실제 Windows 호스트 검증에서 계정 수집을 켠 GNU Agent가
`0xC0000005` 접근 위반으로 종료되는 문제도 발견해 함께 수정했습니다.
`NetLocalGroupEnum`과 `NetLocalGroupGetMembers`의 재개 핸들은 x64에서 64비트인
`PDWORD_PTR`인데 32비트 포인터로 선언돼, 로컬 그룹을 읽을 때 네이티브 API가
스택 경계를 넘겨 쓰는 것이 직접 원인이었습니다. 두 API만 pointer-sized 핸들로
교정하고, 별도로 32비트 `PDWORD`가 맞는 `NetUserEnum` 계약은 유지했습니다.

또한 네이티브 Windows Server가 bootstrap master key와 spool·Agent update
메타데이터를 원자 저장한 뒤 디렉터리에 `File.Sync()`를 호출하면 Windows가
`Access is denied`를 반환해 시작하지 못하던 문제를 플랫폼별 durable-write
경계로 수정했습니다. Linux에서는 rename 후 디렉터리 `fsync`를 계속 수행하고,
Windows에서는 디렉터리의 존재·타입·핸들 종료를 검증한 뒤 지원되지 않는 sync만
생략합니다. 파일 본문의 flush와 원자 교체 규칙은 그대로 유지됩니다.

CI는 Windows 배포 ZIP에 들어가는 것과 동일한 GNU Agent와 네이티브 Go Server를
실제 Windows runner에서 함께 실행합니다. 로컬 계정·그룹 API의 ABI 회귀 테스트와
Server 조기 종료 로그를 포함해, 자동 등록·전체 수집·전송·운영체제 저장·주요
소프트웨어 정규화가 모두 끝나야 통과합니다.

## 2. 프로세스 나열을 주요 소프트웨어 관리로 전환

원시 프로세스는 사고 조사 증거로 유용하지만 CMDB 구성 항목으로는 부적합합니다.
PostgreSQL 한 인스턴스가 여러 PID를 만들고 Windows PC 한 대에 수백 프로세스가
있을 수 있으므로, 각 PID를 관리 대상으로 보여주면 중요한 데이터베이스·웹 서버·보안
Agent가 오히려 묻힙니다.

Server는 매 inventory 처리 트랜잭션 안에서 같은 호스트의 다음 증거를 교차
검증합니다.

| 증거 | 예 | 판단에 사용하는 값 |
|---|---|---|
| 프로세스 | `postgres`, `sqlservr.exe`, `w3wp.exe` | 정규화 이름, 실행 파일 basename·경로 |
| 서비스 | `postgresql@16-main`, `MSSQLSERVER`, `W3SVC` | 서비스명·표시명, 시작/실행 상태, 실행 경로 |
| 설치 패키지 | `postgresql-16`, Microsoft SQL Server 2022 | 패키지명, 버전, Windows publisher/설치 범위 |

내장 카탈로그 `2026.08.1`은 51개 운영상 중요한 제품을 포함합니다. 데이터베이스,
검색, 웹·역방향 프록시, 애플리케이션 서버, 컨테이너·Kubernetes, 메시지 브로커,
관측·보안, 백업, CI/CD, 원격 접속과 게스트 도구에 더해 PC 생산성,
브라우저, 협업, 언어 런타임, Endpoint 관리 영역까지 다룹니다. 대표 예는
PostgreSQL, Microsoft SQL Server, NGINX, IIS, Docker, Kubernetes Node, Kafka,
Prometheus, Splunk, CrowdStrike, Microsoft Office/Microsoft 365, Chrome, Edge,
Firefox, Teams, Zoom, Java, .NET, Microsoft Endpoint Configuration Manager(MECM),
Tanium, BigFix, Elastic Agent와 Wazuh입니다. 따라서 Windows PC와 서버를 하나의
제품 인벤토리 기준으로 비교할 수 있습니다.

### 2.1 제품 단위 자산과 호스트 연결

- 같은 호스트에서 같은 제품의 여러 PID·서비스·패키지는 하나의
  `software_product` 자산으로 합칩니다.
- 제품 자산 키는 Agent와 `product_key`에 대해 안정적이므로 같은 inventory를 다시
  처리해도 중복 자산·원천·관계가 생기지 않습니다.
- 제품은 자동 `runs_on` 관계로 실제 host 자산과 연결되고 host의 environment를
  상속합니다.
- 패키지 또는 서비스가 있으면 `install_state=installed`, 프로세스만 관찰되면
  `observed`로 구분합니다.
- 프로세스나 활성 서비스가 있으면 `runtime_state=running`, 중지 상태가 확인된
  서비스만 있으면 `stopped`, 설치만 확인되면 `unknown`입니다.
- 증거가 사라지면 다음 inventory에서 상태를 재계산합니다. 마지막 증거가 사라진
  제품은 논리 제거되고 `runs_on` 관계도 종료되며, 다시 나타나면 같은 자산이
  재활성화됩니다.
- 생성·변경·제거는 `automatic_software_catalog` 사유의 자산 변경 이력으로
  남습니다.

정규화는 Agent 이벤트와 같은 DB 트랜잭션에서 수행됩니다. 여러 Server Pod가 같은
PostgreSQL을 사용할 때 다른 Pod가 원시 증거만 갱신되고 제품 상태는 이전인 중간
상태를 볼 수 없습니다.

### 2.2 설명 가능한 확신도와 오탐 방지

판별 결과는 결론만 저장하지 않습니다. 제품명·제조사·역할·버전과 함께 매칭된
서비스명, 프로세스명, 패키지명, 인자를 제거한 실행 경로, 원천 자산 ID와 카탈로그
버전을 보존합니다. 운영자는 어떤 신호 때문에 제품으로 판단됐는지 상세 화면에서
확인할 수 있습니다.

오탐을 줄이는 원칙은 다음과 같습니다.

- 카탈로그에 없는 일반 프로세스는 임의의 소프트웨어 제품으로 승격하지 않습니다.
  `explorer.exe`, `agent.exe` 같은 일반 이름은 원시 증거로만 남습니다.
- Chrome·Java처럼 프로세스 이름만으로 제품 주체나 설치 여부를 단정하기 쉬운
  항목은 generic process 한 개만으로 제품 자산으로 승격하지 않습니다. 패키지,
  서비스, 제품 고유 경로와 같은 더 강한 증거를 요구해 공유 runtime·임시 실행
  파일의 오탐을 억제합니다.
- 단순 부분 문자열 대신 정규화한 정확 이름과 제한된 `*` 패턴, 실행 파일 basename,
  제품 고유 경로를 사용합니다.
- PostgreSQL client/common/libs/devel/JDBC/ODBC처럼 서버 제품을 뜻하지 않는 패키지는
  명시적으로 제외합니다.
- 서비스 증거 95%, 패키지 90%, 실행 파일 86%, 경로 84%, 프로세스명 82%를
  기본으로 하며 서로 다른 종류의 증거가 일치하면 확신도를 올립니다. 상한은
  99%이고 80% 미만은 화면에서 “검토 권장”으로 분리합니다.
- 서비스 `ImagePath`는 실행 파일까지만 보존하고 뒤의 인자를 제거합니다. 비밀번호나
  Token이 들어갈 수 있는 명령행은 판별 증거에 복제하지 않습니다.
- 한 제품의 상세 증거는 최대 32개로 제한하되 전체 증거 수와 프로세스 수는 별도로
  집계합니다.

내장 카탈로그는 자동 운영을 위한 보수적 기준입니다. 미식별 프로세스를 억지로
제품화하지 않으며, 조직 전용 제품을 수동 매핑하라고 요구하지 않습니다. 미식별
제품의 자동 확장은 후속 카탈로그 릴리즈로 배포합니다.

## 3. 관리 화면과 API

좌측 메뉴에 **주요 소프트웨어** 화면을 추가했습니다.

- 식별 제품 수, 호스트별 인스턴스, 실행·중지·미확인 상태, 설치 확인, 프로세스
  자동 매핑, 높은 신뢰도와 검토 권장 건수를 제공합니다.
- 제품·호스트·버전·서비스·프로세스 검색과 역할, 실행 상태, 확신도 필터를
  제공합니다.
- 상세 Drawer에서 제조사, 역할, 호스트, 버전, 설치·실행 상태, 실행 경로와 모든
  판별 근거를 확인합니다.
- 운영 현황 Dashboard에도 주요 제품 분포를 표시합니다.
- 일반 **자산** 화면과 Dashboard는 기본적으로 원시 `process` 자산을 제외해 관리
  가능한 구성 항목 중심으로 집계합니다. 원시 데이터는 삭제하지 않으며 **프로세스
  관찰 포함**을 켜면 기존처럼 조회할 수 있습니다.

`assets.read` 권한으로 다음 API를 사용할 수 있습니다.

```http
GET /api/v1/assets/software-products
GET /api/v1/assets?scope=managed
GET /api/v1/dashboard/statistics?scope=managed
```

주요 소프트웨어 API는 `q`, `role`, `vendor`, `runtime_state=running|stopped|unknown`,
`confidence=high|review`, `limit`, `offset`을 지원합니다. 응답은 전체 요약,
상위 제품, 필터 후보, host 연결과 각 제품의 증거를 함께 반환합니다. 기존 자산
API의 기본 응답은 바뀌지 않으며 `scope=managed`를 명시할 때만 process를
제외합니다.

MCP에는 `assets.read` scope의 전용 `software_inventory` 도구를 추가했습니다.
`q`, `role`, `runtime_state`, `confidence`, `limit`, `offset`로 제품을 조회하며,
REST 관리 화면과 같은 요약·host·상태·확신도·evidence 계약을 typed
`structuredContent`로 반환합니다. 범용 `asset_search`는 일상 자산 탐색에서 원시
`process` 관찰을 기본 제외하고, 사고 조사 등 필요할 때만
`include_observations=true`로 명시적으로 조회합니다.

## 4. 검증

다음 검증을 릴리즈 게이트에 포함했습니다.

- Rust 전체 단위 테스트와 Windows release metadata 직렬화·fallback 테스트
- Server가 snapshot과 delta `added`/`updated` 모두에서 Windows metadata를 복구하고
  `agents` 테이블을 갱신하는 ingest 회귀 테스트
- v0.2.13 형식의 저장된 Windows `system` 원천과 raw 소프트웨어 증거만 남긴 뒤
  heartbeat 한 번으로 OS·주요 제품을 backfill하고, 두 번째 heartbeat에서는
  카탈로그 스캔을 반복하지 않는 업그레이드 회귀 테스트
- `x86_64-pc-windows-gnu` Agent 교차 빌드를 GitHub Actions build matrix에 추가
- Windows 로컬 그룹 API의 pointer-sized resume handle 시그니처와 실제 계정
  열거를 네이티브 회귀 테스트로 고정
- Ubuntu cross job이 Windows 배포 ZIP과 동일한 GNU 실행 파일을 만들고, 이를
  Windows GitHub runner로 전달해 Server와 함께 기동합니다. 자동 등록부터
  `system`·`process`·`service`·`software.package` 수집, 전송, 운영체제 표시와
  주요 제품 정규화까지 검증하는 네이티브 E2E를 추가했습니다.
- React production build를 Server embedded UI에 자동 동기화하고 CI에서 차이가
  없음을 확인해, 컨테이너와 독립 Server 실행 파일이 같은 화면을 제공하도록 보장
- Windows 서비스·프로세스·Uninstall 패키지 신호를 결합해 SQL Server와 IIS를
  식별하는 단위 테스트
- 알 수 없는 일반 프로세스와 PostgreSQL client 패키지가 제품으로 오탐되지 않는
  부정 테스트
- SQL Server Management Studio·Native Client·ODBC/JDBC driver·LocalDB·설치
  도구와 IIS Express를 각각 SQL Server·IIS 본체로 오인하지 않는 부정 테스트
- 제품 버전의 자연 정렬 검증(`9.4`보다 `16.0`, `1.9`보다 `1.10`을 최신으로 판정)
- Chrome·Java generic process 단독 신호를 제품으로 승격하지 않는 PC 카탈로그
  부정 테스트
- 실행 인자를 제거해 secret이 정규화 증거로 유출되지 않는 테스트
- 같은 inventory 재처리 멱등성, 증거 제거·제품 논리 제거·재활성화와 host
  `runs_on` 관계 테스트
- 실제 PostgreSQL-backed Server에 Windows 형식 inventory를 전송해 운영체제명이
  저장되고, SQL Server/IIS의 설치·실행 상태와 세 종류의 증거가 API에서 조회되며,
  `scope=managed`에서 원시 process가 숨겨지는 계약 E2E
- 소프트웨어 REST·MCP의 검색·상태·확신도 필터, 요약·host·evidence와 UTC 시각
  계약, `asset_search` 원시 process 기본 제외 테스트

## 5. 호환성 및 업그레이드

- Server 시작 시 migration `006_software_product_inventory`가 자동 적용됩니다.
  관리형 소프트웨어의 검색·요약·필터·페이징을 JSON 전체 스캔 없이 처리하는
  조회용 projection이며, 기존 원천 자산과 증거는 그대로 보존됩니다. PostgreSQL
  멀티 Pod에서는 기존 advisory lock으로 한 Pod만 migration을 수행하므로 수동 SQL
  작업이 필요하지 않습니다.
- 기존 REST 응답을 제거하거나 의미를 바꾸지 않습니다. 새 API와 `scope=managed`는
  additive 변경입니다.
- 기존 원시 process·service·software.package 자산은 삭제하지 않습니다. 새
  화면의 기본 집계에서만 process 관찰을 숨깁니다.
- v0.2.14 Server는 각 Agent의 첫 heartbeat 또는 inventory에서 저장된 현재 증거를
  자동 재조정하므로 별도 카탈로그 이관이나 수동 매핑 작업이 필요하지 않습니다.
  Agent별 카탈로그 버전 완료 마커를 기록해 변화가 없는 장비와 제품이 하나도 없는
  장비도 매 heartbeat마다 다시 스캔하지 않습니다. projection과 마커는 같은
  transaction에서 갱신되므로 여러 Server Pod에서 같은 조회 결과를 제공합니다.
- Windows 운영체제 표시는 Server와 Agent 어느 한쪽을 먼저 올려도 wire 형식이
  깨지지 않습니다. 운영 중인 기존 장비의 자동 복구를 빠르게 확인하려면 Server를
  먼저 올리고 Agent를 순차 업데이트하는 것을 권장합니다.
- 업그레이드 후 **Agent** 화면에서 Windows 운영체제, **주요 소프트웨어** 화면에서
  host 연결과 판별 근거를 확인하십시오. Agent 등록 정보나 `%ProgramData%` 상태를
  삭제하고 재등록할 필요가 없습니다.
