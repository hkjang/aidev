# 회차 노트 2026-09-22-050426-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:04] base pinned — main@d42cc59
- [러너 05:04] autonomy release — 

## 정찰 노트
- guide-shots 백업 실패 후 쓰기 지속을 원본 블록 vm 실행으로 재현하여 선택. runtime/base·auth·workflow 변경 없이 설정 손실을 차단해 다른 후보보다 가치/위험이 유리하다.
- 기존 Node 57건·Go API/runtime-proxy·guide-shots 구문 검사 통과. DB/브라우저 실물 검증은 미확인이며 새 테스트는 구현자가 추가한다.
- 네 API 값의 복원 형식을 섞지 말고 설정 누락 시 seed 전체 중단. 복원 오류 수집·CSP 기록 전체 삭제는 후속으로 남긴다.
- 요청된 세 부서 스킬은 도구/로컬 검색에서 미발견이라 형식 미확인. 프로필의 이미 해결된 Langflow 항목을 갱신했으며 코드는 변경하지 않았다.
- [러너 05:08] scout done — guide-shots 전역 설정 네 가지를 모두 백업한 뒤에만 seed·추적 캡처를 시작한다 (가치 4 / 위험 1 / 작업량 M)

## 구현 노트
- af47666: 실제 guide-shots에 withGuideSettings를 연결해 네 설정 검증·깊은 복제 완료 전 seed/추적/복원 쓰기를 차단했다.
- policy/DLP는 원본 객체, sessionGateway/tracking은 {value: 원본객체}로 finally 복원; skip은 capture만 실행한다.
- 테스트 선작성: 기존 코드 51건 중 41건 실패. 최종 신규 52건+기존 57건=109건, 구문 검사 2개, Go API/runtime-proxy, diff 검사 통과.
- 확신 없는 곳·검증 못 한 것: AGENTHUB_TEST_DSN 미설정으로 실제 Handler+DB와 브라우저 실물 촬영은 미검증. Node는 실제 스크립트 구간을 vm에서 실행하고 fetch/작업을 대역으로 사용한다.
- 일부러 하지 않은 것: 복원 오류 수집·전체 CSP 기록 삭제·요청 제한 시간은 별도 후속. 공유 배포 쓰기, 보호 파일, 버전 변경 없음.
- 다음 역할 주의: 복원 첫 PUT의 네트워크 예외가 나머지 복원을 막는 기존 동작은 남아 있다. 실제 DB 테스트는 격리 DSN·관리자 세션/CSRF 필요.
- 요청된 technology 세 스킬은 callable Skill 도구/정확한 로컬 파일 미발견으로 미적용; 결과 파일은 지정 경로에 작성했다.
- [러너 05:13] brief accepted — 채택 — 현재 코드의 실패 무시·빈 정책 대체·설정 누락 복원 생략과 실제 API의 GET/PUT 계약이 과제서와 일치했다.
- [러너 05:13] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음: 변경 3개 파일, 실제 API 계약, 오류·복원·skip 경로와 보안·법무 범위를 확인; 새 결함 없음.
- Node 109건·구문 2개·diff 검사 통과; main 원본에 새 테스트 적용 시 52건 중 42건 실패하여 회귀 검출 확인.
- 실제 Handler+DB·브라우저 촬영은 미검증; 요청된 세 부서 스킬은 도구·로컬 파일 미발견으로 미적용.
- 후속: 복원 PUT 예외 시 이후 복원 중단, 전체 CSP 기록 삭제·timeout 부재는 기존 한계이며 새 회귀 테스트의 CI 연결도 필요.
- [러너 05:14] review approved — 리뷰 승인 (risk=low)
- [러너 05:15] pr created — https://github.com/hkjang/AgentHub/pull/33
- [러너 05:17] ci passed — 검사 1개 모두 success
- [러너 05:17] merge done — af47666
- [러너 05:17] release missing — 릴리즈 결과 없음/손상: missing
