**README의 통합 테스트 준비를 CI와 같은 `scripts/test-services.sh`로 통일했습니다.** 이전 안내의 수동 PostgreSQL 생성과 고정 DSN은 준비 스크립트의 접속 정보와 달랐고, LDAP·LDAPS 준비도 빠져 있었습니다. 이제 세 서비스를 준비한 같은 셸에서 린트와 테스트를 실행하도록 안내합니다.

### 수정

- 수동 PostgreSQL 생성·고정 DSN 대신 준비 스크립트 → 환경변수 적용 → `make lint` → `make test` 순서를 명시하고, 준비에 실패하면 다음 단계로 진행하지 않도록 했습니다.
- PostgreSQL·LDAP·LDAPS가 모두 필요하며, 성공 종료하더라도 연동 테스트의 SKIP 요약을 확인해야 한다고 안내했습니다.
- 기존 컨테이너 재사용, 선택적 재생성 시 테스트 데이터 삭제, LDAPS 인증서 마운트 경로 재사용과 정리 방법, 디렉터리 연동 개발 가이드 링크를 추가했습니다.

### 확인

- 구현 단계에서 실제 HTTP/PostgreSQL·LDAP·LDAPS race 테스트가 PASS/SKIP 0으로 통과했고, `make lint`, `make test`(Go race·vet, 프런트 29파일/161테스트·빌드), `bash -n`, `git diff --check`가 통과했습니다.
- 릴리즈 준비에서도 `make lint`, `make test`(Go race 전 패키지·연동 SKIP 0·`go vet`·프런트 29파일/161테스트·빌드), `make build VERSION=v0.9.89`, 버전 일치 검사, `bash -n`, `git diff --check`가 통과했습니다.
- 기존 컨테이너 재사용을 검증했습니다. 깨끗한 환경의 새 컨테이너 생성·삭제와 재생성은 실행하지 않았습니다.

### Upgrade notes

개발 및 검증 문서만 달라지며 런타임 동작·인증 정책·DB 스키마와 설정은 그대로입니다. 마이그레이션은 없으며 이전 `v0.9.88` 이미지로 롤백할 수 있습니다. 이전 수동 절차로 만든 테스트 PostgreSQL의 비밀번호가 다르면 준비가 실패할 수 있으므로, 테스트 데이터를 삭제해도 되는 경우에만 README의 재생성 절차를 따르세요. 별도 LDAPS 인증서 디렉터리를 쓰는 환경은 기존 마운트 원본과 같은 `RESSO_TEST_CERT_DIR`를 지정하세요.

**Full Changelog**: https://github.com/hkjang/ReSSO/compare/v0.9.88...v0.9.89
