## 2026-10-08
- 선택: SLA·조치 우선순위 설정의 정수 필드 9개에서 소수 입력을 제한 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `web/src/settings.tsx`의 다섯 선언 위치에 `integer: true`를 추가해 SLA 6개·risk 3개의 기존 NumberInput 옵션을 연결했고, EPSS 소수·범위·기본값·문구·저장 경로를 보존했다(프로덕션 1파일, 테스트 1파일, 커밋 `83d1dad`). 수정 전부터 통과해야 하는 순수 Go 계약 시험은 새 기본 그룹을 사례마다 만들어 9필드 소수 거절·상하한 수락·범위 밖 거절과 EPSS 0/0.1/0.125/1 수락 등 51사례를 검증했으며, Go 1.26.7에서 지정 시험 상위 2개 PASS/0 SKIP, Node 26.11.1에서 npm ci·웹 113 PASS/0 FAIL/0 SKIP·typecheck·build, 원본 312파일 검사와 diff 검사를 통과했다. 실제 앱을 Chromium 1440×1000/390×844에서 합성 API로 실행해 9필드 타이핑/실제 클립보드 붙여넣기/blur/PUT·정수 상하한·EPSS 소수·그룹 저장 후 타 그룹 초안 보존을 확인하고 합성 자료 표시 캡처를 남겼지만 실제 서버 API/DB·Go 실행 파일·원격 CI·배포는 미실행이며, 추가 Prettier 검사의 기존 MCP/OIDC 서식 차이 3곳은 HEAD에서도 동일하여 범위 밖으로 남겼다.
- 실패 재현: `1440 sla.critical_days typing 1.5 -> display=1.5, PUT=1.5` / `AssertionError: sla.critical_days typing submits fractional value 1.5` (프로덕션 수정 전 실제 앱+합성 API 브라우저 시험). 수정 후 통과, 다섯 옵션만 제거하면 같은 실패 재현, 복구 후 전체 데스크톱/모바일 시험 재통과. Go 계약 시험은 처음부터 통과하므로 UI red/green 증거로 세지 않았다.
- 보류 아이디어: inventory SBOM 갱신 검토 기준 정수 제한 (3/1/S) — 별도 그룹이라 제외.
  - 목록 CSV 범위 밖 숫자 타임스탬프 RangeError 방어 (2/1/S) — 1순위 전제가 맞아 차선 미착수.
  - AI max_tokens ≤ context_window 저장 전 안내 (2/2/S) — 서버 교차 제약과 실행 독자 후속 조사.
  - web Node engines 하한 선언 (3/1/S) — 기존 후보 유지, 이번 Node 26 검증과 별개.
- 과제서: 채택 — 9개 정수 옵션 누락·공통 렌더러 배선·서버 정수 거절·설정 원값 PUT 전제가 모두 현재 코드와 맞아 지정된 두 파일만 수정했다.
