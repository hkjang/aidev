---
title: "작업함 — 사람 판단 필요"
description: "사람이 판단해야 할 항목 4건 — 열린 PR(리뷰 보류·보호 파일·CI 실패·승인 대기), 배포 복구, 수정 과제. 각 항목에 변경 요약·실패 근거·권장 조치가 붙어 있다."
last_modified_at: 2026-09-13 15:04:12 +0900
type: report
---
{% raw %}
# 작업함 — 사람 판단 필요

<p class="tldr"><strong>요약.</strong> 사람이 판단해야 할 항목 4건 — 열린 PR(리뷰 보류·보호 파일·CI 실패·승인 대기), 배포 복구, 수정 과제. 각 항목에 변경 요약·실패 근거·권장 조치가 붙어 있다.</p>

<div class="alerts" role="alert"><strong>⛔ 긴급 중지 중</strong><ul><li><code>aiportal-java</code> — 2026-09-08T19:18:43+09:00 by hkjang: JDK 21(javac) 미설치로 gradle 검증 불가 — sudo apt install -y openjdk-21-jdk-headless 후 해제</li></ul><p class="meta">해제: <code>bin/stop.sh &lt;범위&gt; off</code> 또는 <code>stop:</code> 이슈 닫기</p></div>

## 승인 방법

- **승인**: PR 에 라벨 `aidev-approved` → 다음 회차 시작 시 러너가 승인 당시 커밋(SHA)을 기록하고, CI 성공을 확인한 뒤 **그 커밋에만** 머지합니다. 승인 뒤 커밋이 바뀌면 라벨을 떼고 다시 물어봅니다. 승인은 정책 버전과 함께 `state/approvals.jsonl` 에 남습니다.
- **반려**: 라벨 `aidev-rejected` → PR 을 닫고 교훈으로 기록합니다.
- **재실행**: [수동 작업 요청 이슈](https://github.com/hkjang/aidev/issues/new?template=run.yml)로 문제·수용 기준·금지 범위를 적어 요청합니다.
- **긴급 중지**: `bin/stop.sh all|merge|release|<프로젝트> on "사유"` 또는 라벨 `stop`, 제목 `stop: <범위>` 이슈.

<div class="table-wrap"><table class="rt"><thead><tr><th>종류</th><th class="primary">프로젝트</th><th>항목</th><th>변경 요약</th><th>근거</th><th>권장 조치</th></tr></thead><tbody><tr data-status="failed"><td data-label="종류">배포 복구</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a> <span class="pill pill-merged" title="자율화 단계 low-risk — 강등: 롤백 PR  (2026-09-06)">저위험 자동 병합</span> ⬇</td><td data-label="항목"><a href="https://github.com/hkjang/aidev/issues/3">⏪ 배포 복구 필요: relio (60601bd)</a></td><td data-label="변경 요약">이전 정상 릴리즈로 운영 복귀 여부 결정</td><td data-label="근거">롤백 PR 과 별개로 운영 환경 복구가 필요할 수 있음 · <a href="https://github.com/hkjang/aidev/issues/3">증거</a></td><td data-label="권장 조치">이슈 안내대로 복구 후 이슈 닫기</td></tr><tr data-status="merged"><td data-label="종류">수정 과제</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/nexabuilder/">nexabuilder</a> <span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="항목"><a href="https://hkjang.github.io/aidev/projects/nexabuilder/">Spring Boot 4.1 이전: dependabot #5 는 Gradle wrapper 8.14.3 이 들어간 뒤 다시 열어 보고, 플러그인 적용이 되면 Boot 4 의 나머지</a></td><td data-label="변경 요약">릴리즈 워크플로 2회 실패 — 다음 회차 자동 배정</td><td data-label="근거">Spring Boot 4.1 이전: dependabot #5 는 Gradle wrapper 8.14.3 이 들어간 뒤 다시 열어 보고, 플러그인 적용이 되면 Boot 4 의 나머지 변경(Spring Framework 7, jakarta 네임스페이스 정리, 프로퍼티 이름 변경)을 따라간다.</td><td data-label="권장 조치">두면 러너가 수정 회차를 돌림. 급하면 run: 이슈, 멈추려면 stop</td></tr><tr data-status="merged"><td data-label="종류">수정 과제</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/nexabuilder/">nexabuilder</a> <span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="항목"><a href="https://hkjang.github.io/aidev/projects/nexabuilder/">openpdf 2.0.3 → 3.0.5: dependabot #6 이 빌드 실패. com.lowagie → org.openpdf 패키지 이름이 바뀌므로 import 를 옮기고 PD</a></td><td data-label="변경 요약">릴리즈 워크플로 2회 실패 — 다음 회차 자동 배정</td><td data-label="근거">openpdf 2.0.3 → 3.0.5: dependabot #6 이 빌드 실패. com.lowagie → org.openpdf 패키지 이름이 바뀌므로 import 를 옮기고 PDF 출력물이 같은지 확인한다.</td><td data-label="권장 조치">두면 러너가 수정 회차를 돌림. 급하면 run: 이슈, 멈추려면 stop</td></tr><tr data-status="merged"><td data-label="종류">수정 과제</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/nexabuilder/">nexabuilder</a> <span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="항목"><a href="https://hkjang.github.io/aidev/projects/nexabuilder/">jQuery 4 이전: dependabot #8 은 templates/fragments/layout.html 이 /webjars/jquery/3.7.1/... 로 버전을 박아 두어</a></td><td data-label="변경 요약">릴리즈 워크플로 2회 실패 — 다음 회차 자동 배정</td><td data-label="근거">jQuery 4 이전: dependabot #8 은 templates/fragments/layout.html 이 /webjars/jquery/3.7.1/... 로 버전을 박아 두어 의존성만 올리면 404 가 된다. webjars-locator 로 경로에서 버전을 빼고, 548 곳의 jQuery 호출 중 4 에서 제거된 API 를 쓰는 곳을 찾아 고친 뒤 브</td><td data-label="권장 조치">두면 러너가 수정 회차를 돌림. 급하면 run: 이슈, 멈추려면 stop</td></tr></tbody></table></div>

## 현재 경고

- **nexabuilder** — 수정 과제 대기 중 — Spring Boot 4.1 이전: dependabot #5 는 Gradle wrapper 8.14.3 이 들어간 뒤 다시 열어 보고, 플러그인 적용이 되면 Boot 4 의 나머지 변경(Spring Framework 7, jakarta 네임스페이스 정리, 프로퍼티 이름 변경)을 따라간다
- **nexabuilder** — 수정 과제 대기 중 — openpdf 2.0.3 → 3.0.5: dependabot #6 이 빌드 실패. com.lowagie → org.openpdf 패키지 이름이 바뀌므로 import 를 옮기고 PDF 출력물이 같은지 확인한다.
- **nexabuilder** — 수정 과제 대기 중 — jQuery 4 이전: dependabot #8 은 templates/fragments/layout.html 이 /webjars/jquery/3.7.1/... 로 버전을 박아 두어 의존성만 올리면 404 가 된다. webjars-locator 로 경로에서 버전을 빼고, 548 곳의 jQ


[← 대시보드](https://hkjang.github.io/aidev/)

{% endraw %}
