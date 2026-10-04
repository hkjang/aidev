## 정찰 노트
- 고른 이유: `clientAuthUndecided`(oidc.go:875 부근) 주석이 스스로 "that is a change per endpoint, not one made in the helper they share" 라고 다음 할 일을 지목한다. 원래 L 이던 [3/4/L] 을 **token 한 Endpoint** 로 쪼개 프로덕션 파일 1개(oidc.go)로 끝나게 했다. 차선(등록기 붙이기, 0파일)은 가치 2 로 낮아 2순위.
- 제친 것: `RedeemAuthorizationCode` 400 합치기는 **기각**했다 — 5bed9dc 로 구현됐는데 main 에 없고(`TestIntegrationTokenSaysWhenItCouldNotRedeemTheCode` 부재로 확인) 사람이 받지 않은 접근이다. CSP 와 `active` 버리기는 각각 10회·3회 연속 보류 끝에 기각으로 닫았다.
- 확인한 것: 회귀 범위(401 단언은 integration_test.go:1635·1881 둘뿐), `answered()`(1934-1946)가 status 로 걸러내지 않는 것, 8227·10630·3233 이 token 을 지나지 않는 것, 끊긴 호출자 재현 수단(1805 테스트의 LOCK TABLE + context.WithCancel + pg_locks 폴링) — 모두 파일을 열어 확인했다. **미확인: npm audit 4건의 현재 상태, UpdateUser 세션 종료 단언의 존재 여부.**
- 구현자가 조심할 것: ① `r.Context().Err()` 를 두 번 읽지 말 것 — 헬퍼가 `(undecided, ours)` 를 돌려주고 호출자는 `ours` 만 보게 할 것. 끊긴 호출자에게 500 을 쓰면 8f2c73c 가 닫은 구멍을 `resso_http_requests_total{status="500"}` 에 다시 연다. ② `resso_token_errors_total` 을 더하지 말 것 — 이 시점의 `grant_type` 은 미검증 입력이라 라벨이 열린다. ③ 1805 테스트의 `blocker` 락을 밖으로 새게 하면 같은 컨테이너의 뒤 테스트가 전멸한다.
- 프로필은 2일 전 것이라 다시 쓰지 않았다. 다만 그 "현재 공백" 절의 1순위 ①은 위 사유로 **기각**이고, v0.9.97 의 세 커밋(954da4b·59b6c34·8f2c73c)이 반영돼 있지 않다.
