## MOINA 오프라인 이미지

- 파일: `moina-v0.1.29.tar.gz`
- 이미지: `moina:v0.1.29`
- 플랫폼: `linux/amd64`
- SHA256: (태그 푸시 시 `.github/workflows/release.yml`이 이미지를 빌드해 실제 값을 기록합니다)

폐쇄망 반입 전에 위 해시와 파일 해시를 대조하고 `docker load` 하세요. 릴리스 asset은 요청 범위에 따라 서비스 이미지 tar.gz 하나만 포함합니다.

### 이번 릴리스

`v0.1.29`는 시각 회귀 베이스라인을 화면 단위로 부분 갱신할 수 있습니다. `MOINA_VISUAL_ONLY=admin-settings,login`처럼 화면 slug를 나열하면 그 화면의 4장(Light·Dark × Desktop·Mobile)만 다시 찍고, 고르지 않은 entry의 sha256은 기존 `manifest.json`에서 이어받습니다. 부분 갱신은 비교와 같은 manifest 검증(계약·Chromium 버전 일치)을 먼저 통과해야 하며 모르는 slug는 즉시 거절합니다. 같은 변수는 비교에도 적용됩니다. 서비스 코드와 이미지 내용은 `v0.1.28`과 같으며, 변경은 `e2e/visual-regression.mjs`와 `e2e/VISUAL_REGRESSION.md`뿐입니다.
