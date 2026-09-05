## AI Proxy Gateway v0.83.0

- Source commit: [`45f2d11dacb089a1b0e8dc1bc745e2e7fd9293af`](https://github.com/hkjang/vibe-coders/commit/45f2d11dacb089a1b0e8dc1bc745e2e7fd9293af)
- Source tag: [`v0.83.0`](https://github.com/hkjang/vibe-coders/tree/v0.83.0)

### 주요 변경 사항
- **요청 단위 추적 탐색기·추적 조회 권한 및 성능 강화 (v0.83.0)**: `/app/observability/traces` 읽기 전용 미리보기를 추가해 같은 추적 ID로 연결된 요청을 시작 시각·지연 구간·HTTP 상태·모델·안전 공급자 표시명·토큰·비용 기준의 시간축과 표로 비교하고, URL 필터 복원·필터 결합 암호화/서명 양방향 커서·자동 갱신·마지막 정상 데이터·재시도·요청 탐색기 및 기존 화면 연결을 제공. 메뉴·버튼·상태·안내 문구는 한글을 우선하며 키보드와 스크린리더용 의미 구조, 상세 포커스 이동·복원을 유지. 신규 화면은 기존 `/admin/requests`의 `X-Vibe-UI: app` 안전 투영만 재사용해 프롬프트·응답 본문·원시 오류·사용자 에이전트·도구 인자·SQL을 브라우저로 내려보내지 않고, 원문 오류나 Text2SQL 거절 사유가 포함될 수 있는 레거시 상세 추적 API는 호출하지 않음. 안전한 요청 ID는 원문 딥링크로 유지해 `GATEWAY_SECRET` 교체에도 안정적으로 복원하고, 비공개 처리된 중복 표시 ID는 비밀 탐지 규칙과 충돌하지 않는 분절 HMAC 참조와 별도 `selected_ref` URL 공간으로 정확히 구분하며 새로고침 회귀를 검증. 정확한 `trace_id` 조회용 복합 부분 인덱스를 SQLite와 PostgreSQL에 추가하고 20만 행 자연 계획에서 범위 탐색을 검증했으며, 과대한 레거시 식별자를 인덱스에서 제외해 업그레이드 안전성을 유지. 저장된 표기 그대로의 유효한 레거시 IPv6 필터도 정규화로 놓치지 않게 했고, `X-Vibe-App-Requests-Version`을 추가해 헤더 미지정 또는 `1`은 v0.82 안전 투영의 정확한 형태를, `2`는 신규 HMAC 참조·필터 가능 여부를 반환하며 잘못되거나 중복된 버전은 거부하고 두 변형을 `Vary`와 OpenAPI에 명시. 요청 탐색기와 추적 탐색기의 최소 백엔드 계약을 v0.83.0으로 고정하며 서버가 내려주는 더 낮은 런타임 버전으로 하한이 완화되지 않게 해 롤링 배포 중에는 안전하게 기존 화면으로 전환. 기존 요청 추적과 LLM 추적 상세/목록에는 GET 전용 메서드·길이·UTF-8·경로 구분자 검증, 조회 실패 시 안정 오류, exact 팀 소유권 검사, 하위 권한 오류·대체 사유·거절 사유 감사 마스킹, 특수문자 요청 ID 링크 인코딩을 적용해 다른 팀 데이터와 내부 오류가 새지 않도록 강화. 릴리스 파이프라인은 체크섬으로 고정한 Grype 0.117.0으로 실제 Distroless 최종 이미지의 High·Critical 취약점을 패키징 전에 차단하고 `/admin` Legacy 본문도 컨테이너 smoke에서 확인. Text2SQL 검증기의 LIMIT 규칙이 중첩을 고려하지 않고 문장 전체를 훑어 서브쿼리·CTE 본문의 LIMIT이 바깥 쿼리를 대신 대답하던 문제를 수정. 상한 검사는 텍스트 순서상 첫 매치만 보던 방식을 버리고 모든 LIMIT을 `MaxLimit`과 비교해 `(SELECT ... LIMIT 5) ... LIMIT 999999` 형태가 통과하지 못하게 하고, 기본 LIMIT 주입은 괄호 밖 최상위 LIMIT 유무로만 판단해 서브쿼리에만 LIMIT이 있는 무제한 쿼리에도 기본 상한이 붙도록 했으며, 값 파싱은 개행이 섞인 `LIMIT\n999999`나 오버플로 리터럴을 0으로 읽어 상한을 무력화하지 않도록 정규식 캡처와 정수 변환으로 바꾸고 해석할 수 없는 값은 거부

### 배포 파일
| 파일 | 설명 |
|------|------|
| ai-coding-proxy-gateway-v0.83.0.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| ai-coding-proxy-gateway-v0.83.0.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.83.0.md | 오프라인 배포 가이드 |
| SBOM-v0.83.0.spdx.json | Go+npm 통합 SPDX SBOM |
| THIRD_PARTY_LICENSES-v0.83.0.md | Go·Frontend 제3자 라이선스 목록 |
| init-deployment-env-v0.83.0.sh | 운영 env 원자 생성·검증 helper |
| backup-volume-v0.83.0.sh | named volume·env 백업·복구 helper |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c ai-coding-proxy-gateway-v0.83.0.tar.gz | docker load

# 최초 1회 비밀값 파일과 nonroot 호환 named volume 준비
ENV_FILE=/opt/proxy-gateway/gateway.env
install -d -m 0700 "$(dirname "$ENV_FILE")" || exit 1
command -v openssl >/dev/null 2>&1 || { echo "openssl이 필요합니다." >&2; exit 1; }
if [ ! -e "$ENV_FILE" ]; then
  umask 077
  ADMIN_TOKEN_VALUE="$(openssl rand -hex 32)" || exit 1
  GATEWAY_SECRET_VALUE="$(openssl rand -hex 32)" || exit 1
  [ "${#ADMIN_TOKEN_VALUE}" -eq 64 ] && [ "${#GATEWAY_SECRET_VALUE}" -eq 64 ] || exit 1
  printf '%s\n' "$ADMIN_TOKEN_VALUE" | grep -Eq '^[0-9A-Fa-f]{64}$' && 
    printf '%s\n' "$GATEWAY_SECRET_VALUE" | grep -Eq '^[0-9A-Fa-f]{64}$' || exit 1
  UPSTREAM_API_KEY_VALUE="${UPSTREAM_API_KEY:-}"
  if [ -z "$UPSTREAM_API_KEY_VALUE" ]; then
    read -r -s -p 'Upstream API key: ' UPSTREAM_API_KEY_VALUE || exit 1
    echo
  fi
  case "$UPSTREAM_API_KEY_VALUE" in
    ''|*[!A-Za-z0-9._~:/+=-]*) echo 'UPSTREAM_API_KEY 형식이 안전하지 않습니다.' >&2; exit 1 ;;
  esac
  ENV_TMP="$(mktemp "${ENV_FILE}.tmp.XXXXXX")" || exit 1
  trap 'rm -f -- "$ENV_TMP"' EXIT HUP INT TERM
  {
    echo 'UPSTREAM_BASE_URL=https://api.openai.com'
    echo "UPSTREAM_API_KEY=${UPSTREAM_API_KEY_VALUE}"
    echo 'GATEWAY_VERSION=v0.83.0'
    echo "ADMIN_TOKEN=${ADMIN_TOKEN_VALUE}"
    echo "GATEWAY_SECRET=${GATEWAY_SECRET_VALUE}"
    echo 'UI_APP_ENABLED=false'
  } > "$ENV_TMP" || exit 1
  chmod 0600 "$ENV_TMP" && mv -f -- "$ENV_TMP" "$ENV_FILE" || exit 1
  trap - EXIT HUP INT TERM
fi
chmod 0600 "$ENV_FILE" || exit 1
[ "$(grep -Ec '^ADMIN_TOKEN=' "$ENV_FILE")" -eq 1 ] && grep -Eq '^ADMIN_TOKEN=[0-9A-Fa-f]{64}$' "$ENV_FILE" && 
  [ "$(grep -Ec '^GATEWAY_SECRET=' "$ENV_FILE")" -eq 1 ] && grep -Eq '^GATEWAY_SECRET=[0-9A-Fa-f]{64}$' "$ENV_FILE" && 
  [ "$(grep -Ec '^UPSTREAM_API_KEY=' "$ENV_FILE")" -eq 1 ] && grep -Eq '^UPSTREAM_API_KEY=[A-Za-z0-9._~:/+=-]+$' "$ENV_FILE" && 
  ! grep -q '^UPSTREAM_API_KEY=replace-before-start$' "$ENV_FILE" && 
  [ "$(grep -Ec '^GATEWAY_VERSION=' "$ENV_FILE")" -eq 1 ] && 
  grep -qxF 'GATEWAY_VERSION=v0.83.0' "$ENV_FILE" || 
  { echo "gateway.env의 필수 비밀값이 유효하지 않습니다." >&2; exit 1; }
docker volume create proxy-gateway-data >/dev/null || exit 1

docker run -d --name proxy-gateway --restart=always \
  -p 8080:8080 \
  --mount source=proxy-gateway-data,target=/data \
  --env-file "$ENV_FILE" \
  ai-coding-proxy-gateway:v0.83.0
```

- Legacy Stable Console: http://localhost:8080/admin
- Next Console Preview: http://localhost:8080/app/ (/app is OFF unless UI_APP_ENABLED=true)
- React assets are embedded in the Go binary; the runtime image does not contain Node.js.
- Reuse and back up the same 0600 env file; rotating GATEWAY_SECRET without migration makes stored Provider Secrets unreadable.
- Transfer, checksum, and chmod 0700 the bundled init-deployment-env-v0.83.0.sh and backup-volume-v0.83.0.sh helpers.
- The init helper preserves secrets and atomically updates only GATEWAY_VERSION during upgrades.
