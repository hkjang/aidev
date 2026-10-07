서비스 역할(law_app)은 공유 Supabase 에서 NOLOGIN 이고 앱은 공유 게이트웨이
(irumx_gateway)로만 접속한다 — 게이트웨이 상한(30)이 DB 를 지키는 유일한 하드 캡이다.
배포와 마이그레이션이 그 전제를 되돌릴 수 있던 자리를 막고, 공유 Hyperdrive 설정을
건드리지 않게 했다. 링크 안전 검사의 끝 점 우회도 함께 고쳤다.

- 배포 스크립트: bootstrap 연결도 공유 게이트웨이로 로그인해 law_app 으로 전환한다.
  law_app 으로 직접 로그인하면 그 역할의 CONNECTION LIMIT(5)을 옛 Hyperdrive 풀과
  다퉈 'too many connections for role "law_app"' 로 배포가 멈췄다.
- 배포마다 wrangler.toml 의 Hyperdrive 상한을 5로 다시 설정하던 단계를 없앴다. 설정이
  공유(irumx-shared-db)가 된 뒤에는 그것이 모든 서비스의 상한을 5로 내렸다. 이제는
  공유 설정인지만 확인하고, 아니면 배포를 중단한다. 상한은 한 곳에서 관리한다.
- 배포 전 검사와 비밀값: migrate 에 서비스 역할 비밀번호를 넘기지 않고, --secrets 가
  직접 접속 URL 비밀값을 올리지 않는다(그 역할로 상한 밖 연결을 열 수 있었다).
- migrate.mjs: 대상이 Supabase(*.supabase.com/.co)면 앱·게이트웨이 비밀번호를 무시하고
  그렇다고 출력한다(개발·CI 127.0.0.1 은 그대로). cloud.env 에 남은 옛
  <SVC>_APP_DB_PASSWORD 를 보고 서비스 역할의 LOGIN 을 되살리거나, 8개 서비스가 함께
  쓰는 게이트웨이 비밀번호를 바꿔 모두를 끊는 일을 막는다. shared-db-guard.test.ts 가
  판정식과 그 자리들을 시험으로 고정한다.
- isSafeLink: 끝 점(루트 라벨)을 한 번만 떼어 라벨 검사와 차단 호스트 목록 조회에 같은
  값을 쓴다. WHATWG URL 은 IPv4 리터럴에서만 끝 점을 지우므로
  http://metadata.google.internal./ 가 공개 호스트로 통과했다. text.test.ts 의 차단·허용
  표에 끝 점 형태를 더해 과차단이 아닌 것도 함께 고정했다.
- 운영 문서: 공유 DB 에서 서비스 역할은 NOLOGIN 이므로 운영 마이그레이션에 앱 비밀번호를
  넘기지 않는다. 연결 배분·시간 제한·감시의 정본은 yeopjari engineering/shared-db.md.
- 검증: lint·types, 단위 96, 웹·서버 빌드, VERSION·wrangler LAW_VERSION 일치 검사.
  e2e·acceptance·ui-flows 는 이 기계에서 돌리지 못했다(개발 서버가 :8810 을 쓰는 중이고
  PostgreSQL 17 클라이언트가 없다) — check.yml 이 main 푸시마다 그 전부를 돌린다.
