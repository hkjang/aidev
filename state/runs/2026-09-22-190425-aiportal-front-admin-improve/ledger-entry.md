## 2026-09-22
- 선택: 수정 과제 — 릴리즈 실패의 저장소 측 원인 제거: 릴리즈 관례 문서·CHANGELOG·버전 단일화·버전 일치 검사 추가 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 릴리즈 단계가 저장소에서 찾지 못하던 입력(태그·릴리즈 커밋·증가 규칙·릴리즈 노트 관례)을 `docs/RELEASE.md` 로 확정하고, 루트 `CHANGELOG.md`(`[Unreleased]` 에 PR #18~#21 fix 4건)를 추가했으며, 루트 `package.json`/`package-lock.json` 버전을 0.0.0→0.1.0 으로 `upgrade/admin-v2` 와 맞췄다. `upgrade/admin-v2/scripts/check-version.mjs` 가 4개 파일 6개 값을 한 검사로 비교(불일치 시 파일·값 출력 후 exit 1)하고 `npm run check:version` 으로 노출했다. 검증: 변경 전 기준선 `npm ci && npm run verify` 25파일 193테스트 통과 → 변경 후 26파일 202테스트 통과(새 테스트 9개는 실제 스크립트를 node 자식 프로세스로 실행해 6개 값 각각의 불일치·파일 누락·실제 저장소 일치를 확인), `node upgrade/admin-v2/scripts/check-version.mjs` exit 0, 루트 package.json 을 임시로 0.1.9 로 바꾸면 exit 1(파일·값 출력) 후 복원 확인, `/tmp` cwd 에서도 exit 0. 커밋 1d64d8e 하나. 태그·버전 상향·러너 편집은 하지 않았다. 주의: 스킬 도구 미전달(`marketing:product-launch`, `technology:release-and-deployment`)은 외부 러너 소유 결함으로 이 변경으로 없어지지 않으며, 다음 릴리즈가 그 이유로 여전히 실패할 수 있다. 변경 후 첫 `npm run verify` 에서 기존 갱신 표시 테스트 4파일이 1회 실패했으나 이후 부하·shuffle·직렬·cold cache·verify 반복 20회 모두 통과해 원인은 미입증(구현 노트 참고).
- 보류 아이디어:
  - AdvancedPolicyView 저장·새로고침이 미저장 편집을 덮어씀 (3/2/M, pending)
  - check-version 검사를 admin-v2 build 스크립트 앞단에 연결 (2/2/S, pending)
  - 루트 crypto-js local tarball 의존성 제거 (4/4/M, pending)
  - 갱신 표시 테스트 4파일의 간헐 실패 원인 규명 (2/1/S, pending — 이번 회차 1회 관측, 20회 미재현)
- 과제서: 채택 — 과제서 근거(로컬 태그 0개, 관례 문서 없음, 루트/admin-v2 버전 불일치, node_modules 비어 있음)가 현재 코드와 일치해 그대로 구현했고 수용 기준 1~5 를 모두 충족했다.
