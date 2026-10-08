# Invenqor Server·Agent v0.2.12 릴리즈 노트

릴리즈 일자: 2026-07-30
호환 Agent: v0.2.12 (Linux·Windows)

**Windows 서비스가 시작 직후 종료되어 아무것도 수집하지 못했습니다.** Windows
Agent를 설치한 경우 이 릴리즈로 교체하십시오. Server와 Linux Agent는 영향이
없습니다.

## 1. 서비스 상태 핸들이 잘려 서비스가 죽었습니다

`SERVICE_STATUS_HANDLE`은 핸들, 즉 **포인터 크기** 타입입니다. 이를 32비트로
선언해 x64에서 상위 32비트가 잘렸고, 잘린 핸들로 `SetServiceStatus`를 호출하니
모든 상태 보고가 실패했습니다. Service Control Manager는 서비스가 `RUNNING`을
보고하는 것을 끝내 보지 못했고, 시작 제한 시간이 지나면 **시작 실패로 판정해
프로세스를 종료**했습니다.

그래서 첫 수집에 도달하지 못했고, 큐에 아무것도 쌓이지 않았고, 등록도 없었습니다.
`--diagnose`는 이 모든 것과 무관하게 정상을 보고했습니다 — Server 도달 경로만
검사했기 때문입니다.

`isize`로 수정했습니다. 상태 보고가 거부되면 이제 로그에 기록합니다.

## 2. 서비스 열거가 무한 반복될 수 있었습니다

`!ok != 0`은 논리 부정이 아니라 **비트 반전**입니다. 어떤 `i32`에 대해서도 0이
아니므로 모든 결과에 "데이터가 더 있다"고 판정했고, 예상치 못한 오류가 종료 조건이
없는 반복으로 흘러들어갔습니다. 수집기 하나가 멈추면 주기 전체가 멈추므로 큐에
아무것도 쌓이지 않습니다 — 1번과 증상이 같습니다.

조건을 바로잡고 재시도 횟수에 상한을 두었으며, 초기 버퍼를 256 KiB로 키워 대부분의
호스트에서 한 번의 호출로 끝나게 했습니다.

## 3. 무엇보다, 이 문제를 볼 방법이 없었습니다

Windows 서비스의 표준 오류는 어디에도 남지 않습니다. 서비스로 동작하는 동안 Agent가
기록한 모든 것이 버려졌고, 그래서 한 번도 수집하지 못한 서비스가 정상으로 보였습니다.

- **Agent가 로그 파일을 씁니다**: `%ProgramData%\Invenqor\state\agent.log`.
  8 MiB에서 한 번 회전해 최대 16 MiB를 사용합니다.

  ```powershell
  Get-Content "$env:ProgramData\Invenqor\state\agent.log" -Tail 50 -Wait
  ```

- **`--diagnose`가 서비스 자체를 점검합니다.** 이전의 모든 점검은 "이 호스트가
  Server에 도달해 등록할 수 있는가"를 답했고, 그 전부가 통과하는 동안 서비스는
  죽어 있었습니다. 두 항목을 추가했습니다.

  | 항목 | 판정 |
  |---|---|
  | `service` | SCM이 보고하는 서비스 상태와 마지막 종료 코드. 실행 중이 아니면 실패 |
  | `collection activity` | 마지막 수집 완료 시각. 한 번도 없거나 두 주기 이상 지났으면 실패 |

  세 상태(한 번도 없음 / 오래됨 / 정상)를 각각 확인했습니다. 이번 사례에서
  `--diagnose`는 이제 `OK` 대신 실패를 보고합니다.

## 이미 설치된 Windows 호스트

새 패키지에서 `install.ps1`을 다시 실행하면 됩니다. 설정과 미전송 큐는 유지됩니다.

```powershell
.\scripts\install.ps1
& "$env:ProgramFiles\Invenqor\invenqor-agent.exe" `
  --config "$env:ProgramData\Invenqor\config.toml" --diagnose
Get-Content "$env:ProgramData\Invenqor\state\agent.log" -Tail 30
```

`--diagnose`를 관리자 PowerShell에서 실행하면 서비스 상태 항목까지 포함됩니다.
일반 계정에서는 SCM을 조회할 수 없어 해당 항목이 `SKIP`으로 표시됩니다.

## 호환성

- 데이터베이스 마이그레이션이 없습니다. Server와 Linux Agent 동작 변경이 없습니다.
- Linux Agent도 `--diagnose`에 `collection activity` 항목이 추가됩니다. 수집이
  두 주기 이상 멈춘 호스트는 이제 실패로 보고되므로, `--diagnose`의 종료 코드를
  감시에 사용하는 경우 확인하십시오. 서비스 상태 항목은 Windows에만 나타납니다.
- Windows에서 로그 파일이 새로 생성됩니다. 상태 디렉터리 용량 한도와 별개로 최대
  16 MiB를 사용합니다.
