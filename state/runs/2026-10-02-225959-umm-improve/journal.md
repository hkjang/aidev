# 회차 노트 2026-10-02-225959-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:00] base pinned — main@715fd97
- [러너 23:00] autonomy release — 

## 구현 노트
- 지정된 실패 `npm run lint` (exit 1) 는 저장소 루트에 package.json 이 없어 npm 이 체크아웃 밖(`/home/hkjang/package.json`)을 읽은 것. 루트에 private package.json + 의존성 0개 lockfile 을 두어 npm 을 저장소에 못 박고 CI 의 web 스크립트를 `npm --prefix web` 로 전달. 커밋 88cdca1, 새 파일 2개(프로덕션 Go/TS 코드 0줄).
- 확신 없는 곳: ① 루트 package.json 이 **없는 것이 설계**라고 볼 사람이 있다(Go 저장소, `make lint` 가 정본). 다만 바깥 파일이 같은 이름 스크립트를 가지면 그것이 실행된다는 점은 실측으로 증명했다(`npm pkg get dependencies` → @hkjang/openpro, playwright). ② npm 의 PATH 조상 탐색은 이 변경으로 바뀌지 않는다 — `node` 가림은 그대로다. ③ 루트 lockfile 을 함께 둔 판단은 `npm ci`/`npm audit` 가 루트에서 홈 디렉터리를 건드리지 않게 하려는 것이고, 꼭 필요하진 않다(package.json 만으로도 탈출은 막힌다).
- 일부러 하지 않은 것: 조상 node v20 가림(`npm test`·`test:offline-queue` 가 죽음) — `web/` 안에서도 **같은** 오류라 이번 변경과 무관하고, 앞 회차의 고침(2d8ad12)이 이 base 에 없어 같은 접근 재제출이 운영자 규칙 1번에 걸릴 수 있다. 워크플로·Makefile·Dockerfile·README 도 손대지 않았다(README/docs 에 npm 명령 문구가 아예 없음).
- 다음 역할이 조심할 것: 루트 `npm test`·`npm run verify:pwa` 는 이 기계에서 실패한다 — 전자는 위의 node 20 가림, 후자는 `web/dist/asset-manifest.json` 이 없어서(CI 는 build 뒤에 돌린다). 둘 다 `web/` 에서 돌려도 똑같이 실패한다. Go DB 통합 시험은 POSTGRES_DSN 없이 SKIP 됐다(이번에 DB 컨테이너를 띄우지 않았음). 릴리즈 경로는 release.yml 과 같은 형태의 `docker build` 로 실제 확인했다(EXIT=0, 이미지는 삭제).
- [러너 23:09] verify failed — 실패한 검증: npm test --silent (exit 1)
