## AppStore v2.11.4

SSO 전용·Bootstrap 전용·병행 설치의 로그인 UI 회귀 검증을 보강한 patch 릴리스입니다.

- 실제 AuthProvider·API를 경유하는 Vitest 5건과 production bundle E2E 2건을 추가했습니다. 세션과 설정 조회가 끝난 뒤 사용 가능한 로그인 방식과 복구 폼을 검증합니다.
- 구현 단계에서 로그인 분기를 훼손한 mutation 3종에 대해 새 테스트가 실패하고 원본 복원 뒤 통과함을 확인했습니다. 정상 동작의 테스트 보강이며 제품 버그 수정으로 주장하지 않습니다.
- 제품 코드와 데이터베이스 schema 변경은 없습니다. 기존 설치의 로그인 동작은 v2.11.3과 같습니다.
- 버전 표기와 두 가이드 PDF를 갱신했습니다. 화면 변경이 없어 기존 캡처와 SHA-256을 유지하고 manifest 버전만 갱신했습니다.

Offline-loadable Linux/AMD64 service image: `appstore:v2.11.4`.

```bash
gzip -dc appstore-v2.11.4.tar.gz | docker load
docker image inspect appstore:v2.11.4
```

Only the AppStore service image is uploaded as a custom asset. PostgreSQL, Keycloak, and AI provider images are not bundled. GitHub adds its standard source-code links automatically.

태그 푸시 후 release.yml이 아카이브를 빌드·검증하고 실제 SHA-256이 포함된 GitHub Release 본문을 생성합니다. 이 로컬 준비 단계에서는 원격 게시를 수행하지 않았습니다.

검증: 실제 PostgreSQL 16 통합 테스트를 포함한 Go race 테스트, vet·gofmt·build, Vitest 65건, lint·Prettier·production build, 오프라인·환경·문서 검사 통과. Chromium 설치 후 전체 desktop/mobile E2E 71 passed/1 skipped, retry 없음. 실제 Keycloak 및 로컬 Docker image smoke는 수행하지 않았습니다.
