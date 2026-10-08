# 회차 노트 2026-10-08-143823-kanpic-improve — kanpic
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:38] base pinned — main@6e71a7a
- [러너 14:38] autonomy release — 

## 정찰 노트
- 초안(step 부호)을 먼저 저장한 뒤, 정상 일정 Next가 Santiago 자정 DST에서 같은 날짜를 반복하는 실제 결함을 확인해 선택을 변경했다(가치4/위험2/M, 프로덕션1파일).
- 현재 Go 전체 테스트 통과; overlay 관찰은 날짜 커서 비전진과 Next 200ms 미반환을 확인. HTTP/DB 정지와 수정 후 기대 출력은 아직 실행하지 않았다.
- UTC 달력 커서와 현지 후보 시각을 분리하되 기존 DST 건너뛰기·8년 경계·OR 의미는 보존. Apia 생략일 기대값은 구현자가 실제 테스트로 확인할 것.
- 12개 기존 후보 유지/재평가+새2개 기록; 별칭 원문 보존과 요일 step 상한 축소는 기각. 스킬3개는 로컬 원문으로 적용했으며 저장소 코드·커밋은 변경하지 않았다.
- [러너 14:44] scout done — 자정 DST 전환에서 자동화 일정 Next의 날짜 순회가 멈추지 않게 한다 (가치 4 / 위험 2 / 작업량 M)

## 구현 노트
- f2c1ad1: Next의 날짜 커서·8년 경계를 UTC 달력으로 분리하여 현지 자정 정규화에 따른 무한 순회를 제거했다(프로덕션 1파일+테스트 1파일).
- 재현: Santiago 09:00/00:30과 Apia는 수정 전 외부 go test -timeout=10s로 실패; 원본 프로덕션 overlay에서도 Santiago 재실패(test-red.log·red-missing.log·red-apia.log·test-revert.log).
- 실제 ParseSchedule→Next로 수용 기준 전부, 명시한 Apia 12/30의 건너뛰기, 월말·가을 DST 1회·8년 윤일·없는 날짜 ErrInvalid·UTC 반환을 검증했다.
- 지정 표적/패키지/전체 Go 테스트·vet·build·gofmt·release-docs·commit-identities 모두 exit 0; 전체 테스트 일부는 캐시이며 로그는 이 run 디렉터리에 보존했다.
- 확신 없는 곳·미검증: 운영 DB/HTTP 저장·스케줄러 통합, integration 태그, 웹/E2E는 미실행; 서비스의 직접 ParseSchedule→Next 호출 배선만 읽었다. 모든 역사적 시간대 전이를 전수 검증한 것은 아니다.
- 일부러 하지 않음: cron 문법/step/별칭/matchesDay/DB/서비스/HTTP/문서/PDF/릴리즈; 이번 결함은 날짜 순회에 한정된다. ideas.json 기존 14항목을 보존하고 구현·대조 테스트 후보 2개만 done 처리했다.
- 다음 역할: 회귀 테스트는 DB 없이 실제 시간대 자료로 실행하며 hang 검증에는 외부 -timeout을 유지할 것; 지정 스킬 3개는 전용 도구 부재로 로컬 SKILL.md 원문을 읽어 적용했다.
- [러너 14:48] brief accepted — 채택 — 현재 코드에서 Santiago 자정 커서 비전진과 실제 Next timeout이 재현됐고, 지정한 2파일만 수정하여 모든 반환 시각 �
- [러너 14:49] verify passed — 검증 7개 통과 (auto)
- [러너 14:49] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 14:49] pr created — https://github.com/hkjang/kanpic/pull/44
