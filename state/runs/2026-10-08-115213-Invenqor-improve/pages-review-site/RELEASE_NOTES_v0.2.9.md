# Invenqor Server·Agent v0.2.9 릴리즈 노트

릴리즈 일자: 2026-07-30
호환 Agent: v0.2.9

현장에서 확인된 문제 하나를 고칩니다. **패키지 설치 스크립트가 서비스 계정이 읽을
수 없는 설정 디렉터리를 만들었고**, Agent는 그 상태를 "설정 파일이 없다"로
보고하며 내장 기본값으로 동작했습니다. 등록도 전송도 하지 않고 로컬 큐만 늘어나며,
`sudo`로 실행한 `--diagnose`는 root가 파일을 읽을 수 있으므로 **정상으로
보고했습니다**.

## 1. 설치 스크립트가 읽을 수 없는 설정 디렉터리를 만들었습니다

```sh
install -d -m 0750 "$CONFIG_DIR"     # 소유자 지정 없음 → root:root 0750
install -m 0640 -o root -g invenqor-agent "$CONFIG_DIR/config.toml"
```

설정 **파일**의 소유·권한은 맞았지만 **디렉터리**가 `root:root 0750`이라
`invenqor-agent` 서비스 계정은 디렉터리를 통과(traverse)할 수 없었습니다. 파일
권한이 옳아도 도달할 수 없으므로 모든 읽기가 `EACCES`로 실패합니다.
systemd 유닛의 `ProtectSystem=strict`는 `/etc`를 읽기 전용으로 만들 뿐
읽기 자체를 막지 않으므로 원인이 아닙니다.

- `install.sh`가 설정 디렉터리를 `root:invenqor-agent 0750`으로 만듭니다.
- 업그레이드 시 기존 설정 파일의 소유·권한을 `root:invenqor-agent 0640`으로
  복구합니다. 잘못된 권한으로 이미 설치된 호스트가 업그레이드만으로 정상화됩니다.
- 손으로 설치하는 `enrollment.token`, `ca.pem`, `device.pem`도 존재하면 같은
  소유·권한으로 맞춥니다. 이 파일들의 권한이 틀려도 같은 방식으로 조용히
  실패했습니다.

## 2. "읽을 수 없음"을 "없음"으로 보고했습니다

`Path::exists()`는 stat 실패를 모두 부재로 취급하므로, 권한 때문에 열 수 없는
파일도 **없는 파일**이 됩니다. Agent는 기본 경로에 파일이 없으면 내장 기본값으로
계속 동작하는 것이 정상 동작(최초 설치 상태)이므로, 권한 문제가 그 경로로
흘러들어가 아무 오류 없이 흡수되었습니다. 남는 단서는 사실과 다른 경고 한 줄
(`no configuration file was found`)뿐이었습니다.

- 부재와 접근 거부를 구분합니다. 접근이 거부된 설정 파일을 만나면 Agent는 기본값으로
  넘어가지 않고 **거부된 계정과 복구 명령을 출력하며 기동을 거부**합니다.
- `--diagnose`와 `--status`는 이 상태에서도 보고서를 만들어야 하므로 종료하지 않고,
  `[FAIL] configuration file` 항목으로 원인과 조치를 표시합니다.

## 3. root로 실행한 진단이 서비스의 현실을 보고하지 않았습니다

`--diagnose`는 보통 `sudo`로 실행하고 서비스는 `invenqor-agent`로 동작합니다.
root가 파일을 읽을 수 있다는 사실은 서비스가 읽을 수 있는지에 대해 아무것도
말해주지 않는데, 진단은 자기 자신의 접근 결과를 `[PASS]`로 보고했습니다. 문제를
찾기 위해 실행한 도구가 문제를 감추고, 운영자를 엉뚱한 곳으로 보냅니다.

이제 설정 파일에 도달하는 모든 경로의 소유·그룹·모드를 확인해 **서비스 계정이 읽을
수 있는지**를 판정합니다. 읽을 수 없으면 어느 경로가 어떤 모드로 막고 있는지와
복구 명령을 함께 표시합니다.

```text
  [FAIL] configuration file  /etc/invenqor-agent/config.toml is readable by this
                             account but not by the invenqor-agent service account
                             (/etc/invenqor-agent is mode 0750 owned by uid 0 gid 0);
                             the service runs on built-in defaults and never registers
                             fix:  sudo chown root:invenqor-agent /etc/invenqor-agent
                                   /etc/invenqor-agent/config.toml; sudo chmod 0750
                                   /etc/invenqor-agent; sudo chmod 0640
                                   /etc/invenqor-agent/config.toml
```

## 4. 사용할 수 없는 설정에서 진단이 보고 없이 종료했습니다

`--diagnose`가 TOML 오류를 만나면 그 오류만 출력하고 종료했습니다. 오타를 낸
운영자는 오타 위치만 알게 되고, Server 도달 여부나 자격 증명 유효성은 아무것도
확인하지 못했습니다. 이제 사용할 수 없는 설정도 `[FAIL] configuration file`
항목으로 보고하고 나머지 점검을 계속 수행합니다.

## 5. Helm chart의 기본 image tag가 0.2.7에 멈춰 있었습니다

`values.yaml`의 `tag`가 따옴표를 포함한 형태여서 버전 일괄 변경에서 빠졌습니다.
v0.2.8 chart는 `appVersion: 0.2.8`이면서 기본 이미지로 `invenqor-server:0.2.7`을
가리켰습니다. tag를 0.2.9로 맞추고, 버전 변경 도구가 이 형태를 처리하도록
고쳤습니다. chart에서 `image.tag`를 명시하는 배포는 영향이 없습니다.

## 이미 설치된 호스트 복구

업그레이드하면 `install.sh`가 권한을 맞춥니다. 즉시 복구하려면:

```bash
sudo namei -l /etc/invenqor-agent/config.toml
sudo chown root:invenqor-agent /etc/invenqor-agent /etc/invenqor-agent/config.toml
sudo chmod 0750 /etc/invenqor-agent
sudo chmod 0640 /etc/invenqor-agent/config.toml
sudo systemctl restart invenqor-agent
```

큐에 쌓여 있던 이벤트는 등록이 성공하면 다음 주기에 순서대로 전송됩니다. 큐를
삭제하지 마십시오.

확인:

```bash
sudo -u invenqor-agent /opt/invenqor-agent/bin/invenqor-agent \
  --config /etc/invenqor-agent/config.toml --diagnose
```

`--validate-config`와 `--diagnose`는 **서비스 계정으로**(`sudo -u invenqor-agent`)
실행하십시오. root로 실행하면 서비스가 읽지 못하는 파일도 통과합니다.

## 호환성

- 데이터베이스 마이그레이션이 없습니다. Server 동작 변경이 없습니다.
- **읽을 수 없는 설정 파일에서 Agent가 기동을 거부합니다.** 그 상태로 계속 수집만
  하던 호스트는 이제 서비스가 시작되지 않고 저널에 조치 명령이 남습니다. 조용히
  아무것도 등록하지 않는 것보다 낫다는 판단입니다.
- Helm chart 기본 `image.tag`가 0.2.9입니다.
