# 회차 노트 2026-09-23-143408-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:34] base pinned — main@37006f3
- [러너 14:34] autonomy release — 

## 정찰 노트
- 우선 과제(릴리즈)는 10회째 같은 교착이라 재조사하지 않고 6개 근거만 재확인해 pending 유지로 적었다(구현자에게 재조사 금지를 명시). 대신 기본 실행 대상으로 SupportOcr.vue:206 의 미-await `ocrParse` 를 골랐다 — 저장소 19개 `inf.tools.*.call` 중 유일하게 await 이 없고, 형제 SupportStt.vue:131-155 와 docs/03:415 라는 출처 두 개가 정답 형태를 이미 정해 주므로 추측이 적다.
- Alert `showHeader`(차선)를 제친 이유: 기대 계약이 미확정이라 잘못 고치면 화면에 Error 객체가 찍히는 회귀가 난다. 참조 카운트는 선행 누수가 남아 있어 제외.
- 추측으로 적은 것: `/tools/parse` 의 응답 지연 특성(즉시 등록인지 동기 변환인지) 미확인 — await 로 바꾸면 스피너 지속 시간이 그만큼 늘어난다. 백엔드가 이 저장소에 없어 확인 불가이니 구현 노트에 남길 것.
- 구현자 주의: STT 를 그대로 복사해 `finally { stopLoading() }` 만 쓰면 성공 경로에서 스피너가 2초 일찍 꺼져 새 회귀가 난다. 성공 경로의 `setTimeout` 안 stopLoading 은 유지할 것.
- node_modules 가 비어 있어 `npm ci` 가 선행이다. 이번 정찰은 테스트/빌드를 실행하지 않았고 저장소를 변경하지 않았다(`git status --short` 0줄).
- [러너 14:39] scout done — [수정 과제] 릴리즈 버전 결정 입력 복구 — 승인 근거 확보 전까지 pending 유지 (가치 5 / 위험 2 / 작업량 S)

## 구현 노트
- 무엇/왜: `SupportOcr.vue startConversion` 의 `ocrParse` 를 `await` 하도록 고치고(`isSuccess` 실패 분기 + `catch` 의 `stopLoading()` 추가), 성공 경로의 `setTimeout` 안 `stopLoading()` 은 유지했다. 실패가 `catch` 에 닿지 못해 알림 없이 목록으로 이동하고 스피너가 60초 남던 문제다. 커밋 e0aac38.
- 확신 없는 곳: (1) `/tools/parse` 의 응답 지연 특성이 **미확인**이다 — 백엔드가 이 저장소에 없다. 즉시 등록이 아니라 동기 변환이면 `await` 때문에 스피너가 그만큼 길어진다. 근거는 형제 `sttParse` 가 같은 구조로 await 한다는 것과 docs/03:415 뿐이다. (2) 성공 응답이 `isSuccess` 계약(`code 200`/`success:true`)을 실제로 만족하는지 응답 샘플로 확인하지 못했다 — 만족하지 않으면 성공이 실패 알림으로 오인된다. 여기가 이번 변경에서 가장 위험한 지점이다.
- 일부러 안 한 것: `globalLoading` 참조 카운트 도입(60초 자동 해제가 카운터를 안 되돌려 더 나쁜 회귀), STT 처럼 `finally { stopLoading() }` 로 통일(성공 경로 스피너가 2초 일찍 꺼지는 새 회귀), 릴리즈 관련 일체(진입 조건 미충족).
- 다음 역할 주의: 새 스펙은 `vi.useFakeTimers({ toFake: ['setTimeout','clearTimeout'] })` 로 타이머를 **부분만** 가짜로 만든다(전부 가짜로 하면 `flushPromises` 가 멈춘다). `globalLoading` 은 모듈 싱글턴이라 각 케이스가 `stopLoading()` 으로 정리한다. `npm ci` 선행 필요, 빌드 후 `dist/` 삭제.
- [러너 14:43] brief accepted — 채택 — A(릴리즈)는 지시대로 무변경으로 두고, B(SupportOcr ocrParse 미-await)를 수용 기준 1~5 그대로 구현·검증했다.
- [러너 14:43] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 판정 approve / risk low / blocking 없음. diff·log, OCR/STT·응답 계약·인터셉터·폴링·로딩·새 테스트를 확인했고 실제 차단 결함은 찾지 못했다.
- npm test: 24파일 458테스트 통과. 새 테스트의 실패 단언은 기존 코드에서 성립하지 않음을 코드 대조했다(기존 코드 실행은 안 함).
- 실제 /tools/parse 성공 응답·처리 시간과 백엔드 통합·빌드는 미검증. 지연 응답 및 실패 후 2초 경과·폴링 부재 테스트는 보강 여지가 있다.
- 기존 실패 경로도 2초 타이머에서 로딩을 해제하므로 구현 노트의 60초 잔류 설명은 부정확하다. 릴리즈 판단·소스 변경은 하지 않았다.
- [러너 14:44] review approved — 리뷰 승인 (risk=low)
- [러너 14:44] pr created — https://github.com/hkjang/aiportal-front/pull/24
- [러너 14:45] ci passed — 검사 없음 — 정책으로 허용
- [러너 14:45] merge done — e0aac38
- [러너 14:46] release failed — 릴리즈 안 함: 다음 버전을 결정할 입력이 없어 릴리즈를 수행하지 못했다(11회째 동일 교착). 절차 1의 실제 재조사 결과: (1) `git tag` 0개 — 태그 형식·�
