---
title: "교훈 — 깨졌던 변경 모음"
description: "자율 개선 에이전트의 변경이 깨뜨린 것들 4건 — 머지 뒤 CI 실패, 되돌림, 릴리즈 워크플로 반복 실패, 롤백. 다음 회차 프롬프트에 프로젝트별로 주입된다."
last_modified_at: 2026-09-07 10:22:01 +0900
type: report
---
{% raw %}
# 교훈 — 깨졌던 변경 모음

<p class="tldr"><strong>요약.</strong> 자율 개선 에이전트의 변경이 깨뜨린 것들 4건 — 머지 뒤 CI 실패, 되돌림, 릴리즈 워크플로 반복 실패, 롤백. 다음 회차 프롬프트에 프로젝트별로 주입된다.</p>

<ul class="stats"><li><b>1</b><span>release-workflow-failed</span></li><li><b>1</b><span>demoted</span></li><li><b>1</b><span>rolled-back</span></li><li><b>1</b><span>rejected-by-human</span></li></ul>

<div class="table-wrap"><table class="rt" data-filter="1"><thead><tr><th>날짜</th><th class="primary">프로젝트</th><th>종류</th><th>내용</th></tr></thead><tbody><tr data-status="failed"><td data-label="날짜">2026-09-07</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/Clustara/">Clustara</a></td><td data-label="종류">rejected-by-human</td><td data-label="내용">사람이 PR 을 반려함. 같은 접근은 피할 것. <a href="https://github.com/hkjang/clustara/pull/11">링크</a></td></tr><tr data-status="failed"><td data-label="날짜">2026-09-06</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a></td><td data-label="종류">rolled-back</td><td data-label="내용">머지 60601bd 이 릴리즈를 반복해서 깨뜨려 되돌림 PR 을 열었다. 같은 접근은 피할 것.</td></tr><tr data-status="failed"><td data-label="날짜">2026-09-06</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a></td><td data-label="종류">demoted</td><td data-label="내용">자율화 단계 release → low-risk: 롤백 PR </td></tr><tr data-status="failed"><td data-label="날짜">2026-09-06</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a></td><td data-label="종류">release-workflow-failed</td><td data-label="내용">릴리즈 워크플로가 2회 실패(Verify upgrade from previous release). 릴리즈 관련 검증은 머지 전에 로컬에서 재현할 것.</td></tr></tbody></table></div>


[← 대시보드](https://hkjang.github.io/aidev/)

{% endraw %}
