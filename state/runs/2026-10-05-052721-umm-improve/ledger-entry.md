## 2026-10-05
- 선택: 수정 과제 — `npm test --silent` 실행 전 누락된 web 의존성 설치 (가치 4 / 위험 1 / 작업량 S)
- 결과: 변경없음
- 요약: main@6aaf940에서 지정 명령의 exit 1을 재현했고, 원인은 web/node_modules 미설치였으며 기존 래퍼는 이미 정확한 원인과 설치 명령을 안내했습니다. CI와 동일하게 `npm ci --prefix web`만 실행한 뒤 같은 명령이 20파일·218시험 PASS(exit 0)가 되었고, 설치 디렉터리를 잠시 치우면 같은 실패가 재발하며 복원하면 다시 218시험 PASS여서 원인을 입증했습니다. 코드·워크플로·버전 변경은 0파일이며 커밋도 만들지 않았습니다. lint(기존 경고 있음)·typecheck·offline-queue·build·PWA(150 assets)·i18n(1060키)·version 검사·audit high 전부 exit 0이고 audit에는 low 1건이 남습니다.
- 실패 재현: `run-on-supported-node: vitest is not installed: nothing resolves vitest/package.json from /home/hkjang/.cache/auto-improve-wt/umm/web/package.json. Run `npm ci --prefix /home/hkjang/.cache/auto-improve-wt/umm/web` first. This is a missing install, not an unsupported interpreter.` / `EXIT=1`. 새 테스트 추가는 못 함 — 코드 결함이 아니라 설치 전 상태이며 기존 설치 누락 테스트가 이미 3개 있어 동일 테스트를 중복하지 않았습니다. 실제 루트 npm 진입점으로 설치 전/후/재제거/복원을 확인했습니다.
- 보류 아이디어: 러너의 새 체크아웃 검증 전에 `npm ci --prefix web` 선행 — 영구 재발 방지는 러너에서 해야 하며 이번 허용 범위 밖이라 미수정 (가치 4 / 위험 1 / 작업량 S)
  - make test-go를 CI처럼 직렬화 — 격리 DB에서 경합 재현 선행 (가치 3 / 위험 1 / 작업량 S)
  - Node 선언 세 곳의 드리프트 확인 — 릴리즈 이미지 빌드 검증 필요 (가치 2 / 위험 2 / 작업량 S)
  - 캔버스 My Space 대체값 번역 — 실제 UI 경로 재현 필요 (가치 2 / 위험 1 / 작업량 S)
- 과제서: 채택 — 지정된 실패 조사와 로컬 검증 복구를 수행했으나, 저장소 코드 결함이라는 전제는 성립하지 않았습니다. release.yml에는 npm 테스트가 없고 ci.yml은 이미 설치를 선행하므로 워크플로 수정은 불필요합니다. 새 checkout의 외부 러너가 설치를 생략하면 재발하며, 그 러너의 영구 수정·원격 워크플로 실행은 이번에 하지 않았습니다.
