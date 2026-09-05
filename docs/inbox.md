---
title: "작업함 — 사람 판단 필요"
description: "사람이 판단해야 할 항목 5건 — 열린 PR(리뷰 보류·보호 파일·CI 실패·승인 대기), 배포 복구, 수정 과제. 각 항목에 변경 요약·실패 근거·권장 조치가 붙어 있다."
last_modified_at: 2026-09-06 02:19:34 +0900
type: report
---
{% raw %}
# 작업함 — 사람 판단 필요

<p class="tldr"><strong>요약.</strong> 사람이 판단해야 할 항목 5건 — 열린 PR(리뷰 보류·보호 파일·CI 실패·승인 대기), 배포 복구, 수정 과제. 각 항목에 변경 요약·실패 근거·권장 조치가 붙어 있다.</p>

## 승인 방법

- **승인**: PR 에 라벨 `aidev-approved` → 다음 회차 시작 시 러너가 승인 당시 커밋(SHA)을 기록하고, CI 성공을 확인한 뒤 **그 커밋에만** 머지합니다. 승인 뒤 커밋이 바뀌면 라벨을 떼고 다시 물어봅니다. 승인은 정책 버전과 함께 `state/approvals.jsonl` 에 남습니다.
- **반려**: 라벨 `aidev-rejected` → PR 을 닫고 교훈으로 기록합니다.
- **재실행**: [수동 작업 요청 이슈](https://github.com/hkjang/aidev/issues/new?template=run.yml)로 문제·수용 기준·금지 범위를 적어 요청합니다.
- **긴급 중지**: `bin/stop.sh all|merge|release|<프로젝트> on "사유"` 또는 라벨 `stop`, 제목 `stop: <범위>` 이슈.

<div class="table-wrap"><table class="rt"><thead><tr><th>종류</th><th class="primary">프로젝트</th><th>항목</th><th>변경 요약</th><th>근거</th><th>권장 조치</th></tr></thead><tbody><tr data-status="merged"><td data-label="종류">PR</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a> <span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="항목"><a href="https://github.com/hkjang/aiportal-front/pull/5">auto-improve: fix: 마크다운 렌더링 코드블록/표 래퍼 속성 유실 및 외부 링크 하드닝</a> <code>7ff4f45</code></td><td data-label="변경 요약">2파일 +185/−8 · CI no-ci, PR open https://github.com/hkjang/aiportal-front/pull/5</td><td data-label="근거">review: approved — 리뷰 승인 (risk=low); ci: no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단) · <a href="https://github.com/hkjang/aidev/tree/main/state/runs/2026-09-06-015043-aiportal-front-improve">증거</a></td><td data-label="권장 조치">승인: PR 라벨 aidev-approved / 반려: aidev-rejected — 다음 회차에 러너가 승인 당시 커밋에만 CI 확인 후 머지·닫기</td></tr><tr data-status="merged"><td data-label="종류">PR</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/Quantoss/">Quantoss</a> <span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="항목"><a href="https://github.com/hkjang/Quantoss/pull/13">auto-improve: 스윙 실행기가 마지막 봉만 보고 체결·청산하던 버그 수정</a> <code>e63952b</code></td><td data-label="변경 요약">2파일 +254/−39 · CI no-ci, PR open https://github.com/hkjang/Quantoss/pull/13</td><td data-label="근거">review: approved — 리뷰 승인 (risk=medium); ci: no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단) · <a href="https://github.com/hkjang/aidev/tree/main/state/runs/2026-09-06-003042-Quantoss-improve">증거</a></td><td data-label="권장 조치">승인: PR 라벨 aidev-approved / 반려: aidev-rejected — 다음 회차에 러너가 승인 당시 커밋에만 CI 확인 후 머지·닫기</td></tr><tr data-status="merged"><td data-label="종류">PR</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/Clustara/">Clustara</a> <span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="항목"><a href="https://github.com/hkjang/clustara/pull/11">auto-improve: fix(capacity): finished pods held node capacity, cost and GPU slots (v0.9.272)</a> <code>6ce336f</code></td><td data-label="변경 요약">9파일 +337/−74 · review held, PR open https://github.com/hkjang/clustara/pull/11</td><td data-label="근거">review: rejected — 리뷰 거절: internal/analyzer/resourcesummary.go:42-47 PodResourceNumbers now applies · <a href="https://github.com/hkjang/aidev/tree/main/state/runs/2026-09-06-000042-Clustara-improve">증거</a></td><td data-label="권장 조치">승인: PR 라벨 aidev-approved / 반려: aidev-rejected — 다음 회차에 러너가 승인 당시 커밋에만 CI 확인 후 머지·닫기</td></tr><tr data-status="merged"><td data-label="종류">PR</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/Quantoss/">Quantoss</a> <span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="항목"><a href="https://github.com/hkjang/Quantoss/pull/12">auto-improve: 미국 종목 가격이 정수로 뭉개지던 표시 일괄 수정 (신호 사유·진입 이벤트·상태 표)</a> <code>212afd4</code></td><td data-label="변경 요약">12파일 +156/−15 · CI no-ci, PR open https://github.com/hkjang/Quantoss/pull/12</td><td data-label="근거">review: approved — 리뷰 승인 (risk=low); ci: no-ci — 이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단) · <a href="https://github.com/hkjang/aidev/tree/main/state/runs/2026-09-05-173039-Quantoss-improve">증거</a></td><td data-label="권장 조치">승인: PR 라벨 aidev-approved / 반려: aidev-rejected — 다음 회차에 러너가 승인 당시 커밋에만 CI 확인 후 머지·닫기</td></tr><tr data-status="merged"><td data-label="종류">PR</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/AgentHub/">AgentHub</a> <span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="항목"><a href="https://github.com/hkjang/AgentHub/pull/10">auto-improve: fix: the order the policy rules were written in never reached the Pod</a> <code>235f78b</code></td><td data-label="변경 요약">18파일 +640/−81 · guarded files, PR open https://github.com/hkjang/AgentHub/pull/10</td><td data-label="근거">guard: held — compose.yaml deploy/kubernetes/crd.yaml deploy/kubernetes/ollama-bridge.yaml  · <a href="https://github.com/hkjang/aidev/tree/main/state/runs/2026-09-05-163039-AgentHub-improve">증거</a></td><td data-label="권장 조치">승인: PR 라벨 aidev-approved / 반려: aidev-rejected — 다음 회차에 러너가 승인 당시 커밋에만 CI 확인 후 머지·닫기</td></tr></tbody></table></div>

## 현재 경고

- **moina** — 릴리즈 v0.1.21 가 GitHub 에 없음 — 워크플로 Release offline image: failure
- **relio** — 릴리즈 v1.11.16 가 GitHub 에 없음 — 워크플로 Release Offline Docker Image: failure
- **relio** — 릴리즈 v1.11.17 가 GitHub 에 없음 — 워크플로 Release Offline Docker Image: failure
- **Invenqor** — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요
- **AgentHub** — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요
- **AgentHub** — 최신 릴리즈 v0.234.0 자산 0개 (이전 v0.233.0: 10개)


[← 대시보드](https://hkjang.github.io/aidev/)

{% endraw %}
