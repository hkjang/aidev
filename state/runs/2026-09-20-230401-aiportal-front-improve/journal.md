# 회차 노트 2026-09-20-230401-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:04] base pinned — main@dd65af7
- [러너 23:04] autonomy release — 

## 정찰 노트
- 수정 과제: aidev run_agent→run_codex의 headcount 원문 경로 전달 누락 선택; 앱 CI 실패 증거가 없어 무관한 앱 변경은 제외. 외부 저장소 이관 필요.
- 스킬 5개는 /mnt/c/Users/USER/projects/headcount/plugins 아래 실제 존재하여 읽음. 기존 “로컬 없음”은 검색 누락; callable Skill 도구는 없음.
- bash -n 및 외부 gate 27건 통과. 앱 build/test·신규 회귀·전체 sim은 미실행; 코드는 수정하지 않았으므로 수정 후 통과를 주장하지 않음.
- 버전 0.0.0/문서 0.0.1 모순과 증가 관례는 여전히 미확정. 외부 릴리즈 지침 완화·임의 태그/버전 신설 금지; 원장에 수정 과제/외부 차단을 구분 기록할 것.
- [러너 23:09] scout done — [수정 과제] Codex 폴백에 headcount 스킬의 실제 파일 경로 전달 복구 (가치 5 / 위험 2 / 작업량 M)

## 구현 노트
- [수정 과제/차선/변경없음] 외부 aidev 구현 회차로 이관되지 않아 코드 변경 없이 ledger-entry.md와 ideas.json에 이관 근거를 기록했다.
- 요청된 technology 스킬 3개는 실제 파일로 읽음; callable Skill 도구 없음. debugging/TDD에는 Return contract가 있으므로 정찰의 “별도 반환 형식 없음”은 보완한다.
- run_agent→run_codex의 Claude 전용 안내 유지·경로 누락을 정적으로 확인; registry release 2개/scout 3개 원문은 실제 읽을 수 있다.
- 검증: bash -n /mnt/c/Users/USER/projects/aidev/bin/run.sh 통과. gate 27건은 정찰 결과이며 이번 재실행 아님.
- 확신 없는 곳·검증 못 한 것: quota 폴백 동적 재현, 신규 회귀, Agents/전체 sim, 앱 test/build 미실행; 동일 원인 두 실패 독립 확인 못 함.
- 외부 구현·빈 커밋·실 모델 호출·릴리즈는 하지 않음. 미수정 결함을 done 또는 릴리즈 성공/skipped로 기록하지 않는다.
- 다음 역할: 별도 aidev 회차에서 fixture/도구 스텁으로 Red→Green 검증; 앱 0.0.0/문서 0.0.1은 별도 장애로 유지하고 gate·릴리즈 규칙을 완화하지 말 것.
- [러너 23:11] brief fallback — 차선 — 지정 구현 파일은 외부 aidev 저장소에 있고 이번 회차에서 직접 수정하지 말라는 지시가 유지되어, 허용된 회차 �
- [러너 23:11] improve no-change — 커밋 없음
