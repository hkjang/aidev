# 회차 노트 2026-09-23-150431-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:04] base pinned — main@d74730d
- [러너 15:04] autonomy release — 

## 정찰 노트
- 골랐다: guide-shots 복원 루프의 개별 예외 처리. 지난 두 회차가 "쓰기 전 백업" 을 닫았으므로 남은 실패는 나가는 길 한 곳뿐이고, 기존 vm 하니스가 그대로 재사용돼 45분에 맞는다. 제친 후보 — 호출 제한시간(같은 파일·범위 겹침), CSP 위반 전체 삭제(폐기 배포 전제라 피해 작음), reporter.send non-2xx(BASE_VERSION 상향 부담), 릴리즈 워크플로/CI postgres(보호 경로라 기각 고정).
- 추측으로 적은 것: `page.evaluate` 안 fetch 의 reject 가 실제 운영에서 얼마나 자주 나는지는 미확인(코드상 가능하다는 것만 확인). 본문 오류 우선 전파 방식은 내 설계 제안이며 기존 테스트 78-92행 호환을 근거로 골랐다.
- 구현자 주의: 전부 note 로만 바꿔 아무것도 던지지 않으면 "복원 실패인데 성공" 이 되어 반려 사유가 된다. 본문 오류는 래핑 없이 원본 그대로 재전파할 것.
- 이번 환경에서 node/go 테스트는 실행 승인을 받지 못해 미실행 — 프로필의 통과 기록은 어제 것이며 이번 확인이 아니다.
- 프로필은 1일 전이고 VERSION 0.250.0 / BASE_VERSION 0.26.0 / HEAD d74730d 로 지금 코드와 맞아 다시 쓰지 않았다.
- [러너 15:07] scout done — guide-shots 복원 루프가 PUT 한 건의 예외로 중단되지 않고 네 설정을 모두 되돌린다 (가치 3 / 위험 1 / 작업량

## 구현 노트
- `withGuideSettings` 의 finally 복원 루프만 고쳤다(7ea3ee5): 각 PUT 을 개별 try/catch 로 감싸 실패를 note 로 남기고 계속 진행, 본문 오류는 래핑 없이 재전파, 본문 성공 시에만 `복원 실패: <경로> (<사유>)` 를 던진다. 백업 사전 검증 블록·복원 순서·`{value:…}` 래핑은 손대지 않았다.
- 확신 없는 곳: (1) 본문 성공 + 복원 실패일 때 예외가 guide-shots.mjs:113 의 problems 요약 블록을 건너뛴다 — 종료 코드는 0 이 아니지만 운영자는 목록 대신 스택을 본다. 과제서가 "던져라" 를 요구해 그대로 했고 보류 아이디어로 적었다. (2) 실제 브라우저 촬영·클러스터·DB live 는 이 환경에서 미실행이라 `page.evaluate` fetch 의 실제 reject 동작은 하니스 재현으로만 확인했다. (3) 요청된 세 technology 스킬은 이번엔 Skill 도구로 불렸고 절차를 따랐으나 반환 형식은 기존 원장 형식에 맞춰 적었다.
- 일부러 하지 않은 것: 차선 후보인 요청별 AbortSignal(같은 파일, 범위 겹침), package.json 스크립트 목록 수정(별도 보류), guide-shots 의 problems 요약 위치 변경(동작 변경이라 별도 과제).
- 다음 역할 주의: 새 테스트는 DB·브라우저 없이 `cd web && node --test scripts/guide-settings-check.test.mjs` 로 돈다(64건). 하니스는 실제 guide-shots.mjs 의 소스 블록을 vm 으로 실행하므로 그 파일의 `const call =` ~ `if (problems.length)` 구간 형태가 바뀌면 같이 깨진다. 수정 전 코드에서 8건이 실패하는 것을 먼저 확인했다.
- [러너 15:11] brief accepted — 채택 — finally 루프의 개별 catch 부재, 하니스가 non-GET 실패를 흉내낼 수 없던 점, 기존 테스트 78-92행의 원본 오류 기대까�
- [러너 15:12] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 확인함: main 헬퍼 + 새 테스트를 임시 디렉터리에 조합해 돌려 8건이 실제로 실패하고 HEAD 에서 64건 전부 통과하는 것을 봤다. 테스트는 변경을 진짜로 고정한다. 범위 이탈 없음(web/scripts 3파일, guide-shots 는 주석만), 소비자는 guide-shots.mjs 하나, revert 로 완전히 되돌아온다.
- 못 본 것: 실제 브라우저·클러스터·DB live. page.evaluate fetch 의 실운영 reject 빈도는 여전히 미확인이며 vm 재현만 근거다.
- 승인이어도 남는 우려 1 — 본문 성공 + 복원 실패 시 throw 가 guide-shots.mjs:114 요약을 건너뛴다. note() 가 각 FAIL 을 이미 즉시 출력하고 종료 코드도 0 이 아니라 손실은 마지막 목록뿐이지만, 릴리즈 노트에 적고 다음 회차 후보로 남긴다.
- 승인이어도 남는 우려 2 — 새 http 변형 4건은 수정 전 코드에서도 통과해 변별력이 없다(중복 커버리지). 또 이 테스트 파일은 여전히 CI(test:sso)에 없어 회귀가 조용히 지나갈 수 있다(기존 공백).
- 보안·법무 차단 없음: 인증·인가·식별자·비밀값 미접촉, 오류 메시지에 설정 본문 미포함, 새 개인정보 수집 없음(시드는 가공 데이터 + DISPOSABLE 게이트).
- [러너 15:15] review approved — 리뷰 승인 (risk=low)
- [러너 15:15] pr created — https://github.com/hkjang/AgentHub/pull/34
- [러너 15:17] ci passed — 검사 1개 모두 success
- [러너 15:17] merge done — 7ea3ee5
- [러너 15:23] release published — v0.251.0
- [러너 15:29] assets verified — v0.251.0 자산 4개 (이전 v0.250.0: 8)
