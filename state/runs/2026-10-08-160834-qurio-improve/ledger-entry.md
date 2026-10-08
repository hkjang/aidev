## 2026-10-08
- 선택: 웹 SSE 종료·예외 경로의 reader 취소와 잠금 해제 보장 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: parseServerSentEvents의 읽기 루프와 마지막 dispatch에 수명 정리가 없어 정상 EOF·읽기 오류에서 잠금이 남고 서버/콜백 예외에서 열린 스트림이 취소되지 않는 원인을 재현한 뒤, 예외 시 reader.cancel을 await하여 best-effort 처리하고 원래 예외를 다시 던지며 finally에서 releaseLock하도록 수정했다. 공개 apiClient → 실제 파서와 Response/ReadableStream을 통과하는 회귀 7건을 추가하고 기존 EOF·CRLF·마지막 빈 줄 없는 이벤트·agent/legacy error→done 테스트를 보강했으며, `npm test --prefix web -- --run src/lib/api.test.ts`는 수정 전 11 failed/8 passed → 수정 후 19 passed, 정리 코드만 되돌리면 동일 11 failed/8 passed를 확인하고 복원한 현재 트리에서 `npm test --prefix web -- --run`은 27파일/141테스트 통과(exit 0), `npm run typecheck --prefix web`·`npm run lint --prefix web`·`npm run build --prefix web -- --outDir /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-08-160834-qurio-improve/web-dist`도 exit 0, `node --experimental-transform-types /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-08-160834-qurio-improve/sse-probe.mjs`는 exit 1 → 0이었다. 커밋 b5fd4c0(hkjang, 트레일러 없음)은 프로덕션 1파일+테스트 1파일뿐이며 작업 트리는 깨끗하고, build의 외부 outDir 자동 비우기 생략 안내 외 검증 문제는 없었으나 브라우저 실제 연결 잔존 시간·자원 절감량과 Go·DB·E2E는 검증하지 않았다.
- 실패 재현: `AssertionError: expected "vi.fn()" to be called 1 times, but got 0 times` / `AssertionError: expected true to be false // Object.is equality` — sse-red.log, 취소 누락 5건·잠금 잔존 6건; 오류 객체·ApiError 필드와 기존 이벤트 순서 단정은 먼저 통과했다.
- 보류 아이디어: SSE UTF-8 멀티바이트 네트워크 분할 회귀 테스트 (가치 3 / 위험 1 / 작업량 S) — 정찰의 신규 차선 유지, 이번엔 reader 수명만 변경.
- 보류 아이디어: 다운로드 Content-Disposition의 잘못된 percent 인코딩에 filename 폴백 (가치 2 / 위험 1 / 작업량 S) — 정찰의 신규 후보 유지, 이번 구현에서는 재측정하지 않음.
- 보류 아이디어: 웹 스위트 전수 감사 — 응답 전 기본 상태로도 충족되는 findBy 단정 (가치 4 / 위험 1 / 작업량 M) — 이번 전체 실행은 통과했지만 전수 감사·flake 부재 증명은 아님.
- 보류 아이디어: AnalyzeDialect 사유 개수 상한 (가치 3 / 위험 3 / 작업량 S) — 사유 집합 계약을 바꾸는 독립 과제로 유지, 이번 웹 변경에 혼합하지 않음.
- 과제서: 채택 — 기준 cbbf578과 reader 정리 누락을 현재 코드·실행으로 확인했고 지정한 두 파일에서 수용 기준을 충족했으며 차선으로 바꿀 이유가 없었다.
