첨부를 내려받을 때 **`filename*`을 읽지 못하는 클라이언트가 파일을 확장자 없이 저장하던 것**을 고쳤습니다. 화면은 바뀌지 않았고, 마이그레이션도 설정 변경도 없습니다.

## 고친 것

### 첨부 내려받기의 ASCII fallback 이름에 확장자가 없던 것

내려받기 응답의 `Content-Disposition`은 파일 이름을 두 벌로 내보냅니다. 한글과 공백을 담을 수 있는 `filename*`(RFC 8187)과, 그것을 읽지 못하는 클라이언트를 위한 ASCII `filename` fallback입니다. 구형 브라우저, 단순 HTTP 클라이언트, 폐쇄망에서 쓰는 내려받기 도구는 뒤쪽만 봅니다.

muni에서 이름을 두 벌로 내보내는 내려받기는 다섯입니다. **그 가운데 첨부 내려받기 하나만 fallback이 확장자 없는 상수였습니다.**

| 내려받기 | ASCII fallback (이전) | ASCII fallback (지금) |
| --- | --- | --- |
| 문서 내보내기 | `document.md` | 그대로 |
| 프레젠테이션 | `presentation.pdf` | 그대로 |
| 워크스페이스 ZIP | `workspace-20260929.zip` | 그대로 |
| 문서 넘기기 | `document.xlsx` | 그대로 |
| **첨부 내려받기** | **`attachment`** | **`attachment.xlsx`** |

그래서 `회의록.xlsx` 첨부를 그런 클라이언트로 받으면 **확장자 없는 `attachment` 파일**이 되어, 내용은 온전한데도 연결 프로그램이 열지 못하고 사용자가 이름을 손으로 고쳐야 했습니다.

### 확장자를 그대로 붙일 수는 없습니다

첨부 이름은 올린 사람의 파일 이름을 240룬으로 자른 값이며, **어떤 문자도 치환되지 않습니다**. 그 이름에서 뽑은 확장자를 quoted-string에 그대로 넣으면 큰따옴표 하나가 매개변수를 미리 끝내고, 헤더 전체가 `mime: invalid media parameter`로 거부됩니다. 그러면 확장자만 없던 오늘보다 나쁩니다 — `filename*`이 싣고 있던 **진짜 이름까지 잃습니다**.

그래서 **인용도 인코딩도 필요 없는 확장자일 때만** 채택합니다: 점으로 시작하고, 나머지가 전부 ASCII 영숫자이며, 점을 포함해 12자 이하일 때입니다.

| 올린 이름 | ASCII fallback | `filename*`이 싣는 이름 |
| --- | --- | --- |
| `회의록.xlsx` | `attachment.xlsx` | `회의록.xlsx` |
| `회의록` | `attachment` | `회의록` |
| `보고서.p"g` | `attachment` | `보고서.p"g` |
| `자료.한글확장자` | `attachment` | `자료.한글확장자` |
| `자료.abcdefghijkl` | `attachment` | `자료.abcdefghijkl` |

채택하지 않는 경우는 **이전과 똑같이 `attachment`로 남습니다.** 어느 경우에도 `filename*`은 손대지 않았으므로, 그것을 읽는 클라이언트(요즘 브라우저와 muni 화면)가 저장하는 이름은 이번 릴리스로 **전혀 바뀌지 않습니다.**

### 남아 있는 것

**12자 상한은 판단으로 고른 값입니다.** muni가 읽고 쓰는 모든 형식과 `.properties`(11자)까지 통과하지만, 그보다 긴 정당한 확장자가 있다면 fallback만 조용히 확장자를 잃습니다 — 그때의 결과는 이번 수정 이전과 같습니다.

`extValueEscape`, `safeFilename`, 240룬 절단, 나머지 네 헤더 자리, inline/attachment 분기, CSP sandbox 헤더, 감사 기록은 **그대로 두었습니다.** 저장되는 값도 바뀌지 않으므로 이미 올린 첨부에 다시 손댈 필요가 없습니다.

## 검증

- 전용 PostgreSQL 16에서 `go test -count=1 -json ./...`: 하위 테스트 포함 **719 PASS**, 외부 코퍼스·수동 출력 4 SKIP, 실패 0. HTTP API 테스트는 SKIP 없이 통과했습니다.
- 실제 라우트로 표-주도 검증을 추가했습니다: 위 표의 다섯 이름을 멀티파트로 올린 뒤 `GET /api/v1/attachments/{id}`의 `Content-Disposition`을 받아, fallback 문자열을 확인하고 **프로덕션과 같은 파서로 `filename*`을 되읽어** 올린 이름과 일치하는지 매 행마다 확인합니다. 고치기 전에는 첫 행이 실패하고 나머지 네 행은 통과하는 것을 먼저 확인했습니다.
- 프런트 `npm run lint`(`tsc -b`), `npm test` 42파일·297건, `npm run build` 통과.
- `go vet ./...`, Go 포맷 검사, placeholder 검사, `git diff --check` 통과.
- `CGO_ENABLED=0 go build -trimpath`, Docker 이미지 빌드 통과.
- 브라우저 E2E와 실제 운영 배포는 이번 릴리즈 준비에서 실행하지 않았습니다. 배포용 archive는 태그 푸시 워크플로가 생성합니다.

## 업그레이드

기존 DB와 환경변수를 그대로 사용합니다. 이미지를 불러온 뒤 사용하는 Compose 파일의 `image`를 `muni:v0.52.0`으로 지정하세요. 저장소의 예제는 `muni:v0.1.0`을 가리키므로 운영 버전에 맞춰 변경해야 합니다.

```bash
gzip -dc muni-v0.52.0.tar.gz | docker load
# compose.example.yaml의 image를 muni:v0.52.0으로 지정합니다.
docker compose -f compose.example.yaml --env-file .env up -d
```

배포 담당자는 먼저 검증 환경에서 `/healthz`·`/readyz`와 첨부 올리기·내려받기를 확인한 뒤 운영에 적용하세요. 확장자가 있는 첨부를 내려받아 **브라우저가 저장하는 이름이 올린 이름 그대로인지**(즉 `filename*` 경로가 바뀌지 않았는지) 함께 확인합니다. 두 상태 점검이 실패하거나 저장되는 이름이 이전과 다르면 배포를 중단하고, 보관한 이전 이미지 태그로 되돌려 같은 Compose 명령을 실행합니다. 이번 변경에 따른 DB 되돌리기는 필요하지 않습니다.

배포 후 첫 주에는 첨부 내려받기 관련 문의를 확인합니다. 확인 기준은 `filename*`을 읽는 클라이언트의 저장 이름이 0건 바뀌고, 그것을 읽지 못하는 클라이언트에서 확장자 없는 `attachment`로 저장된다는 문의가 더 오지 않는 것입니다. 실제 운영 적용과 사후 관찰은 배포 담당자가 수행합니다.

## 오프라인 설치

워크플로가 게시할 릴리스 asset에는 `muni:v0.52.0` Docker 이미지가 포함됩니다.

```bash
gzip -dc muni-v0.52.0.tar.gz | docker load
docker image inspect muni:v0.52.0

# compose.example.yaml의 image를 muni:v0.52.0으로 지정합니다.
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
- 문서 넘기기: `/handoff?source&claim` (받는 쪽, 세션 필요, 허용 목록의 오리진만) · `/api/v1/handoff/claims/{claim}` (내주는 쪽, 인증 없이 5분짜리 단일 사용 표로만 열립니다)
- 메일 알림: 사내 SMTP 릴레이로 1분마다 배경 발송, 기본 꺼짐 · 발송 기록 `/api/v1/admin/mail/deliveries` (관리자 인증 필요)
- OpenAPI: `/api/openapi.yaml` — 실제 라우트와 일치하며, 테스트가 그것을 지킵니다
- Prometheus: `/metrics` — 관리자 인증 필요
- MCP: `/mcp`
- Momento 프록시: `/momento/*` — 방문 추적을 Momento 로 켜 둔 동안에만 답합니다
- 지원 DB: PostgreSQL 15 이상
- 이미지에는 PDF Export용 Chromium과 Noto CJK 글꼴이 포함되어 있습니다.

## 릴리스 파일 검증

- 파일: `muni-v0.52.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.52.0`

```bash
sha256sum muni-v0.52.0.tar.gz
```

태그 푸시 후 GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 수행하고 이 asset을 게시합니다. 게시 완료 여부는 해당 워크플로에서 확인하세요.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [사용자 가이드](https://github.com/hkjang/muni/blob/v0.52.0/docs/USER_GUIDE.md)
- [관리자 가이드](https://github.com/hkjang/muni/blob/v0.52.0/docs/ADMIN_GUIDE.md)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.52.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.52.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.52.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.51.0...v0.52.0)
