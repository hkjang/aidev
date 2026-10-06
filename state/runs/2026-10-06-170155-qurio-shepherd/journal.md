# PR 처리기 노트 2026-10-06-170155-qurio-shepherd — qurio PR #35
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-06-155834-qurio-improve)
# 회차 노트 2026-10-06-155834-qurio-improve — qurio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:58] base pinned — main@afd3882
- [러너 15:58] autonomy release — 

## 정찰 노트
- 1순위는 `appendUnique` quadratic 제거 **재시도**다. 이번 base `afd3882` 에서 `git diff bd3f310 afd3882 -- internal/domain/sqlsafe/` 가 무출력이고 `grep -n reasonSet` 이 0건 — 즉 2026-10-06 회차가 설계·증명까지 끝낸 그 수정이 **main 에 없다**(러너 판정 verify-failed). 설계와 실측치(1MiB = pg 10.9s/oracle 11.0s)가 그대로 재사용되고 프로덕션 파일 1개·판정 불변이라, 남은 후보 중 가치/위험/확신이 가장 좋다.
- 제친 후보: Oracle `WITH (col,…) AS` 면제(3bf8799 재시도)는 같은 파일이고 **판정을 바꾸는**(통과시키는) 방향이라 위험 3 — 차선으로 뒀다. 웹 스위트 전수 감사는 가치 4지만 M·개방형이고 게이트가 지금 녹색이라 급하지 않다. 사유 개수 상한은 1순위와 같은 함수를 건드리고 판정 불변 증명을 깬다.
- 추측으로 적은 것(과제서에도 표시): ① verify-failed 의 원인을 CI 로그로 확인하지 못했다 — 같은 날 `ab53169` 가 고친 web flake 2건이라는 것은 정황 추정이다. ② 성능 수치는 전부 직전 회차 실측을 인용했고 이번 회차에 재측정하지 않았다(코드가 바이트 동일하므로 유효하다고 판단). ③ `cd web && npm test` 녹색도 재실행하지 않았다.
- 구현자가 조심할 것: `list()` 를 맵 순회로 만들면 판정 불변이 즉시 깨진다. 병합 루프는 `result.Reasons` 로 시드해야 순서가 유지된다. 스케일 테스트는 16000개가 아니라 **64000개**로 재야 base 가 red 가 된다. 임시 덤프·퍼즈·base 사본을 지우고 `git status` 로 변경 2파일만 남기라.
- 이 저장소에서 반복되는 실패는 과제 선정이 아니라 **머지 실패**다(a8f0775·12223b0·3bf8799·reasonSet 네 건). 프로필의 "미머지 커밋" 절을 그에 맞게 갱신했다.
- [러너 16:05] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 커밋 feb607a — `internal/domain/sqlsafe/sqlsafe.go` 1파일(+94/-15) + 테스트 1파일. 위반 사유 문자열이 질의에서 뽑은 식별자를 품어 서로 다른 사유 개수가 질의 길이로만 묶이는데, `appendUnique` 가 새 사유를 이미 모은 사유 전부와 비교했고 `AnalyzeDialect` 병합 루프가 같은 quadratic 을 또 치렀다. 순서 보존 슬라이스 + lazy 멤버십 맵(`reasonSet`, 임계값 8)으로 교체. **내 실측**: 요청 한도 1,048,566B 에서 pg 13.55s→125ms, oracle 13.16s→105ms; 사유 개수·바이트는 전후 완전 동일.
- **확신 없는 곳·검증 못 한 것**: ① 실 Oracle 인스턴스가 없어 Oracle 은 게이트 판정 수준까지만 증명했다(판정이 바이트 단위로 불변이므로 한계는 가볍다). ② 웹 스위트·E2E·`make build` 미실행 — Go 전용 변경이고 SPA 자산과 교집합이 없다. ③ **직전 회차가 왜 verify-failed 였는지는 여전히 확인하지 못했다** — CI 로그에 접근하지 못했고 `ab53169` 의 web flake 2건이라는 것은 정찰의 정황 추정을 그대로 물려받은 것이다. 이번 변경 자체의 근거는 ledger 의 ①~⑧ 이다. ④ `reasonSetIndexThreshold = 8` 은 측정으로 고른 값이 아니라 '평범한 질의는 맵을 할당하지 않는다' 를 만족하는 보수적 값이다 — 1·2·8·maxint 에서 판정이 동일함은 코드로 확인했다.
- **일부러 하지 않은 것**: 사유 개수 상한(1MiB 질의가 여전히 사유 96,332개·약 7.98MB 를 응답에 담는다)은 사유 집합을 바꿔 '판정 불변' 증명을 깨므로 섞지 않고 ideas.json 에 유지했다. `appendUnique` 는 카디널리티가 키워드 표로 묶인 `Analyze` 호출부(76·88)와 단발 호출(114·122)에 그대로 남겼다 — 거기선 quadratic 이 아니다. `postgresAliasColumnList` 의 `AS` 분기, Oracle `WITH (col,…)` 면제(3bf8799)는 둘 다 **판정을 바꾸는** 변경이라 이 순수 성능 회차에 섞지 않았다.
- **다음 역할이 조심할 것**: 새 테스트 3개는 각각 약 693KB 질의를 분석하며 **1초 절대 예산**을 쓴다 — 느린/과부하 러너에서 흔들릴 수 있다(이 환경 실측은 0.07~0.08s 로 예산의 1/12, base 는 5.3~5.8s). 전부 순수 CPU 이고 **DB 는 필요 없다**. `TestAnalyzeDialectDeduplicatesAndOrdersViolationReasons` 는 사유 6건의 **정확한 순서**를 고정한다 — 사유 문자열이나 스캔 순서를 바꾸면 여기서 깨지는 것이 의도된 동작이다(네거티브 컨트롤에서 이 테스트가 시드 없는 병합을 실제로 잡았다). 이 회차의 증명 장치(base 사본 패키지 `zzbase`, 덤프·퍼즈 `zz_tmp_*_test.go`)는 전부 삭제했으므로 재검증하려면 다시 만들어야 한다: `git show afd3882:internal/domain/sqlsafe/sqlsafe.go | sed 's/^package sqlsafe$/package zzbase/'`.
- [러너 16:22] verify passed — 검증 9개 통과 (auto)
- [러너 16:36] review timeout — 단계 제한 시간 초과
- [러너 16:36] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 16:37] pr created — https://github.com/hkjang/qurio/pull/35

## 심사 노트
- HEAD feb607a / base afd3882: 세 부서 스킬을 로컬 문서로 적용했고 변경 2파일·실제 호출부를 검토; 코드 수정 없음.
- Go 전체 -race -count=1 및 vet 통과; 실제 base/HEAD 분석 31,284건 동일, 새 성능 테스트는 base와 인덱스 비활성 사본에서만 실패.
- 1,048,572바이트 입력에서 pg 15.19초→111ms / oracle 14.66초→135ms, 105,424개 사유와 직렬화 결과 동일.
- 실 DB·웹/E2E·빌드/릴리즈·이전 CI 실패 원인은 미확인; 변경 범위 밖. 시간 제한/대량 응답은 비차단 참고. 새 보안·개인정보 차단 소견 없어 approve/merge, risk medium.
