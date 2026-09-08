---
title: "작업함 — 사람 판단 필요"
description: "사람이 판단해야 할 항목 1건 — 열린 PR(리뷰 보류·보호 파일·CI 실패·승인 대기), 배포 복구, 수정 과제. 각 항목에 변경 요약·실패 근거·권장 조치가 붙어 있다."
last_modified_at: 2026-09-08 21:20:04 +0900
type: report
---
{% raw %}
# 작업함 — 사람 판단 필요

<p class="tldr"><strong>요약.</strong> 사람이 판단해야 할 항목 1건 — 열린 PR(리뷰 보류·보호 파일·CI 실패·승인 대기), 배포 복구, 수정 과제. 각 항목에 변경 요약·실패 근거·권장 조치가 붙어 있다.</p>

<div class="alerts" role="alert"><strong>⛔ 긴급 중지 중</strong><ul><li><code>aiportal-java</code> — 2026-09-08T19:18:43+09:00 by hkjang: JDK 21(javac) 미설치로 gradle 검증 불가 — sudo apt install -y openjdk-21-jdk-headless 후 해제</li></ul><p class="meta">해제: <code>bin/stop.sh &lt;범위&gt; off</code> 또는 <code>stop:</code> 이슈 닫기</p></div>

## 승인 방법

- **승인**: PR 에 라벨 `aidev-approved` → 다음 회차 시작 시 러너가 승인 당시 커밋(SHA)을 기록하고, CI 성공을 확인한 뒤 **그 커밋에만** 머지합니다. 승인 뒤 커밋이 바뀌면 라벨을 떼고 다시 물어봅니다. 승인은 정책 버전과 함께 `state/approvals.jsonl` 에 남습니다.
- **반려**: 라벨 `aidev-rejected` → PR 을 닫고 교훈으로 기록합니다.
- **재실행**: [수동 작업 요청 이슈](https://github.com/hkjang/aidev/issues/new?template=run.yml)로 문제·수용 기준·금지 범위를 적어 요청합니다.
- **긴급 중지**: `bin/stop.sh all|merge|release|<프로젝트> on "사유"` 또는 라벨 `stop`, 제목 `stop: <범위>` 이슈.

<div class="table-wrap"><table class="rt"><thead><tr><th>종류</th><th class="primary">프로젝트</th><th>항목</th><th>변경 요약</th><th>근거</th><th>권장 조치</th></tr></thead><tbody><tr data-status="failed"><td data-label="종류">배포 복구</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a> <span class="pill pill-merged" title="자율화 단계 low-risk — 강등: 롤백 PR  (2026-09-06)">저위험 자동 병합</span> ⬇</td><td data-label="항목"><a href="https://github.com/hkjang/aidev/issues/3">⏪ 배포 복구 필요: relio (60601bd)</a></td><td data-label="변경 요약">이전 정상 릴리즈로 운영 복귀 여부 결정</td><td data-label="근거">롤백 PR 과 별개로 운영 환경 복구가 필요할 수 있음 · <a href="https://github.com/hkjang/aidev/issues/3">증거</a></td><td data-label="권장 조치">이슈 안내대로 복구 후 이슈 닫기</td></tr></tbody></table></div>

## 현재 경고

- **Clustara** — 릴리즈 자산 누락 — 이전 릴리즈엔 있던 파일이 이번엔 없음
- **Invenqor** — 릴리즈 에이전트가 결과를 남기지 못함
- **ai-admin** — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요
- **git-ctx** — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요
- **jikim** — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요
- **ai-admin** — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요
- **igame** — 최신 릴리즈 v0.7.7 자산 0개 (이전 v0.7.6: 1개)


[← 대시보드](https://hkjang.github.io/aidev/)

{% endraw %}
