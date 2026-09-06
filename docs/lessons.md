---
title: "교훈 — 깨졌던 변경 모음"
description: "자율 개선 에이전트의 변경이 깨뜨린 것들 1건 — 머지 뒤 CI 실패, 되돌림, 릴리즈 워크플로 반복 실패, 롤백. 다음 회차 프롬프트에 프로젝트별로 주입된다."
last_modified_at: 2026-09-06 19:02:11 +0900
type: report
---
{% raw %}
# 교훈 — 깨졌던 변경 모음

<p class="tldr"><strong>요약.</strong> 자율 개선 에이전트의 변경이 깨뜨린 것들 1건 — 머지 뒤 CI 실패, 되돌림, 릴리즈 워크플로 반복 실패, 롤백. 다음 회차 프롬프트에 프로젝트별로 주입된다.</p>

<ul class="stats"><li><b>1</b><span>release-workflow-failed</span></li></ul>

<div class="table-wrap"><table class="rt" data-filter="1"><thead><tr><th>날짜</th><th class="primary">프로젝트</th><th>종류</th><th>내용</th></tr></thead><tbody><tr data-status="failed"><td data-label="날짜">2026-09-06</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a></td><td data-label="종류">release-workflow-failed</td><td data-label="내용">릴리즈 워크플로가 2회 실패(Verify upgrade from previous release). 릴리즈 관련 검증은 머지 전에 로컬에서 재현할 것.</td></tr></tbody></table></div>


[← 대시보드](https://hkjang.github.io/aidev/)

{% endraw %}
