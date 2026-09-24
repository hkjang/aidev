# PR 처리기 노트 2026-09-23-114056-releasedock-shepherd — releasedock PR #22
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-23-110433-releasedock-improve)
# 회차 노트 2026-09-23-110433-releasedock-improve — releasedock
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:04] base pinned — main@31f64a9
- [러너 11:04] autonomy release — 

## 구현 노트
- 무엇/왜: `SimpleRunDetailPage` 의 SSE effect 가 `useAsync` 의 매 렌더 새 객체(`run`)를 의존성에 둬서 로그 한 줄마다 EventSource 를 재연결했다(실제 라우트 렌더 테스트에서 4줄 → 연결 5개). 서버는 사용자당 3스트림·해제는 최대 1초 뒤라 초당 몇 줄이면 429 로 로그가 조용히 멈춘다. 의존성을 안정적인 `run.reload` 로 바꿨다.
- 함께 처리: 고치고 나니 서버 30분 상한의 `end {"reason":"max_duration"}` 뒤 재연결(옛 폭주가 우연히 해주던 복구)이 사라져, end 프레임을 읽어 종료 상태면 닫힌 채 두고 이유만 있으면 상태 재조회 후 재연결하도록 명시했다(`streamEndedRun` + `streamAttempt`).
- 확신 없는 곳: 실제 브라우저·실제 서버로 30분 초과 실행을 돌려 재연결을 본 적은 없다(테스트의 EventSource 는 jsdom 에 없어 만든 대역이고, 페이지·effect·useAsync·라우팅은 프로덕션 그대로). 429 가 실제로 발생하는 장면도 재현하지 않았고 서버 `acquireLogStream` 코드 읽기로만 근거를 세웠다.
- 일부러 안 한 것: EventSource `error` 를 화면에 표시하는 것(별건, 보류 목록에 남김), `SimpleDeployPage`·`ReleaseDetailPage` 의 스트림(둘 다 의존성이 안정적인 원시값이라 같은 결함 없음 — 확인함), 문서 수정(docs/simple-mode.md 288줄이 이미 약속한 동작이 비로소 사실이 된 것이라 문구 변경 없음).
- 다음 역할 주의: 웹 테스트만으로 검증되는 변경이다. `web/src/test/setup.ts` 에 jsdom 용 `scrollIntoView` 정의를 추가했으니 다른 컴포넌트 테스트에도 영향이 간다(없으면 로그 페이지 렌더가 예외로 죽는다). `make test` 의 backend 통합 테스트는 `TEST_POSTGRES_DSN` 이 있어야 돈다 — 도커 PostgreSQL 16 으로 채워 129 PASS 를 확인했다.
- [러너 11:14] verify passed — 검증 7개 통과 (auto)
- [러너 11:28] review timeout — 단계 제한 시간 초과
- [러너 11:28] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 11:28] pr created — https://github.com/hkjang/releasedock/pull/22

## 심사 노트
- 확인: origin/main 소스로 되돌리면 신규 스트림 테스트가 연결 5개로 실패(수정 후 1개) — 회귀가 실제로 고정된다. streamEndedRun 의 종료 판정을 변이시키면 해당 테스트가 실패하므로 새 로직도 변이에 잡힌다.
- 확인: 클라이언트 TERMINAL 과 simple.go:1211/1221 의 end 프레임 두 형태(status / reason:max_duration), migrations/017 의 status CHECK 집합이 모두 일치. 두 경로가 같은 값을 같게 읽는다.
- 확인: npm ci 후 vitest 98건 통과, tsc -b --noEmit 통과. 보호 파일·엔드포인트·의존성·비밀값 변화 없음 → security/legal 차단 소견 없음.
- 못 본 것: 실제 브라우저·실제 서버의 30분 초과 재연결과 429 발생 장면(EventSource 는 jsdom 에 없어 전송 계층만 대역, 페이지·effect·useAsync·라우팅은 프로덕션). backend/runner 테스트는 diff 무관이라 미실행.
- 권고 근거: 결함을 못 찾았고 프런트 한정·revert 로 완전히 되돌아온다. 종료 end 뒤 reload 가 실패하면 자동 재시도가 없다는 점은 429 폭주를 없애기 위한 의도된 교환으로 판단해 notes 에만 남겼다 → approve/merge.
