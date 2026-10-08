## 2026-10-09
- 선택: Node StaticFiles의 실제 파일 기반 캐시·HEAD·SPA·경로 격리 회귀 테스트 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: apps/server/src/__tests__/static.test.ts 한 파일에 실제 임시 디렉터리·파일·Request/Response·StaticFiles를 사용하는 14건을 추가했다. 자산 GET/HEAD의 MIME·UTF-8 길이·캐시, 셸 GET/HEAD의 보안 헤더·CSP·no-store, 같은 크기 파일의 utimes 기반 캐시 갱신, 잘못된 인코딩·NUL·encoded slash 형제 디렉터리 접근의 셸 fallback을 검사하며 모든 GET 본문을 소비하고 afterEach로 fixture를 정리한다. Node v22.23.1에서 npm ci, 대상 테스트(최초 4/4 → 최종 14/14), npm run check(린트·core/server/web 및 workers/db-watch 타입 검사·32파일 274건), npm run build 모두 exit 0; 프로덕션 변경 0개, 커밋 45c4e76.
- 실패 재현: 못 함 — 정찰 과제서가 명시한 현재 정상 동작의 회귀 방지 과제이며 추가 테스트는 최초 실행부터 통과했다. 결함 수정이나 red→green을 주장하지 않고, static.ts·index.ts·Vitest 설정도 수정하지 않았다.
- 보류 아이디어: PR CI(lint/typecheck/test) [5/3/M] — 보호 경로이며 이번 범위 밖.
- 보류 아이디어: 연봉 RLS E2E의 쿼리·파싱 실패를 성공 처리하지 않게 한다 [4/2/M] — 격리 DB 재현 필요.
- 보류 아이디어: 사진 입력 바이트·형식 한계 AppError 회귀 테스트 [3/1/S] — 이전 사진 과제와 충돌 여부 확인 필요.
- 보류 아이디어: 공개 문서 생성물의 버전 동기화 [2/1/S] — 지정 1순위를 완료하여 차선은 수행하지 않음.
- 과제서: 채택 — 현재 StaticFiles와 생산 호출부의 URL pathname 전달 및 Vitest 구성이 정찰 근거와 일치하며 지정 테스트 한 파일만으로 수용 기준을 검증했다.
