# 회차 노트 2026-10-08-133829-insight.yeopjari.bid-improve — insight.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:38] base pinned — main@cbc8cb5
- [러너 13:38] autonomy release — 

## 정찰 노트
- 실제 parseLinks 실행에서 501자 URL이 500자로 잘리는 것을 확인해 선택했다. 운영 호출처 없는 시간 유틸 테스트·204 캐시 후보보다 직접 영향이 크고 프로덕션 1개 파일로 끝난다.
- brief.md 초안을 먼저 쓴 뒤 실제 호출처 확인으로 최종 후보를 바꿨다. 기존 6개 아이디어를 유지·재평가하고 신규 3개를 추가했다.
- 실서버 저장·조회는 미확인. npm run test는 node_modules 부재로 vitest: not found; 단위 함수 재현만 실행했으며 전체 통과를 주장하지 않는다.
- 구현자는 parseLinks만 좁혀 고치고 공용 text/다른 파서·auth·migrations는 건드리지 말 것. e2e로 400과 저장 상태 보존을 증명하고 npm ci로 lockfile을 보호한다.
- [러너 13:43] scout done — 과제 링크가 500자를 넘으면 잘라 저장하지 말고 입력 오류로 거절하기 (가치 4 / 위험 2 / 작업량 M)

## 구현 노트
- parseLinks만 수정: 정규화 후 500자 초과 URL을 자르지 않고 원래 인덱스의 validation_failed로 거절. 변경 3개 파일(프로덕션 1개), 원문 로그/오류 추가 없음.
- TDD: 지정 rules 테스트 3 실패→32 통과, 수정 되돌림에도 같은 3 실패 확인. 최종 npm run check: lint/typecheck + 63개 통과, npm run build 성공.
- 실 Postgres 16(:55482)·빌드 서버(:18789)·기존 OAuth(:18999)에서 e2e 176 passed, 0 failed. 500자 PUT→GET 보존, 501자 400/원래 인덱스, 제목·링크·연구 전체 상태 불변 검증.
- 첫 e2e 1 실패는 JSONB 키 순서에 의존한 문자열 비교였음; 실 API probe로 길이·원문·라벨 일치를 확인하고 isDeepStrictEqual로 수정한 뒤 전체 재실행 통과.
- 확신 없는 곳·검증 못 한 것: 브라우저 스크린샷/운영 Cloudflare 실행은 미실행(서버 입력 검증 과제 범위 밖). 공용 text/isWebLink/judgeSubmission 및 기존 저장 데이터 보정은 계약·범위 보존을 위해 변경하지 않음.
- 다음 역할: e2e에는 폐기 가능한 실 DB·마이그레이션·최신 빌드 서버가 필요. .env 파일 생성 없이 예제 설정을 메모리로 읽었고 테스트 서버/컨테이너와 e2e 임시 credential fixture는 정리함. npm ci 사용, lockfile 무변경, git diff --check 통과.
- 요청한 세 technology 스킬은 Skill 도구 부재로 로컬 SKILL.md를 읽어 적용. 세부 명령·실패 출력은 ledger-entry.md, 기존 아이디어 9개는 유지하고 선택 과제만 done 처리.
- [러너 13:49] brief accepted — 채택 — 기준 코드에서도 절단 결함이 재현되었고 지정 3개 파일과 수용 기준 그대로 구현했다.
- [러너 13:49] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음. 지정 세 스킬은 Skill 도구 부재로 로컬 SKILL.md를 읽어 적용; diff/log·과제서·원장·구현 미검증 항목을 확인했다.
- npm run check 63개 통과; main/HEAD 공개 함수 독립 실행으로 501·700자 절단→400 거절과 499·500자 보존 확인. 저장 전 검증·권한·원문 비노출·범위·revert 가능성을 검토했다.
- 기록된 실제 e2e 176/0 및 기업 격리 통과 출력을 확인했으나 이번 리뷰에서 실 DB e2e·브라우저·운영 Cloudflare는 재실행하지 않았다.
- 후속 참고: 기존 LinksEditor는 task_links[i] 오류를 표시하지 않아 화면은 일반 입력 오류만 안내한다. API 검증 과제 범위 밖의 UI 개선이며, 기존 절단 데이터 복구도 별도다.
- [러너 13:51] review approved — 리뷰 승인 (risk=low)
- [러너 13:52] pr created — https://github.com/hkjang/insight.yeopjari.bid/pull/2
