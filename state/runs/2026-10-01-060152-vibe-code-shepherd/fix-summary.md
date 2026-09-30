# 수리 결과: 고치지 않음 — CI 실패가 이 브랜치의 코드 결함이 아니다

- **재현 시도**: `git archive HEAD`(af9b04b) 를 `/tmp/pr11` 에 펼쳐 CI `check` job 4단계(`npm ci` → `npm run check` → `node --check dist/extension.js` → `node --check dist/extension.core.js`)를 그대로 돌렸다. **Node 20.19.2(engines 핀)와 Node 20.20.2(CI `node-version: 20` 의 실제 해석값) 양쪽에서 4단계 전부 exit 0, 88/88 tests 통과.** TZ=UTC·LC_ALL=C 에서도 88/88(신규 테스트가 한국어 문자열·KST 스탬프를 단언하므로 로케일/타임존 변수를 일부러 제거했다). `npm warn EBADENGINE` 만 나오고 exit 0 이다.
- **판정**: 지적(CI 실패)은 **틀렸다** — 정확히는 코드 결함을 가리키지 않는다. 러너가 준 실패 로그 본문은 **비어 있었고**(단계 출력 0줄), 보고된 소요는 4s 다. 프로필에 기록된 2026-09-29 run 36518589343 과 같은 신호(`steps=[]`, `runner_id=0`, "The job was not started because recent account payments have failed or your spending limit needs to be increased.")로, job 자체가 시작되지 않은 계정/결제 측 실패다. PR #6~#11 여섯 회차 연속 같은 양상이다.
- **원격 확인 불가(정직하게 명시)**: `gh auth status` = 미로그인, GitHub API 는 저장소부터 익명 404(비공개). 따라서 위 "job 미시작" 은 로그 부재·소요시간·과거 동일 신호에 근거한 **추론**이고, run 36761215096 의 annotation 을 직접 읽어 확정하지는 못했다. 결제/지출한도 해소 후 `workflow_dispatch` 재실행이 유일한 확정 수단이다.
- **하지 않은 것**: 커밋 없음(워크트리 clean, tracked 파일 무변경). 테스트·단언·`ci.yml` 은 손대지 않았다. 고칠 결함이 없어 고치지 않는 것이 규칙이다.
- **비평가에게**: 코드 쪽을 다시 볼 필요가 있다면 재현 로그는 `/tmp/pr11/{ci.log,check.log,ci20.log,check20.log,check-locale.log}` 에 있다(휘발성 경로). 앞선 비평의 approve 를 뒤집을 증거는 찾지 못했다.
