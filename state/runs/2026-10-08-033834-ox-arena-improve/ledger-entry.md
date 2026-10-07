## 2026-10-08
- 선택: Admin 설정 폼(`RoomSettings`)의 "선택 시간(초)" 입력 깨짐과 서버 값 미반영 고치기 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `dur` 을 숫자 state → 문자열 state 로 바꿔 `Number('') === 0` 리셋을 없애고, 저장 시점에만 `Number()` 로 한 번 변환한다. 이름이 공백뿐이거나 선택 시간이 3~60 밖이면 저장 버튼을 비활성화하고 한국어 안내를 띄워, `rooms.duration_sec` 의 `check (between 3 and 60)` 을 위반하는 값이 `admin_update_room` 으로 나갈 경로를 코드에서 없앴다(= `errMsg` 에 매핑 없는 영문 Postgres 메시지가 `.err` 바에 뜰 수 없다). 서버 값 동기화는 원시값 의존 `useEffect` 3개로 했다. 검증: 기준선 `npm i` + `npm run build` EXIT 0(434.14 kB) 확인 후, 임시 Playwright 하네스로 프로덕션 `RoomSettings` 를 실제 `react-dom/client` 에 마운트해 15개 검사를 레드→그린 실측, 이어 `npm run build` EXIT 0(434.50 kB, 89 modules 동일 = 하네스가 번들에 안 섞인 증거). 하네스 삭제 후 `git status --porcelain` = `M src/pages/Admin.tsx` 한 줄, `git diff --stat` = 1파일(+20/-4).
- 실패 재현: 수정 전 하네스 실행 — 15개 중 13개 레드. 과제서의 예측이 전부 실측으로 확인됐고 `015` 는 React 가 교정하지 않고 그대로 남는다:
  `[FAIL] T1a 전부 지우면 빈 칸으로 남는다 :: 지운 뒤 input.value='0'` / `[FAIL] T1b 지운 뒤 15 를 치면 15 다 :: 타이핑 후 input.value='015'`
  `[FAIL] T2c 빈 칸 상태로 저장을 눌러도 범위 밖 p_duration 이 안 나간다 :: 전송된 호출=[{"fn": "admin_update_room", "args": {"p_name": "A방", "p_duration": 0, "p_elimination": false}}]` (← DB check 제약 위반이 실제로 전송됨)
  `[FAIL] T3a 서버 duration_sec 변경이 폼에 반영된다 :: input.value='015' (기대 '20')` / `T3b name 'A방'(기대 'B방')` / `T3c checked=False` → 수정 후 `=== 15 passed / 0 failed ===`.
  추가로 과제서가 1순위 위험으로 지목한 "4초 폴링이 편집 중 입력을 날린다" 가 실재함을 **변이 테스트**로 증명했다. 의존성만 `[room.duration_sec]` → `[room]` 으로 바꾸고 값이 같은(=폴링과 동일한) 새 객체를 넘기니 `[FAIL] T5 :: input.value='20' (기대 '45')` — 운영자가 타이핑한 45 가 날아갔다. 되돌리니 다시 15/15 그린.
- 보류 아이디어: ①Questions 가져오기 입력 검증(`Admin.tsx:231` `importFile` 이 `Array.isArray` 검사 없이 `JSON.parse` 결과를 그대로 서버로 보냄 — 차선 후보였고 같은 1파일, 다음 회차 최우선) ②탈락 모드 정답률·미선택 분모 불일치(`0003:114` `v_total` 이 탈락자 포함 — SQL 보호 경로, 사람 검토 붙는 단독 회차로) ③`Admin.tsx:52` / `Play.tsx:61` 의 `useNow(200)` 전역 루프 단계별 정지(`useNow` 의 `null` 지원이 머지 대기 PR 에 있어 그 전엔 불가) ④`main.tsx` 라우트별 코드 스플리팅(여전히 434 kB 단일 청크, 전/후 청크 크기 실측을 수용 기준에 넣어야) ⑤카운트다운 첫 프레임 `Math.ceil` 반올림 아티팩트(`Timer.tsx:11` — 머지 대기 경로)
- 과제서: 채택 — 1순위를 지정된 1파일로 지정된 방식(문자열 state + 저장 가드 + 원시값 의존 `useEffect`)대로 구현하고 수용 기준 1~5 를 모두 실측으로 충족했다. 선택 과제였던 공백 이름 가드도 같은 함수 안이라 포함했다.
