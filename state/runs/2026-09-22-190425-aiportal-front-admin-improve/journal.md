# 회차 노트 2026-09-22-190425-aiportal-front-admin-improve — aiportal-front-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:04] base pinned — main@01fedba
- [러너 19:04] autonomy release — 

## 정찰 노트
- 선택 이유: 릴리즈 실패의 두 원인 중 저장소가 고칠 수 있는 쪽(관례 문서·CHANGELOG·버전 불일치 0.0.0/0.1.0)을 골랐다. 외부 러너 스킬 미전달을 겨냥한 과제서는 6회 연속 no-change 였으므로 이번엔 그 방향을 버렸다.
- 확신 없는 곳: 원격 태그·GitHub Release 존재 여부 미확인(로컬 `git tag` 만 비어 있음 확인). `npm ci` 가 이 환경에서 레지스트리에 닿는지 미확인(node_modules 비어 있음). 첫 릴리즈를 v0.1.1 로 정한 것은 정찰의 제안이며 사람이 PR 리뷰에서 바꿀 수 있다.
- 구현자가 조심할 것: 태그·버전 0.1.1 상향·러너 편집 금지. 루트 package.json 은 version 만 바꾸고 crypto-js 줄은 손대지 말 것. check-version.mjs 는 4개 파일 6개 값 전부를 한 검사에서 비교해야 한다(한쪽만 넓히지 말 것 — 운영자 규칙). 과제 완료 후에도 러너 스킬 결함으로 다음 릴리즈가 실패할 수 있음을 원장에 솔직히 적을 것.
- [러너 19:08] scout done — 수정 과제 — 릴리즈 실패의 저장소 측 원인 제거: 릴리즈 관례 문서·CHANGELOG·버전 단일화·버전 일치 검�

## 구현 노트
- 무엇/왜: 릴리즈 단계가 저장소에서 못 찾던 관례를 `docs/RELEASE.md`(버전 원본 `upgrade/admin-v2/package.json`, `v<version>` 주석 태그, `chore(release): v<version>`, patch 증가·첫 릴리즈 v0.1.1, CHANGELOG 절=Release 본문·자산 없음)로 확정하고 `CHANGELOG.md`(`[Unreleased]` fix 4건)를 추가, 루트 버전 0.0.0→0.1.0 동기화, `scripts/check-version.mjs`(4파일 6값 한 검사)+`npm run check:version`. 커밋 1d64d8e 하나.
- 검증: 변경 전 기준선 verify 25파일 193테스트 통과 → 변경 후 26파일 202테스트 통과. `check-version.mjs` 실제 저장소 exit 0, 루트 0.1.9 로 바꾸면 exit 1 후 복원, `/tmp` cwd 에서도 exit 0. 새 테스트 9개는 대역 없이 실제 스크립트를 node 자식 프로세스로 실행한다(임시 디렉터리 픽스처 + 실제 저장소 1건).
- **확신 없는 곳**: 변경 후 첫 `npm run verify` 에서 기존 갱신 표시 테스트 4파일이 실패(마지막은 `OperationsView.test.ts:95` `toHaveBeenCalledTimes(2)`). 이후 일반 5회·CPU 부하 3회·shuffle 3회·직렬 2회·cold cache 3회·verify 4회 = 20회 전부 통과해 원인 미입증. 실패 로그는 tail 만 남아 정확한 4파일 이름은 미확정. 내 테스트가 자식 프로세스 9개를 동기 실행해 부하를 만든 것이 가설이었으나 32코어 부하 재현에도 실패. 비평가는 이 부분을 먼저 보길 바람.
- **검증 못 한 것**: 원격 태그·GitHub Release 존재 여부(RELEASE.md 에 미확인으로 명시). `check-version.test.ts` 는 tsconfig include(src/**) 밖이라 vue-tsc typecheck 를 받지 않고 vitest 로만 돈다.
- 일부러 하지 않은 것: 태그·0.1.1 상향·러너/스킬 편집(과제서 금지), `verify`/`build` 본문 변경(check:version 은 별도 스크립트로만), 루트 `crypto-js` file: 의존성·루트 `npm ci`, README '검증 상태' 절의 낡은 수치(21 unit) 수정.
- 다음 역할이 조심할 것: 스킬 도구 미전달(`marketing:product-launch`, `technology:release-and-deployment`)은 외부 러너 결함이라 다음 릴리즈가 같은 이유로 실패할 수 있음. 릴리즈 단계는 RELEASE.md 대로 6개 버전 값을 한꺼번에 0.1.1 로 올리고 `check-version.mjs` 로 확인해야 하며, 루트에서 `npm version` 을 쓰면 안 됨(crypto-js tarball 부재).
- [러너 19:18] brief accepted — 채택 — 과제서 근거(로컬 태그 0개, 관례 문서 없음, 루트/admin-v2 버전 불일치, node_modules 비어 있음)가 현재 코드와 일치�
- [러너 19:18] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인함: diff 9파일 전부 읽음. verify(vue-tsc+vitest 26/202) 통과, check-version.mjs 두 cwd 에서 exit 0, 새 테스트 9개는 실제 스크립트를 자식 프로세스로 실행해 각 값의 불일치를 exit code·stderr 로 단언 — 수정 전 코드에선 통과 불가. CHANGELOG 4건은 main 이력과 일치. 인증·비밀값·개인정보·의존성 변경 없음 → 차단 없음, approve.
- 구현자가 의심한 flakiness 를 시험함: 새 테스트 포함 9회 중 1회 실패(기존 Catalog/ContentAccess race 테스트 3건), 제외하고 15회 0회 실패. 새 테스트의 execFileSync 9회 부하가 setImmediate 기반 flushPromises 경합을 키우는 것으로 보이나 통계적 확정은 아님.
- 못 봄: 원격 태그·GitHub Release 존재 여부(git ls-remote 자격증명 없음). 
- 릴리즈·다음 회차: verify 가 릴리즈 전 검사 게이트이므로 간헐 실패가 릴리즈 단계를 막을 수 있음 — 실패 시 재실행하되, 다음 회차에서 check-version.test.ts 부하 축소(비동기 execFile/케이스 축소) 또는 race 테스트 flush 결정화 권장. RELEASE.md 의 `github_release: true` 는 러너 용어라 문서 정리 대상. 첫 릴리즈 v0.1.1 노트는 #18~#21 만 담고 #15·#16 이전 fix 는 빠짐.
- [러너 19:21] review approved — 리뷰 승인 (risk=medium)
- [러너 19:21] pr created — https://github.com/hkjang/aiportal-front-admin/pull/22
- [러너 19:22] ci passed — 검사 없음 — 정책으로 허용
- [러너 19:22] merge done — 1d64d8e
- [러너 19:25] release published — v0.1.1
- [러너 19:25] gh-release created — GitHub Release v0.1.1
- [러너 19:25] assets n/a — 이전 릴리즈에도 자산 없음
