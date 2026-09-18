# 과제서 (2026-09-19) — 수정 과제

## 우선 과제 진단 결과: 'error / hold: budget' 은 저장소 결함이 아니다

- 직전 회차 `2026-09-18-162344` 의 `evidence.json` 을 읽었다. `stages.improve = { state: "hold", reason: "회차 예산($22)이 오늘 남은 상한을 넘음" }`, `base_sha`·`head_sha`·`pr` 모두 빈 값, `usage: []`. 즉 **구현 단계가 시작조차 하지 않았다** — 러너가 자기 일일 예산 상한 때문에 회차를 보류한 것을 원장이 `outcome: error` 로 적은 것이다.
- 그 앞 회차 `2026-09-17-023310` 은 `outcome: review-pending`(정상 완료, 커밋 304e89e), 오늘 새벽 `2026-09-19-035516-…-shepherd` 는 `outcome: approve`. 같은 이유로 두 번 실패한 "릴리즈 워크플로" 는 없다.
- 이 저장소에는 `.github/workflows` 가 없다. 유일한 CI 파일 `.gitlab-ci.yml` 은 사내 GitLab(`gitlab.koreacb.com`) 에서 main/develop 푸시 시 `npm run build:*` 후 dist 를 PVC 로 복사하는 배포 파이프라인이며, 이 러너가 돌리는 것이 아니고 실패 기록도 없다(러너의 CI 산출물은 `ci-<sha>.json` 이며 09-17 회차까지 모두 통과).
- 따라서 **코드·워크플로에서 고칠 원인이 없다.** 러너 예산 정책은 이 저장소 밖(aidev 러너 설정)이고 정찰·구현 에이전트의 권한 밖이다. 아래에 원장 기록용 문구를 적고, 이번 회차는 예산을 적게 쓰는 S 과제를 배정한다.

### 원장에 적을 '수정 과제' 기록 (구현자가 그대로 옮길 것)
> 수정 과제: 2026-09-18 회차의 `error` 는 러너 예산 보류(`hold: budget`, "회차 예산($22)이 오늘 남은 상한을 넘음")로 구현이 시작되지 않은 것이며, 저장소 코드·CI 에는 대응하는 실패가 없다(`.github/workflows` 부재, `.gitlab-ci.yml` 은 사내 배포 파이프라인, 09-17 회차 `ci-*.json` 통과). 저장소 측 조치 없음. 러너 측 조치 필요: 예산 보류를 `error` 가 아니라 `hold` 로 분류하도록 원장 분류기 수정을 운영자에게 요청.

---

- 과제: `monitoringParser.toPod` 의 restarts 가 kubectl 스타일 문자열(`"3 (5d ago)"`)에서 NaN 으로 노출되는 문제 수정 (가치 2 / 위험 1 / 작업량 S)
- 왜: `upgrade/admin-v2/src/api/monitoring.ts:43` 이 `Number(row.restarts ?? row.restartCount ?? restartSum)` 이라 backend 가 kubectl 출력 그대로 `"3 (5d ago)"` 같은 문자열을 주면 `NaN` 이 되어 운영 현황 표 칸(`OperationsView.vue:122` `{{ pod.restarts }}`)에 "NaN" 이 그대로 찍힌다. 재시작 MetricCard(`OperationsView.vue:23`)는 `Number(pod.restarts || 0)` 으로 NaN 을 0 으로 접어 합계에서 그 POD 의 재시작이 조용히 빠진다. 선행 정수만 파싱하면 표와 합계가 같은 값을 읽게 된다.
- 수용 기준:
  1) `toPod({ restarts: '3 (5d ago)' }, 'ns').restarts === 3`, `toPod({ restartCount: '12' }, 'ns').restarts === 12`, 숫자 `7` 은 그대로 `7`.
  2) 파싱 불가(`'abc'`, `''`, `null` 이면서 containerStatuses 도 없음)는 `0` — 표에 NaN 이 절대 나오지 않는다.
  3) `status.containerStatuses[].restartCount` 합산 경로(`restartSum`)도 같은 파서를 거쳐 `"2 (1h ago)"` 같은 값이 와도 합이 맞는다(같은 값을 읽는 경로가 둘이므로 둘 다 고친다).
  4) 테스트: `src/api/monitoring.test.ts` 에 위 세 경우를 추가하고, 파서를 원래 `Number(...)` 로 되돌리면 1)·3) 이 실제로 실패하는지 확인한 뒤 되돌린다. OperationsView 쪽은 이미 `pod.restarts` 를 소비하므로 컴포넌트 테스트 추가는 선택(기존 `OperationsView.test.ts` 에 `'3 (5d ago)'` 를 넣은 응답으로 표 셀이 "3", 합계 카드가 그 값을 포함하는지 1개 더하면 end-to-end 증명이 된다 — 권장).
- 건드릴 파일:
  - `upgrade/admin-v2/src/api/monitoring.ts:toPod` — 선행 정수 파서(예: `parseRestarts(value: unknown): number` — 숫자면 `Number.isFinite` 검사 후 반환, 문자열이면 `/^\s*(\d+)/` 매치, 실패 시 0)를 두고 `:43` 과 `:34`(`restartSum` reduce) 양쪽에 적용.
  - `upgrade/admin-v2/src/api/monitoring.test.ts` — 케이스 추가.
  - (선택) `upgrade/admin-v2/src/modules/operations/OperationsView.test.ts` — end-to-end 1개.
- 검증 명령 (모두 `upgrade/admin-v2` 에서):
  - `npm run verify` (vue-tsc + vitest 전체; 09-17 기준 156개 통과)
  - `npm run build` (verify → vite build → runtime-config 검증 → offline 검사 → integrity manifest 19개) — 최종 확인용. 이번 정찰은 예산 때문에 로컬 재실행하지 않았다(미확인). 09-17 회차의 `verify.json`/`ci-*.json` 은 통과 기록.
- 위험과 피할 것:
  - `src/auth/*`, `src/app/router.ts`, `ARCHITECTURE.md` 의 "세션 만료 신호" 절은 건드리지 말 것 — 09-17 커밋 304e89e(PR 승인됨, main 미병합)가 그 파일들을 바꾸므로 충돌 여지.
  - `.gitlab-ci.yml` 은 건드리지 말 것(보호 경로, 사내 배포). 다만 이 파일에 GitLab PAT 가 평문으로 커밋되어 있다 — 코드로 고칠 일이 아니라 운영자에게 토큰 회전·CI 변수화를 알릴 사안이므로 PR 본문에 한 줄 언급만 하고 값은 절대 옮겨 적지 말 것.
  - 루트 앱(`/src`, `package.json` 의 crypto-js tarball)은 테스트가 없으니 손대지 말 것.
  - 효과 없는 변경 금지(운영자 지침): `Number()` 를 감싸기만 하고 문자열 케이스를 실제로 바꾸지 않는 식의 수정은 반려 대상. 파서가 표 셀과 합계 카드 두 경로의 값을 실제로 바꾸는지 테스트로 보일 것.
- 차선 후보: 결과 0건인 탭을 누를 때마다 같은 조회를 되풀이하는 문제 (2/1/S) — `ContentAccessView.changeTab`(:78-79)·`CatalogView`(:140-141)·`DirectoryView`(:73-74)가 `!items.length` 로 재조회를 판단하므로 탭별 "조회 완료" 플래그를 기존 `tabLoading`·`tabError` 레코드 옆에 두면 된다. 세 화면 모두 컴포넌트 테스트가 있어 "0건 응답 후 탭을 오가도 API 가 1번만 호출" 로 증명 가능.
