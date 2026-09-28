## AppStore v2.11.10

앱 카탈로그의 2페이지 이후에서 그리드·목록 보기를 바꾸면 1페이지로 돌아가던 문제를 고친 patch 릴리스입니다.

- 보기 전환 시 URL의 page 값을 유지해 현재 페이지와 검색·카테고리·정렬 상태가 보존됩니다.
- 실제 Provider·라우터·API를 경유하는 Vitest 5건과 desktop/mobile 브라우저 회귀 검증을 추가했습니다.
- 서버와 schema 변경은 없습니다. 기존 설치는 image만 교체하면 됩니다.

Offline-loadable Linux/AMD64 service image: `appstore:v2.11.10`.

태그 워크플로가 `appstore-v2.11.10.tar.gz`를 빌드·load·smoke 검증하고 SHA-256을 포함한 기존 영문 형식의 GitHub Release 본문과 함께 게시합니다. 이 파일은 로컬 릴리즈 변경 기록이며, 자산 해시는 CI에서 생성됩니다.

```bash
gzip -dc appstore-v2.11.10.tar.gz | docker load
docker image inspect appstore:v2.11.10
```

Only the AppStore service image is uploaded as a custom asset. PostgreSQL, Keycloak, and AI provider images are not bundled. GitHub adds its standard source-code links automatically.
