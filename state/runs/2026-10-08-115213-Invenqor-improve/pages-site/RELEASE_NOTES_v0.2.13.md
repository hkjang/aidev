# Invenqor Server·Agent v0.2.13 릴리즈 노트

릴리즈 일자: 2026-07-30
호환 Agent: v0.2.13 (Linux·Windows)

v0.2.12에서 Windows 서비스는 실행되지만 **수집 주기가 한 번도 완료되지 않는** 사례를
확인했습니다. 원인은 수집기 하나가 반환하지 않으면 주기 전체가 영구히 멈추는
구조였습니다.

## 1. 멈춘 수집기가 주기 전체를 세웠습니다

설계 원칙은 "수집기 하나가 실패해도 나머지는 계속 수집한다"였고, 이는 수집기가
**오류를 반환할 때만** 성립했습니다. **반환하지 않을 때**는 성립하지 않았습니다.
`collect_all`이 각 수집기를 기한 없이 기다렸으므로, 블로킹 시스템 호출 하나가
영원히 대기하면 수집도 큐 적재도 전송도 등록도 일어나지 않고, 서비스는 완전히
정상으로 보였습니다.

- 수집기별 **60초 기한**을 둡니다. 초과하면 해당 수집기를 포기하고 오류로 보고하며
  나머지 주기는 계속 진행합니다.
- 기한을 초과한 수집기는 **이후 주기에서 호출하지 않습니다.** 블로킹 작업은 취소할
  수 없어 스레드가 계속 점유되므로, 매 주기 다시 호출하면 스레드 풀이 고갈됩니다.
  Agent를 재시작하면 다시 시도합니다.
- 멈추는 수집기와 정상 수집기를 함께 실행해, 정상 수집기의 레코드가 그대로 도착하고
  멈춤이 오류로 보고되며 다음 주기에는 건너뛰는 것을 테스트로 고정했습니다.

이 변경은 Linux에도 적용됩니다. 응답하지 않는 NFS 마운트나 멈춘 `rpm` 호출에서
같은 일이 일어날 수 있습니다.

## 2. Windows에서 네트워크 대기를 유발하던 두 호출

- **네트워크 드라이브의 용량과 레이블을 조회하지 않습니다.** 매핑된 공유가 사라진
  경우 `GetDiskFreeSpaceEx`와 `GetVolumeInformation`이 SMB 제한 시간까지 대기합니다.
  드라이브 자체는 계속 보고하되 크기는 측정하지 않습니다. 어차피 다른 호스트의
  저장 공간입니다.
- **로컬 그룹 구성원 조회를 권한 관련 그룹으로 한정합니다.** `NetLocalGroupGetMembers`
  레벨 3은 구성원 SID를 `DOMAIN\이름`으로 변환하며, 도메인 가입 호스트에서는 이것이
  도메인 컨트롤러 호출입니다. 내장 그룹 전체(약 25개)에 대해 수행하면 느리거나
  도달 불가한 DC의 영향이 그만큼 커집니다. 인벤토리가 답해야 하는 질문은 "어떤
  계정이 권한을 가졌는가"이므로 Administrators, Remote Desktop Users,
  Backup Operators, Power Users, Remote Management Users만 조회합니다.

## 원인 확정 방법

기한이 생겼으므로 이제 어느 수집기가 멈추는지 보고서에 남습니다. 다음 주기 이후
`--diagnose`와 로그에서 확인하십시오.

```powershell
Get-Content "$env:ProgramData\Invenqor\state\agent.log" -Tail 40
& "$env:ProgramFiles\Invenqor\invenqor-agent.exe" `
  --config "$env:ProgramData\Invenqor\config.toml" --status --json |
  ConvertFrom-Json | Select-Object -ExpandProperty collection
```

수집기별 오류는 `--status --json`의 `collection.collector_errors`와 Server의
수집 이벤트에 함께 담깁니다.

## 호환성

- 데이터베이스 마이그레이션이 없습니다. Server 동작 변경이 없습니다.
- 수집기 기한(60초)이 Linux에도 적용됩니다. 정상적인 수집기는 훨씬 빠르게 끝나므로
  영향이 없지만, 매우 느린 사용자 환경이 있다면 확인하십시오.
- Windows에서 네트워크 드라이브의 `total_bytes`·`free_bytes`·`filesystem`·`label`이
  비어 있게 됩니다. `drive_type`은 계속 `network`로 보고합니다.
