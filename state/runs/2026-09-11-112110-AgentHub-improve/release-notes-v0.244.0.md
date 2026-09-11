## 이번 릴리즈의 변경

- **수정** — 런타임으로 인계된 실행(ErrHandedOff)이 실행 기록에 `실패 · 이유 없음` 으로 남던 결함을 고쳤습니다. `Execute` 의 상태 분기를 `runStatus` 로 꺼내 승인 대기와 같은 자리에서 인계를 다루고, 일곱 경우 표 테스트를 추가했습니다. (43ac34a)
- **문서** — 사용자·관리자 가이드의 실행 기록·작업 대기열·런타임 화면을 실제 워커가 만든 항목이 있는 상태로 다시 찍었습니다(실행 상세 서랍 `runs-detail.png` 추가). 클러스터 없이도 워커와 모델 대역(`web/scripts/guide-model-stub.mjs`)으로 실행을 돌리며, `guide-shots.mjs` 의 DLP·sessionGateway 설정 복원이 실제로 동작하도록 고쳤습니다. PDF 둘을 다시 구웠습니다. (541bbbe)

런타임 이미지 입력은 바뀌지 않았습니다(`release-catalog-images.sh check-versions` 통과). 제어 이미지 `agenthub:v0.244.0` 만 새로 게시됩니다.
