## AI Proxy Gateway v0.82.2

- Source commit: [`4e469a877ad6c9f8d68e53157f59b3b3220f6994`](https://github.com/hkjang/vibe-coders/commit/4e469a877ad6c9f8d68e53157f59b3b3220f6994)
- Source tag: [`v0.82.2`](https://github.com/hkjang/vibe-coders/tree/v0.82.2)

### 주요 변경 사항
- **모델 단가 음수 차단·구조화 응답 토큰 집계 정정 (v0.82.2)**: 운영자가 단가를 넣는 두 경로가 모두 음수 KRW를 그대로 받아들여 해당 모델의 비용이 음수로 계산되던 문제를 수정. `POST /admin/pricing`은 필드 존재 여부만 검사했고 `MODEL_PRICING_KRW_PER_1M`은 범위 검사 없이 적재되었는데, 음수 비용은 `cost > limit` 형태인 키별 예산·비용 가드를 무조건 통과시키고 일별·월별 쿼터 누적에서 요청분을 더하는 대신 빼므로 강제 자체가 무력화된다. 관리 API는 입력·출력·캐시 단가의 음수를 400으로 거부하고 잘못된 가격 버전을 기록하지 않으며, 환경변수 단가 맵에 음수가 있으면 기동을 차단. 과금하지 않는 모델을 표기하는 0은 계속 허용. 응답 분석기가 문자열 content만 평탄화해 `[{"type":"text","text":"..."}]` 구조화 content로 답하는 공급자의 응답에서 completion 텍스트가 비어, usage 블록이 없는 요청이 completion 토큰 0·비용 0으로 기록되던 문제를 함께 수정하고, 텍스트가 없는 파트는 원문 JSON 대신 아무것도 더하지 않아 추정치가 부풀지 않도록 유지. 스트리밍 tool call 이름을 누적 map 순회로 내보내 `tool_invocations` 행 순서가 실행마다 달라지던 문제도 `delta.tool_calls`의 index 순 정렬로 안정화

### 배포 파일
| 파일 | 설명 |
|------|------|
| ai-coding-proxy-gateway-v0.82.2.tar.gz | Docker 이미지 패키지 (linux/amd64) |
| ai-coding-proxy-gateway-v0.82.2.tar.gz.sha256 | SHA256 체크섬 |
| README-offline-v0.82.2.md | 오프라인 배포 가이드 |
| SBOM-v0.82.2.spdx.json | Go+npm 통합 SPDX SBOM |
| THIRD_PARTY_LICENSES-v0.82.2.md | Go·Frontend 제3자 라이선스 목록 |
| init-deployment-env-v0.82.2.sh | 운영 env 원자 생성·검증 helper |
| backup-volume-v0.82.2.sh | named volume·env 백업·복구 helper |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c ai-coding-proxy-gateway-v0.82.2.tar.gz | docker load

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
    echo 'GATEWAY_VERSION=v0.82.2'
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
  grep -qxF 'GATEWAY_VERSION=v0.82.2' "$ENV_FILE" || 
  { echo "gateway.env의 필수 비밀값이 유효하지 않습니다." >&2; exit 1; }
docker volume create proxy-gateway-data >/dev/null || exit 1

docker run -d --name proxy-gateway --restart=always \
  -p 8080:8080 \
  --mount source=proxy-gateway-data,target=/data \
  --env-file "$ENV_FILE" \
  ai-coding-proxy-gateway:v0.82.2
```

- Legacy Stable Console: http://localhost:8080/admin
- Next Console Preview: http://localhost:8080/app/ (/app is OFF unless UI_APP_ENABLED=true)
- React assets are embedded in the Go binary; the runtime image does not contain Node.js.
- Reuse and back up the same 0600 env file; rotating GATEWAY_SECRET without migration makes stored Provider Secrets unreadable.
- Transfer, checksum, and chmod 0700 the bundled init-deployment-env-v0.82.2.sh and backup-volume-v0.82.2.sh helpers.
- The init helper preserves secrets and atomically updates only GATEWAY_VERSION during upgrades.
