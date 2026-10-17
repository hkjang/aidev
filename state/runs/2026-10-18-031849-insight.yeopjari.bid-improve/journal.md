# 회차 노트 2026-10-18-031849-insight.yeopjari.bid-improve — insight.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:18] base pinned — main@601659d
- [러너 03:18] autonomy release — 
- [러너 03:18] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- 무엇/왜: check.yml 의 `setup-node` `cache: npm` 을 떼고, 같은 캐시를 `actions/cache/restore@v4` + `save@v4` 두 step(`continue-on-error: true`, 새 키 이름공간 `npm-1-...`)으로 옮겼다. 로그의 BOM+`BlobNotFound`+`Time:...9857639Z` 는 Azure Blob = Actions 캐시 서비스이고, 실패 step 이름이 '' 인 것은 이름 없는 step(checkout/setup-node/`npm ci`) 중 하나라는 뜻이며, 이 잡에서 Azure Blob 을 건드리는 경로는 캐시뿐이다(lockfile resolved 278개 전부 registry.npmjs.org). 키가 lockfile 해시라 같은 죽은 블롭을 매 실행 재요청 → 하루째 같은 실패. 게이트는 하나도 끄지 않았다(테스트가 그걸 단정한다).
- 확신 없는 곳: **실제 GitHub 실행이 녹색이 되는지 확인 못 했다**(푸시 권한·`gh` 인증 없음, 로그 원문 미열람). 캐시가 원인이 아니었다면 이 변경은 무해하지만 CI 는 계속 깨진 채다 — 그 경우 다음 후보는 '잡 시작 전'(postgres:16-alpine 서비스 컨테이너 초기화)이다. 또 `setup-node` 가 캐시 복원 실패를 치명적으로 처리하는지 액션 내부 코드로 확인하지 못했다(가설은 "같은 죽은 키를 매번 재요청" 쪽에 기대고 있다).
- 일부러 안 한 것: `npm ci`·check·build·migrate·e2e·playwright·screens step 과 타임아웃·서비스 컨테이너 설정은 손대지 않았다(원인과 무관하고 게이트 약화 위험). 앱 코드 0 파일.
- 다음 역할 주의: 새 테스트 `packages/core/src/__tests__/ci-gate.test.ts` 는 DB·네트워크 없이 돌지만 **`.github/workflows/check.yml` 의 문자열 모양에 의존**한다(step 은 6칸 들여쓰기 `- `, `continue-on-error` 는 8칸). 워크플로를 고칠 때 이 테스트가 같이 깨질 수 있고, 그건 "게이트를 soft 로 바꾸지 말라"는 가드가 문 것일 수도 있으니 무력화하지 말고 이유를 먼저 보라. 변경 후 e2e·screenshots 는 재실행하지 않았다(앱 코드 무변경).
- [러너 03:26] verify passed — 검증 6개 통과 (auto)
- [러너 03:26] pr created — https://github.com/hkjang/insight.yeopjari.bid/pull/7
- [러너 03:26] guard held — .github/workflows/check.yml 
