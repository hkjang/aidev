## 2026-10-07
- 선택: Markdown 내부 링크의 역슬래시 우회 차단 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: safeHref의 내부 경로 분기에서 원문 역슬래시를 거절해 외부 목적지로 해석되는 경로가 속성 없는 내부 anchor로 출력되지 않도록 했다. 실제 Markdown export를 createElement + renderToStaticMarkup으로 실행하는 13개 시험에서 슬래시·역슬래시 변형의 span 표시, 정상 내부 경로 및 http(s) 속성 보존, javascript:/data: 거절을 검증했다. npm ci 성공(ESLint 지원 종료 경고 1건), 지정 시험 13개·npm run check(린트/세 workspace 타입 검사/전체 54개 시험)·npm run build(웹/서버) 모두 통과했으며 커밋 c1f9a2b에 프로덕션 1파일+시험 1파일만 포함했다.
- 실패 재현: `Tests  4 failed | 9 passed (13)`
  `Received: "<div class="bz-prose"><p><a href="/\example.org/path">링크</a></p></div>"`
- 보류 아이디어:
  - 원장 CSV 수식 중립화와 인용 처리의 API 회귀 시험 (가치 4 / 위험 2 / 작업량 M): DB 및 fixture 준비가 필요해 기존 pending 유지.
  - 달력 날짜 검증의 윤년·월말 회귀 시험 보강 (가치 3 / 위험 1 / 작업량 S): 1순위 결함을 재현하여 차선은 보류.
  - 클립보드 복사 폴백 예외 시 textarea 정리 (가치 2 / 위험 1 / 작업량 S): 브라우저 예외 재현과 실패 반환 계약 확인을 다음 회차로 유지.
  - 상대 시간 표기의 분·시·일 경계 확인과 회귀 시험 (가치 2 / 위험 1 / 작업량 S): 제품의 반올림 의도 확인이 필요하여 보류.
- 과제서: 채택 — 지정된 실제 컴포넌트 출력 시험에서 결함을 재현했고 내부 경로 허용 조건만 좁혀 모든 수용 기준을 만족했다.
- 원인 검증: Node URL이 원문 /\example.org/path를 https://example.org/path로 해석함을 확인했다. 추가 조건만 임시 제거하자 동일 4개 시험이 다시 실패했고 복원 후 13개 통과; 시험 구성 오류나 다른 파서 변경에 따른 효과가 아님을 확인했다.
- 한계: 실제 브라우저 클릭·DB/OAuth e2e·원격 CI는 실행하지 않았다. 이번 검증 경계는 공유 컴포넌트의 최종 HTML이며, 저장 문서에 의도된 역슬래시 링크가 존재하는지는 미확인이다. percent-decode·URL 정규화·문법 확대·다른 파서 통합 및 의존성 변경은 하지 않았다.
- 스킬: 전용 Skill 도구가 없어 technology의 completion-verification, systematic-debugging, test-driven-development SKILL.md를 로컬 파일로 읽고 실패→최소 수정→통과→조건 제거 재실패→복원 통과 및 전체 검증 절차를 적용했다.

