# 구현 체크포인트
1. 완료 — 두 누락·Read 배선·upload accept·App 인증 후 /guide 연결 재확인. npm ci/typecheck/build/test exit 0 (web-baseline.log); docs 테스트 PASS 3.323초.
2. 완료 — GuidePage.tsx 두 곳만 수정. typecheck/build/test exit 0, 48파일·271개 PASS (web-final.log). diff --check PASS, diff --stat 1파일 4+/3-. /presentations 링크·#import·목차·기존 PPTX/PDF 설명은 diff에서 보존 확인했으며 실제 UI 동작 증거는 아님.
3. 미확인 — npm run dev 기동 후 실제 Chromium에서 /guide#import 접근을 전후 실행. API 8080 미기동(curl exit 7), 로그인 화면의 502 관찰. #import는 렌더되지 않아 두 형식·목차·링크 동작 검증 불가. guide-baseline-access.json/png 및 guide-final-access.json/png 보관. 인증 모킹이나 독립 컴포넌트 렌더링으로 대체하지 않음.
- 기존 경고: npm ci moderate 2건 및 whatwg-encoding deprecated, build 500kB 초과 번들 경고, Vitest environmentMatchGlobs deprecated. 변경 전후 동일 종류이며 의존성·설정 변경 없음; 별도 npm audit·전체 Go race/vet·DB 통합 테스트는 이번 문구 범위에서 실행하지 않음.
- 스킬: 전용 Skill 도구가 없어 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/{completion-verification,systematic-debugging,test-driven-development}/SKILL.md를 읽음. 원인·변경·실행 결과·검증 한계를 기록. 새 테스트 금지와 API 환경 부재로 TDD 실패 재현 및 되돌림 검증 미실시를 명시.
- 아이디어 선택·추가·평가 절차 1~4는 사용자 지시에 따라 정찰 과제서로 갈음. 기존 ideas.json의 13항목(정찰 신규 3항목 포함)을 보존하고 선택 항목 done 및 이번 검증 근거만 갱신.
