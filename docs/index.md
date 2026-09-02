---
title: "aidev 자율 개선 대시보드"
description: "Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 1회차·릴리즈 1건, 누적 51회차·릴리즈 29건."
last_modified_at: 2026-09-03 00:17:49 +0900
---

<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "WebSite",
 "name": "aidev 자율 개선 대시보드",
 "url": "https://hkjang.github.io/aidev/",
 "description": "Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 1회차·릴리즈 1건, 누적 51회차·릴리즈 29건.",
 "inLanguage": "ko",
 "author": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-03T00:17:49"
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
    "text": "에이전트는 10분 간격 스케줄로 사실상 연속 실행되며, 회차가 하나 끝날 때마다 이 사이트가 다시 만들어집니다. 보통 1~2분 안에 반영됩니다."
   }
  },
  {
   "@type": "Question",
   "name": "한 회차에서 에이전트는 무엇을 하나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "저장소를 파악하고 개선 아이디어 5개를 가치·위험·작업량으로 채점해 하나를 고른 뒤 구현하고, 테스트·린트·빌드를 실제로 실행해 통과한 경우에만 커밋합니다. 러너가 PR 을 열어 머지하고, 릴리즈 에이전트가 이전 릴리즈 방식(태그·버전 파일·CHANGELOG·워크플로)을 확인해 같은 방식으로 다음 버전을 냅니다."
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
   "name": "원본 데이터는 어디서 보나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "회차 기록은 runs.jsonl(한 줄에 한 회차, JSON), 프로젝트별 원장은 GitHub 저장소 hkjang/aidev 의 state/ 폴더, 러너와 프롬프트는 같은 저장소의 bin/ 과 prompt.md 에 있습니다."
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
   "name": "일일 보고 2026-09-03",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-03/"
  },
  {
   "@type": "ListItem",
   "position": 2,
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
  }
 ]
}
</script>

# aidev 자율 개선 대시보드

<p class="tldr"><strong>한 줄 요약.</strong> Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 1회차·릴리즈 1건, 누적 51회차·릴리즈 29건. 회차가 끝날 때마다 자동 갱신됩니다 (마지막 갱신 2026-09-03 00:17).</p>

[운영 문서](https://github.com/hkjang/aidev#readme) · [원장](https://github.com/hkjang/aidev/tree/main/state) · [실행 이력](https://github.com/hkjang/aidev/commits/main) · [원본 데이터](https://hkjang.github.io/aidev/data/runs.jsonl)

## 오늘 (2026-09-03)

<ul class="stats"><li><b>1</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>1</b><span>릴리즈</span></li><li><b>0</b><span>머지(릴리즈 없음)</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실패</span></li></ul>

[2026-09-03 보고 자세히 보기 →](https://hkjang.github.io/aidev/reports/2026-09-03/)

| 시각 | 프로젝트 | 결과 |
|---|---|---|
| 00:17 | AgentHub | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/AgentHub/pull/3), released [v0.228.0](https://github.com/hkjang/AgentHub/releases/tag/v0.228.0) |

## 일일 보고

| 날짜 | 회차 | 릴리즈 | 머지 | 변경 없음 | 실패 |
|---|---|---|---|---|---|
| [2026-09-03](https://hkjang.github.io/aidev/reports/2026-09-03/) | 1 | 1 | 0 | 0 | 0 |
| [2026-09-02](https://hkjang.github.io/aidev/reports/2026-09-02/) | 50 | 28 | 16 | 3 | 0 |

## 프로젝트별 현황

| 프로젝트 | 마지막 회차 | 결과 | 최근 릴리즈 |
|---|---|---|---|
| [AgentHub](https://github.com/hkjang/AgentHub) | 2026-09-03 00:17 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/AgentHub/pull/3), released [v0.228.0](https://github.com/hkjang/AgentHub/releases/tag/v0.228.0) | [v0.228.0](https://github.com/hkjang/AgentHub/releases/tag/v0.228.0) |
| [ai-admin](https://github.com/hkjang/ai-admin) | 2026-09-02 13:17 | ✅ 머지 merged [PR #1](https://github.com/hkjang/ai-admin/pull/1), release released | [v1.2.2](https://github.com/hkjang/ai-admin/releases/tag/v1.2.2) |
| [aiportal-front-admin](https://github.com/hkjang/aiportal-front-admin) | 2026-09-02 13:24 | ✅ 머지 merged [PR #1](https://github.com/hkjang/aiportal-front-admin/pull/1), release skipped | skipped |
| [aiportal-java](https://github.com/hkjang/aiportal-java) | 2026-09-02 13:39 | ✅ 머지 merged [PR #1](https://github.com/hkjang/aiportal-java/pull/1), release skipped | skipped |
| [aiportal-py](https://github.com/hkjang/aiportal-py) | 2026-09-02 13:45 | ✅ 머지 merged [PR #1](https://github.com/hkjang/aiportal-py/pull/1), release skipped | skipped |
| [appstore](https://github.com/hkjang/appstore) | 2026-09-02 13:56 | 🚀 릴리즈 merged [PR #1](https://github.com/hkjang/appstore/pull/1), released [v2.1.1](https://github.com/hkjang/appstore/releases/tag/v2.1.1) | [v2.1.2](https://github.com/hkjang/appstore/releases/tag/v2.1.2) |
| [Clustara](https://github.com/hkjang/clustara) | 2026-09-02 12:45 | ✅ 머지 merged (옛 러너 경로에서 수행된 회차 원장 이관) | [v0.9.263](https://github.com/hkjang/clustara/releases/tag/v0.9.263) |
| [dataworks](https://github.com/hkjang/dataworks) | 2026-09-02 14:00 | ➖ 변경 없음 no change | [v0.9.36](https://github.com/hkjang/dataworks/releases/tag/v0.9.36) |
| [git-ctx](https://github.com/hkjang/git-ctx) | 2026-09-02 14:10 | ➖ 변경 없음 no change |  |
| [igame](https://github.com/hkjang/igame) | 2026-09-02 14:20 | ➖ 변경 없음 no change |  |
| [Invenqor](https://github.com/hkjang/invenqor) | 2026-09-02 18:20 | 🚀 릴리즈 merged [PR #2](https://github.com/hkjang/invenqor/pull/2), released [v0.2.19](https://github.com/hkjang/invenqor/releases/tag/v0.2.19) | [v0.2.19](https://github.com/hkjang/invenqor/releases/tag/v0.2.19) |
| [kanpic](https://github.com/hkjang/kanpic) | 2026-09-02 14:38 | 🚀 릴리즈 merged [PR #1](https://github.com/hkjang/kanpic/pull/1), released [v0.227.0](https://github.com/hkjang/kanpic/releases/tag/v0.227.0) | [v0.229.0](https://github.com/hkjang/kanpic/releases/tag/v0.229.0) |
| [moina](https://github.com/hkjang/moina) | 2026-09-02 14:48 | 🚀 릴리즈 merged [PR #1](https://github.com/hkjang/moina/pull/1), released [v0.1.16](https://github.com/hkjang/moina/releases/tag/v0.1.16) | [v0.1.17](https://github.com/hkjang/moina/releases/tag/v0.1.17) |
| [moyro](https://github.com/hkjang/moyro) | 2026-09-02 15:00 | 🚀 릴리즈 merged [PR #1](https://github.com/hkjang/moyro/pull/1), released [v0.2.10](https://github.com/hkjang/moyro/releases/tag/v0.2.10) | [v0.2.12](https://github.com/hkjang/moyro/releases/tag/v0.2.12) |
| [muni](https://github.com/hkjang/muni) | 2026-09-02 15:18 | 🚀 릴리즈 merged [PR #1](https://github.com/hkjang/muni/pull/1), released [v0.22.0](https://github.com/hkjang/muni/releases/tag/v0.22.0) | [v0.23.0](https://github.com/hkjang/muni/releases/tag/v0.23.0) |
| [pii-masker](https://github.com/hkjang/pii-masker) | 2026-09-02 15:26 | ✅ 머지 merged [PR #1](https://github.com/hkjang/pii-masker/pull/1), release released | [v1.0.5](https://github.com/hkjang/pii-masker/releases/tag/v1.0.5) |
| [ptium](https://github.com/hkjang/ptium) | 2026-09-02 15:50 | ✅ 머지 merged [PR #1](https://github.com/hkjang/ptium/pull/1), release released |  |
| [releasedock](https://github.com/hkjang/releasedock) | 2026-09-02 16:07 | 🚀 릴리즈 merged [PR #1](https://github.com/hkjang/releasedock/pull/1), released [v0.5.1](https://github.com/hkjang/releasedock/releases/tag/v0.5.1) | [v0.5.2](https://github.com/hkjang/releasedock/releases/tag/v0.5.2) |
| [relio](https://github.com/hkjang/relio) | 2026-09-02 16:17 | 🚀 릴리즈 merged [PR #1](https://github.com/hkjang/relio/pull/1), released [v1.11.8](https://github.com/hkjang/relio/releases/tag/v1.11.8) | [v1.11.9](https://github.com/hkjang/relio/releases/tag/v1.11.9) |
| [ReSSO](https://github.com/hkjang/ReSSO) | 2026-09-02 16:40 | 🚀 릴리즈 release-only, released [v0.9.66](https://github.com/hkjang/ReSSO/releases/tag/v0.9.66) |  |
| [ReSSO merged PR #2; weekly released v0.281.0 (수동 동기화](https://github.com/hkjang/ReSSO merged PR #2; weekly released v0.281.0 (수동 동기화) | 2026-09-02 12:43 | • 러너 git 식별자 미설정 수정) |  |
| [umm](https://github.com/hkjang/umm) | 2026-09-02 16:30 | ✅ 머지 merged [PR #131](https://github.com/hkjang/umm/pull/131), release released | [v0.66.0](https://github.com/hkjang/umm/releases/tag/v0.66.0) |
| [Vendra](https://github.com/hkjang/Vendra) | 2026-09-02 13:08 | ✅ 머지 merged [PR #101](https://github.com/hkjang/Vendra/pull/101), release released | [v0.7.36](https://github.com/hkjang/Vendra/releases/tag/v0.7.36) |
| [Vendra/ai-admin/pii-masker/ptium](https://github.com/hkjang/Vendra/ai-admin/pii-masker/ptium) | 2026-09-02 16:32 | • 태그 사후 푸시 및 Release 생성 |  |
| [vibe-coders](https://github.com/hkjang/vibe-coders) | 2026-09-02 23:03 | • PR  |  |
| [visitflow](https://github.com/hkjang/visitflow) | 2026-09-02 17:02 | ✅ 머지 merged [PR #1](https://github.com/hkjang/visitflow/pull/1), release missing | [v2.6.1](https://github.com/hkjang/visitflow/releases/tag/v2.6.1) |
| [weekly](https://github.com/hkjang/weekly) | 2026-09-02 17:27 | 🚀 릴리즈 merged [PR #2](https://github.com/hkjang/weekly/pull/2), released [v0.282.0](https://github.com/hkjang/weekly/releases/tag/v0.282.0) | [v0.283.0](https://github.com/hkjang/weekly/releases/tag/v0.283.0) |

## FAQ

<details><summary>이 페이지는 무엇인가요?</summary><p>hkjang 의 GitHub 프로젝트들을 Claude Code 자율 개선 에이전트가 스스로 분석해 개선하고, 테스트를 통과시킨 뒤 PR 을 main 에 머지하고, 각 저장소의 기존 관례대로 릴리즈한 결과를 회차마다 자동으로 갱신하는 일일 보고 대시보드입니다.</p></details>
<details><summary>얼마나 자주 갱신되나요?</summary><p>에이전트는 10분 간격 스케줄로 사실상 연속 실행되며, 회차가 하나 끝날 때마다 이 사이트가 다시 만들어집니다. 보통 1~2분 안에 반영됩니다.</p></details>
<details><summary>한 회차에서 에이전트는 무엇을 하나요?</summary><p>저장소를 파악하고 개선 아이디어 5개를 가치·위험·작업량으로 채점해 하나를 고른 뒤 구현하고, 테스트·린트·빌드를 실제로 실행해 통과한 경우에만 커밋합니다. 러너가 PR 을 열어 머지하고, 릴리즈 에이전트가 이전 릴리즈 방식(태그·버전 파일·CHANGELOG·워크플로)을 확인해 같은 방식으로 다음 버전을 냅니다.</p></details>
<details><summary>어떤 프로젝트가 대상인가요?</summary><p>최근 30일 안에 커밋이 있고, 작업트리가 깨끗하며, GitHub 원격이 있는 저장소만 후보가 됩니다. 사람이 작업 중인(미커밋 변경이 있는) 저장소는 자동으로 제외됩니다.</p></details>
<details><summary>'머지(릴리즈 없음)'는 무슨 뜻인가요?</summary><p>개선은 머지됐지만 릴리즈가 만들어지지 않은 회차입니다. 릴리즈 이력이 전혀 없는 신규 저장소(관례를 새로 정하지 않음)이거나, 릴리즈 단계가 실패·미완료된 경우입니다.</p></details>
<details><summary>원본 데이터는 어디서 보나요?</summary><p>회차 기록은 runs.jsonl(한 줄에 한 회차, JSON), 프로젝트별 원장은 GitHub 저장소 hkjang/aidev 의 state/ 폴더, 러너와 프롬프트는 같은 저장소의 bin/ 과 prompt.md 에 있습니다.</p></details>

---
러너·프롬프트·원장은 [https://github.com/hkjang/aidev](https://github.com/hkjang/aidev) 에서 관리한다. 이 페이지는 회차가 끝날 때마다 `bin/report.py` 가 다시 만든다.
