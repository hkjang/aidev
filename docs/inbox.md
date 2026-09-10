---
title: "작업함 — 사람 판단 필요"
description: "사람이 판단해야 할 항목 3건 — 열린 PR(리뷰 보류·보호 파일·CI 실패·승인 대기), 배포 복구, 수정 과제. 각 항목에 변경 요약·실패 근거·권장 조치가 붙어 있다."
last_modified_at: 2026-09-10 11:56:44 +0900
type: report
---
{% raw %}
# 작업함 — 사람 판단 필요

<p class="tldr"><strong>요약.</strong> 사람이 판단해야 할 항목 3건 — 열린 PR(리뷰 보류·보호 파일·CI 실패·승인 대기), 배포 복구, 수정 과제. 각 항목에 변경 요약·실패 근거·권장 조치가 붙어 있다.</p>

<div class="alerts" role="alert"><strong>⛔ 긴급 중지 중</strong><ul><li><code>aiportal-java</code> — 2026-09-08T19:18:43+09:00 by hkjang: JDK 21(javac) 미설치로 gradle 검증 불가 — sudo apt install -y openjdk-21-jdk-headless 후 해제</li></ul><p class="meta">해제: <code>bin/stop.sh &lt;범위&gt; off</code> 또는 <code>stop:</code> 이슈 닫기</p></div>

## 승인 방법

- **승인**: PR 에 라벨 `aidev-approved` → 다음 회차 시작 시 러너가 승인 당시 커밋(SHA)을 기록하고, CI 성공을 확인한 뒤 **그 커밋에만** 머지합니다. 승인 뒤 커밋이 바뀌면 라벨을 떼고 다시 물어봅니다. 승인은 정책 버전과 함께 `state/approvals.jsonl` 에 남습니다.
- **반려**: 라벨 `aidev-rejected` → PR 을 닫고 교훈으로 기록합니다.
- **재실행**: [수동 작업 요청 이슈](https://github.com/hkjang/aidev/issues/new?template=run.yml)로 문제·수용 기준·금지 범위를 적어 요청합니다.
- **긴급 중지**: `bin/stop.sh all|merge|release|<프로젝트> on "사유"` 또는 라벨 `stop`, 제목 `stop: <범위>` 이슈.

<div class="table-wrap"><table class="rt"><thead><tr><th>종류</th><th class="primary">프로젝트</th><th>항목</th><th>변경 요약</th><th>근거</th><th>권장 조치</th></tr></thead><tbody><tr data-status="merged"><td data-label="종류">PR</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a> <span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="항목"><a href="https://github.com/hkjang/aiportal-front-admin/pull/15">auto-improve: fix(admin-v2): 세션이 만료되면 라우트 이동을 기다리지 않고 복구한다</a> <code>c1a4d7b</code></td><td data-label="변경 요약">8파일 +339/−12 · guarded files, PR open https://github.com/hkjang/aiportal-front-admin/pull/15</td><td data-label="근거">guard: held — upgrade/admin-v2/src/auth/guard.test.ts upgrade/admin-v2/src/auth/guard.ts upgra · <a href="https://github.com/hkjang/aidev/tree/main/state/runs/2026-09-10-111305-aiportal-front-admin-improve">증거</a></td><td data-label="권장 조치">승인: PR 라벨 aidev-approved / 반려: aidev-rejected — 다음 회차에 러너가 승인 당시 커밋에만 CI 확인 후 머지·닫기</td></tr><tr data-status="merged"><td data-label="종류">PR</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/Vendra/">Vendra</a> <span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="항목"><a href="https://github.com/hkjang/Vendra/pull/117">auto-improve: fix: price a bid in the currency the tender was put out in, and say which one every amount is in</a> <code>2e1abb1</code></td><td data-label="변경 요약">12파일 +687/−25 · review held, PR open https://github.com/hkjang/Vendra/pull/117</td><td data-label="근거">review: rejected — 리뷰 거절: internal/httpapi/sourcing.go:311 `in.Currency = tenderCurrency` overwrite · <a href="https://github.com/hkjang/aidev/tree/main/state/runs/2026-09-10-094115-Vendra-improve">증거</a></td><td data-label="권장 조치">승인: PR 라벨 aidev-approved / 반려: aidev-rejected — 다음 회차에 러너가 승인 당시 커밋에만 CI 확인 후 머지·닫기</td></tr><tr data-status="failed"><td data-label="종류">배포 복구</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a> <span class="pill pill-merged" title="자율화 단계 low-risk — 강등: 롤백 PR  (2026-09-06)">저위험 자동 병합</span> ⬇</td><td data-label="항목"><a href="https://github.com/hkjang/aidev/issues/3">⏪ 배포 복구 필요: relio (60601bd)</a></td><td data-label="변경 요약">이전 정상 릴리즈로 운영 복귀 여부 결정</td><td data-label="근거">롤백 PR 과 별개로 운영 환경 복구가 필요할 수 있음 · <a href="https://github.com/hkjang/aidev/issues/3">증거</a></td><td data-label="권장 조치">이슈 안내대로 복구 후 이슈 닫기</td></tr></tbody></table></div>

## 현재 경고

- **aiportal-front-admin** — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요


[← 대시보드](https://hkjang.github.io/aidev/)

{% endraw %}
