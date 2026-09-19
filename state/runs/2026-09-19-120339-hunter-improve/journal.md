# 회차 노트 2026-09-19-120339-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:03] base pinned — main@96f7414
- [러너 12:03] autonomy release — 

## 정찰 노트
- 자동 배정된 수정 과제(릴리즈 실패)를 그대로 골랐다. 근거: 2026-09-18-090342 `release.json` 이 `github_release:false, assets:[]`, 다음 회차 2026-09-18-164351 은 예산 hold($22 초과)로 미착수.
- 확신 없는 곳: 정찰 샌드박스에서 `gh`·curl·python3 실행이 모두 차단되어 release.yml 의 **어느 단계가 왜 실패했는지 미확인**. 과제서의 원인 후보(디스크·digest·release-notes·gh release create)는 워크플로 파일 읽기에서 나온 추측이다. 구현자는 `gh run list --workflow release.yml` 부터 시작할 것.
- 조심할 것: 워크플로 완화 금지, 자산은 tar.gz 하나, 태그 v1.12.0 재생성은 운영자 승인 대상, 전체 Go 스위트(650초+)는 이 과제에 불필요 — 예산 hold 재발 방지.
- 워크플로 파일·release.sh·Dockerfile 을 읽었고 코드는 바꾸지 않았다. profile.md 를 새로 썼다(이전 프로필 없음).
- [러너 12:07] scout done — v1.12.0 릴리즈 워크플로(`Release offline image`) 실패 원인 확인·수정 — 태그 v1.12.0 의 첨부 자산이 비어 있는 �
- [러너 12:14] brief unstated — 구현자가 과제서 판정을 적지 않음
- [러너 12:14] improve no-change — 커밋 없음
