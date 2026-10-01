# Momento v0.34.53 — `npm test` 가 npm 자신의 Node 로 테스트를 돌립니다

콘솔 화면과 서버는 달라지지 않았습니다. 이 릴리스는 **테스트 실행 방식 하나**를 고칩니다.

## 테스트 대부분이 조용히 돌지 않을 수 있었습니다

`web/test/*.test.mjs` 는 순수 로직 모듈을 `.ts` 확장자까지 적어 import 하므로, 테스트를 돌리는 Node 가 타입 스트리핑을 할 수 있어야 합니다.

그런데 npm 은 lifecycle 스크립트의 PATH 에 **상위 디렉터리의 `node_modules/.bin` 을 모두 앞에 붙입니다.** 홈이나 상위 경로 어딘가에 `node` 패키지가 하나 깔려 있으면, `npm test` 안에 적힌 `node` 는 npm 자신을 돌리는 Node 가 아니라 그 쪽으로 해석됩니다.

이 저장소의 릴리즈 검증이 두 번 그렇게 멈췄습니다.

| | |
|---|---|
| npm 을 돌리는 Node | v22.23.1 |
| 스크립트 안의 `node` | `~/node_modules/.bin/node` → v20.19.2 |
| 결과 | 38개 파일 중 **30개**가 `ERR_UNKNOWN_FILE_EXTENSION ".ts"` |

자식 프로세스가 찍은 `# Node.js v20.19.2` 가 결정적 증거였습니다. 명령 한 번만 `node` 라고 적혀 있을 뿐인데 테스트 대부분이 못 돈 셈입니다.

## 고친 것

- `web/package.json` 의 test 명령이 PATH 의 `node` 대신 **npm 이 알려주는 인터프리터 경로**(`npm_node_execpath`)를 씁니다.
- npm 없이 셸에서 바로 돌릴 때(`make test` 등)를 위해 `node` 로 되돌아갑니다.
- 요구하는 Node 범위를 `engines` 에 적었습니다 — `^22.18 || >=24`. 확장자 없는 타입 스트리핑이 기본으로 켜지는 범위입니다.

`.github/workflows` 와 `node --test` 자체, 테스트 단언은 **아무것도 느슨하게 하지 않았습니다.**

## 고정

`web/test/testCommand.test.mjs` 가 `package.json` 의 실제 문자열을 읽어 이 성질을 확인합니다(176 → 178건).

- 타입 스트리핑을 못 하는 `node` 를 PATH 앞에 세우고 `.ts` 를 import 하는 진짜 테스트 파일을 돌립니다.
- 설정된 명령은 통과해야 하고, **같은 PATH 에서 맨 `node` 는 깨져야 합니다** — 가림이 진짜임을 보이는 대조입니다.

환경에 기대지 않는지 두 Node 메이저와 되돌림 경로로 확인했습니다.

| 조합 | 결과 |
|---|---|
| nvm v22.23.1 / npm 10.9.8 | 178 / 178 |
| `/usr/bin/npm` 11.12.1 / Node v25.9.0 | 178 / 178 |
| `npm_node_execpath` 없이 셸에서 직접 | 178 / 178 |

Go·API·데이터베이스 변경은 없습니다. 데이터베이스 마이그레이션도 없습니다.
