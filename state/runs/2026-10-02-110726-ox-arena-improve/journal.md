# 회차 노트 2026-10-02-110726-ox-arena-improve — ox-arena
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:07] base pinned — main@f68a6fa
- [러너 11:07] autonomy release — 

## 정찰 노트
- 테스트 0개·test 스크립트 없음을 확인하고 Vitest 하네스를 1순위로 골랐다. 더 "진짜 버그"인 탈락 모드 정답률 분모 오류(0003_elimination.sql:114 v_total ↔ Display.tsx:50 rv.total ↔ useRoom counts.alive, 세 경로가 분모가 다름)는 마이그레이션=보호 경로이고 이 저장소에 SQL 검증 수단이 전혀 없어 차선으로 미뤘다.
- 추측으로 적은 것: vitest 버전(vite 5 호환은 ^2 계열로 적었으나 **미확인**), vitest 무설정 기본값으로 src/lib/*.test.ts 가 잡힌다는 것(문서 기억 기반, 실행 미확인), src/lib/supabase.ts 의 env fallback(파일 미열람).
- `npm ci`/`npm i` 가 이 샌드박스에서 승인 거부로 막혀 **build 도 test 도 한 번도 실행해 보지 못했다.** 구현자는 손대기 전에 `npm i && npm run build` 로 기준선 녹색부터 확인할 것.
- 조심할 것: rankPlayers 는 정렬에 correct 를 쓰고 랭크는 score 만 본다 — README 의 "동점 공동순위" 와 맞는 현재 동작이니 테스트로 **고정**하고 바꾸지 말 것. counts 추출은 순수 이동만, useRoom 의 폴링/채널은 건드리지 말 것.
- [러너 11:12] scout done — Vitest 도입 + 순수 게임 로직(rankPlayers / derivePhase / 집계) 회귀 테스트 (가치 4 / 위험 2 / 작업량 M)
