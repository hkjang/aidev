# 회차 노트 2026-10-18-022851-apt.irumx.app-improve — apt.irumx.app
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:28] base pinned — main@b14cc46
- [러너 02:28] autonomy release — 
- [러너 02:28] scout skipped — 요청된 작업이 있어 정찰을 건너뛴다 (과제는 이미 정해져 있다)

## 구현 노트
- 무엇/왜: `.github/workflows/ci.yml` 의 npm 캐시 복원을 `actions/setup-node` 의 `cache: npm` 에서 떼어내 `actions/cache@v4` + `continue-on-error: true` 로 옮기고 키에 `github.run_id` 를 넣었다. setup-node 안의 복원 실패는 액션 자신의 setFailed 라 잡을 죽이고, 키가 lockfile 해시 하나라 블롭이 사라진 뒤 모든 실행이 같은 없는 블롭을 다시 요청한다 — 코드 변경 없이 1일째 같은 BlobNotFound 로 실패한 것이 이것으로 설명된다.
- 확신 없는 곳: **실제 CI 로그를 못 봤다**(gh 미인증, 실패 단계 이름도 '' 로 비어 있었다). 그래서 "setup-node 캐시 복원" 은 증명이 아니라 최선의 추론이다 — 근거는 (a) 이 워크플로의 Azure 블롭 접근이 setup-node 캐시와 upload-artifact 둘뿐이고 (b) upload-artifact 는 실행마다 새 블롭을 쓰므로 하루 내내 같은 모양으로 반복 실패하지 않는데 캐시 키는 lockfile 이 안 바뀌는 동안 고정이라 반복이 설명된다, (c) 수정 전 트리에서 CI 명령 전부가 로컬에서 통과했다. 틀렸다면 범인은 upload-artifact 이고, 그쪽은 손대지 않았다.
- 테스트는 ci.yml 을 읽어 구조를 검사한다(런타임 검증이 아니다). CI 설정은 설정 자체가 동작이라 이게 최선이었지만, "소스 문자열 검사" 범주에 가깝다는 점은 인정한다. 실제 YAML 파서(/home/hkjang/node_modules/yaml, 프로젝트 의존성 아님)로 13개 단계를 교차 확인해 손으로 쓴 파서가 같은 결과를 내는 것은 확인했다.
- 일부러 안 한 것: `actions/setup-node`·`checkout` 의 v5 승격(이게 BlobNotFound 를 고친다는 근거를 확인할 수 없음), `cache` 삭제(최적화를 버리는 것), docs/evidence·verification.md 기록(릴리즈 세션의 일).
- 다음 역할 주의: `tests/ci-workflow.ts` 는 DB 가 필요 없지만 **cwd 가 저장소 루트**여야 한다(`.github/workflows/ci.yml` 을 상대 경로로 읽음). ci.yml 을 prettier 2칸 형식에서 벗어나게 고치면 파서가 깨진다 — 그때는 테스트가 거짓 실패한다. `npm test` 는 매번 **새** 127.0.0.1:55462 컨테이너에서 돌려야 한다(재사용 시 누적 데이터로 깨지는 미해결 건 있음).
- [러너 02:53] verify passed — 검증 6개 통과 (auto)
- [러너 02:54] pr created — https://github.com/hkjang/apt.irumx.app/pull/2
- [러너 02:54] guard held — .github/workflows/ci.yml 
