# Hunter v1.10.0

v1.9.0의 자동 SSO 진입을 SILENT-SSO-STANDARD.md 정합에 맞춰 보완한 릴리즈입니다. 자동 진입은 관리자가 명시적으로 켠 경우에만 동작하고, 브라우저 저장소를 읽을 수 없으면 재시도를 억제합니다. 일반 PostgreSQL·네 환경변수·서비스 Docker 이미지 하나의 기본 배포를 유지합니다.

## 사용자가 달라지는 점

- OIDC 자동 진입(`auto_login`)의 기본값이 **꺼짐**입니다. 설정 키가 없거나 불리언이 아니면 꺼진 것으로 취급해 기본 설치에서는 prompt=none 요청을 보내지 않습니다. 서버 기본 설정, 관리자 설정 화면, OpenAPI 계약이 같은 기본값을 사용합니다.
- 자동 진입이 꺼진 상태의 `?mode=auto` 요청은 IdP에 접속하지 않고 평범한 로그인 화면으로 돌아갑니다. 관리자가 켜면 기존과 같이 prompt=none으로 사내 인증 세션을 확인합니다.
- 사생활 보호 모드 등으로 sessionStorage를 읽을 수 없으면 브라우저는 자동 시도를 **이미 한 것**으로 간주해 매 진입마다 리디렉션을 반복하지 않습니다.
- state·nonce·PKCE·서명 토큰·현재 역할 검사, 10분 자동 시도·24시간 명시 로그아웃 억제, `/login?local=1` 복구 경로와 `return_to` 검증은 v1.9.0과 같습니다.

## 운영 조건과 한계

v1.9.0에서 자동 진입을 사용하던 조직은 업그레이드 후 **관리자 → 서비스 설정 → OIDC**에서 자동 진입을 다시 켜야 합니다. 기존 설정에 `auto_login: true`가 저장되어 있으면 그대로 유지됩니다. 실제 운영 ReSSO 계정 연동 검증은 v1.9.0과 마찬가지로 미수행이며 사내 환경에서 OIDC Discovery·Code·PKCE·prompt=none 지원을 확인해야 합니다.

## 검증과 배포

Go 테스트 `TestOIDCAutomaticLoginIsOptInByDefault`(키 없음·비불리언·`?mode=auto` 요청이 IdP 접속 없이 로그인 화면으로 복귀, 켜면 prompt=none 시작)와 web 테스트(차단된 저장소·저장소 없음 → 억제)를 추가했습니다. 릴리즈 커밋에서 `go vet`, 임시 PostgreSQL 컨테이너의 `go test -race`, `npm test`, `npm run build`(tsc 포함), `node scripts/check-docs.mjs`, `bash -n scripts/release.sh`, `python3 -m py_compile scripts/release-notes.py`, `node scripts/verify-pentagi.mjs`를 실행했습니다. 최종 게시 커밋의 CI와 공개 아카이브 다운로드 검증은 실제 완료한 뒤 [릴리즈 본문](https://github.com/hkjang/hunter/releases/tag/v1.10.0)에 기록합니다.

화면 캡처는 v1.9.0의 검증 장면을 보존했습니다. 관리자 가이드(md·html·pdf)·README의 자동 진입 기본값 설명을 갱신했습니다.

배포 이미지: `hunter:v1.10.0` · 유일한 첨부 자산: `hunter-v1.10.0.tar.gz`.

[사용자 가이드](guides/user-guide.html) · [관리자 가이드](guides/admin-guide.html) · [공식 조사와 적용 범위](research-sso-tracking.md) · [v1.9.0 릴리즈 노트](release-v1.9.0.md)
