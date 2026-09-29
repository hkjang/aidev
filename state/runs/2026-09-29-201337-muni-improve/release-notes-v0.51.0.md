문서를 가져오거나 첨부를 올리거나 문서를 넘겨받을 때 **제목과 첨부 이름이 240룬을 넘으면 AI 컨텍스트 생략 문구가 함께 저장되던 것**을 고쳤습니다. 화면은 바뀌지 않았고, 마이그레이션도 설정 변경도 없습니다.

## 고친 것

### 저장하는 제목·첨부 이름의 240룬 절단에 AI 안내 문구가 붙던 것

muni에는 240룬을 넘는 값을 자르는 자리가 여러 개 있고, 그 가운데 **저장되는 값을 자르는 여섯 자리가 AI 프롬프트용 절단기를 쓰고 있었습니다**. 그 절단기는 값을 줄이면 줄인 사실을 모델에게 알리려고 줄바꿈과 문구를 덧붙입니다.

```
긴 제목…긴 제목
[…문서 컨텍스트가 길어 일부 생략됨…]
```

`documents.title`과 `attachments.name`은 둘 다 `text`라 길이 제한이 없으므로, 이 문구는 잘리지 않고 **그대로 DB에 남았습니다**. 제목은 문서 목록과 제목으로 만드는 검색 색인, 워크스페이스 ZIP의 안내 파일 `목록.md`에 그대로 나갔고, 첨부 이름은 첨부 목록과 내려받기 헤더에 실려 나갔습니다.

이제 여섯 자리 모두 **문구 없이 자르기만 하는 절단기**를 씁니다. 240이라는 길이는 바뀌지 않았습니다.

| 경로 | 저장되는 값 | 이전 | 지금 |
| --- | --- | --- | --- |
| `POST /api/v1/import` | `documents.title` | 240룬 + 줄바꿈 + 문구 | 240룬 |
| `POST /api/v1/documents/{id}/import` | 편집기가 제목 칸에 넣는 값 | 240룬 + 줄바꿈 + 문구 | 240룬 |
| `POST /api/v1/documents/{id}/attachments` | `attachments.name` | 240룬 + 줄바꿈 + 문구 | 240룬 |
| 가져오기가 만드는 첨부 | `attachments.name` | 240룬 + 줄바꿈 + 문구 | 240룬 |
| 문서 넘겨받기 | `documents.title` | 240룬 + 줄바꿈 + 문구 | 240룬 |

`attachments.name`을 쓰는 세 자리를 함께 바꿨습니다. 한 자리만 고치면 같은 이름의 같은 파일이 **올린 경로에 따라 다르게 저장**됩니다. 제목을 만드는 두 경로도 마찬가지로, 240룬을 넘는 같은 제목을 새 문서로 가져오는 것과 열어 둔 문서로 가져오는 것이 이제 같은 제목을 만듭니다.

내려받기 헤더는 깨지지 않았습니다. `filename*`을 만드는 인코딩이 줄바꿈을 `%0A`로 감싸므로 헤더 자체는 파싱되고, **받는 쪽 파서가 되돌려주는 파일 이름에 문구와 줄바꿈이 그대로 실려 나왔습니다**. 즉 증상은 저장된 이름과 사람이 내려받는 파일 이름 양쪽이며, 전송 실패는 아닙니다.

AI 호출부의 절단, 머리말·꼬리말 200룬 절단, 넘겨받기 안내 문장은 그대로 두었습니다 — 그 자리에서는 문구가 의도된 동작입니다.

### 남아 있는 것

**이미 저장된 제목과 첨부 이름은 자동으로 바뀌지 않습니다.** 이번 변경은 새로 저장되는 값에만 적용되며, 마이그레이션은 없습니다. 기존 값을 정리해야 하면 해당 문서의 제목을 다시 저장하거나 첨부를 다시 올리세요.

240룬을 넘는 제목을 마크다운으로 가져올 때 **본문 첫 제목(H1)이 그대로 남는 것**은 이번 수정이 고치지 않았습니다. 본문 H1 제거는 확정된 제목과 정확히 일치할 때만 일어나고, 241룬 제목은 240룬으로 잘려 일치하지 않습니다. 수정 전에도 문구 때문에 같은 불일치였으므로 **동작은 달라지지 않았습니다**.

## 검증

- 전용 PostgreSQL 16에서 `go test -count=1 -json ./...`: 하위 테스트 포함 713 PASS, 외부 코퍼스·수동 출력 4 SKIP. HTTP API 테스트는 SKIP 없이 통과했습니다.
- 실제 라우트로 표-주도 검증을 추가했습니다: 241룬 파일 이름을 올려 저장된 `attachments.name`을 읽고, 각 행마다 `GET /api/v1/attachments/{id}`의 `Content-Disposition`을 프로덕션과 같은 파서로 되읽어 저장값과 일치하는지 확인합니다. 241룬 제목 가져오기도 같은 방식으로 확인합니다.
- `make test`: Go 테스트와 프런트 42파일·297건 통과.
- `go vet ./...`, Go 포맷 검사, placeholder 검사, `git diff --check` 통과.
- `npm run build`, `CGO_ENABLED=0 go build -trimpath`, Docker 이미지 빌드 통과.
- 브라우저 E2E와 실제 운영 배포는 이번 릴리즈 준비에서 실행하지 않았습니다. 배포용 archive는 태그 푸시 워크플로가 생성합니다.

## 업그레이드

기존 DB와 환경변수를 그대로 사용합니다. 이미지를 불러온 뒤 사용하는 Compose 파일의 `image`를 `muni:v0.51.0`으로 지정하세요. 저장소의 예제는 `muni:v0.1.0`을 가리키므로 운영 버전에 맞춰 변경해야 합니다.

```bash
gzip -dc muni-v0.51.0.tar.gz | docker load
# compose.example.yaml의 image를 muni:v0.51.0으로 지정합니다.
docker compose -f compose.example.yaml --env-file .env up -d
```

배포 담당자는 먼저 검증 환경에서 `/healthz`·`/readyz`와 문서 가져오기·첨부 올리기를 확인한 뒤 운영에 적용하세요. 두 상태 점검이 실패하거나 240룬을 넘는 이름·제목이 위 표와 다르게 저장되면 배포를 중단하고, 보관한 이전 이미지 태그로 되돌려 같은 Compose 명령을 실행합니다. 이번 변경에 따른 DB 되돌리기는 필요하지 않습니다.

배포 후 첫 주에는 첨부 이름과 문서 제목 관련 문의를 확인합니다. 확인 기준은 새로 저장되는 제목·첨부 이름에 생략 문구가 0건이고, 240룬 이하의 이름은 이전과 똑같이 저장되는 것입니다. 실제 운영 적용과 사후 관찰은 배포 담당자가 수행합니다.

## 오프라인 설치

워크플로가 게시할 릴리스 asset에는 `muni:v0.51.0` Docker 이미지가 포함됩니다.

```bash
gzip -dc muni-v0.51.0.tar.gz | docker load
docker image inspect muni:v0.51.0

# compose.example.yaml의 image를 muni:v0.51.0으로 지정합니다.
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

- 파일: `muni-v0.51.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.51.0`

```bash
sha256sum muni-v0.51.0.tar.gz
```

태그 푸시 후 GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 수행하고 이 asset을 게시합니다. 게시 완료 여부는 해당 워크플로에서 확인하세요.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [사용자 가이드](https://github.com/hkjang/muni/blob/v0.51.0/docs/USER_GUIDE.md)
- [관리자 가이드](https://github.com/hkjang/muni/blob/v0.51.0/docs/ADMIN_GUIDE.md)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.51.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.51.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.51.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.50.0...v0.51.0)
