# 회차 노트 2026-09-20-110355-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:03] base pinned — main@96f7414
- [러너 11:03] autonomy release — 

## 정찰 노트
- 선택 이유: 보류 목록 중 유일하게 main 에 코드가 있고(메일 계열은 브랜치 미머지), 환경(Keycloak·ptium)도 필요 없고, 보호 경로(auth·workflows)를 안 건드리는 S 과제. 어제 과제(릴리즈 워크플로)는 gh 가 샌드박스에서 막혀 no-change 로 끝났으므로 같은 유형을 피했다.
- 확신 없는 곳: 관리자 가이드 파일명·§5.7 위치는 grep 으로만 추정(미확인). 429 상태 코드 선택은 기존 로그인 제한 관례를 따른 것이며 저장소에 handoff 용 선례는 없다.
- 구현자 주의: 전체 Go 스위트 금지(-run 'TestHandoff' 만), DSN 없는 skip 을 통과로 쓰지 말 것, 프런트 미변경이므로 dist 재복사 불필요, 검증 직후 먼저 커밋.
- 프로필은 1일 전 것이 현재 코드와 일치해 다시 쓰지 않았다.
- [러너 11:06] scout done — 넘기기 표(handoff claim) 발급에 사용자별 미사용 표 개수 상한(20개) 추가 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 바꾼 것: `internal/app/handoff.go` 발급 트랜잭션에서 만료 정리 DELETE 직후 사용자별 live 행 COUNT → 20 이상이면 429·행/감사 없음(상수 `handoffClaimsPerUser`, 잠금 없는 근사 상한). 커밋 b9b0243(코드·openapi·md·validation) + 4f1ba72(가이드 html/pdf 재생성).
- 검증한 것: 임시 PostgreSQL 17 로 `go test -race -run TestHandoff ./internal/app` 4개 통과(skip 없음). 새 테스트는 구현 전에 먼저 실패(21번째 201)를 확인함. gofmt·vet·build·check-docs·git diff --check 통과. 전체 Go 스위트·web 테스트는 돌리지 않음(Go 변경은 handoff.go 한 함수, 프런트 무변경).
- 확신 없는 곳: 429 메시지가 프런트 실행 상세의 보내기 메뉴에서 어떻게 표시되는지는 확인하지 않음(서버 error 문자열이 그대로 알림으로 보일 것으로 추정, web/src 미변경). 동시 발급 경합은 과제서대로 정확히 20 을 보장하지 않음(주석에 명시).
- 일부러 하지 않은 것: FOR UPDATE/advisory lock, 새 인덱스, 설정 키, redeem·targets·프런트 변경, dist 재복사(프런트 무변경).
- 다음 역할 주의: `TestHandoffClaimsPerUserCap` 은 `HUNTER_TEST_DSN` 이 있어야 돌고 없으면 skip 됨. `user-guide.pdf` 도 render-guides 가 함께 재생성해 바뀌었음(이전 docs 커밋과 같은 관례).
- [러너 11:12] brief accepted — 채택 — 과제서의 근거(트랜잭션에 COUNT 없음, 파일·행 위치, 헬퍼 이름)가 현재 코드와 정확히 맞아 그대로 구현했다. 가
- [러너 11:12] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 판정 approve, risk low, blocking 없음. main 대비 두 커밋의 상한·롤백·권한·수령/만료·실제 API/DB 단언과 프런트 429 전달 경로 확인; 머지 차단 결함 없음.
- 직접 검증: TestHandoff 비DB 2개 통과·DB 2개 DSN 없어 skip; 원본312파일·check-docs·diff --check 통과(Node 22.23.1).
- 남는 우려: 관리자 가이드의 20개 설명에는 동시 발급 시 초과 가능한 근사 상한임을 보충 권장(과제에서 허용된 동작).
- 미확인: DB 통합 재현·실제 브라우저 오류 표시·PDF 육안·전체 회귀·배포; 코드 변경 없음. 상세 판정은 review.json.
- [러너 11:13] review approved — 리뷰 승인 (risk=low)
- [러너 11:13] pr created — https://github.com/hkjang/hunter/pull/7
