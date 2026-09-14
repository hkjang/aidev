# Postra v0.19.1 — OIDC 콜백 실패를 로그인 화면으로

v0.19.0의 조용한 SSO(silent SSO) 흐름에서 남아 있던 콜백 경로의 틈을 메운 패치 릴리즈입니다. 기능 추가나 스키마 변경은 없으며, 바이너리 또는 이미지만 교체하면 됩니다.

## 고친 문제

- 조용한 거절(`login_required`)이 아닌 진짜 실패 — 제공자 오류(`invalid_request` 등), 플로우 쿠키 없음·`state` 불일치, 토큰 교환·검증 실패(잘못된 client secret, 허용되지 않은 scope·redirect_uri, 비활성 사용자, 자동 생성 꺼짐) — 를 `/ui/auth/oidc/callback?code=…&state=…` 위에 인라인으로 그렸습니다. 브라우저가 콜백 주소에 머물러 새로고침마다 이미 쓴 코드를 다시 보내고, 그때마다 "토큰 교환 실패" 장애 기록이 하나씩 쌓였습니다.
- 이제 콜백은 플로우 쿠키를 지우고 사유를 서명된 HttpOnly 일회용 쿠키(`postra_oidc_error`, `Path=/ui/login`, 60초, 1KB 룬 경계 절단)에 담아 `/ui/login?sso=error` 로 보냅니다. 깊은 링크가 있으면 `return_to` 로 함께 넘깁니다.
- 로그인 화면은 사유를 한 번 보여 주고 쿠키를 지웁니다. 주소에 `sso=` 표시(`none`·`error`·`signed_out`)가 있으면 조용한 로그인을 다시 시도하지 않으므로 콜백 → 로그인 → 콜백 되돌이표가 생기지 않습니다.
- 사유 쿠키는 플로우 쿠키와 같은 키로 서명하되 도메인 분리(`oidc-error:` 접두)를 두어 서로 재생할 수 없습니다. 위조·만료된 쪽지는 일반 메시지로 대체됩니다.
- 관리자 가이드 3.3절에 `sso=error` 동작을 추가하고 PDF를 다시 생성했습니다.

## 검증

- webui 테스트 갱신 및 신규 추가: 플로우 쿠키 없음 → 302 와 쿠키 삭제, 사유 1회 표시·삭제·조용한 시도 없음·`return_to` 유지, 위조 쪽지 미반영, 맨 링크의 일반 메시지, 일반 로그인 화면 불변.
- application 테스트 추가: 사유 쿠키 왕복, 도메인 분리, 서명 위조, 만료, 빈 메시지, 긴 메시지 절단.
- `go build`, `go vet`, `go test -race ./...` 전부 통과. 실바이너리에서 302 → 사유 표시 → 재방문 시 일반 메시지를 확인했습니다.

## 기존 릴리즈와의 호환

v0.19.0의 React AI 업무 워크스페이스(`/app/`)와 HTML 서식 메일, v0.18.7의 `auth.oidc.auto_login`(기본 꺼짐)·탭당 한 번의 `prompt=none` 시도·안전한 `return_to`, 실행 파일/이미지의 커밋·빌드 시각·OCI 라벨을 그대로 유지합니다. `/app/` 의 SSO 자동 로그인도 같은 콜백을 쓰므로 함께 적용됩니다.

## 배포

기존 데이터베이스 스키마 변경이나 새 환경변수는 없습니다. 기존 볼륨·DB·Secret 설정을 유지한 채 바이너리 또는 이미지를 교체하세요.

- `postra-0.19.1-linux-amd64`: 정적 실행 파일
- `postra-0.19.1-linux-amd64-image.tar.gz`: 오프라인 `docker load`용 이미지 (`postra:0.19.1`)
- `postra-0.19.1-sbom.cdx.json`: CycloneDX SBOM
- `postra-0.19.1-frontend-sbom.cdx.json`: 프런트엔드 CycloneDX SBOM
- `SHA256SUMS.txt`: 배포 파일 무결성 검증
