# 회차 노트 2026-09-29-081231-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:12] base pinned — main@1d13889
- [러너 08:12] autonomy release — 

## 정찰 노트
- 선택: ab36254의 CI e2e 실패 요약 복원. 현재 main에 없고 patch 사전검사 통과; 새 Go 요약·로그 보강보다 검증 이력과 두 파일 범위가 명확하다.
- 확인: make check 통과, 원본 script 문법·파일 누락 exit 0. 실제 e2e·패키지·원격 CI/릴리즈 및 작업 시간은 미확인; 과거 성공과 이번 실행을 과제서에 구분했다.
- 주의: ci.yml 진단 단계만 복원하고 gate/renderer/기본 임계값은 유지. 새 접근성 blocking 유실·커스텀 출력 경로 후보는 별도 회차로 분리한다.
- 세 요청 스킬을 headcount 로컬 경로에서 읽어 적용(전용 Skill 도구 없음); 초안 후 갱신 완료. 기존 12후보는 아직 유효해 pending 유지, 새 후보 2개 추가; 이전 프로필의 main 미반영 정보를 수정했다.
- [러너 08:18] scout done — 유실된 CI e2e 실패 요약 커밋 `ab36254`를 현재 main에 복원 (가치 4 / 위험 2 / 작업량 M)

## 구현 중 체크포인트
- 세 요청 스킬 로컬 SKILL.md 적용. 검증 전 커밋 금지와 과제서 cherry-pick을 함께 만족시키기 위해 --no-commit 적용 후 마지막에 원본 메시지로 커밋 예정. 원본 두 파일 byte-identical, 149줄 추가.
- 상태 디렉터리 check-summary.py로 실제 workflow run 명령을 읽어 실행: 복원 전 summary 누락 assertion 실패 → 복원 후 통과 → 옛 workflow 재주입 시 재실패 → 복원 후 통과. GitHub runner 조건 평가는 로컬에서 재현하지 않음.
- 최초 make image 실패: 기존 AuthContext.test.tsx:92 expected true to be false, 249/250 통과. 별도 effect 완료 경합 가설; 범위 밖으로 수정하지 않음. 같은 빌드 한 번 재확인 중이며 최초 실패를 성공으로 숨기지 않는다.

## 구현 노트
- df9dd4b: ab36254의 두 파일 byte-identical 복원(149줄, ci.yml 7줄); if: failure() 요약 append가 기존 artifact 앞에 위치. cherry-pick --no-commit → 검증 → 원본 메시지 커밋, 작업 트리 clean.
- TDD: check-summary.py로 옛 workflow summary 누락 assertion RED → 복원 GREEN → 원복 RED → 재복원 GREEN; 실제 YAML run 명령 실행 및 기존 summary 보존. 테스트는 두 파일 범위를 지키려고 상태 디렉터리에만 보존.
- 정상 검증: make check(120 route)·node --check·누락/깨진 JSON exit 0; 공식 noble Node 24 + 새 PG17 정상 npm test 52 visual·36 Axe·9 keyboard/media/reflow·27 smoke 통과, normal-summary.md는 3개 모두 통과.
- 실제 실패: 새 DB, 명령 한정 maxDiffRatio=0에서 26/52 실패(exit 1), version=v0.0.0-invalid smoke exit 1; failure-summary.md/actual-workflow-summary.md에 dark-desktop-admin-smtp 0.345%·diff 경로·locator·browser-smoke.mjs:112:154 확인, 통과 접근성·화면은 실패 목록 제외.
- 확신 없는 곳: 최초 make image는 기존 AuthContext.test.tsx:92 실패(249/250), 동일 조건 2회차 250/250·빌드 통과. effect 시점 경합은 가설이며 해결하지 않음(image.log/image-second.log); 원격 source/image CI·실제 릴리즈·Actions summary 공개 가시성 미검증.
- make package/verify-package 각 1회 exit 0(상태 디렉터리 package/); 커밋 전 빌드여서 OCI revision은 base HEAD. 다음 역할은 PR source/image CI와 release.yml exact commit CI gate·실제 릴리즈 성공을 따로 확인할 것.
- 임계값·baseline·테스트 exit·artifact·cleanup·release/auth/production은 무변경. 정상 suite 1회·각 의도 실패 1회만 실행, 전용 컨테이너/볼륨/네트워크 제거. 접근성 blocking 실패·커스텀 경로·임의 schema 안전성은 이번 미검증/별도 후보.
- [러너 08:27] brief accepted — 채택 — 현재 파일과 근거가 일치해 원본 두 파일을 그대로 복원했고, 검증 전 커밋 금지 지시를 지키기 위해 cherry-pick --n
- [러너 08:28] verify passed — 검증 7개 통과 (auto)
- [러너 08:28] pr created — https://github.com/hkjang/moina/pull/36
- [러너 08:28] guard held — .github/workflows/ci.yml 
