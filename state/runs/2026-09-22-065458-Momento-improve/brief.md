# 정찰 차단 — 구현 과제 미선정

- 과제: 미선정. 필수 스킬 부재로 정찰 선행 조건을 충족하지 못함 (가치·위험·작업량 미평가).
- 왜: 요청된 `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration`을 제공하는 Skill/skills.list/skills.read 도구가 현재 도구 목록에 없고, 검색한 로컬 경로에서도 원문을 찾지 못했다. 개발 지침은 명시적으로 요청된 필수 스킬을 찾지 못하면 중단하도록 요구하므로 검증되지 않은 구현 과제를 넘기지 않는다.
- 수용 기준: 1) 실행 환경에서 세 스킬 원문을 제공한다. 2) 스킬을 읽은 새 정찰이 현재 코드 근거로 후보를 재평가한다. 3) 그 정찰에서 실제 검증 명령과 관찰 가능한 구현 수용 기준을 작성한다.
- 건드릴 파일: 저장소 파일 없음. 구현자는 이 문서를 코드 변경 지시로 취급하지 않는다.
- 검증 명령: 미확인. 저장소 코드·테스트·CI를 읽거나 실행하지 않았다.
- 위험과 피할 것: 필수 스킬을 다른 패키지 스킬로 대체하거나, 과거 과제의 근거가 지금도 성립한다고 추정하지 않는다. auth/migrations/workflows를 포함한 저장소 변경 및 커밋 없음.
- 차선 후보: 미선정. 선행 조건 부재는 후보 변경으로 해결되지 않는다.

확인 범위: 현재 도구 이름 목록, MCP 리소스/템플릿 목록, /home/hkjang/.codex·.claude·.agents, /mnt/c/Users/USER/.codex·.claude, /mnt/c/Users/USER/projects/aidev, /home/hkjang/.cache 및 현재 저장소의 파일명 검색. 해당 스킬명과 일치하는 원문을 발견하지 못했다. 접근 불가능하거나 검색 범위 밖에 있는 원문의 존재는 미확인이다.

절차 이행 상태: 코드 파악·후보 재평가·신규 후보 2개·과제 선정은 미실시. ideas.json은 state/Momento.ideas.json을 의미 변경 없이 보존했으며 날짜·상태·평가를 갱신하지 않았다. profile.md는 현재 코드를 검증하지 못해 새로 작성하지 않았다.
