# Invenqor Server·Agent v0.2.6 릴리즈 노트

릴리즈 일자: 2026-07-29
호환 Agent: v0.2.6 (이번 릴리즈부터 Server와 Agent 버전을 동일하게 관리합니다)

## 이번 릴리즈가 해결하는 문제

자동 등록이 되지 않을 때 **원인을 확인할 방법이 없었습니다.** Agent가 Server에
도달조차 하지 못하면 Server에는 아무 기록도 남지 않고, Agent 쪽에서도 등록 실패와
전송 실패가 같은 문장으로 기록되었으며 실제 원인은 로그에 출력되지 않았습니다.

v0.2.6은 등록·연동 실패를 **양쪽 모두에서, 실패한 상태에서도** 확인할 수 있게
만듭니다.

## 핵심 변경

### 1. 등록이 조용히 실패하지 않습니다 (Agent)

- `server.url`이 없으면 기동 시 경고를 남깁니다. 이전에는 로그 한 줄 없이 수집만
  하고 등록을 시도하지 않아 정상 Agent와 구분할 수 없었습니다. 배포 패키지의
  `config.toml`은 `url`이 주석 처리된 상태로 제공되므로 이 경우가 가장 흔했습니다.
- 기동 시 `agent transport configured`에 적용된 Server URL, 인증 방식, 상태
  디렉터리, 등록 상태를 함께 기록합니다.
- 등록 실패와 전송 실패를 분리해 기록합니다. 이전에는 등록 실패도
  `delivery failed`로 표시되었습니다.
- 모든 실패에 안정적인 원인 코드, Server `request_id`, 조치 문구가 붙습니다.
- 운영체제 수준 원인이 로그에 나타납니다. `Connection refused`,
  `certificate has expired` 같은 근본 원인이 이전에는 유실되었습니다.
- `install.sh`는 `server.url`이 없을 때 다음 단계와 검증 명령을 출력합니다.
- `--once`는 Server가 설정돼 있는데 전달에 실패하면 종료 코드 2를 반환합니다.
  이전에는 실패해도 0이었기 때문에 설치 자동화가 실패를 감지할 수 없었습니다.

### 2. `--diagnose`: 상태를 바꾸지 않는 등록 자체 진단 (Agent)

```bash
invenqor-agent --config /etc/invenqor-agent/config.toml --diagnose
```

설정 파일 → 상태 디렉터리 → 큐 → 저장된 자격 증명 → URL 형식 → TLS →
이름 해석 → 도달성 → Server 신원 → **Server가 인식한 출처 IP** → 등록 정책 →
장비 자격 증명 유효성 순으로 점검하고, 처음 실패한 항목에 원인 코드와 조치를
함께 출력합니다. 등록·전송·자격 증명 교체를 하지 않으므로 운영 중에도 안전하며,
실패가 있으면 0이 아닌 코드로 종료합니다. `--json`으로 기계 판독 출력을 얻습니다.

### 3. `status.json`과 `--status` (Agent)

`/var/lib/invenqor-agent/status.json`에 등록 상태, 마지막 실패 코드와 Server
`request_id`, 큐 깊이, 전송 실적을 매 주기 기록합니다(권한 `0600`). 저널을
열람할 수 없는 환경에서도 원인을 확인할 수 있고, `--status`는 사람이 읽는 요약을
출력하며 정상 여부를 종료 코드로 반환합니다.

### 4. `GET /v1/agent/preflight` (Server)

자격 증명 없이 호출할 수 있는 사전 점검 API입니다. 상태를 만들지 않고 다음을
돌려줍니다.

- Server가 인식한 **출처 IP**(신뢰 프록시 판정 적용 후)
- 등록 모드, Token 요구 여부, 네트워크 허용 여부
- `would_enroll`과 거부 사유 코드 — 실제 등록 엔드포인트와 **동일한 판정 로직**을
  사용하므로 사전 점검 결과와 실제 등록 결과가 어긋나지 않습니다
- Authorization 헤더로 장비 Token을 함께 보내면 그 Token의 유효성

호출은 진단 로그에 `AGENT_PREFLIGHT_READY` 또는 `AGENT_PREFLIGHT_BLOCKED`로
기록되므로, **등록에 실패해 콘솔에 나타나지 않는 장비의 시도도** 관리자가 확인할
수 있습니다.

### 5. 잘못된 URL이 콘솔 HTML을 반환하지 않습니다 (Server)

`server.url`에 경로가 섞이면(`https://host:7070/invenqor`) 이전에는 콘솔 SPA가
HTTP 200과 HTML로 응답하고 Agent는 JSON 파싱 오류만 남겼습니다. 이제 경로에
`/v1/agent/`가 포함된 요청은 JSON 404 `AGENT_ENDPOINT_NOT_FOUND`로 응답하고
진단 로그에 기록되며, Agent는 이를 `SERVER_RESPONSE_NOT_JSON` 또는
`AGENT_ENDPOINT_NOT_FOUND`로 분류해 조치 문구를 출력합니다. 잘못된 HTTP 메서드는
`AGENT_ENDPOINT_METHOD_NOT_ALLOWED`로 기록됩니다.

### 6. 등록 진단 요약 API와 콘솔 패널 (Server)

`GET /api/v1/admin/diagnostics/enrollment` (`agents.read`)는 지정 기간의 등록
활동을 집계합니다.

- 등록 성공·거부, 사전 점검 차단, 전송 실패 건수
- 원인 코드별 발생 횟수와 **조치 문구**
- **출처 IP·Agent별 마지막 판정**(실패한 출처를 먼저 정렬)
- **등록은 됐지만 첫 수집 이벤트가 없는 Agent** 목록

같은 내용을 콘솔 **Agent 관리 → 등록 진단** 패널에서 확인할 수 있습니다. 기존
**Server 진단 로그**는 무엇을 검색해야 하는지 이미 아는 경우에만 유용했지만, 이
패널은 검색어 없이 실패 중인 등록을 보여줍니다.

### 7. 교차 조회용 `X-Request-Id`

모든 Server 응답에 `X-Request-Id`가 포함됩니다. Agent는 이 값을 로그와
`status.json`에 기록하므로 Agent 로그, 프록시 로그, Server 진단 로그를 하나의
식별자로 연결할 수 있습니다.

## 오류 코드와 조치

| 코드 | 조치 |
|---|---|
| `SERVER_UNREACHABLE` | URL·라우팅·방화벽(7070/TCP outbound) 확인 |
| `SERVER_TIMEOUT` | `timeout_seconds` 상향, 중간 프록시 점검 |
| `TLS_REJECTED` | `ca_file`에 사설 CA 인증서 지정 |
| `SERVER_RESPONSE_NOT_JSON` | `server.url`을 scheme·host·port만으로 설정 |
| `AGENT_ENDPOINT_NOT_FOUND` | 같은 조치 |
| `AGENT_AUTO_ENROLLMENT_DISABLED` | 콘솔에서 자동 등록 활성화 |
| `AGENT_SOURCE_NOT_ALLOWED` | 허용 목록에 출처 IP/CIDR 추가 |
| `AGENT_ENROLLMENT_UNAUTHORIZED` | 발급 Token을 `enrollment_token_file`에 기록 |
| `AGENT_ALREADY_CLAIMED` | 복제본의 `agent-id`, `enrollment-claim.json` 삭제 |
| `AGENT_BLOCKED` | 콘솔에서 차단 해제 |
| `AGENT_UNAUTHORIZED` | 자동으로 재등록됩니다. 반복되면 Token 회전 확인 |

## 호환성

- 데이터베이스 마이그레이션이 없습니다. 기존 `diagnostic_logs` 테이블을 사용합니다.
- 기존 Agent(v0.2.5 이하)는 그대로 동작합니다. 새 진단 기능만 사용할 수 없습니다.
- v0.2.6 Agent는 이전 Server에도 접속합니다. `/v1/agent/preflight`가 없으면
  `--diagnose`가 해당 항목만 `WARN`으로 표시하고 나머지 점검은 계속합니다.
- 등록 정책, Token, 장비 자격 증명, 큐 형식은 변경되지 않았습니다.
- `--once`의 종료 코드가 0에서 2로 바뀔 수 있습니다. 전달 실패를 무시해야 하는
  자동화가 있다면 종료 코드 처리를 확인하십시오.

## 확인 방법

```bash
# Agent 장비에서
invenqor-agent --config /etc/invenqor-agent/config.toml --diagnose
invenqor-agent --config /etc/invenqor-agent/config.toml --status

# 임의 장비에서
curl -s https://invenqor.example.com:7070/v1/agent/preflight | jq .enrollment

# 콘솔에서
Agent 관리 → 등록 진단
Server 진단 로그 → component: Agent 등록 / Agent 전송
```
