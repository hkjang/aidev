# fix-summary (수리 시도 2) — docs/OPERATIONS.md:79

- 비평 두 건 모두 **재현했고 맞았다**: 디렉터리 없이 `git archive HEAD | tar -x -C /tmp/pages-ws` 는 exit 2(`Cannot open: No such file or directory`), 디렉터리 재활용 시 심어둔 파일이 두 번째 추출 뒤에도 그대로 남아 구 파일이 살아남는 것을 확인했다.
- 고친 것: 추출 전 `rm -rf /tmp/pages-ws && mkdir -p /tmp/pages-ws` 한 줄 추가 + 왜 빼면 안 되는지(exit 2 / 조용한 유령 실패) 한 문단. 비평가가 비차단으로 짚은 두 가지도 같이 처리했다 — concurrency 취소와 러너 미배정을 가르는 단서(뒤 실행 존재 여부 확인), `GITHUB_TOKEN` 은 스코프 없는 토큰이면 충분(권한 있는 PAT 금지).
- **수리 중 추가로 발견해 고친 것**: 컨테이너가 `_site/` 를 root 소유로 써서, `rm -rf` 만 추가한 1차 수정본도 2회차에 `Permission denied` + exit 1 로 `&&` 사슬이 끊겼다. `docker run` 에 `--user "$(id -u):$(id -g)"` 를 더해 해결(빌드 exit 0, `_site/` 호출자 소유 확인).
- 검증(실제 실행): 문서 코드블록을 awk 로 그대로 뽑아 2회 연속 실행 → 둘 다 exit 0, Liquid/에러 0건. HEAD 에 없는 깨진 `docs/*.md` 를 심고 재실행 → 지워지고 exit 0(비평가가 말한 유령 실패가 실제로 막힘). 음성 대조로 미종결 Liquid 태그는 exit 1 확인 → exit 0 이 유의미함. 토큰은 로컬 API 스텁으로 대체(문서가 명시한 환경 한계).
- 변경은 `docs/OPERATIONS.md` 1파일 +37줄, 커밋 `94ffb94`. 미확인으로 남는 것: 실제 GitHub Pages 실행과 러너 미배정의 내부 사유, 진짜 `site.github` 값에만 의존하는 실패(스텁으로는 못 본다).
