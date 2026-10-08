# 릴리즈 검증 결과

- 버전/제목/주석 태그: 1.2.36 / ai-admin v1.2.36 / v1.2.36
- 커밋: dc0f868cff028b36207a3a0687fb145a236fc075 (detached HEAD, 작성자/태거 hkjang, 트레일러 없음)
- 변경 범위: 기존 릴리즈와 동일한 18개 버전 메타데이터·CHANGELOG 파일. 기능 변경 없음.
- Go 1.26.6: 전체 race 테스트 PASS, 서버 167.650초, 실제 Keycloak E2E 포함, SKIP 없음. go build PASS.
- Node 26: npm ci / 18파일 81테스트 / TypeScript·Vite 빌드 PASS.
- make lint / verify-version / git diff --check / Pages 44개 스크린샷 참조 검사 PASS.
- package-offline.sh / verify-offline.sh PASS. 아카이브 11,218,238 bytes, linux/amd64, OCI version v1.2.36, OCI revision dc0f868cff02 일치, SHA256SUMS 검증 PASS.
- 검증용 로컬 패키지와 체크섬은 지정된 assets/에 보존. release.yml이 태그 푸시로 패키지·체크섬과 GitHub Release를 자동 생성하므로 release.json은 지시대로 github_release=false, assets=[]이다. 로컬 검증 산출물은 수동 업로드 대상으로 전달하지 않는다.
- notes_file은 기존 GitHub 자동 생성 본문 형식에 맞춘 release-notes.md이다.
- 전용 PostgreSQL·Keycloak·웹 테스트 컨테이너 제거 완료. 작업 트리 깨끗함.
- 원격 push·Release 생성/업로드·패키지 배포·운영 배포는 수행하지 않았다. 후속 게시 작업은 외부 러너와 기존 워크플로가 수행한다.
