## 2026-10-08
- 선택: 상속된 객체 속성이 문의 유형 검증을 통과하는 오류 수정 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: validate의 유형 검사 한 줄을 Object.hasOwn(TYPE_LABELS, v.type)으로 바꿔 직접 등록된 여섯 유형만 허용하고, 실제 TS 모듈을 상대 import하는 단위 테스트 24건과 토큰 없는 constructor 요청의 API 회귀 1건을 추가했다. 단위 검증은 수정 전 16 통과·8 실패 → 수정 후 24 통과이며, 원래 조건 복원 시 같은 8건 실패와 수정 재적용 후 24건 통과까지 확인했다. npm ci → npm run test:build(타입·산출물 검사 포함) → npx playwright install chromium → npx playwright test tests/contact-send.spec.ts --project=desktop --grep '문의 API'(7 passed) → npm test --silent(78 passed, 기존 폭 조건 4 skipped, 31.8초)는 모두 exit 0이고, 커밋 caf9e01(작성자 hkjang, 프로덕션 1개·테스트 2개 파일) 후 작업 트리는 깨끗하다.
- 실패 재현: `not ok 7 - 미등록 유형 "constructor"은 거절한다 (needsReply=false)` / `not ok 8 - 미등록 유형 "__proto__"은 거절한다 (needsReply=false)` — 실제 `{}` / 기대 `{ type: '문의 유형을 골라 주세요.' }`, exit 1; 전체 출력 assets/unit-red.log. 정찰 재현 파일도 2 통과·2 실패.
- 보류 아이디어: 서비스 HTML에서 사이트맵 누락 동적 감지 (가치 3 / 위험 1 / 작업량 S)
  - 나머지 inquiry 순수 함수 계약·경계값 단위 테스트 (가치 3 / 위험 1 / 작업량 S)
  - README·배포 문서의 Worker 경로와 메일 관문 설명 정합성 (가치 2 / 위험 1 / 작업량 S)
  - gzip;q=0 압축 협상 회귀 검증 (가치 3 / 위험 3 / 작업량 S)
- 과제서: 채택 — 상속 키를 허용하는 현 조건과 Worker의 검증 후 토큰 검사 순서가 코드에 그대로 있었고, 지정한 세 파일과 세 수용 기준을 충족했다.
