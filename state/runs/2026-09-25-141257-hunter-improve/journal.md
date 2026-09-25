# 회차 노트 2026-09-25-141257-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:13] base pinned — main@4ae4034
- [러너 14:13] autonomy release — 
- [러너 14:17] scout done — PR #10 / v1.15.0 태그에서 실패한 Verify·Release 워크플로의 실제 실패 단계를 확인하고, `web/src/tracking-state.ts` �

## 구현 노트
- 배정된 "실패한 워크플로" 는 이미 고쳐져 있었다. 공개 Actions API 로 확정: 실패 run 은 `36085120189`(PR 브랜치 691a7530, 실패 단계 6 `npm ci && npm test && npm run build`) 하나뿐이고 `e503f24` 가 고쳤다(그 뒤 run 3건 success, main@4ae4034 Verify success). 로그 본문은 403(admin 필요)이라 `git archive 691a7530` → `node:26-bookworm-slim` 으로 재현해 실패 문구를 확인했다: `origin/invalid-punycode: http://xn--a.internal !== null`. 릴리즈 워크플로는 실패가 아니라 **한 번도 실행되지 않았다** — 태그가 v1.10.0 까지만 존재한다(세션 규칙상 태그·버전 미조작).
- 그래서 릴리즈 경로의 실제 공백을 고쳤다: CI 가 `release-notes.py` 를 `py_compile` 로만 확인해 본문이 한 번도 생성되지 않았고, 열린 `version >= (1,N,0)` 체인이 다음 마이너에 이전 릴리즈 안내를 그대로 재게시했다(v1.16.0 생성 시 v1.15.0 안내 + `docs/release-v1.15.0.md` 링크, 실측). 마이너 동등 비교로 바꾸고 `scripts/check-release-notes.mjs` 를 CI 에 추가했다.
- **확신 없는 곳**: (1) 과제서의 수용기준 1(실제 로그 마지막 줄 인용)은 로그 다운로드 403 때문에 **인용이 아니라 Node 26 재현**으로 대체했다. (2) `gh release create/upload` 는 GitHub 쓰기 권한이 없어 미수행 — 릴리즈 워크플로의 마지막 단계만 미검증이다. (3) v1.16.0 처럼 분기 없는 태그가 기능 안내 없이 릴리즈되는 것이 맞는 동작이라고 판단했는데(하드 실패는 릴리즈를 깨뜨리므로 피함), 대신 CI 가 먼저 실패하도록 했다 — 이 정책 선택은 내 판단이다.
- **일부러 하지 않은 것**: `web/src/tracking-state.ts` 는 건드리지 않았다(Node 26.10.0 에서 95/95 통과, 263~276행이 파서 결과를 Go 규칙으로 재검사해 수용기준 2 를 이미 충족). `docs/validation.md` 도 손대지 않았다 — 항목이 릴리즈별 기록이고 세션 규칙이 릴리즈 기록 편집을 금지한다. 차선 후보(OIDC return_to)는 착수하지 않았다.
- **다음 역할이 조심할 것**: 새 검사기는 `python3` 를 실행한다(`hashlib.file_digest` 이므로 Python 3.11+ 필요, Node 26.10.0 + Python 3.11.2 컨테이너에서 통과 확인). VERSION 을 1.16.0 으로 올리는 릴리즈 세션은 `release-notes.py` 에 `elif series == (1, 16):` 분기를 **반드시** 추가해야 하고, 없으면 CI 의 `Verify deployment scripts` 가 실패한다(의도된 게이트). 전체 Go 스위트는 PostgreSQL 이 있어야 돌고(이번엔 로컬 55432 의 postgres:17 컨테이너), 결과는 exit 0 · internal/app 672.403s · pentagicore 1.833s · SKIP 0 · DATA RACE 0 이다 — 이 커밋은 Go 코드를 바꾸지 않았으므로 CI 동등성 확인용이다.
- [러너 14:31] brief rejected — 기각 — 1순위 근거가 이미 해결됐다(과제서가 지목한 Node 26 xn-- 실패는 `e503f24` 로 고쳐졌고 main CI 는 success, Node 26.10.0 에�
- [러너 14:32] verify passed — 검증 9개 통과 (auto)
- [러너 14:32] pr created — https://github.com/hkjang/hunter/pull/11
- [러너 14:32] guard held — .github/workflows/ci.yml 
