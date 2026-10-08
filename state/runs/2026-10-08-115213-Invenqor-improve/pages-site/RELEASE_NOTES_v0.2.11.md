# Invenqor Server·Agent v0.2.11 릴리즈 노트

릴리즈 일자: 2026-07-30
호환 Agent: v0.2.11 (Linux·Windows)

**v0.2.10의 Windows 설치 스크립트가 실행되지 않습니다.** v0.2.10으로 Windows
Agent를 설치하려던 경우 이 릴리즈로 교체하십시오. Linux Agent와 Server는 영향이
없습니다.

## 설치 스크립트가 Windows PowerShell에서 즉시 실패했습니다

```text
'$IsWindows' 변수는 아직 설정되지 않았으므로 검색할 수 없습니다.
```

`install.ps1`이 플랫폼 확인에 `$IsWindows`를 사용했습니다. 이 자동 변수는
**PowerShell 6 이상에만 존재**하며, Windows에 기본 포함되어 운영자가 실제로 여는
셸인 **Windows PowerShell 5.1에는 없습니다.** 스크립트가 `Set-StrictMode`를
사용하므로 설정되지 않은 변수를 읽는 것은 종료 오류이고, 따라서 아무것도 설치되지
않은 상태로 첫 줄에서 중단됐습니다.

`[System.Environment]::OSVersion.Platform`으로 교체했습니다. 모든 에디션·모든
플랫폼에 존재하므로 에디션 판별이 필요하지 않습니다. `uninstall.ps1`에도 같은
문제가 있어 함께 고쳤습니다.

**왜 놓쳤는가**: 검증을 컨테이너의 PowerShell 7로만 수행했습니다. 7에는
`$IsWindows`가 있으므로 문제가 드러나지 않았습니다. 정작 대상 셸인 5.1에서는
확인하지 않았습니다.

## 같은 종류의 결함이 다시 나가지 못하게 했습니다

`packaging/build-zip.sh`가 패키지를 만들기 전에 `verify-scripts.ps1`을 실행하고,
실패하면 **패키지를 만들지 않습니다.**

1. 스크립트가 파싱되는지
2. PSScriptAnalyzer 오류가 없는지
3. PSScriptAnalyzer 호환성 규칙이 5.1에 대해 아무것도 보고하지 않는지
4. **5.1 이후에 추가된 자동 변수를 참조하지 않는지**

네 번째가 핵심입니다. PSScriptAnalyzer의 호환성 규칙은 구문과 cmdlet을 검사하지만
자동 변수는 검사하지 않으므로 `$IsWindows`를 잡아내지 못합니다. 실제로 이 결함을
되살려 확인했더니 1~3번은 모두 통과했습니다. 잡으려던 결함을 놓치는 검사는 없는
것보다 나쁘므로, 5.1에 없는 자동 변수(`$IsWindows`, `$IsLinux`, `$IsMacOS`,
`$IsCoreCLR`, `$PSStyle` 등)를 구문 트리에서 직접 찾습니다.

결함을 되살린 상태로 패키징을 실행해 `build-zip.sh`가 종료 코드 1로 중단하고
zip을 만들지 않는 것을 확인했습니다. PowerShell을 사용할 수 없는 빌드 환경에서는
검증을 건너뛴다는 사실을 출력하고 실패합니다. 의도적으로 건너뛰려면
`SKIP_SCRIPT_VERIFY=1`이 필요합니다.

## 호환성

- 데이터베이스 마이그레이션이 없습니다. Server와 Linux Agent 동작 변경이 없습니다.
- Windows Agent 실행 파일 자체는 v0.2.10과 기능이 같습니다. 변경은 설치 스크립트에
  한정됩니다. 이미 v0.2.10을 설치하는 데 성공했다면(PowerShell 7에서 실행한 경우)
  교체하지 않아도 동작하지만, 다음 업그레이드부터 이 릴리즈의 스크립트를
  사용하십시오.
