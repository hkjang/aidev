# 원격 push 가 막혀 있습니다 — 한 번만 돌리면 풀립니다

**증상.** 회차·PR·머지·릴리즈는 정상인데 `origin/main` 이 2026-09-21 11:44 에서 멈춰 있고
로컬에만 커밋이 350건 넘게 쌓입니다. 대시보드(GitHub Pages)와 논문 데이터도 그 시점에 멈춥니다.

**원인.** 2026-09-21·22 회차가 작업 디렉터리에 남긴 빌드 산출물 두 개가 커밋에 들어갔습니다.

- `state/runs/2026-09-21-115415-appstore-improve/appstore-check` — 162MB
- `state/runs/2026-09-22-124431-jikim-improve/tools/node_modules/node/bin/node` — 121MB

GitHub 은 100MB 를 넘는 파일이 들어 있는 push 를 pre-receive 단계에서 거부합니다. 파일을
지금 지워도 **커밋 기록에 남아 있는 한** 계속 거부되므로, 기록에서 빼야 풀립니다.

## 푸는 명령 (한 번, 2~5분)

```bash
cd /mnt/c/Users/USER/projects/aidev
bin/stop.sh all on "히스토리 정리"

FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch -f --index-filter \
  'git rm -r --cached --ignore-unmatch -q \
     "state/runs/2026-09-21-115415-appstore-improve/appstore-check" \
     "state/runs/2026-09-22-124431-jikim-improve/tools/node_modules" \
   >/dev/null 2>&1 || true' \
  -- 1f6041fd1..HEAD

git push origin main && bin/stop.sh all off "정리 완료"
```

- 회차가 돌고 있지 않을 때 실행하세요 (첫 줄의 `stop.sh` 가 새 회차를 막습니다).
- 커밋 메시지는 전부 보존됩니다. 파일 내용도 위 두 개 말고는 그대로입니다.
- 되돌리려면 `git reset --hard pre-squash-2026-09-24` — 정리 전 히스토리를 태그로 박아 뒀습니다.
- push 가 성공하면 `state/.sync-blocked` 가 자동으로 지워지고, 그 뒤로는 러너가 알아서 올립니다.

## 다시 생기지 않게 한 것 (2026-09-24)

- `sync_repo` 가 커밋 전에 `state/runs` 아래 50MB 초과 파일과 `node_modules` 를 지웁니다
- `.gitignore` 에 회차 산출물 경로를 넣었습니다
- push 가 100MB 제한으로 거부되면 재시도를 멈추고, 알림에 위 명령을 그대로 실어 보냅니다
- 헬스체크가 "원격에 못 올린 커밋 N개" 를 문제로 보고합니다 (30분마다)
