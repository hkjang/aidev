## 터미널 명령 위험 분류기가 우회당하고, 동시에 정상 조회를 차단했습니다

`analyzer.ParseCommandRisk` 는 Pod 터미널/exec 요청이 통과하는 위험 등급기이고, 그 결과는 라벨이 아니라 **게이트**입니다 — `evaluateTerminalPolicy` 는 `critical` 이면 정책을 **한 줄도 읽기 전에** `Allowed=false` 로 돌려보내고, `ClassifyTerminalAccessMode` 는 `low` 를 넘는 순간 guided 티어와 승인을 강제합니다. 그런데 구현이 명령 전체 문자열에 대한 substring 매칭이라 양쪽으로 다 틀렸습니다.

### ① 루트 삭제가 플래그 표기만 바꾸면 등급을 벗어났습니다

`rm -rf /` 라는 정확한 바이트열만 critical 이어서, 플래그를 나눠 쓴 `rm -f -r /` 는 **아무 규칙에도 걸리지 않아 `low`(승인이 필요 없는 read_only 티어)** 로 분류됐습니다. 공백 하나가 다른 `rm  -rf /` 도 `low` 였고, `rm -rf  /`·`rm -rf "/"`·`sh -c "rm -rf /"` 는 하드 블록에서 `high` 로 내려앉아 allowlist 항목 하나만 있으면 실행됐습니다.

이제 프로그램·플래그·대상을 각각 보고 판정하므로 `/bin/rm`·`sudo rm`·`xargs rm`·`--no-preserve-root` 표기도 같은 등급이며, `rm -rf /data` 는 그대로 high 입니다.

### ② 인자에 단어가 들어 있다는 이유로 조회가 하드 블록됐습니다

`reboot`·`shutdown`·`halt`·`mkfs` 를 substring 으로 찾았기 때문에 `cat /var/log/reboot-analysis.log`, `grep -i halt /var/log/app.log`, `tail /var/log/shutdown.log` 가 전부 critical 로 차단됐습니다 — 지난 재부팅 로그를 **읽는** 것이 재부팅을 **시키는** 것으로 취급된 셈입니다.

프로그램 자리에 있는 토큰만 보도록 바꿨고(프록시 deny 경로의 `terminalProgramTokenMatches` 가 이미 쓰던 판정입니다), `/sbin/reboot`·`sudo shutdown -h now`·`mkfs.ext4` 는 계속 critical 입니다.

### ③ pipe-to-shell 판정이 "sh" 라는 글자를 찾고 있었습니다

조건이 "명령 어딘가에 `|` 와 `sh` 와 `curl` 이 있으면" 이라, `curl -s http://api/health | grep crash` 처럼 **crash·flush·shard 안의 sh** 만으로 원격 코드 실행으로 판정돼 하드 블록됐습니다. 이제 파이프 뒤 **프로그램 자리**에 셸이 와야 하며(`curl x|sh`, `wget -qO- x | sudo bash` 는 그대로 critical), 덤으로 `a || b` 를 파이프로 세던 것도 고쳤습니다.

### ④ 같은 명령의 차단 사유가 실행할 때마다 달라졌습니다

규칙 표가 map 이라 findings 순서와 그 첫 항목을 쓰는 `CommandRiskReason` — 즉 evaluator 가 저장하는 차단 사유이자 감사 로그에 남는 문구 — 가 매 실행 바뀌었습니다. `mkfs.ext4 /dev/sdb && reboot` 이 "mkfs" 로 막히기도 하고 "reboot" 으로 막히기도 했습니다. 정렬된 슬라이스로 바꿨습니다.

부수적으로 `>/dev/sda`(공백 없는 리다이렉트)가 critical 로 올라오고, 재귀 삭제 하나가 `rm -rf`·`rm -r` 두 건으로 중복 보고되던 것도 사라졌습니다.

### 초록불을 믿지 않았습니다

신규 테스트 8개를 **고치기 전 코드에 되돌려 붙여** 확인했습니다 — 일곱 개가 각 결함을 **정확히 지목하며** 실패했습니다(여덟 번째는 정상 조회가 계속 `low` 로 남는지 지키는 오탐 회귀 테스트입니다).

검증: `go build ./...` · `go vet ./...` · `go test ./...` 전부 통과(19 패키지).

---
- 이미지: `clustara:v0.9.269`
- 배포 압축본: `clustara-v0.9.269.tar.gz`
