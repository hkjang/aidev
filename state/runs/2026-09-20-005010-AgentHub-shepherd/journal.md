# PR 처리기 노트 2026-09-20-005010-AgentHub-shepherd — AgentHub PR #29
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## 수리 노트
- 세 지적 모두 맞았다. 모르는 등급의 Class 가 Class·Label 로 그대로 실렸고(재현함), 64룬 컷이 83룬 sk- 샘플에서 `Reported(s) != s` 였고(수정 전 새 테스트 실패로 재현), 문서는 서버·툴·방향이 Pod 의 말인데도 "무엇을 보내든" 이라 적었다.
- 고친 방법: 모르는 등급은 `dlp.UnknownClass`("unknown") + 고정 라벨 + 빈 샘플, 모르는 조치는 `audit`; 룬 컷 제거(`mask` 는 어떤 길이에서도 자기 출력에 no-op, 본문 1 MiB 한도가 길이를 묶음); 문서를 "어느 칸이 누구의 말인지" 로 고쳐 쓰고 같은 md2pdf 로 PDF 재생성. 커밋 bd3a982.
- 확신 없는 곳: (1) 모르는 `Action` 을 `audit` 로 바꾸는 것은 지적 밖의 한 줄 — 같은 JSON 에 Pod 문자열이 남는 걸 막으려 넣었다. (2) 컷을 없앴으니 Pod 가 1 MiB 별표 샘플을 보내면 그대로 저장된다 — main 에서도 이미 그랬으니 퇴행은 아니다. (3) live 테스트는 Postgres 없이 못 돌렸다.

## 심사 노트
- 확인한 것: bf1e062 의 `Reported` 에 스크래치 테스트를 붙여 83룬 sk- 컷과 모르는 등급의 Class·Label 에코를 둘 다 재현했고, HEAD 에서는 새 테스트 두 개가 정확히 그 두 동작을 잡는다. 모든 검출기에 대해 `mask` 가 자기 출력에 no-op 임을 최소 길이 기준으로 따져 봤다.
- 실행한 것: 임시 postgres:16 컨테이너로 `TestAGatewaysFindingReachesTheTrail` 을 프로덕션 라우터·실제 store 로 돌려 통과(원문 RRN 을 실은 보고가 audit_events 에 `900101********` 로만 남음). `go vet`, `go test ./...` 전부 통과. 컨테이너는 지웠다.
- 못 본 것: 콘솔 UI 가 감사 details 의 findings 를 등급별로 그리는 곳은 없어(JSON 그대로) 확인할 렌더러가 없었다. 서버·툴·방향 문자열의 길이 한도는 이번 PR 범위 밖으로 두었다(문서가 이제 정직하게 적음).
- 판단: 결함 없음, 범위 이탈 없음(BASE_VERSION 은 Dockerfile.base 가 internal/ 을 복사하므로 관례). 감사 데이터를 만지므로 risk=medium, approve/merge.
