화면을 쓰는 사람과 그 화면을 띄워 놓고 지키는 사람을 위한 **사용자 가이드**와 **관리자 가이드**가 생겼습니다. 실린 그림 스무 장은 모두 실제로 띄운 muni 에서 찍은 것입니다.

## 추가

### 사용자 가이드와 관리자 가이드

muni 에는 README 와 운영 안내만 있었습니다. 설치하는 사람이 볼 글은 있었지만, 로그인해서 문서를 쓰고 댓글을 달고 내보내는 사람이 볼 글도, 계정을 만들고 사고가 났을 때 무엇을 확인할지 적은 글도 없었고 — 화면 캡처는 한 장도 없었습니다.

[사용자 가이드](https://github.com/hkjang/muni/blob/v0.37.0/docs/USER_GUIDE.md)([PDF](https://github.com/hkjang/muni/blob/v0.37.0/docs/USER_GUIDE.pdf))는 로그인부터 홈·워크스페이스·편집기·댓글·버전·내보내기·공유·검색·빠른 이동·개인 설정까지를, [관리자 가이드](https://github.com/hkjang/muni/blob/v0.37.0/docs/ADMIN_GUIDE.md)([PDF](https://github.com/hkjang/muni/blob/v0.37.0/docs/ADMIN_GUIDE.pdf))는 구성 요소·설치·설정·계정과 권한·운영·장애 대응·보안을 다룹니다. README 에서 두 문서로 가는 길을 냈습니다.

**문서에 적은 것은 코드에서 읽었습니다.** 환경 변수 표는 `internal/config` 에서, API 의 메서드와 경로는 라우트가 등록된 자리에서, 오류 메시지는 서버가 실제로 내는 문구에서 가져왔습니다. 겹치는 자리는 옮겨 적지 않고 링크합니다 — 백업·복구·키 교체·퇴사자 정리는 [운영 안내](https://github.com/hkjang/muni/blob/v0.37.0/docs/OPERATIONS.md)가 정본이고, 관리자 가이드는 그것을 가리키며, 운영 안내는 관리자 가이드를 되가리킵니다. 정본은 하나입니다.

### 화면 캡처 스크립트

그림은 손으로 찍지 않았습니다. `frontend/scripts/guide-screenshots.mjs`(`npm run screenshots`)가 데모 데이터 — 회사 하나, 팀 하나, 문서 다섯, 사람 셋 — 를 채우고 1440x900 headless Chromium 으로 사용자 화면 열두 장과 관리 화면 여덟 장을 찍습니다. 화면이 바뀌면 다시 돌려 다시 찍습니다.

데모 데이터를 **만드는** 스크립트이므로 조심스럽게 두었습니다. 대상 주소는 e2e 와 공유하지 않는 전용 변수(`MUNI_GUIDE_BASE_URL`)로만 받고 로컬이 아니면 멈춥니다. 자격 증명은 환경 변수로만 받고, 데모 계정의 비밀번호는 실행할 때마다 새로 만들어 어디에도 적히지 않습니다.

## 업그레이드

마이그레이션과 설정 변경은 필요하지 않습니다. 이번 릴리스는 문서만 더했고 서버와 화면의 동작은 v0.36.0 과 같습니다.

```bash
gzip -dc muni-v0.37.0.tar.gz | docker load
docker compose -f compose.example.yaml --env-file .env up -d
```

## 오프라인 설치

릴리스 asset 에는 `muni:v0.37.0` Docker 이미지가 포함되어 있습니다.

```bash
gzip -dc muni-v0.37.0.tar.gz | docker load
docker image inspect muni:v0.37.0

cp .env.example .env
# .env의 네 값을 운영 환경에 맞게 변경합니다.
docker compose -f compose.example.yaml --env-file .env up -d
```

애플리케이션이 반드시 필요로 하는 런타임 환경변수는 다음 네 개입니다.

| 환경변수                   | 설명                                 |
| -------------------------- | ------------------------------------ |
| `POSTGRES_DSN`             | PostgreSQL 접속 문자열               |
| `BOOTSTRAP_ADMIN`          | 최초 관리자 아이디 또는 이메일       |
| `BOOTSTRAP_ADMIN_PASSWORD` | 최초 관리자 비밀번호(12자 이상)      |
| `ENCRYPTION_KEY`           | base64로 인코딩한 32-byte master key |

PDF 변환 동작만 조정하는 선택 환경변수가 두 개 있습니다.

| 환경변수               | 기본값    | 설명                                       |
| ---------------------- | --------- | ------------------------------------------ |
| `MUNI_CHROMIUM_PATH`   | 자동 탐색 | PDF Export에 사용할 headless 브라우저 경로 |
| `MUNI_PDF_CONCURRENCY` | `2`       | 동시에 실행할 Chromium 프로세스 수(1~32)   |

## 운영 정보

- 서비스 포트: `8080`
- Liveness: `/healthz`
- Readiness: `/readyz`
- REST API: `/api/v1`
- 공개 링크: `/s/{token}` — 인증 없이 열리는 유일한 화면입니다
- OpenAPI: `/api/openapi.yaml` — 실제 라우트와 일치하며, 테스트가 그것을 지킵니다
- Prometheus: `/metrics` — 관리자 인증 필요
- MCP: `/mcp`
- 지원 DB: PostgreSQL 15 이상
- 이미지에는 PDF Export용 Chromium과 Noto CJK 글꼴이 포함되어 있습니다.

## 릴리스 파일 검증

- 파일: `muni-v0.37.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.37.0`

```bash
sha256sum muni-v0.37.0.tar.gz
```

GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 완료한 뒤 이 asset을 게시했습니다.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [사용자 가이드](https://github.com/hkjang/muni/blob/v0.37.0/docs/USER_GUIDE.md)
- [관리자 가이드](https://github.com/hkjang/muni/blob/v0.37.0/docs/ADMIN_GUIDE.md)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.37.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.37.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.37.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.36.0...v0.37.0)
