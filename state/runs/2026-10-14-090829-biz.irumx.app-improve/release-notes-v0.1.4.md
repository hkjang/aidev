v0.1.4 금액·정수·소수 입력이 '0x10'·'1e3' 을 조용히 받던 것 차단

v0.1.3 뒤 한 갈래다. 프로덕션 변경은 1파일이고, 공급가·수금액·차변/대변이 지나는
입력 검증에서 비십진 리터럴이 검증 오류 없이 금액으로 통과하던 것을 막았다.

- util/validate.ts: Number() 는 16진·2진·8진·지수 리터럴을 모두 수로 바꾸므로
  공급가에 '0x10' 이 들어오면 거절이 아니라 16원으로 통과해 불변 원장에 기록됐다
  ('1e3'→1000, '0b101'→5, '0o17'→15). 십진 리터럴 정규식(DECIMAL_RE)을 통과한
  문자열만 변환하는 asNumber 로 좁히고, 리터럴이 아니면 원래 문자열을 그대로 돌려줘
  기존 typeof n !== 'number' 분기가 기존 한국어 문구(… 원 단위 정수여야 합니다. /
  … 정수여야 합니다. / … 숫자여야 합니다.)로 거절하게 했다 — 새 메시지·새 분기 0개.
  쉼표·공백 제거 규칙은 경로별로 그대로다(money·decimal 은 제거, int 는 trim 만).
- 라우트가 실제로 쓰는 진입점 parse/parseOne 과 실제 검증기 구성을 지나는 시험
  24개를 더했다. 수정 전 15개 실패 → 수정 후 전원 통과한다.
- 버전: VERSION, 워크스페이스 4개와 package-lock, scripts/ci-env.mjs 와 wrangler.toml
  의 BIZ_VERSION, docs/openapi.yaml(npm run routes).
- 검증: 단위 138, e2e 300, ui-flows 16. npm run check(ESLint + tsc 3개)와 npm run build.

