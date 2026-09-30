# 회차 노트 2026-09-30-230312-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:03] base pinned — main@dcb5b8a
- [러너 23:03] autonomy release — 

## 정찰 노트
- 고른 이유: 바로 앞 회차가 허용 출처 **한 항목**의 상한을 닫으면서 목록 전체는 열어 뒀다. 항목마다 세 지시문에 들어가므로 목록 n 룬이 헤더 3n 룬이 되는 증폭을 코드에서 직접 확인했고(tracking.go:330-332 → api/tracking.go:101-111), 프로덕션 파일 한 개·쓰기 경로 거절 한 줄로 끝난다. guide-shots 요약 건은 값이 낮고(로그 출력만), CSP 위반 전체 삭제·CI 스크립트 편입은 각각 개별 삭제 API 부재와 보호 경로 때문에 계속 미뤄 왔다.
- 추측으로 적은 것: 상한 2048 룬/64 항목은 내가 고른 숫자다 — 실사용 허용 목록 크기 통계는 없고, 프록시의 응답 헤더 8KB 관례도 이 저장소에서 확인한 값이 아니다. 구현자는 숫자를 바꿔도 되지만 근거를 상수 주석에 남길 것.
- 조심할 것: `PolicySources`/`pagePolicy` 에서 자르지 말 것 — 저장값과 헤더가 갈라진다. 거절이지 정규화가 아니다(앞 회차가 같은 이유로 그렇게 골랐다).
- 미확인: live 테스트는 DSN 이 없어 이 세션에서 돌리지 않았다. `go test ./internal/tracking ./internal/api` 가 DSN 없이 통과하는 것만 확인했다(api 1.6초).
- [러너 23:07] scout done — 추적 허용 출처 **목록 전체**의 크기에 상한을 두어 정책 헤더가 무한히 커지지 못하게 한다 (가치 3 / 위�

## 구현 노트
- 무엇을 왜: `Settings.Validate()` 에 허용 출처 **목록**의 상한(`MaxAllowedHostEntries=64` / `MaxAllowedHostsTotalRunes=4096`, 룬 단위)을 항목별 루프 뒤에 더했다. 항목마다 세 지시문에 들어가 목록 n 룬이 헤더 3n 룬이 되는데 목록 크기에는 제한이 없었다. 프로덕션 파일은 `internal/tracking/tracking.go` 하나, 거절이지 정규화가 아니며 `PolicySources`/`pagePolicy` 는 손대지 않았다.
- 확신 없는 곳: **숫자 4096/64 는 측정값이 아니다.** 프록시의 8KB 응답 헤더 관례는 이 저장소에서 확인한 값이 아니고(정찰도 추측이라 적었다), 실제 허용 목록 크기 통계도 없다. 과제서의 2048 은 쓸 수 없었다 — `tracking_test.go:181-189` 가 40개×81룬(3240룬) 목록을 "목록에는 상한이 없다" 며 통과시키므로 수용 기준 2 와 충돌한다. 4096 은 그 목록이 안쪽에 남는 가장 작은 2의 거듭제곱으로 골랐고 헤더는 약 12.5KB 로 묶인다 — 8KB 보다는 크다. 숫자를 낮추려면 그 기존 테스트를 함께 판단해야 한다.
- 일부러 하지 않은 것: 콘솔 UI(`AdminSettings.tsx`)에 상한 안내를 넣지 않았다(프로덕션 파일 1개 유지 — ideas.json 에 신규 후보로 적음). `PolicySources` 중복 제거도 그대로 뒀다.
- 다음 역할이 조심할 것: live 서브테스트("the list as a whole stops at its limit on both routes")는 DB 가 있어야 돈다 — 이 세션에서는 `docker run --rm -d -p 55447:5432 postgres:16-alpine` 로 실제 실행해 통과와 되돌림 실패를 모두 확인했다(컨테이너는 제거했다). 한 번 클릭 허용 라우트는 과제서의 `/api/v1/admin/tracking/allow` 가 아니라 `/api/v1/admin/tracking/violations/allow` 다. `runtime-images.json` 의 어떤 이미지도 `internal/tracking`·`internal/api` 를 sourcePaths 에 두지 않아 BASE_VERSION 상향은 불필요(매니페스트 직접 확인; check-versions 스크립트는 실행하지 않았다).
- [러너 23:16] brief accepted — 채택 — 3n 증폭과 무제한 목록, 두 쓰기 경로의 공통 지점이 현재 코드와 정확히 일치했다. 두 가지는 과제서와 달랐다: (
- [러너 23:16] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 확인: diff 는 tracking.go 한 개 + 테스트 3개, 범위 이탈·마이그레이션·의존성 변경 없음. 두 쓰기 경로(routes.go:1546, api/tracking.go:274)가 모두 Validate() 를 지나고 우회하는 설정 복원 경로는 internal/api 에서 찾지 못했다. 경계는 정확하다 — 64개/4096룬 정확히는 통과, 개수만 초과·합계만 1룬 초과·한국어(룬 vs 바이트) 세 방향 모두 단언이 있고, 헤더 증가분을 3*(4096+64)=12480 으로 정확히 고정한다. 원장의 실패 재현 줄 번호가 새 테스트 실제 줄과 일치해 수정 전 실패가 이번 증상과 맞는다.
- 못 봄: live 서브테스트는 DSN 이 없어 돌리지 않았다(구현자의 일회용 postgres 결과에 의존). 실물 브라우저·프록시에서의 헤더 수용 여부도 미확인.
- 승인이어도 남는 우려 ①: 64/4096 은 측정값이 아니라 기존 테스트(40개x81룬)가 안쪽에 남는 값으로 고른 것이라, 허용 목록만으로도 헤더가 약 12.5KB — 주석이 스스로 인용한 8KB 프록시 예산은 여전히 넘는다. 증가를 묶을 뿐 기본 예산 안에 든다는 보장이 아니다.
- 승인이어도 남는 우려 ②(릴리즈 노트감): 이미 상한을 넘긴 목록이 저장된 배포는 이후 tracking 설정 PUT 이 무관한 필드만 고쳐도 400 이 된다. 오류가 목록과 개수를 알려 주므로 조치 가능하고 revert 는 코드뿐이지만, 운영자에게 미리 알릴 값어치가 있다.
- 다음 회차: pagePolicy 가 Sources.all() 을 쓰지 않아 ProviderCustom 스니펫 출처는 여전히 무제한 3배(8KB 스니펫 → 약 24KB 헤더)로 열려 있다. 콘솔 UI 상한 안내도 미반영.
- [러너 23:20] review approved — 리뷰 승인 (risk=low)
- [러너 23:20] pr created — https://github.com/hkjang/AgentHub/pull/39
- [러너 23:22] ci passed — 검사 1개 모두 success
- [러너 23:22] merge done — 6496e46
- [러너 23:34] release published — v0.256.0
- [러너 23:41] assets verified — v0.256.0 자산 8개 (이전 v0.255.0: 8)
