# 회차 노트 2026-10-09-052836-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:28] base pinned — main@ef9081b
- [러너 05:28] autonomy release — 

## 정찰 노트
- GPU 공급자 누락을 선택: 사용자에게 행이 사라지는 결함이며 프로덕션 1파일로 한정 가능. 알림 연속 변경·inventory/dedup 반복 차선 유형을 피했다.
- podRequestGPU의 비용 경로 공유를 확인해 초안을 수정했다. nodePackingAndGPU 안에서만 세 키를 합산하고 기존 파서 통합은 금지한다.
- AMD·Intel HTTP 신규 실패 및 브라우저 렌더는 미확인; 구현 첫 단계에서 실제 SQLite+Server.Routes로 재현해야 한다. 외부 GPU 표준 변경이 아니라 기존 노드 모니터링 키 계약에 맞추는 과제다.
- 보류 10개는 여전히 유효하여 유지했고 메트릭 오류 폐기·비대상 Kind의 용량 조회 창 점유 2개를 추가했다. 과제서 초안→확정본 저장, 코드·커밋 변경 없음.
- [러너 05:33] scout done — 용량 API에서 AMD·Intel GPU 노드와 요청 수량 누락 수정 (가치 3 / 위험 2 / 작업량 M)

## 구현 노트
- 08cfc7f: 용량 함수 안에서 NVIDIA·AMD·Intel allocatable/requests 합산. 총 4파일, 프로덕션은 capacity.go 1파일이며 작업 트리 깨끗함.
- 실제 SQLite+InventoryFromObject+Server.Routes HTTP에서 수정 전 AMD/Intel 빈 행 실패를 확인; 수정 후 AMD 4/1/3·Intel 2/1/1, 함대/필터·격리·정렬·CPU·미배치/미등록 노드 제외를 검증했다.
- 단위 15케이스로 문자열/숫자·혼합 공급자/컨테이너·init 합산·빈 GPU·미지원 키·limits 제외·음수 Idle 검증; NVIDIA-only 목록으로 일시 복원하면 같은 회귀 테스트가 재실패했다.
- 검증: 지정 좁은 테스트, gofmt 무출력, go build ./..., 서버 단독 빌드, go vet ./..., go test ./... -count=1 모두 통과(proxy 75.762s, store 19.815s). 서버 빌드 산출물 제거.
- 확신 없는 곳·검증 못 한 것: 실제 Kubernetes/GPU 장치·PostgreSQL·브라우저 렌더링은 실행하지 않음. 이번 종단 검증은 실제 SQLite 저장소에서 HTTP JSON까지다.
- 의도적 제외: 비용 podRequestGPU/단가, 모니터링 함수, quantity 파서, MIG/신규 키, limits fallback, init 스케줄러 의미, 조회 상한, 릴리즈 파일은 범위 밖이라 유지.
- 다음 역할 주의: capacity Idle은 음수 가능하고 monitoring Available은 0 clamp이므로 두 잔여값 동등 비교 금지. HTTP 테스트는 외부 DB 없이 임시 SQLite로 실행된다.
- [러너 05:38] brief accepted — 채택 — NVIDIA 전용 용량 집계와 세 공급자 노드 모니터링의 차이가 현재 코드 및 수정 전 실제 HTTP 실패로 확인되어 지정�
- [러너 05:40] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- approve / low / blocking 없음: 4파일 diff·커밋·원장 실패 출력·집계 호출 경로·문서 일치와 실제 HTTP 단언을 확인했다.
- 직접 검증: 관련 회귀, gofmt, diff --check, build, vet, 전체 go test 통과(proxy 66.216s, store 16.836s); 수정 전 재실행은 하지 않았다.
- 보안·법무: 기존 인가·클러스터 격리·SQL 바인딩 확인; 신규 개인정보 처리·외부 전송·권한 확대·의존성·마이그레이션 없음, 코드 revert 가능.
- 미검증: 실제 GPU/Kubernetes·PostgreSQL·브라우저. 릴리즈/후속 회차는 비용의 NVIDIA 전용 유지, init 합산·조회 상한 유지 및 Idle 음수 계약을 유의.
- [러너 05:42] review approved — 리뷰 승인 (risk=low)
- [러너 05:42] pr created — https://github.com/hkjang/clustara/pull/39
- [러너 05:43] ci passed — 검사 없음 — 정책으로 허용
- [러너 05:43] merge done — 08cfc7f
