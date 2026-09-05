---
title: "aidev 자율 개선 대시보드"
description: "Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 27회차·릴리즈 19건, 누적 250회차·릴리즈 143건, 주의 필요 15건."
last_modified_at: 2026-09-05 16:24:12 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "WebSite",
 "name": "aidev 자율 개선 대시보드",
 "url": "https://hkjang.github.io/aidev/",
 "description": "Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 27회차·릴리즈 19건, 누적 250회차·릴리즈 143건, 주의 필요 15건.",
 "inLanguage": "ko",
 "author": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-05T16:24:12+09:00"
}
</script>

<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "FAQPage",
 "mainEntity": [
  {
   "@type": "Question",
   "name": "이 페이지는 무엇인가요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "hkjang 의 GitHub 프로젝트들을 Claude Code 자율 개선 에이전트가 스스로 분석해 개선하고, 테스트를 통과시킨 뒤 PR 을 main 에 머지하고, 각 저장소의 기존 관례대로 릴리즈한 결과를 회차마다 자동으로 갱신하는 일일 보고 대시보드입니다."
   }
  },
  {
   "@type": "Question",
   "name": "얼마나 자주 갱신되나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "에이전트는 10분 간격 스케줄로 사실상 연속 실행되며, 회차가 하나 끝날 때마다 이 사이트가 다시 만들어집니다. 보통 1~2분 안에 반영됩니다. Atom 피드(/feed.xml)를 구독하면 일일 보고를 받아볼 수 있습니다."
   }
  },
  {
   "@type": "Question",
   "name": "한 회차에서 에이전트는 무엇을 하나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "저장소를 파악하고 개선 아이디어 5개를 가치·위험·작업량으로 채점해 하나를 고른 뒤 구현하고, 테스트·린트·빌드를 실제로 실행해 통과한 경우에만 커밋합니다. 러너가 PR 을 열고 CI 통과를 확인한 뒤 머지하며, 릴리즈 에이전트가 이전 릴리즈 방식(태그·버전 파일·CHANGELOG·워크플로·첨부 자산)을 확인해 같은 방식으로 다음 버전을 냅니다."
   }
  },
  {
   "@type": "Question",
   "name": "릴리즈 워크플로가 실패하면 어떻게 되나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "러너가 한 번 자동으로 재실행합니다. 그래도 실패하면 실패 단계와 로그 요지를 수정 과제 큐에 넣고, 다음 회차에서 그 프로젝트를 우선 배정해 에이전트가 원인을 고칩니다(검증을 느슨하게 만드는 것은 금지). 큐에 있는 동안 '주의 필요'에 표시됩니다."
   }
  },
  {
   "@type": "Question",
   "name": "어떤 프로젝트가 대상인가요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "최근 30일 안에 커밋이 있고, 작업트리가 깨끗하며, GitHub 원격이 있는 저장소만 후보가 됩니다. 사람이 작업 중인(미커밋 변경이 있는) 저장소는 자동으로 제외됩니다."
   }
  },
  {
   "@type": "Question",
   "name": "'머지(릴리즈 없음)'는 무슨 뜻인가요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "개선은 머지됐지만 릴리즈가 만들어지지 않은 회차입니다. 릴리즈 이력이 전혀 없는 신규 저장소(관례를 새로 정하지 않음)이거나, 릴리즈 단계가 실패·미완료된 경우입니다."
   }
  },
  {
   "@type": "Question",
   "name": "'주의 필요'에는 무엇이 뜨나요? 알림은 어디로 오나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "최근 2일 회차 중 CI 실패로 머지되지 않은 PR, 릴리즈 실패, 이전 릴리즈에는 있던 첨부 자산이 빠진 릴리즈, 수정 과제 대기처럼 사람이 확인해야 할 항목입니다. 새 경고가 생기면 GitHub Issue(라벨 alert)에 기록하고, 설정돼 있으면 Slack·이메일·Windows 알림으로도 보냅니다."
   }
  },
  {
   "@type": "Question",
   "name": "머지 전에 검토는 없나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "있습니다. 구현과 다른 세션의 리뷰 에이전트가 diff 만 읽고 '머지하면 안 되는 이유'(검증하지 않는 테스트, 논리 오류, 설명과 다른 동작, 위험한 변경)를 찾습니다. 거절하면 PR 에 사유를 달고 열어 둡니다. 또 워크플로·마이그레이션·인증·결제·배포 파일(보호 파일)을 건드린 PR 은 자동 머지하지 않습니다."
   }
  },
  {
   "@type": "Question",
   "name": "깨진 변경은 어떻게 되나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "머지 2시간 뒤부터 main CI 실패와 되돌림 커밋을 확인해 '교훈'으로 기록하고, 그 프로젝트의 다음 회차 프롬프트에 주입해 같은 실수를 피하게 합니다. 릴리즈 워크플로가 반복 실패하고 수정 회차도 실패하면 원래 머지를 되돌리는 롤백 PR 을 자동으로 엽니다(머지는 사람). 프로젝트마다 최근 14일의 실패·경고·회귀로 건강 등급 A~D 를 매깁니다."
   }
  },
  {
   "@type": "Question",
   "name": "비용은 어떻게 계산되나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "각 회차의 claude -p 세션이 보고한 추정 비용(USD)과 소요 시간·턴 수·토큰을 그대로 합산합니다. 정액제 구독에서는 실제 청구가 아닌 참고값입니다."
   }
  },
  {
   "@type": "Question",
   "name": "원본 데이터는 어디서 보나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "회차 기록은 runs.jsonl, 사용량은 usage.jsonl, 요약은 data/summary.json, 프로젝트별 원장은 GitHub 저장소 hkjang/aidev 의 state/ 폴더, 러너와 프롬프트는 같은 저장소의 bin/ 과 prompt.md 에 있습니다."
   }
  }
 ]
}
</script>

<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "ItemList",
 "name": "일일 보고",
 "itemListOrder": "Descending",
 "itemListElement": [
  {
   "@type": "ListItem",
   "position": 1,
   "name": "일일 보고 2026-09-05",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-05/"
  },
  {
   "@type": "ListItem",
   "position": 2,
   "name": "일일 보고 2026-09-04",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-04/"
  },
  {
   "@type": "ListItem",
   "position": 3,
   "name": "일일 보고 2026-09-03",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-03/"
  },
  {
   "@type": "ListItem",
   "position": 4,
   "name": "일일 보고 2026-09-02",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-02/"
  }
 ]
}
</script>

<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "Dataset",
 "name": "aidev 회차 기록 (runs.jsonl)",
 "description": "자율 개선 에이전트의 회차별 기록. 한 줄에 한 회차, 필드: ts, date, project, result.",
 "url": "https://hkjang.github.io/aidev/data/runs.jsonl",
 "license": "https://opensource.org/license/mit",
 "inLanguage": "ko",
 "creator": {
  "@type": "Person",
  "name": "hkjang"
 },
 "encodingFormat": "application/x-ndjson",
 "distribution": [
  {
   "@type": "DataDownload",
   "encodingFormat": "application/x-ndjson",
   "contentUrl": "https://hkjang.github.io/aidev/data/runs.jsonl"
  },
  {
   "@type": "DataDownload",
   "encodingFormat": "application/x-ndjson",
   "contentUrl": "https://hkjang.github.io/aidev/data/usage.jsonl"
  },
  {
   "@type": "DataDownload",
   "encodingFormat": "application/json",
   "contentUrl": "https://hkjang.github.io/aidev/data/summary.json"
  }
 ]
}
</script>

# aidev 자율 개선 대시보드

<p class="tldr"><strong>한 줄 요약.</strong> Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 27회차·릴리즈 19건, 누적 250회차·릴리즈 143건, 주의 필요 15건. 회차가 끝날 때마다 자동 갱신됩니다 (마지막 갱신 <time datetime="2026-09-05T16:24:12+09:00" data-rel>2026-09-05 16:24</time> KST).</p>

<div class="alerts" role="alert"><strong>⚠️ 주의 필요 15건</strong> <span class="meta">— 새 경고는 GitHub Issue·Slack·이메일·Windows 알림으로도 보냅니다</span><ul><li><a href="https://hkjang.github.io/aidev/projects/moina/">moina</a> — 릴리즈 v0.1.21 가 GitHub 에 없음 — 워크플로 Release offline image: failure <span class=meta>(2026-09-05 05:15)</span></li><li><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a> — 릴리즈 v1.11.16 가 GitHub 에 없음 — 워크플로 Release Offline Docker Image: failure <span class=meta>(2026-09-05 07:33)</span></li><li><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a> — 릴리즈 v1.11.17 가 GitHub 에 없음 — 워크플로 Release Offline Docker Image: failure <span class=meta>(2026-09-05 08:12)</span></li><li><a href="https://hkjang.github.io/aidev/projects/Invenqor/">Invenqor</a> — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요 <span class=meta>(2026-09-05 11:20)</span></li><li><a href="https://hkjang.github.io/aidev/projects/weekly/">weekly</a> — 릴리즈 에이전트가 결과를 남기지 못함 <span class=meta>(2026-09-04 02:53)</span></li><li><a href="https://hkjang.github.io/aidev/projects/ai-admin/">ai-admin</a> — 릴리즈 v1.2.6 자산 0개 — 워크플로 Offline release: failure <span class=meta>(2026-09-04 05:13)</span></li><li><a href="https://hkjang.github.io/aidev/projects/dataworks/">dataworks</a> — 자산 빌드 실패 <span class=meta>(2026-09-04 07:00)</span></li><li><a href="https://hkjang.github.io/aidev/projects/ptium/">ptium</a> — 자산 빌드 결과 없음 <span class=meta>(2026-09-04 07:04)</span></li><li><a href="https://hkjang.github.io/aidev/projects/Invenqor/">Invenqor</a> — 자산 빌드 결과 없음 <span class=meta>(2026-09-04 07:06)</span></li><li><a href="https://hkjang.github.io/aidev/projects/igame/">igame</a> — 릴리즈 단계 실패 <span class=meta>(2026-09-04 07:20)</span></li><li><a href="https://hkjang.github.io/aidev/projects/igame/">igame</a> — 릴리즈 v0.7.5 가 GitHub 에 없음 — 워크플로 Release offline image: failure <span class=meta>(2026-09-04 08:18)</span></li><li><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a> — 릴리즈 v1.4.0 가 GitHub 에 없음 — 워크플로 오프라인 이미지 릴리스: failure <span class=meta>(2026-09-04 08:40)</span></li><li><a href="https://hkjang.github.io/aidev/projects/moina/">moina</a> — 릴리즈 에이전트가 결과를 남기지 못함 <span class=meta>(2026-09-04 09:27)</span></li><li><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a> — 릴리즈 v1.11.15 가 GitHub 에 없음 — 워크플로 Release Offline Docker Image: failure <span class=meta>(2026-09-04 16:19)</span></li><li><a href="https://hkjang.github.io/aidev/projects/AgentHub/">AgentHub</a> — 최신 릴리즈 v0.234.0 자산 0개 (이전 v0.233.0: 10개)</li></ul></div>

<div class="alerts ok" role="status"><strong>🩺 러너 정상</strong> <span class="meta">— 마지막 회차 4시간 30분 전 · 스케줄러 준비 · 다음 실행 2026-09-05 오후 4:20:00 · 디스크 68% · 최근 7일 회귀 0건 · 점검 16:15</span></div>

[운영 문서](https://github.com/hkjang/aidev#readme) · [원장](https://github.com/hkjang/aidev/tree/main/state) · [실행 이력](https://github.com/hkjang/aidev/commits/main) · [경고 이슈](https://github.com/hkjang/aidev/issues?q=label%3Aalert) · [교훈 0건](https://hkjang.github.io/aidev/lessons/) · [Atom 피드](https://hkjang.github.io/aidev/feed.xml) · [summary.json](https://hkjang.github.io/aidev/data/summary.json)

## 오늘 (2026-09-05)

<ul class="stats"><li><b>27</b><span>회차</span></li><li><b>25</b><span>프로젝트</span></li><li><b>19</b><span>릴리즈</span></li><li><b>5</b><span>머지(릴리즈 없음)</span></li><li><b>1</b><span>변경 없음</span></li><li><b>0</b><span>실패</span></li><li><b>$23.65</b><span>비용</span></li><li><b>57분</b><span>에이전트 시간</span></li></ul>

[2026-09-05 보고 자세히 보기 →](https://hkjang.github.io/aidev/reports/2026-09-05/)

<div class="table-wrap"><table class="rt" data-filter="1"><caption class="meta">오늘 전체 회차 (KST)</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="merged"><td data-label="시각">00:10</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/8">PR #8</a>, release skipped</td></tr><tr data-status="merged"><td data-label="시각">00:31</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/4">PR #4</a>, release skipped</td></tr><tr data-status="merged"><td data-label="시각">00:51</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a></td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/9">PR #9</a>, release skipped</td></tr><tr data-status="merged"><td data-label="시각">01:10</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a></td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-py/pull/6">PR #6</a>, release skipped</td></tr><tr data-status="released"><td data-label="시각">01:43</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/appstore/">appstore</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/appstore/pull/5">PR #5</a>, released <a href="https://github.com/hkjang/appstore/releases/tag/v2.5.1">v2.5.1</a></td></tr><tr data-status="released"><td data-label="시각">02:05</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/dataworks/">dataworks</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/dataworks/pull/7">PR #7</a>, released <a href="https://github.com/hkjang/dataworks/releases/tag/v0.9.42">v0.9.42</a> +1 assets</td></tr><tr data-status="released"><td data-label="시각">02:45</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/git-ctx/">git-ctx</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/git-ctx/pull/18">PR #18</a>, released <a href="https://github.com/hkjang/git-ctx/releases/tag/v0.77.5">v0.77.5</a></td></tr><tr data-status="released"><td data-label="시각">03:23</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/igame/">igame</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/igame/pull/7">PR #7</a>, released <a href="https://github.com/hkjang/igame/releases/tag/v0.7.6">v0.7.6</a>, <strong>ASSETS MISSING</strong></td></tr><tr data-status="released"><td data-label="시각">03:52</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jikim/">jikim</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/jikim/pull/14">PR #14</a>, released <a href="https://github.com/hkjang/jikim/releases/tag/v0.2.2">v0.2.2</a></td></tr><tr data-status="released"><td data-label="시각">04:17</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/jupiq/pull/3">PR #3</a>, released <a href="https://github.com/hkjang/jupiq/releases/tag/v1.4.4">v1.4.4</a></td></tr><tr data-status="released"><td data-label="시각">04:35</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/kanpic/">kanpic</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/kanpic/pull/9">PR #9</a>, released <a href="https://github.com/hkjang/kanpic/releases/tag/v0.236.0">v0.236.0</a></td></tr><tr data-status="released"><td data-label="시각">05:15</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/moina/">moina</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/moina/pull/7">PR #7</a>, released <a href="https://github.com/hkjang/moina/releases/tag/v0.1.21">v0.1.21</a>, <strong>ASSETS MISSING</strong></td></tr><tr data-status="released"><td data-label="시각">05:43</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/muni/">muni</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/muni/pull/6">PR #6</a>, released <a href="https://github.com/hkjang/muni/releases/tag/v0.27.0">v0.27.0</a></td></tr><tr data-status="released"><td data-label="시각">06:02</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/pii-masker/">pii-masker</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/pii-masker/pull/8">PR #8</a>, released <a href="https://github.com/hkjang/pii-masker/releases/tag/v1.0.11">v1.0.11</a> +1 assets</td></tr><tr data-status="released"><td data-label="시각">06:30</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/ptium/">ptium</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/ptium/pull/9">PR #9</a>, released <a href="https://github.com/hkjang/ptium/releases/tag/v1.69.27">v1.69.27</a> +7 assets</td></tr><tr data-status="released"><td data-label="시각">06:55</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/releasedock/">releasedock</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/releasedock/pull/8">PR #8</a>, released <a href="https://github.com/hkjang/releasedock/releases/tag/v0.5.8">v0.5.8</a></td></tr><tr data-status="released"><td data-label="시각">07:33</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/relio/pull/9">PR #9</a>, released <a href="https://github.com/hkjang/relio/releases/tag/v1.11.16">v1.11.16</a>, <strong>ASSETS MISSING</strong></td></tr><tr data-status="released"><td data-label="시각">08:12</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/relio/pull/10">PR #10</a>, released <a href="https://github.com/hkjang/relio/releases/tag/v1.11.17">v1.11.17</a>, <strong>ASSETS MISSING</strong></td></tr><tr data-status="released"><td data-label="시각">08:43</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/umm/">umm</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/umm/pull/144">PR #144</a>, released <a href="https://github.com/hkjang/umm/releases/tag/v0.71.2">v0.71.2</a></td></tr><tr data-status="released"><td data-label="시각">09:12</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/vibe-coders/">vibe-coders</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/vibe-coders/pull/14">PR #14</a>, released <a href="https://github.com/hkjang/vibe-coders/releases/tag/v0.83.0">v0.83.0</a> +7 assets</td></tr><tr data-status="released"><td data-label="시각">09:39</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/visitflow/">visitflow</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/visitflow/pull/8">PR #8</a>, released <a href="https://github.com/hkjang/visitflow/releases/tag/v2.6.5">v2.6.5</a></td></tr><tr data-status="nochange"><td data-label="시각">10:00</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/weekly/">weekly</a></td><td data-label="결과"><span class="pill pill-nochange">➖ 변경 없음</span> no change</td></tr><tr data-status="released"><td data-label="시각">10:46</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/AgentHub/">AgentHub</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/AgentHub/pull/9">PR #9</a>, released <a href="https://github.com/hkjang/AgentHub/releases/tag/v0.234.0">v0.234.0</a>, <strong>ASSETS MISSING</strong></td></tr><tr data-status="released"><td data-label="시각">11:08</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/Clustara/">Clustara</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/clustara/pull/9">PR #9</a>, released <a href="https://github.com/hkjang/clustara/releases/tag/v0.9.270">v0.9.270</a> +3 assets</td></tr><tr data-status="other"><td data-label="시각">11:20</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/Invenqor/">Invenqor</a></td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/invenqor/pull/9">PR #9</a></td></tr><tr data-status="merged"><td data-label="시각">11:44</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/Quantoss/">Quantoss</a></td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/Quantoss/pull/11">PR #11</a>, release skipped</td></tr><tr data-status="other"><td data-label="시각">16:24</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/pii-masker/">pii-masker</a></td><td data-label="결과"><span class="pill pill-other">• 기타</span> CI no-ci, PR open <a href="https://github.com/hkjang/pii-masker/pull/9">PR #9</a><div class="meta">6파일 <span style="color:var(--good)">+393</span>/<span style="color:var(--bad)">−31</span> · 테스트 2 — Bound how many synchronous masks run at once</div></td></tr></tbody></table></div>

## 최근 14일

<div class="chart"><svg viewBox="0 0 640 200" role="img" aria-labelledby="chart-t chart-d"><title id="chart-t">최근 14일 회차 수</title><desc id="chart-d">날짜별 회차 수를 릴리즈·머지·변경 없음·실패로 나눠 쌓은 막대. 같은 값은 아래 표에 있다.</desc><line class="grid" x1="28" x2="636" y1="172.0" y2="172.0"/><text x="22" y="176.0" text-anchor="end">0</text><line class="grid" x1="28" x2="636" y1="155.8" y2="155.8"/><text x="22" y="159.8" text-anchor="end">10</text><line class="grid" x1="28" x2="636" y1="139.6" y2="139.6"/><text x="22" y="143.6" text-anchor="end">20</text><line class="grid" x1="28" x2="636" y1="123.4" y2="123.4"/><text x="22" y="127.4" text-anchor="end">30</text><line class="grid" x1="28" x2="636" y1="107.2" y2="107.2"/><text x="22" y="111.2" text-anchor="end">40</text><line class="grid" x1="28" x2="636" y1="91.0" y2="91.0"/><text x="22" y="95.0" text-anchor="end">50</text><line class="grid" x1="28" x2="636" y1="74.8" y2="74.8"/><text x="22" y="78.8" text-anchor="end">60</text><line class="grid" x1="28" x2="636" y1="58.6" y2="58.6"/><text x="22" y="62.6" text-anchor="end">70</text><line class="grid" x1="28" x2="636" y1="42.4" y2="42.4"/><text x="22" y="46.4" text-anchor="end">80</text><line class="grid" x1="28" x2="636" y1="26.2" y2="26.2"/><text x="22" y="30.2" text-anchor="end">90</text><line class="grid" x1="28" x2="636" y1="10.0" y2="10.0"/><text x="22" y="14.0" text-anchor="end">100</text><g><title>2026-08-23: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title></g><g><title>2026-08-24: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title><text x="92.7" y="186" text-anchor="middle">08/24</text></g><g><title>2026-08-25: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title></g><g><title>2026-08-26: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title><text x="179.0" y="186" text-anchor="middle">08/26</text></g><g><title>2026-08-27: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title></g><g><title>2026-08-28: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title><text x="265.3" y="186" text-anchor="middle">08/28</text></g><g><title>2026-08-29: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title></g><g><title>2026-08-30: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title><text x="351.6" y="186" text-anchor="middle">08/30</text></g><g><title>2026-08-31: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title></g><g><title>2026-09-01: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title><text x="437.9" y="186" text-anchor="middle">09/01</text></g><g><title>2026-09-02: 회차 50 · 릴리즈 28 · 머지 16 · 변경 없음 3 · 실패 0</title><rect x="469.0" y="126.6" width="24.0" height="43.4" fill="#0ca30c"/><rect x="469.0" y="100.7" width="24.0" height="23.9" fill="#fab219"/><path d="M469.0,100.7 V98.3 Q469.0,95.9 471.4,95.9 H490.6 Q493.0,95.9 493.0,98.3 V100.7 Z" fill="#9ca3af"/></g><g><title>2026-09-03: 회차 96 · 릴리즈 63 · 머지 25 · 변경 없음 8 · 실패 0</title><rect x="512.1" y="69.9" width="24.0" height="100.1" fill="#0ca30c"/><rect x="512.1" y="29.4" width="24.0" height="38.5" fill="#fab219"/><path d="M512.1,29.4 V20.5 Q512.1,16.5 516.1,16.5 H532.1 Q536.1,16.5 536.1,20.5 V29.4 Z" fill="#9ca3af"/><text x="524.1" y="186" text-anchor="middle">09/03</text></g><g><title>2026-09-04: 회차 77 · 릴리즈 33 · 머지 9 · 변경 없음 1 · 실패 0</title><rect x="555.3" y="118.5" width="24.0" height="51.5" fill="#0ca30c"/><rect x="555.3" y="104.0" width="24.0" height="12.6" fill="#fab219"/><path d="M555.3,104.0 V103.1 Q555.3,102.3 556.1,102.3 H578.5 Q579.3,102.3 579.3,103.1 V104.0 Z" fill="#9ca3af"/></g><g><title>2026-09-05: 회차 27 · 릴리즈 19 · 머지 5 · 변경 없음 1 · 실패 0</title><rect x="598.4" y="141.2" width="24.0" height="28.8" fill="#0ca30c"/><rect x="598.4" y="133.1" width="24.0" height="6.1" fill="#fab219"/><path d="M598.4,133.1 V132.3 Q598.4,131.5 599.2,131.5 H621.6 Q622.4,131.5 622.4,132.3 V133.1 Z" fill="#9ca3af"/><text x="610.4" y="127.5" text-anchor="middle">27</text><text x="610.4" y="186" text-anchor="middle">09/05</text></g><line class="grid" x1="28" x2="636" y1="172" y2="172"/></svg><ul class="legend" aria-hidden="true"><li><i style="background:#0ca30c"></i>🚀 릴리즈</li><li><i style="background:#fab219"></i>✅ 머지</li><li><i style="background:#9ca3af"></i>➖ 변경 없음</li><li><i style="background:#d03b3b"></i>❌ 실패</li></ul></div>

## 일일 보고

<div class="table-wrap"><table class="rt"><thead><tr><th class="primary">날짜</th><th class="num">회차</th><th class="num">릴리즈</th><th class="num">머지</th><th class="num">변경 없음</th><th class="num">실패</th><th class="num">비용</th></tr></thead><tbody><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-05/">2026-09-05</a></td><td data-label="회차" class="num">27</td><td data-label="릴리즈" class="num">19</td><td data-label="머지" class="num">5</td><td data-label="변경 없음" class="num">1</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">$23.65</td></tr><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-04/">2026-09-04</a></td><td data-label="회차" class="num">77</td><td data-label="릴리즈" class="num">33</td><td data-label="머지" class="num">9</td><td data-label="변경 없음" class="num">1</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">—</td></tr><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-03/">2026-09-03</a></td><td data-label="회차" class="num">96</td><td data-label="릴리즈" class="num">63</td><td data-label="머지" class="num">25</td><td data-label="변경 없음" class="num">8</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">—</td></tr><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-02/">2026-09-02</a></td><td data-label="회차" class="num">50</td><td data-label="릴리즈" class="num">28</td><td data-label="머지" class="num">16</td><td data-label="변경 없음" class="num">3</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">—</td></tr></tbody></table></div>

## 주간·월간 보고

### 주간

<div class="table-wrap"><table class="rt"><thead><tr><th class="primary">주</th><th class="num">활동일</th><th class="num">회차</th><th class="num">릴리즈</th><th class="num">실패</th><th class="num">비용</th><th class="num">시간</th></tr></thead><tbody><tr data-status="released"><td data-label="주" class="primary"><a href="https://hkjang.github.io/aidev/weekly/2026-W36/">2026-W36</a></td><td data-label="활동일" class="num">4</td><td data-label="회차" class="num">250</td><td data-label="릴리즈" class="num">143</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">$23.65</td><td data-label="시간" class="num">57분</td></tr></tbody></table></div>

### 월간

<div class="table-wrap"><table class="rt"><thead><tr><th class="primary">월</th><th class="num">활동일</th><th class="num">회차</th><th class="num">릴리즈</th><th class="num">실패</th><th class="num">비용</th><th class="num">시간</th></tr></thead><tbody><tr data-status="released"><td data-label="월" class="primary"><a href="https://hkjang.github.io/aidev/monthly/2026-09/">2026-09</a></td><td data-label="활동일" class="num">4</td><td data-label="회차" class="num">250</td><td data-label="릴리즈" class="num">143</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">$23.65</td><td data-label="시간" class="num">57분</td></tr></tbody></table></div>

## 프로젝트별 현황

<div class="table-wrap"><table class="rt" data-filter="1"><caption class="meta">프로젝트 이름을 누르면 원장 전체와 회차 이력을 볼 수 있습니다.</caption><thead><tr><th class="primary">프로젝트 · 건강</th><th>마지막 회차</th><th>결과</th><th>최근 릴리즈</th><th class="num">누적 비용</th></tr></thead><tbody><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/AgentHub/">AgentHub</a> <span class="pill pill-merged" title="14일: 릴리즈 9, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="마지막 회차">2026-09-05 10:46</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/AgentHub/pull/9">PR #9</a>, released <a href="https://github.com/hkjang/AgentHub/releases/tag/v0.234.0">v0.234.0</a>, <strong>ASSETS MISSING</strong></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/AgentHub/releases/tag/v0.234.0">v0.234.0</a> <span class="meta">자산 0</span> <span class="pill pill-failed">❌ 누락</span></td><td data-label="누적 비용" class="num">$4.52</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/ai-admin/">ai-admin</a> <span class="pill pill-merged" title="14일: 릴리즈 6, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="마지막 회차">2026-09-04 23:57</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/ai-admin/pull/9">PR #9</a>, released <a href="https://github.com/hkjang/ai-admin/releases/tag/v1.2.9">v1.2.9</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/ai-admin/releases/tag/v1.2.9">v1.2.9</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="merged"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span></td><td data-label="마지막 회차">2026-09-05 00:31</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/4">PR #4</a>, release skipped</td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="merged"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span></td><td data-label="마지막 회차">2026-09-05 00:10</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/8">PR #8</a>, release skipped</td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="merged"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span></td><td data-label="마지막 회차">2026-09-05 00:51</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-java/pull/9">PR #9</a>, release skipped</td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="merged"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span></td><td data-label="마지막 회차">2026-09-05 01:10</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-py/pull/6">PR #6</a>, release skipped</td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/appstore/">appstore</a> <span class="pill pill-released" title="14일: 릴리즈 5, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="마지막 회차">2026-09-05 01:43</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/appstore/pull/5">PR #5</a>, released <a href="https://github.com/hkjang/appstore/releases/tag/v2.5.1">v2.5.1</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/appstore/releases/tag/v2.5.1">v2.5.1</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/Clustara/">Clustara</a> <span class="pill pill-released" title="14일: 릴리즈 9, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="마지막 회차">2026-09-05 11:08</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/clustara/pull/9">PR #9</a>, released <a href="https://github.com/hkjang/clustara/releases/tag/v0.9.270">v0.9.270</a> +3 assets</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/clustara/releases/tag/v0.9.270">v0.9.270</a> <span class="meta">자산 3</span></td><td data-label="누적 비용" class="num">$5.09</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/dataworks/">dataworks</a> <span class="pill pill-merged" title="14일: 릴리즈 6, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="마지막 회차">2026-09-05 02:05</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/dataworks/pull/7">PR #7</a>, released <a href="https://github.com/hkjang/dataworks/releases/tag/v0.9.42">v0.9.42</a> +1 assets</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/dataworks/releases/tag/v0.9.42">v0.9.42</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/git-ctx/">git-ctx</a> <span class="pill pill-merged" title="14일: 릴리즈 3, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="마지막 회차">2026-09-05 02:45</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/git-ctx/pull/18">PR #18</a>, released <a href="https://github.com/hkjang/git-ctx/releases/tag/v0.77.5">v0.77.5</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/git-ctx/releases/tag/v0.77.5">v0.77.5</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/igame/">igame</a> <span class="pill pill-failed" title="14일: 릴리즈 5, 실패 0, 경고 4, 회귀 0">건강 D</span></td><td data-label="마지막 회차">2026-09-05 03:23</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/igame/pull/7">PR #7</a>, released <a href="https://github.com/hkjang/igame/releases/tag/v0.7.6">v0.7.6</a>, <strong>ASSETS MISSING</strong></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/igame/releases/tag/v0.7.6">v0.7.6</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/Invenqor/">Invenqor</a> <span class="pill pill-merged" title="14일: 릴리즈 7, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="마지막 회차">2026-09-05 11:20</td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/invenqor/pull/9">PR #9</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/invenqor/releases/tag/v0.2.25">v0.2.25</a></td><td data-label="누적 비용" class="num">$4.72</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/jikim/">jikim</a> <span class="pill pill-released" title="14일: 릴리즈 1, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="마지막 회차">2026-09-05 03:52</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/jikim/pull/14">PR #14</a>, released <a href="https://github.com/hkjang/jikim/releases/tag/v0.2.2">v0.2.2</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/jikim/releases/tag/v0.2.2">v0.2.2</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a> <span class="pill pill-merged" title="14일: 릴리즈 3, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="마지막 회차">2026-09-05 04:17</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/jupiq/pull/3">PR #3</a>, released <a href="https://github.com/hkjang/jupiq/releases/tag/v1.4.4">v1.4.4</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/jupiq/releases/tag/v1.4.4">v1.4.4</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/kanpic/">kanpic</a> <span class="pill pill-merged" title="14일: 릴리즈 8, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="마지막 회차">2026-09-05 04:35</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/kanpic/pull/9">PR #9</a>, released <a href="https://github.com/hkjang/kanpic/releases/tag/v0.236.0">v0.236.0</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/kanpic/releases/tag/v0.236.0">v0.236.0</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/moina/">moina</a> <span class="pill pill-merged" title="14일: 릴리즈 6, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="마지막 회차">2026-09-05 05:15</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/moina/pull/7">PR #7</a>, released <a href="https://github.com/hkjang/moina/releases/tag/v0.1.21">v0.1.21</a>, <strong>ASSETS MISSING</strong></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/moina/releases/tag/v0.1.21">v0.1.21</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/moyro/">moyro</a> <span class="pill pill-merged" title="14일: 릴리즈 6, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="마지막 회차">2026-09-04 10:01</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/moyro/pull/7">PR #7</a>, released <a href="https://github.com/hkjang/moyro/releases/tag/v0.2.15">v0.2.15</a>, <strong>ASSETS MISSING</strong></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/moyro/releases/tag/v0.2.15">v0.2.15</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/muni/">muni</a> <span class="pill pill-released" title="14일: 릴리즈 6, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="마지막 회차">2026-09-05 05:43</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/muni/pull/6">PR #6</a>, released <a href="https://github.com/hkjang/muni/releases/tag/v0.27.0">v0.27.0</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/muni/releases/tag/v0.27.0">v0.27.0</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/pii-masker/">pii-masker</a> <span class="pill pill-released" title="14일: 릴리즈 7, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="마지막 회차">2026-09-05 16:24</td><td data-label="결과"><span class="pill pill-other">• 기타</span> CI no-ci, PR open <a href="https://github.com/hkjang/pii-masker/pull/9">PR #9</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/pii-masker/releases/tag/v1.0.11">v1.0.11</a></td><td data-label="누적 비용" class="num">$5.05</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/ptium/">ptium</a> <span class="pill pill-merged" title="14일: 릴리즈 7, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="마지막 회차">2026-09-05 06:30</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/ptium/pull/9">PR #9</a>, released <a href="https://github.com/hkjang/ptium/releases/tag/v1.69.27">v1.69.27</a> +7 assets</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/ptium/releases/tag/v1.69.27">v1.69.27</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="merged"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/Quantoss/">Quantoss</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span></td><td data-label="마지막 회차">2026-09-05 11:44</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/Quantoss/pull/11">PR #11</a>, release skipped</td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$4.27</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/releasedock/">releasedock</a> <span class="pill pill-merged" title="14일: 릴리즈 7, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="마지막 회차">2026-09-05 06:55</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/releasedock/pull/8">PR #8</a>, released <a href="https://github.com/hkjang/releasedock/releases/tag/v0.5.8">v0.5.8</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/releasedock/releases/tag/v0.5.8">v0.5.8</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a> <span class="pill pill-merged" title="14일: 릴리즈 10, 실패 0, 경고 3, 회귀 0">건강 C</span></td><td data-label="마지막 회차">2026-09-05 08:12</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/relio/pull/10">PR #10</a>, released <a href="https://github.com/hkjang/relio/releases/tag/v1.11.17">v1.11.17</a>, <strong>ASSETS MISSING</strong></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/relio/releases/tag/v1.11.17">v1.11.17</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/ReSSO/">ReSSO</a> <span class="pill pill-merged" title="14일: 릴리즈 5, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="마지막 회차">2026-09-04 23:18</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/ReSSO/pull/9">PR #9</a>, released <a href="https://github.com/hkjang/ReSSO/releases/tag/v0.9.70">v0.9.70</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/ReSSO/releases/tag/v0.9.70">v0.9.70</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/umm/">umm</a> <span class="pill pill-released" title="14일: 릴리즈 7, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="마지막 회차">2026-09-05 08:43</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/umm/pull/144">PR #144</a>, released <a href="https://github.com/hkjang/umm/releases/tag/v0.71.2">v0.71.2</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/umm/releases/tag/v0.71.2">v0.71.2</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/Vendra/">Vendra</a> <span class="pill pill-released" title="14일: 릴리즈 6, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="마지막 회차">2026-09-04 23:39</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/Vendra/pull/107">PR #107</a>, released <a href="https://github.com/hkjang/Vendra/releases/tag/v0.7.41">v0.7.41</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/Vendra/releases/tag/v0.7.41">v0.7.41</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/vibe-coders/">vibe-coders</a> <span class="pill pill-merged" title="14일: 릴리즈 3, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="마지막 회차">2026-09-05 09:12</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/vibe-coders/pull/14">PR #14</a>, released <a href="https://github.com/hkjang/vibe-coders/releases/tag/v0.83.0">v0.83.0</a> +7 assets</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/vibe-coders/releases/tag/v0.83.0">v0.83.0</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/visitflow/">visitflow</a> <span class="pill pill-merged" title="14일: 릴리즈 5, 실패 0, 경고 3, 회귀 0">건강 C</span></td><td data-label="마지막 회차">2026-09-05 09:39</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/visitflow/pull/8">PR #8</a>, released <a href="https://github.com/hkjang/visitflow/releases/tag/v2.6.5">v2.6.5</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/visitflow/releases/tag/v2.6.5">v2.6.5</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="nochange"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/weekly/">weekly</a> <span class="pill pill-merged" title="14일: 릴리즈 6, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="마지막 회차">2026-09-05 10:00</td><td data-label="결과"><span class="pill pill-nochange">➖ 변경 없음</span> no change</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/weekly/releases/tag/v0.289.0">v0.289.0</a></td><td data-label="누적 비용" class="num">$0.00</td></tr></tbody></table></div>

## 오늘 상한

<dl class="kv">
<dt>비용</dt><dd><span class="meta">23.65 / 80</span> <span style="display:inline-block;width:6rem;height:.5rem;background:var(--card-2);border-radius:4px;vertical-align:middle"><span style="display:block;width:29%;height:100%;background:var(--good);border-radius:4px"></span></span></dd>
<dt>회차</dt><dd><span class="meta">27 / 60</span> <span style="display:inline-block;width:6rem;height:.5rem;background:var(--card-2);border-radius:4px;vertical-align:middle"><span style="display:block;width:45%;height:100%;background:var(--good);border-radius:4px"></span></span></dd>
<dt>릴리즈</dt><dd><span class="meta">19 / 30</span> <span style="display:inline-block;width:6rem;height:.5rem;background:var(--card-2);border-radius:4px;vertical-align:middle"><span style="display:block;width:63%;height:100%;background:var(--good);border-radius:4px"></span></span></dd>
<dt>휴면 규칙</dt><dd>변경 없음 3회 연속이면 7일 제외</dd>
<dt>수동 실행</dt><dd><a href="https://github.com/hkjang/aidev/issues/new?labels=run&title=run%3A+%3C%ED%94%84%EB%A1%9C%EC%A0%9D%ED%8A%B8%3E">이슈 만들기</a> — 제목 <code>run: &lt;프로젝트&gt;</code>, 라벨 <code>run</code> → 다음 회차 우선 실행</dd>
</dl>

설정: `state/caps.env`. 상한에 닿으면 그날은 새 회차를 시작하지 않고 알린다.

## 비용·사용량

<ul class="stats"><li><b>$23.65</b><span>오늘 비용</span></li><li><b>57분</b><span>오늘 에이전트 시간</span></li><li><b>11</b><span>오늘 세션</span></li><li><b>$23.65</b><span>누적 비용</span></li><li><b>57분</b><span>누적 시간</span></li><li><b>23.8M/212K</b><span>누적 토큰 입력/출력</span></li></ul>

claude -p 가 세션마다 보고한 추정값(정액제에서는 참고값). 회차별 내역은 각 일일 보고와 프로젝트 페이지, 원본은 [usage.jsonl](https://hkjang.github.io/aidev/data/usage.jsonl).

## FAQ

<details><summary>이 페이지는 무엇인가요?</summary><p>hkjang 의 GitHub 프로젝트들을 Claude Code 자율 개선 에이전트가 스스로 분석해 개선하고, 테스트를 통과시킨 뒤 PR 을 main 에 머지하고, 각 저장소의 기존 관례대로 릴리즈한 결과를 회차마다 자동으로 갱신하는 일일 보고 대시보드입니다.</p></details>
<details><summary>얼마나 자주 갱신되나요?</summary><p>에이전트는 10분 간격 스케줄로 사실상 연속 실행되며, 회차가 하나 끝날 때마다 이 사이트가 다시 만들어집니다. 보통 1~2분 안에 반영됩니다. Atom 피드(/feed.xml)를 구독하면 일일 보고를 받아볼 수 있습니다.</p></details>
<details><summary>한 회차에서 에이전트는 무엇을 하나요?</summary><p>저장소를 파악하고 개선 아이디어 5개를 가치·위험·작업량으로 채점해 하나를 고른 뒤 구현하고, 테스트·린트·빌드를 실제로 실행해 통과한 경우에만 커밋합니다. 러너가 PR 을 열고 CI 통과를 확인한 뒤 머지하며, 릴리즈 에이전트가 이전 릴리즈 방식(태그·버전 파일·CHANGELOG·워크플로·첨부 자산)을 확인해 같은 방식으로 다음 버전을 냅니다.</p></details>
<details><summary>릴리즈 워크플로가 실패하면 어떻게 되나요?</summary><p>러너가 한 번 자동으로 재실행합니다. 그래도 실패하면 실패 단계와 로그 요지를 수정 과제 큐에 넣고, 다음 회차에서 그 프로젝트를 우선 배정해 에이전트가 원인을 고칩니다(검증을 느슨하게 만드는 것은 금지). 큐에 있는 동안 &#x27;주의 필요&#x27;에 표시됩니다.</p></details>
<details><summary>어떤 프로젝트가 대상인가요?</summary><p>최근 30일 안에 커밋이 있고, 작업트리가 깨끗하며, GitHub 원격이 있는 저장소만 후보가 됩니다. 사람이 작업 중인(미커밋 변경이 있는) 저장소는 자동으로 제외됩니다.</p></details>
<details><summary>&#x27;머지(릴리즈 없음)&#x27;는 무슨 뜻인가요?</summary><p>개선은 머지됐지만 릴리즈가 만들어지지 않은 회차입니다. 릴리즈 이력이 전혀 없는 신규 저장소(관례를 새로 정하지 않음)이거나, 릴리즈 단계가 실패·미완료된 경우입니다.</p></details>
<details><summary>&#x27;주의 필요&#x27;에는 무엇이 뜨나요? 알림은 어디로 오나요?</summary><p>최근 2일 회차 중 CI 실패로 머지되지 않은 PR, 릴리즈 실패, 이전 릴리즈에는 있던 첨부 자산이 빠진 릴리즈, 수정 과제 대기처럼 사람이 확인해야 할 항목입니다. 새 경고가 생기면 GitHub Issue(라벨 alert)에 기록하고, 설정돼 있으면 Slack·이메일·Windows 알림으로도 보냅니다.</p></details>
<details><summary>머지 전에 검토는 없나요?</summary><p>있습니다. 구현과 다른 세션의 리뷰 에이전트가 diff 만 읽고 &#x27;머지하면 안 되는 이유&#x27;(검증하지 않는 테스트, 논리 오류, 설명과 다른 동작, 위험한 변경)를 찾습니다. 거절하면 PR 에 사유를 달고 열어 둡니다. 또 워크플로·마이그레이션·인증·결제·배포 파일(보호 파일)을 건드린 PR 은 자동 머지하지 않습니다.</p></details>
<details><summary>깨진 변경은 어떻게 되나요?</summary><p>머지 2시간 뒤부터 main CI 실패와 되돌림 커밋을 확인해 &#x27;교훈&#x27;으로 기록하고, 그 프로젝트의 다음 회차 프롬프트에 주입해 같은 실수를 피하게 합니다. 릴리즈 워크플로가 반복 실패하고 수정 회차도 실패하면 원래 머지를 되돌리는 롤백 PR 을 자동으로 엽니다(머지는 사람). 프로젝트마다 최근 14일의 실패·경고·회귀로 건강 등급 A~D 를 매깁니다.</p></details>
<details><summary>비용은 어떻게 계산되나요?</summary><p>각 회차의 claude -p 세션이 보고한 추정 비용(USD)과 소요 시간·턴 수·토큰을 그대로 합산합니다. 정액제 구독에서는 실제 청구가 아닌 참고값입니다.</p></details>
<details><summary>원본 데이터는 어디서 보나요?</summary><p>회차 기록은 runs.jsonl, 사용량은 usage.jsonl, 요약은 data/summary.json, 프로젝트별 원장은 GitHub 저장소 hkjang/aidev 의 state/ 폴더, 러너와 프롬프트는 같은 저장소의 bin/ 과 prompt.md 에 있습니다.</p></details>

---
러너·프롬프트·원장은 [https://github.com/hkjang/aidev](https://github.com/hkjang/aidev) 에서 관리한다. 이 페이지는 회차가 끝날 때마다 `bin/report.py` 가 다시 만든다.

{% endraw %}
