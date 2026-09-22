워크스페이스를 ZIP 으로 내보낼 때 **`.` 이나 `..` 라는 이름의 폴더가 압축 파일 밖을 가리키던 것**을 고쳤습니다. 화면은 바뀌지 않았고, 마이그레이션도 설정 변경도 없습니다.

## 고친 것

### 아카이브 밖을 가리키던 `.` · `..` 폴더

muni 의 폴더 이름은 공백만 아니면 무엇이든 받으므로 `..` 라는 폴더를 만들 수 있습니다. 워크스페이스 내보내기(`GET /api/v1/workspaces/{id}/export.zip`)는 폴더 이름을 `safeFilename` 에만 통과시킨 뒤 `path.Join` 으로 이어 붙였는데, `safeFilename` 은 경로 구분자만 지우고 `.` 과 `..` 는 그대로 돌려줍니다. 그래서 그런 폴더 하나가 ZIP 전체의 경로를 흔들었습니다.

- `..` 폴더 안의 문서는 ZIP 항목 이름이 `../문서.md` 가 되어, **순진한 압축 풀기 도구는 그 파일을 대상 폴더 밖에 씁니다.**
- 휴지통 격리가 무너졌습니다. 휴지통 문서는 `휴지통/` 아래로 모이는데 `path.Join("휴지통", "..")` 은 `.` 으로 접히므로, 버린 문서가 살아 있는 문서 옆에 섞였습니다.
- `.` 폴더는 부모로 접혀, 서로 다른 두 폴더의 문서가 한 디렉터리에 들어갔습니다.

이제 폴더 세그먼트 전용 `safeFolderSegment` 이 **`safeFilename` 결과가 정확히 `.` 이나 `..` 일 때만** 앞에 `_` 를 붙입니다(`_.`, `_..`). 그 두 이름은 경로를 읽는 모든 도구가 "이동" 으로 해석하는 이름이고, 그것만 막으면 나머지는 손댈 곳이 없습니다.

**바뀌지 않은 것:** 다른 폴더 이름은 한 바이트도 달라지지 않습니다 — 한글, 공백, 가운데나 끝에 점이 있는 이름(`v1.2`, `보고서...`)은 전과 똑같습니다. `safeFilename` 자체는 문서 하나를 내려받을 때의 파일 이름과 함께 쓰므로 건드리지 않았고, 문서 제목은 `..` 이어도 `...md` 가 되어 원래 안전했으므로 그대로입니다. 데이터베이스와 화면의 폴더 이름도 그대로입니다 — 앞에 붙는 `_` 는 내보낸 ZIP 안에서만 보입니다.

입력→기대 세그먼트 12건을 보는 표-주도 단위 테스트, 세그먼트를 `path.Join` 으로 이어 붙여도 `.`·`..` 요소가 남지 않고 휴지통 접두가 유지되는지 보는 단위 테스트, 그리고 실제 라우트로 `..`·`.`·하위 폴더를 만들어 ZIP 항목 이름을 읽는 live 테스트가 이것을 지킵니다. 세 테스트 모두 고치기 전에 먼저 실패하는 것을 확인했습니다.

## 업그레이드

마이그레이션과 설정 변경은 필요하지 않습니다. 화면은 손대지 않았고, 서버는 워크스페이스 ZIP 의 폴더 경로를 만드는 자리만 v0.42.0 과 다릅니다.

```bash
gzip -dc muni-v0.43.0.tar.gz | docker load
docker compose -f compose.example.yaml --env-file .env up -d
```

**이미 내려받은 ZIP 은 그대로입니다.** 이 고침은 내보내는 순간에만 적용되므로, `.`·`..` 폴더가 있는 워크스페이스는 다시 내보내 주세요.

## 오프라인 설치

릴리스 asset 에는 `muni:v0.43.0` Docker 이미지가 포함되어 있습니다.

```bash
gzip -dc muni-v0.43.0.tar.gz | docker load
docker image inspect muni:v0.43.0

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

- 파일: `muni-v0.43.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.43.0`

```bash
sha256sum muni-v0.43.0.tar.gz
```

GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 완료한 뒤 이 asset을 게시했습니다.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [사용자 가이드](https://github.com/hkjang/muni/blob/v0.43.0/docs/USER_GUIDE.md)
- [관리자 가이드](https://github.com/hkjang/muni/blob/v0.43.0/docs/ADMIN_GUIDE.md)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.43.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.43.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.43.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.42.0...v0.43.0)
