## AI Proxy Gateway v0.82.0

- Source commit: [`8d8f16f9943cbaa838b981f2f96bbac733f98970`](https://github.com/hkjang/vibe-coders/commit/8d8f16f9943cbaa838b981f2f96bbac733f98970)
- Source tag: [`v0.82.0`](https://github.com/hkjang/vibe-coders/tree/v0.82.0)

### 주요 변경 사항
- **Next Admin Console Phase 1 Provider·Model Preview (v0.82.0)**: `/app/gateway/providers`, `/app/gateway/models` 읽기 전용 Preview를 추가해 URL 필터·페이지 복원, 안정 정렬, 상세 Dialog, Loading·부분 실패·last-known-good·Request ID·Retry, 키보드/접근성, Legacy Bridge를 제공. 기존 Provider·Model 관리 API를 그대로 공동 사용하고, `/admin/models` 정규화 카탈로그와 익명 `/v1/models` 집계·호환 fallback에 Provider fan-out cache·singleflight·공유 동시성/시간/행/압축 해제 후 응답 크기 상한, bounded Provider·Agent Route projection, 안전한 부분 실패·메타데이터를 적용. Provider URL의 credential·민감 query 노출을 차단하고 Agent Route 모델 shadow/deprecation 조회를 fail-closed로 강화. `/app` redirect에서 query를 폐기하고 React bootstrap 전 query·hash·history state·`return_to`의 Secret을 제거하되 브라우저 바인딩된 Keycloak 1회 교환 코드는 유지하며, Keycloak callback의 브라우저·감사 오류는 안정된 코드로 마스킹. Provider·MCP의 내부 원본 식별자는 집계 정확도를 위해 유지하면서 외부 응답·SSE·사용자 receipt·ClickHouse fact에는 opaque ref·안전 라벨·안정 오류만 노출하고, 과거 ClickHouse retry payload는 configured-table allowlist와 크기·행·스키마 검증 후 재투영하며 잘못된 batch를 보존 격리. MCP discovery 실패는 데이터베이스 오류 문자열 대신 안정된 `mcp_discovery_failed` 코드만 반환하고, `PROXY_API_KEYS`는 `name:key:owner:team` 각 필드를 트림하며 시크릿이 빈 항목을 건너뛰어 조용한 인증 실패와 의도치 않은 키 필수 전환을 막아 `/admin`과 Gateway API의 안전한 동작을 유지

### 배포 파일
| 파일 | 설명 |
|------|------|
| ai-coding-proxy-gateway-v0.82.0.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| ai-coding-proxy-gateway-v0.82.0.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.82.0.md | 오프라인 배포 가이드 |
| SBOM-v0.82.0.spdx.json | Go+npm 통합 SPDX SBOM |
| THIRD_PARTY_LICENSES-v0.82.0.md | Go·Frontend 제3자 라이선스 목록 |
| init-deployment-env-v0.82.0.sh | 운영 env 원자 생성·검증 helper |
| backup-volume-v0.82.0.sh | named volume·env 백업·복구 helper |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c ai-coding-proxy-gateway-v0.82.0.tar.gz | docker load

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
    echo 'GATEWAY_VERSION=v0.82.0'
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
  grep -qxF 'GATEWAY_VERSION=v0.82.0' "$ENV_FILE" || 
  { echo "gateway.env의 필수 비밀값이 유효하지 않습니다." >&2; exit 1; }
docker volume create proxy-gateway-data >/dev/null || exit 1

docker run -d --name proxy-gateway --restart=always \
  -p 8080:8080 \
  --mount source=proxy-gateway-data,target=/data \
  --env-file "$ENV_FILE" \
  ai-coding-proxy-gateway:v0.82.0
```

- Legacy Stable Console: http://localhost:8080/admin
- Next Console Preview: http://localhost:8080/app/ (/app is OFF unless UI_APP_ENABLED=true)
- React assets are embedded in the Go binary; the runtime image does not contain Node.js.
- Reuse and back up the same 0600 env file; rotating GATEWAY_SECRET without migration makes stored Provider Secrets unreadable.
- Transfer, checksum, and chmod 0700 the bundled init-deployment-env-v0.82.0.sh and backup-volume-v0.82.0.sh helpers.
- The init helper preserves secrets and atomically updates only GATEWAY_VERSION during upgrades.
