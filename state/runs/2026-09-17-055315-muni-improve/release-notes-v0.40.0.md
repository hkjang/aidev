편집기를 **새로고침하거나 문서 링크로 바로 열면** 「문제가 생겼습니다」로 깨지던 것을 고쳤습니다. 목록에서 눌러 들어가면 멀쩡했고 주소창으로 들어가면 깨졌던, 여섯 릴리스를 살아남은 결함입니다. 서버는 바뀌지 않았습니다.

## 고친 것

### 새로고침·링크로 바로 열기·넘겨받은 문서의 착지

편집기를 **하드 로드**하면 — 새로고침, 문서 링크를 새 탭에서 열기, v0.39.0 의 문서 넘기기로 넘겨받은 문서가 `/handoff` 에서 `/docs/{id}` 로 착지하는 것 — 문서 대신 「문제가 생겼습니다」가 먼저 나왔습니다. 콘솔에는 `TypeError: Cannot read properties of null (reading 'commands')`, Tiptap `Editor` 의 `get commands` 안입니다.

편집기의 `commandManager` 가 null 인 것은 **destroy 된 편집기**뿐입니다. `useEditor` 는 편집기를 렌더 도중에 만들고, 1ms 안에 페이지가 마운트되지 않으면 마운트를 놓친 렌더의 뒤처리로 그것을 스스로 지웁니다. 목록에서 눌러 들어가는 SPA 이동은 동기 갱신이라 마운트 효과가 곧바로 흘러 그 1ms 안에 들지만, 하드 로드는 lazy 청크의 재시도 렌더라 효과가 다음 태스크로 미뤄지고 그 사이에 창이 닫힙니다. 첫 커밋의 효과는 그 죽은 편집기를 들고 돌며 제목 번호를 매기는 `editor.commands.setHeadingNumbering(...)` 이 거기서 터졌습니다.

이제 `immediatelyRender: false` 로 편집기를 **마운트 효과에서** 만듭니다. 그 창 자체가 없어지고, 편집기 없는 첫 렌더는 이미 로딩 화면이 덮고 있어 눈에 보이는 것은 같습니다.

### 문서에서 문서로 옮기기

같은 자리를 보다가 둘째 경로를 찾았습니다. 편집기 안에서 다른 문서로 옮기면 — 「문서 복제」가 그렇습니다 — `useEditor` 가 옛 편집기를 지우고 새것을 만드는데, 그 커밋의 효과는 아직 옛것을 들고 있습니다. 제목 번호 매김은 문서와 함께 바뀌므로 그 효과가 돌고, 같은 오류가 났습니다. 편집기를 쓰는 효과들이 `isDestroyed` 를 보고 물러나며, 새 편집기는 다음 렌더에 와서 그때 번호 매김을 받습니다.

실제로 띄워 확인했습니다 — 빌드한 바이너리 + PostgreSQL + headless Chromium 으로 하드 로드 8/8 이 같은 TypeError 로 깨지는 것을 먼저 보고, 고친 뒤 0/8. `immediatelyRender` 만 고친 빌드에서는 「문서 복제」가 같은 오류로 깨지는 것을 보고 둘째 고침을 넣어 둘 다 통과합니다. 고친 뒤 제목 번호 40개, 입력·자동 저장·새로고침 뒤 보존, 콘솔 오류 0 입니다. 테스트는 EditorPage 를 jsdom 에 올려(인증·협업 훅·fetch 를 대신함) 바로 연 문서에 번호가 붙는 것과, 문서에서 문서로 옮겨도 편집기가 남는 것을 봅니다 — 둘째는 고치기 전에 0.2초 만에 같은 TypeError 로 실패합니다.

## 업그레이드

마이그레이션과 설정 변경은 필요하지 않습니다. Go 쪽은 손대지 않았으므로 서버의 동작은 v0.39.0 과 같고, 화면은 새로고침해도 깨지지 않는 것만 다릅니다.

```bash
gzip -dc muni-v0.40.0.tar.gz | docker load
docker compose -f compose.example.yaml --env-file .env up -d
```

**v0.39.0 에서 문서 넘기기를 켜셨다면 올려 주세요.** 넘겨받은 문서는 언제나 하드 로드로 착지하므로, 이 결함이 그 기능을 받는 쪽에서 매번 「문제가 생겼습니다」로 보이게 했습니다.

## 오프라인 설치

릴리스 asset 에는 `muni:v0.40.0` Docker 이미지가 포함되어 있습니다.

```bash
gzip -dc muni-v0.40.0.tar.gz | docker load
docker image inspect muni:v0.40.0

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
- OpenAPI: `/api/openapi.yaml` — 실제 라우트와 일치하며, 테스트가 그것을 지킵니다
- Prometheus: `/metrics` — 관리자 인증 필요
- MCP: `/mcp`
- Momento 프록시: `/momento/*` — 방문 추적을 Momento 로 켜 둔 동안에만 답합니다
- 지원 DB: PostgreSQL 15 이상
- 이미지에는 PDF Export용 Chromium과 Noto CJK 글꼴이 포함되어 있습니다.

## 릴리스 파일 검증

- 파일: `muni-v0.40.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.40.0`

```bash
sha256sum muni-v0.40.0.tar.gz
```

GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 완료한 뒤 이 asset을 게시했습니다.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [사용자 가이드](https://github.com/hkjang/muni/blob/v0.40.0/docs/USER_GUIDE.md)
- [관리자 가이드](https://github.com/hkjang/muni/blob/v0.40.0/docs/ADMIN_GUIDE.md)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.40.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.40.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.40.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.39.0...v0.40.0)
