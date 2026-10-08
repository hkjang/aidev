# 회차 노트 2026-10-08-190833-Vendra-improve — Vendra
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:08] base pinned — main@d8591b9
- [러너 19:08] autonomy release — 

## 정찰 노트
- 신규 업무 제출이 진행 중 자동 저장을 await한 뒤 currentTarget을 읽어 실패함을 실제 Objects+fetch 경계 테스트로 재현했다(1 failed/1 error, POST 없음). 안내 개선보다 핵심 저장 동작 복구가 우선이며 프로덕션 1파일이다.
- 먼저 저장된 보기 오류 과제 초안을 썼고, 위 증거 확보 후 최종 과제를 교체했다. 수정 후 통과·전체 웹/DB 테스트는 미확인; 구현자가 수행한다.
- 첫 await 전에 FormData 스냅샷을 확보하되 PUT 대기→업무 POST→초안 DELETE 순서를 지킬 것. React 이벤트/FormData 대역 금지, api export만 mock하는 함정은 fetch 경계 mock으로 피할 것.
- Skill 도구 미노출로 요청한 세 SKILL.md를 로컬에서 직접 읽었다. 진단 복사본·로그와 최종 brief/ideas/profile만 run 디렉터리에 기록했고 저장소 코드는 변경하지 않았다.
- [러너 19:17] scout done — 자동 저장 중 신규 업무를 제출하면 폼 이벤트가 비워져 저장되지 않는 문제 수정 (가치 4 / 위험 1 / 작업�

## 구현 노트
- fd56d79: NewObject.save의 FormData 생성을 첫 await 앞으로 옮겨 제출 당시 값을 보존했다. 프로덕션 1파일 + 신규 테스트 1파일만 커밋.
- 실제 Objects·React DOM·FormData·원본 API 배선에서 수정 전 POST 0회/FormData TypeError 2건을 재현했고, 수정 되돌림에서도 재현했다(assets/draft-submit-red.log, draft-submit-revert.log).
- 신규 4건은 제출 스냅샷·PUT/POST/DELETE/재조회 순서, 즉시 제출, PUT 500, POST 실패 재시도를 검증한다. 지정 3파일 15건 및 전체 25파일 113건 통과, tsc·eslint·build 통과.
- 검증 못 한 것: jsdom/fetch 경계 테스트이며 실제 브라우저·Go/DB 통합은 미실행. DELETE 실패 정책, 다중 PUT, 대기 중 중복 제출은 이번 테스트 범위 밖이다.
- 일부러 하지 않은 것: 자동 저장 구조·700ms·DELETE 실패 정책·busy 시점·목록 정렬·의존성 변경 및 릴리즈 작업. 중복 제출은 별도 후보 유지.
- 다음 역할: API export mock 대신 fetch 경계를 유지할 것. 기록과 로그는 지정 run 디렉터리에 있고, 기존 테스트 파일은 무변경이다.
- Skill 도구가 없어 요청한 completion-verification/systematic-debugging/test-driven-development의 로컬 SKILL.md를 읽고 실패→수정→되돌림→전체 검증 절차를 적용했다.
- [러너 19:22] brief accepted — 채택 — 실제 저장소에서 PUT 대기 후 FormData 오류와 POST 부재가 재현됐고, 제시한 첫 await 전 스냅샷 확보만으로 수용 기준
- [러너 19:23] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- approve / low / blocking=[]: main...HEAD 두 파일·커밋·미검증 노트·수정 전/되돌림 실패 로그와 실제 DOM/API 테스트 배선을 확인했다.
- 직접 검증: 전체 웹 25파일·113건, tsc --noEmit, eslint src, diff --check 통과; 저장소 소스는 변경하지 않았다.
- 미검증: 실제 브라우저·Go/DB·사용자 간 격리 실행 및 빌드 재실행. 기존 중복 제출·다중 PUT·DELETE 실패 정책은 다음 회차 검증 대상으로 남긴다.
- 세 요청 스킬은 Skill 도구 부재로 로컬 SKILL.md를 읽어 적용; 새 개인정보 흐름·권한 확대·비가역 변경 및 이번 diff의 차단 결함은 발견하지 못했다.
- [러너 19:25] review approved — 리뷰 승인 (risk=low)
- [러너 19:25] pr created — https://github.com/hkjang/Vendra/pull/144
