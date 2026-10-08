# 회차 노트 2026-10-09-000843-pii-masker-improve — pii-masker
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:08] base pinned — main@19b4426
- [러너 00:08] autonomy release — 

## 정찰 노트
- 정규화 bbox 폭 1 보정 결함을 선택: 실제 cmd→HTTP upstream→최종 PNG에서 502 및 잘못된 200/completed(좌상단 4픽셀만 검정)를 재현했다. 낮은 가치의 테스트 보강보다 우선하고 프로덕션 1파일로 제한했다.
- brief.md 초안을 먼저 저장한 뒤 assets/probe-results.json의 실행 증거·픽셀 수용 기준·26~35분 기본 산정과 8분 contingency로 덮어썼다. 기존 9후보 재평가, 신규 2후보 및 이전 완료 1항목을 ideas.json에 기록했다.
- 미확인: 제안 수정 후 green, PDF 최종 시각 출력, 386용 8000×8000 대체 픽스처. 기본 전체 테스트·vet·build 통과, 기존 386 테스트 실패 재확인.
- 원본 bbox 정규화 여부에서만 분기하고 비정규화 최소 폭·오류 분류·MIME·페이지 파서는 건드리지 말 것. 이미지 페이지 guard는 성공 기록(88e8ace)이 있지만 pinned HEAD에 없으므로 재구현하지 않는다.
- [러너 00:17] scout done — 정규화 bbox의 부분 마스킹에서 좌표 단위를 보존하여 엉뚱한 위치의 성공 출력을 막기 (가치 4 / 위험 2 / �
- [러너 00:17] brief unstated — 구현자가 과제서 판정을 적지 않음
- [러너 00:17] improve no-change — 커밋 없음
