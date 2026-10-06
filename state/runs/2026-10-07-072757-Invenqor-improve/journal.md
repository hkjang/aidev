# 회차 노트 2026-10-07-072757-Invenqor-improve — Invenqor
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:28] base pinned — main@ab7c89b
- [러너 07:28] autonomy release — 

## 구현 노트
- 두 번 실패한 검사는 저장소 워크플로가 아니라 GitHub Pages 의 pages-build-deployment 였다.
  docs/RELEASE_NOTES_v0.2.45.md:150 의 `LIKE '{%'` 가 Jekyll 의 Liquid 파싱을 세웠다. 질의를
  substr(resource_id,1,1)='{' 로 고치고(뜻 동일, 두 방언 공통), scripts/check-docs-liquid.sh 를
  ci.yml server 잡에 붙여 다음에는 CI 가 먼저 잡게 했다. 워크플로를 느슨하게 한 곳은 없다.
- 확신 없는 곳: Jekyll/Liquid 자체는 로컬에서 돌리지 못했다(ruby 없음, 암호 없는 sudo 불가).
  고친 SQL 이 Pages 에서 실제로 렌더되는 것은 머지 후 pages-build-deployment 로만 확인된다.
  가드가 '{%' 만 보는 것은 의도적이다 — '{{' 는 Liquid 가 빈 문자열로 바꿔 빌드를 세우지 않는다.
- 일부러 안 한 것: docs/SERVER_INSTALLATION.md 의 '{{.Id}}' (발행 사이트에서 빈 문자열로 렌더되지만
  빌드는 통과하고, 고치면 PDF 재생성이 따라붙어 구현 회차 범위 밖). docs/index.html 의 .md 링크 404
  의심도 미확인으로 ideas.json 에만 남겼다.
- 다음 역할 주의: 이 워크트리는 core.fileMode=false 다. 새 스크립트는 git update-index --chmod=+x
  가 필요했고(100755 확인), 빠뜨리면 CI 가 Permission denied 로 떨어진다.
- [러너 07:36] verify passed — 검증 8개 통과 (auto)
- [러너 07:36] pr created — https://github.com/hkjang/invenqor/pull/34
- [러너 07:36] guard held — .github/workflows/ci.yml 
