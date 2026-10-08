# PR 처리기 노트 2026-10-08-165701-pii-masker-shepherd — pii-masker PR #36
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-08-152851-pii-masker-improve)
# 회차 노트 2026-10-08-152851-pii-masker-improve — pii-masker
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:29] base pinned — main@19b4426
- [러너 15:29] autonomy release — 

## 정찰 노트
- 이미지 page 3이 실제 cmd 서버에서 동기 200/completed·비동기 completed/download_url로 재현돼 선택. 386/테스트 헬퍼 후보보다 사용자 결과에 직접 영향이 있고 프로덕션 1파일로 제한 가능하다.
- brief.md를 초안 저장 후 갱신했고 9개 기존 아이디어를 유지·재평가하고 신규 2개를 추가했다. 전체 go test·vet·build 통과, 386 document 테스트 기존 실패 재확인; 저장소 변경 없음.
- 주의: HTTP page 생략/0은 수집기가 1로 보정한다. 직접 함수의 0 호환성도 유지하고 파서·PDF·오류 매핑을 함께 고치지 않는다. 정상 안쪽 좌표와 실제 app.New 배선으로 회귀를 증명할 것.
- 미확인: 새 guard 적용 후 결과·race, 차선의 8000×8000 픽스처, 신규 정규화 좌표/소수 페이지 후보의 최종 출력. Skill 도구 부재로 세 로컬 SKILL.md를 읽어 대안·계획·산정을 과제서에 반영했다.
- [러너 15:38] scout done — 이미지에 존재하지 않는 페이지의 마스킹 영역을 성공으로 처리하지 않기 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- 88e8ace: MaskImageFile 루프에서 음수/1 초과 페이지를 기존 placementError로 거절. 프로덕션 1파일(4줄), 테스트 2파일, README 1파일 변경.
- 직접 2/3/-1·혼합 영역 거절 및 nil 바이트, 0/1 전체 PNG 픽셀, 실제 app.New HTTP PNG/PDF 실패·download_url 없음·결과 404, HTTP 생략/0/1 픽셀 검증.
- 구현 전 실패 → guard 후 통과 → guard만 제거해 동일 실패 → 복원 후 전체 검증 통과. red/reverted/compatibility-before 로그는 assets/에 저장. 초기 색상 비교의 Gray16 타입 불일치는 테스트에서 수정했으며 재현 근거에서 제외.
- 검증: 대상/전체 go test -count=1(전체 9패키지 ok), go vet ./..., go build ./..., 대상 race(1.084s/1.990s), gofmt -l ./cmd ./internal 및 git diff --check 모두 통과.
- 확신 없는 곳·검증 못 한 것: Docker/Windows 실환경·실제 외부 Upstage는 미검증. GOARCH=386 기존 실패는 정찰 기록만 유지하고 이번 재실행/수정 안 함.
- 일부러 하지 않은 것: 파서/PDF/service/server·공통 multipart 헬퍼·인코딩·릴리즈 변경 없음. HTTP 음수는 기존 수집기 보정 계약이므로 거절 요구를 추가하지 않음.
- 다음 역할 주의: HTTP 테스트는 config 직접 조립 후 app.New와 실제 리스너를 통과하며 config.Load 자체 검증은 아님. Skill 도구 부재로 요청한 세 technology SKILL.md 로컬 원문을 읽어 적용. 원장·아이디어 파일 갱신 완료.
- [러너 15:42] brief accepted — 채택 — 현재 코드에 페이지 검사 누락이 그대로 있었고 지정된 정상 좌표·프로덕션 배선 테스트에서 잘못된 성공을 재�
- [러너 15:42] verify passed — 검증 3개 통과 (policy)
- [러너 15:43] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 15:43] pr created — https://github.com/hkjang/pii-masker/pull/36
