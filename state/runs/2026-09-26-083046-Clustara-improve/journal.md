# 회차 노트 2026-09-26-083046-Clustara-improve — Clustara
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:30] base pinned — main@adc3294
- [러너 08:30] autonomy release — 

## 정찰 노트
- 골랐다: 알림 설정 POST 의 쓰기 검증이 읽기 파서보다 느슨한 결함(quiet_hours 범위·team_channels 객체). 최근 4회차가 모두 "클러스터별 격리" 계열이었고 조인 결함은 소진됐으므로 같은 파일에서 남은 유형인 "쓰기/읽기 계약 불일치" 로 옮겼다. seccompProfile·classifyPodSecurity·취약점 import 는 포스처 점수·정책 게이트 파급 산정이 먼저라 S 로 끝나지 않고, .github CI 는 보호 경로라 제쳤다.
- 확인한 것: k8s_notify.go 의 inQuietHours(27-45)·handleK8sNotifyConfig(163-206)·notify 클로저(108-160), admin_ui.go 의 입력(14611)·저장(14828-14836 — api() 가 비2xx 에서 throw 하므로 400 이 화면에 보인다), `k8s_quiet_hours` 는 이 파일 3곳에서만 쓰임, k8s_notify_scan_test.go 의 실제 배선과 postJSON(bearer "") 통과.
- 추측/미확인: 스케줄러가 notify/scan 을 주기 실행하는 경로는 읽지 않았다. `22-22` 를 400 으로 거절할지는 구현자 판단으로 남겼다. team_channels 플래그를 notify 밖 다른 Mattermost 경로가 읽는지 미확인 — 그래서 쓰기만 엄격하게 하라고 적었다.
- 조심할 것: 읽기(inQuietHours)를 엄격하게 바꾸면 이미 저장된 잘못된 값이 갑자기 창을 만들어 운영 알림이 사라진다. 고치기 전 코드에서 새 테스트 3개가 각각 200/억제없음/200 으로 실패하는 것을 먼저 볼 것.
- 신규 아이디어로 quiet_hours 의 타임존 부재(now.Hour() 가 컨테이너 TZ 의존)를 ideas.json 에 남겼다 — M 이라 이번엔 제외.
- [러너 08:34] scout done — 알림 설정 저장 시 `quiet_hours`·`team_channels` 를 읽는 쪽과 같은 계약으로 검증하기 (가치 3 / 위험 1 / 작업량 
