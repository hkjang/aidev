## Data Works v0.9.57

### 주요 변경 사항
- **`npm run build` 가 추적 파일 `web/dist/.gitkeep` 을 지워 작업 트리를 더럽히고 `go build` 를 깨뜨리던 문제 수정 (v0.9.57)**: `web/vite.config.ts` 에 아무 조치가 없어 `vite build` 의 기본 `emptyOutDir` 이 `web/dist` 를 통째로 비우면서 추적 파일 `web/dist/.gitkeep` 을 삭제해 빌드마다 ` D web/dist/.gitkeep` 이 남았고(README 는 빌드 뒤 `git checkout -- web/dist/.gitkeep` 으로 수동 복구하라고 안내), 복원을 잊으면 `web/embed.go` 의 `//go:embed all:dist` 가 빈 디렉터리를 가리켜 `go build ./...` 가 실패했음. `emptyOutDir` 은 그대로 두고(끄면 stale 청크가 바이너리에 임베드됨) 인라인 플러그인 `keepDistPlaceholder`(`apply: 'build'`)를 추가해 `configResolved` 에서 `config.root`+`config.build.outDir` 로 경로를 잡고 비우기 전에 원본 내용을 읽어 두었다가 `closeBundle` 에서 같은 바이트로 다시 쓴다(outDir 에 없었으면 추적 파일과 같은 기본 문구 생성). SPA 핸들러는 dotfile 경로를 400 으로 거부하고 준비 판정은 `index.html` 존재로 하므로 Docker 빌드에서 `.gitkeep` 이 함께 임베드돼도 동작 변화가 없음(`base`·proxy·alias·`.gitignore`·embed 지시문·Dockerfile 은 손대지 않음). vitest 2건(`web/src/test/keep-dist-placeholder.test.ts`, node 환경)이 실제 `vite.config.ts` 를 `configFile` 로 넘겨 임시 root 에 실제 `vite build` 를 돌리고 파일로 확인 — placeholder 원본 바이트 복원·outDir 에 없던 경우 기본 문구 생성·stale 청크 제거(플러그인을 `plugins` 에서 빼면 2건 모두 실패). README 의 수동 복구 안내를 새 동작으로 바꾸고 `web/README.md`·`web/embed.go` 주석에 한 줄씩 적음. `dataworks:v0.9.57` 이미지를 `dataworks-v0.9.57.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.

### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.57.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.57.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.57
```
