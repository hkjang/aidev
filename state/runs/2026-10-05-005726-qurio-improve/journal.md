# 회차 노트 2026-10-05-005726-qurio-improve — qurio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:57] base pinned — main@201bf2e
- [러너 00:57] autonomy release — 

## 정찰 노트
- 보류 아이디어 중 flake(M, 지난 회차가 가설 3개를 반증하고도 재현 실패)와 `-count=1` 워크플로 복구(보호 경로, 이미 한 번 머지 실패)를 제치고 **측정으로 확정된** 성능 결함을 골랐다: `postgresCTEColumnList` 의 역방향 루프가 4000/8000/16000 terms 에서 112ms/446ms/1.76s = 바이트 2배마다 정확히 4배. 허용목록 함수(`upper`)로 모양을 만들어 차단 경로 비용을 배제했고, `AS` 접미사만 떼어 루프를 끄면 1.996s→0.257s 가 되는 것으로 인과까지 좁혔다.
- 과제서에서 **추측인 것 하나**: 1MiB 한도에서 ~18초라는 수치는 324KB/1.76s 에서 제곱 외삽한 추정이며 측정하지 않았다. 구현자는 그 값을 단정하지 말 것.
- 구현자가 조심할 것 ①: 루프의 답은 (커서, columnDepth) 의 함수다 — 이 루프는 깊이가 다른 토큰을 *건너뛰기만* 하므로 바깥 그룹을 빠져나가 형제 그룹 안의 같은 깊이 토큰까지 본다. 커서만으로 memo 키를 잡으면 깊이가 다른 사이트끼리 답을 훔쳐 **fail-open** 이 된다.
- 구현자가 조심할 것 ②: 같은 측정에서 **두 번째 quadratic** 을 발견했다 — `postgresReadOnlyViolations` 의 `appendUnique` 가 서로 다른 함수 이름 수에 대해 O(n²)(16000개에서 0.26s). 선재 결함이고 경로가 달라 범위 밖으로 뺐으니 같이 고치지 말 것(ideas.json 에 별 항목으로 올렸다).
- 이 base 에 **3bf8799(Oracle WITH CTE 면제)와 12223b0(`-count=1`)이 미머지**다. `cteColumnList` 심볼이 없고 `scripts/ci_workflow_test.go` 도 없다 — 그 두 파일·심볼을 전제한 과제서는 이 base 에서 무효다.
- [러너 01:04] scout done — `postgresCTEColumnList` 의 역방향 WITH 추적 루프를 메모이제이션해 PostgreSQL 게이트의 quadratic 제거 (가치 4 / 위험 2 / 작업량 S)

## 구현 노트
- `postgresCTEColumnList`(sqlsafe.go:952)의 역방향 WITH 루프를 `cteWithList` 로 떼어내 **(커서, columnDepth) 키**의 lazy memo+trail 로 감쌌다 — 걸음은 한 줄도 바꾸지 않고 답만 재사용. 16000 terms 1.754s → 0.020s, 4000/8000/16000 = 5.8/10.4/19.7ms(선형). 판정 불변은 182판정 덤프 md5 동일 + 차별 퍼즈 92만 비교 drift 0 으로 증명.
- **확신 없는 곳 / 검증 못 한 것**: (a) 과제서의 "1MiB 에서 ~18초" 는 외삽 추정치이고 나도 **측정하지 않았다** — 실측 상한은 325KB/1.76s. (b) 깊이 한도 8 의 값 자체는 임의다. 한도 0·1·1000 에서 전체 스위트가 모두 통과(한도 0 은 스케일 테스트만 실패)하므로 **판정에는 영향이 없고 성능/메모리 트레이드오프일 뿐**이지만, 깊이 8 초과 + 사이트 다수인 질의는 여전히 quadratic 이다(그런 질의를 만들어 측정하지는 않았다). (c) 실 Oracle 인스턴스 없음 — 다만 Oracle 은 이 헬퍼를 호출하지 않으므로(3bf8799 미머지) 영향이 없고 182판정 덤프의 Oracle 91행이 바이트 동일함으로 확인했다. (d) **journal.md 의 정찰 노트를 한 번 덮어썼다가 `stages.json`·`agent-scout.txt` 로 복원했다** — 위 정찰 노트 5줄과 러너 3줄은 그 복원본이며 내용은 원문과 같다(마지막 러너 줄만 stages.json 의 잘린 reason 을 과제서로 보완했다).
- **일부러 하지 않은 것**: `appendUnique` 의 두 번째 quadratic(서로 다른 이름 16000개 = 0.26s, 양 방언) — 경로가 다르고 과제서가 범위 밖으로 못박았다. `fromItemContext` 와의 memo 공통화 — 걸음이 서로 달라 fail-open 위험이 이득보다 크다(ideas.json 에 rejected 로 기록). Oracle 비대칭·`-count=1` 워크플로 복구도 손대지 않았다.
- **다음 역할이 조심할 것**: 새 테스트 `TestAnalyzeDialectScalesLinearlyOverAliasedPostgresCalls` 는 **시간 단정**이라 극단적으로 부하가 걸린 머신에서는 흔들릴 수 있다(예산 500ms, 실측 10–20ms = 25배 여유, 기존 스케일 테스트와 같은 관례로 `t.Parallel()` 없이 직렬). DB 불필요. 나머지 새 테스트 12건은 순수 단위 테스트다.
- **비평가가 먼저 볼 곳**: 이 루프에는 기존 커버리지가 **0** 이었다(단일 CTE 는 967행 지름길로 빠지고, 기존 CTE 테스트 2건이 전부 그 경로다). 그래서 수용 기준 4 의 "메모 고정 → 기존 테스트 red" 가 **성립하지 않았고**, 루프를 실제로 지나는 테스트 12건을 새로 써서(= base 201bf2e 에서도 통과하므로 판정 불변) 그 위에서 인과를 확인했다. 깊이 키를 붕괴시키면 `SELECT * FROM (WITH p (x) AS (…), q (y) AS (…) …) sub, side_effect_fn (3) AS z` 가 **blocked → allowed 로 fail-open** 되는 것을 실측했고 그 질의를 차단 테스트에 고정했다 — 이 키가 이 변경의 유일한 위험 지점이다.
- [러너 01:19] brief accepted — 채택 — 지목한 행(`postgresCTEColumnList` 952, 루프 976-987, `postgresTokenIndex` 829, `build()` 846, 스케일 테스트 530)이 모두 코드와 정�
- [러너 01:22] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 판정 **approve** (risk low, blocking 없음). 실패 재현을 직접 했다: base 201bf2e 로 되돌리면 스케일 테스트만 1.743s 로 FAIL, 나머지 신규 2건은 PASS(판정 불변) — 수정본에서 전부 PASS. 판정 불변은 독립 검증했다(패키지 복사 후 깊이 한도를 var 로 돌려 memo off/on 비교: 무작위 40만 + 중첩 WITH 12만, 각 × 2방언, drift 0). 깊이 키를 `cteMemoFor(0)` 으로 붕괴시키면 실제로 blocked→allowed 가 되고 새 차단 테스트가 잡는다 — 키는 하중을 받고 고정돼 있다.
- 구현 노트의 미검증 지점 (b) 를 측정해 확정했다: **깊이 한도 8 은 적대적 DoS 를 닫지 못한다.** `SELECT 1, upper (x) AS p0, (` 를 8겹(겹당 29바이트) 앞에 붙여 memo 슬롯을 소진시키면 quadratic 이 복귀한다 — prefix=8 에서 2000/4000/8000 terms = 32/129/497ms(2배당 4배), **1MiB 에서 18.6초**. 같은 1MiB 가 prefix 없으면 59ms. base 는 prefix=0 17.3s / prefix=8 16.9s 이므로 최악값은 사실상 불변(+10%)이고 평범한 질의만 290배 빨라진다. 회귀도 새 경로도 아니라 차단하지 않지만 — **릴리즈 노트에 'DoS 수정' 으로 쓰면 사실과 다르다. '평범한 대형 projection 의 quadratic 제거' 로 써야 한다.** (구현 노트의 미검증 (a) 외삽치도 실측으로 맞음을 확인: base 1MiB = 17.3s.)
- 한도를 없앨 수는 없다 — 중첩마다 배열이 생겨 N² 메모리(1MiB 에서 수 GB)가 된다. 즉 CPU/메모리 실제 트레이드오프이고 이 지점은 방어 가능하다. 근본 해결은 (cursor,depth) 희소 memo 또는 payload 한도·검증 타임아웃 하향이며 다음 회차 아이디어 감이다.
- 시간 단정 여유는 구현 노트의 '25배' 보다 좁다: `-race` 에서 130ms / 예산 500ms = 3.8배(기존 chained 스케일 테스트는 같은 조건 170ms). CI `-p=1 -race` 에서 흔들리면 이 테스트부터 보라.
- 못 본 것: 실 Oracle 인스턴스 없음(`TestOracleLive` 미실행; 위 퍼즈의 oracle 52만 비교로 대체), 통합/웹/E2E 는 변경이 닿지 않아 미실행. 범위 밖 `appendUnique` quadratic 과 미머지 `-count=1`·3bf8799 는 그대로다. 워크트리는 검증용 임시 파일을 모두 제거해 clean 로 되돌렸다.
- [러너 01:30] review approved — 리뷰 승인 (risk=low)
- [러너 01:30] pr created — https://github.com/hkjang/qurio/pull/33
- [러너 01:47] ci passed — 검사 1개 모두 success
- [러너 01:47] merge done — 5d8a1b5
- [러너 02:13] release published — v1.4.12
- [러너 02:39] assets verified — v1.4.12 자산 1개 (이전 v1.4.10: 1)
