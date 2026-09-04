---
title: "aidev 자율 개선 대시보드"
description: "Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 3회차·릴리즈 0건, 누적 226회차·릴리즈 124건."
last_modified_at: 2026-09-05 00:51:47 +0900
---

<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "WebSite",
 "name": "aidev 자율 개선 대시보드",
 "url": "https://hkjang.github.io/aidev/",
 "description": "Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 3회차·릴리즈 0건, 누적 226회차·릴리즈 124건.",
 "inLanguage": "ko",
 "author": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-05T00:51:47"
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
  }
 ]
}
</script>

# aidev 자율 개선 대시보드

<p class="tldr"><strong>한 줄 요약.</strong> Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 3회차·릴리즈 0건, 누적 226회차·릴리즈 124건. 회차가 끝날 때마다 자동 갱신됩니다 (마지막 갱신 2026-09-05 00:51).</p>

[운영 문서](https://github.com/hkjang/aidev#readme) · [원장](https://github.com/hkjang/aidev/tree/main/state) · [실행 이력](https://github.com/hkjang/aidev/commits/main) · [원본 데이터](https://hkjang.github.io/aidev/data/runs.jsonl)

## 오늘 (2026-09-05)

<ul class="stats"><li><b>3</b><span>회차</span></li><li><b>3</b><span>프로젝트</span></li><li><b>0</b><span>릴리즈</span></li><li><b>3</b><span>머지(릴리즈 없음)</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실패</span></li></ul>

[2026-09-05 보고 자세히 보기 →](https://hkjang.github.io/aidev/reports/2026-09-05/)

| 시각 | 프로젝트 | 결과 |
|---|---|---|
| 00:10 | aiportal-front-admin | ✅ 머지 merged [PR #8](https://github.com/hkjang/aiportal-front-admin/pull/8), release skipped |
| 00:31 | aiportal-front | ✅ 머지 merged [PR #4](https://github.com/hkjang/aiportal-front/pull/4), release skipped |
| 00:51 | aiportal-java | ✅ 머지 merged [PR #9](https://github.com/hkjang/aiportal-java/pull/9), release skipped |

## 일일 보고

| 날짜 | 회차 | 릴리즈 | 머지 | 변경 없음 | 실패 |
|---|---|---|---|---|---|
| [2026-09-05](https://hkjang.github.io/aidev/reports/2026-09-05/) | 3 | 0 | 3 | 0 | 0 |
| [2026-09-04](https://hkjang.github.io/aidev/reports/2026-09-04/) | 77 | 33 | 9 | 1 | 0 |
| [2026-09-03](https://hkjang.github.io/aidev/reports/2026-09-03/) | 96 | 63 | 25 | 8 | 0 |
| [2026-09-02](https://hkjang.github.io/aidev/reports/2026-09-02/) | 50 | 28 | 16 | 3 | 0 |

## 프로젝트별 현황

| 프로젝트 | 마지막 회차 | 결과 | 최근 릴리즈 |
|---|---|---|---|
| [AgentHub](https://github.com/hkjang/AgentHub) | 2026-09-04 03:25 | 🚀 릴리즈 merged [PR #7](https://github.com/hkjang/AgentHub/pull/7), released [v0.232.0](https://github.com/hkjang/AgentHub/releases/tag/v0.232.0) | [v0.233.0](https://github.com/hkjang/AgentHub/releases/tag/v0.233.0) |
| [ai-admin](https://github.com/hkjang/ai-admin) | 2026-09-04 05:13 | 🚀 릴리즈 merged [PR #6](https://github.com/hkjang/ai-admin/pull/6), released [v1.2.6](https://github.com/hkjang/ai-admin/releases/tag/v1.2.6), ASSETS MISSING | [v1.2.9](https://github.com/hkjang/ai-admin/releases/tag/v1.2.9) |
| [aiportal-front](https://github.com/hkjang/aiportal-front) | 2026-09-05 00:31 | ✅ 머지 merged [PR #4](https://github.com/hkjang/aiportal-front/pull/4), release skipped | skipped |
| [aiportal-front-admin](https://github.com/hkjang/aiportal-front-admin) | 2026-09-05 00:10 | ✅ 머지 merged [PR #8](https://github.com/hkjang/aiportal-front-admin/pull/8), release skipped | skipped |
| [aiportal-java](https://github.com/hkjang/aiportal-java) | 2026-09-05 00:51 | ✅ 머지 merged [PR #9](https://github.com/hkjang/aiportal-java/pull/9), release skipped | skipped |
| [aiportal-py](https://github.com/hkjang/aiportal-py) | 2026-09-04 06:09 | ✅ 머지 merged [PR #5](https://github.com/hkjang/aiportal-py/pull/5), release skipped | skipped |
| [appstore](https://github.com/hkjang/appstore) | 2026-09-03 02:16 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/appstore/pull/3), released [v2.1.3](https://github.com/hkjang/appstore/releases/tag/v2.1.3) | [v2.1.4](https://github.com/hkjang/appstore/releases/tag/v2.1.4) |
| [Clustara](https://github.com/hkjang/clustara) | 2026-09-04 03:41 | 🚀 릴리즈 merged [PR #7](https://github.com/hkjang/clustara/pull/7), released [v0.9.268](https://github.com/hkjang/clustara/releases/tag/v0.9.268) | [v0.9.269](https://github.com/hkjang/clustara/releases/tag/v0.9.269) |
| [dataworks](https://github.com/hkjang/dataworks) | 2026-09-04 06:20 | 🚀 릴리즈 merged [PR #6](https://github.com/hkjang/dataworks/pull/6), released [v0.9.41](https://github.com/hkjang/dataworks/releases/tag/v0.9.41) | [v0.9.36](https://github.com/hkjang/dataworks/releases/tag/v0.9.36) |
| [git-ctx](https://github.com/hkjang/git-ctx) | 2026-09-04 06:56 | 🚀 릴리즈 merged [PR #17](https://github.com/hkjang/git-ctx/pull/17), released [v0.77.4](https://github.com/hkjang/git-ctx/releases/tag/v0.77.4) | [v0.77.4](https://github.com/hkjang/git-ctx/releases/tag/v0.77.4) |
| [igame](https://github.com/hkjang/igame) | 2026-09-04 07:20 | ✅ 머지 merged [PR #5](https://github.com/hkjang/igame/pull/5), release failed | [v0.7.5](https://github.com/hkjang/igame/releases/tag/v0.7.5) |
| [Invenqor](https://github.com/hkjang/invenqor) | 2026-09-04 04:01 | 🚀 릴리즈 merged [PR #7](https://github.com/hkjang/invenqor/pull/7), released [v0.2.24](https://github.com/hkjang/invenqor/releases/tag/v0.2.24) | [v0.2.25](https://github.com/hkjang/invenqor/releases/tag/v0.2.25) |
| [jupiq](https://github.com/hkjang/jupiq) | 2026-09-04 08:40 | 🚀 릴리즈 merged [PR #2](https://github.com/hkjang/jupiq/pull/2), released [v1.4.0](https://github.com/hkjang/jupiq/releases/tag/v1.4.0), ASSETS MISSING | [v1.4.0](https://github.com/hkjang/jupiq/releases/tag/v1.4.0) |
| [kanpic](https://github.com/hkjang/kanpic) | 2026-09-04 09:11 | 🚀 릴리즈 merged [PR #8](https://github.com/hkjang/kanpic/pull/8), released [v0.235.0](https://github.com/hkjang/kanpic/releases/tag/v0.235.0) | [v0.235.0](https://github.com/hkjang/kanpic/releases/tag/v0.235.0) |
| [moina](https://github.com/hkjang/moina) | 2026-09-04 09:27 | ✅ 머지 merged [PR #6](https://github.com/hkjang/moina/pull/6), release missing |  |
| [moyro](https://github.com/hkjang/moyro) | 2026-09-04 10:01 | 🚀 릴리즈 merged [PR #7](https://github.com/hkjang/moyro/pull/7), released [v0.2.15](https://github.com/hkjang/moyro/releases/tag/v0.2.15), ASSETS MISSING | [v0.2.15](https://github.com/hkjang/moyro/releases/tag/v0.2.15) |
| [muni](https://github.com/hkjang/muni) | 2026-09-03 04:00 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/muni/pull/3), released [v0.24.0](https://github.com/hkjang/muni/releases/tag/v0.24.0) | [v0.26.0](https://github.com/hkjang/muni/releases/tag/v0.26.0) |
| [pii-masker](https://github.com/hkjang/pii-masker) | 2026-09-04 00:11 | 🚀 릴리즈 merged [PR #6](https://github.com/hkjang/pii-masker/pull/6), released [v1.0.9](https://github.com/hkjang/pii-masker/releases/tag/v1.0.9) | [v1.0.10](https://github.com/hkjang/pii-masker/releases/tag/v1.0.10) |
| [ptium](https://github.com/hkjang/ptium) | 2026-09-04 00:28 | 🚀 릴리즈 merged [PR #6](https://github.com/hkjang/ptium/pull/6), released [v1.69.24](https://github.com/hkjang/ptium/releases/tag/v1.69.24) | [v1.69.19](https://github.com/hkjang/ptium/releases/tag/v1.69.19) |
| [Quantoss](https://github.com/hkjang/Quantoss) | 2026-09-04 04:14 | ✅ 머지 merged [PR #6](https://github.com/hkjang/Quantoss/pull/6), release skipped | skipped |
| [releasedock](https://github.com/hkjang/releasedock) | 2026-09-04 00:33 | ➖ 변경 없음 no change | [v0.5.7](https://github.com/hkjang/releasedock/releases/tag/v0.5.7) |
| [relio](https://github.com/hkjang/relio) | 2026-09-04 00:50 | 🚀 릴리즈 merged [PR #6](https://github.com/hkjang/relio/pull/6), released [v1.11.13](https://github.com/hkjang/relio/releases/tag/v1.11.13) | [v1.11.15](https://github.com/hkjang/relio/releases/tag/v1.11.15) |
| [ReSSO](https://github.com/hkjang/ReSSO) | 2026-09-04 04:46 | 🚀 릴리즈 merged [PR #8](https://github.com/hkjang/ReSSO/pull/8), released [v0.9.69](https://github.com/hkjang/ReSSO/releases/tag/v0.9.69) | [v0.9.70](https://github.com/hkjang/ReSSO/releases/tag/v0.9.70) |
| [ReSSO merged PR #2; weekly released v0.281.0 (수동 동기화](https://github.com/hkjang/ReSSO merged PR #2; weekly released v0.281.0 (수동 동기화) | 2026-09-02 12:43 | • 러너 git 식별자 미설정 수정) |  |
| [umm](https://github.com/hkjang/umm) | 2026-09-04 01:10 | 🚀 릴리즈 merged [PR #138](https://github.com/hkjang/umm/pull/138), released [v0.67.2](https://github.com/hkjang/umm/releases/tag/v0.67.2) | [v0.71.1](https://github.com/hkjang/umm/releases/tag/v0.71.1) |
| [Vendra](https://github.com/hkjang/Vendra) | 2026-09-04 23:39 | 🚀 릴리즈 merged [PR #107](https://github.com/hkjang/Vendra/pull/107), released [v0.7.41](https://github.com/hkjang/Vendra/releases/tag/v0.7.41) | [v0.7.41](https://github.com/hkjang/Vendra/releases/tag/v0.7.41) |
| [Vendra/ai-admin/pii-masker/ptium](https://github.com/hkjang/Vendra/ai-admin/pii-masker/ptium) | 2026-09-02 16:32 | • 태그 사후 푸시 및 Release 생성 |  |
| [vibe-coders](https://github.com/hkjang/vibe-coders) | 2026-09-04 01:51 | 🚀 릴리즈 merged [PR #12](https://github.com/hkjang/vibe-coders/pull/12), released [v0.82.2](https://github.com/hkjang/vibe-coders/releases/tag/v0.82.2) +7 assets | [v0.82.2](https://github.com/hkjang/vibe-coders/releases/tag/v0.82.2) |
| [visitflow](https://github.com/hkjang/visitflow) | 2026-09-04 02:09 | 🚀 릴리즈 merged [PR #6](https://github.com/hkjang/visitflow/pull/6), released [v2.6.3](https://github.com/hkjang/visitflow/releases/tag/v2.6.3) | [v2.6.4](https://github.com/hkjang/visitflow/releases/tag/v2.6.4) |
| [weekly](https://github.com/hkjang/weekly) | 2026-09-04 02:53 | ✅ 머지 merged [PR #7](https://github.com/hkjang/weekly/pull/7), release missing | [v0.289.0](https://github.com/hkjang/weekly/releases/tag/v0.289.0) |

## FAQ

<details><summary>이 페이지는 무엇인가요?</summary><p>hkjang 의 GitHub 프로젝트들을 Claude Code 자율 개선 에이전트가 스스로 분석해 개선하고, 테스트를 통과시킨 뒤 PR 을 main 에 머지하고, 각 저장소의 기존 관례대로 릴리즈한 결과를 회차마다 자동으로 갱신하는 일일 보고 대시보드입니다.</p></details>
<details><summary>얼마나 자주 갱신되나요?</summary><p>에이전트는 10분 간격 스케줄로 사실상 연속 실행되며, 회차가 하나 끝날 때마다 이 사이트가 다시 만들어집니다. 보통 1~2분 안에 반영됩니다.</p></details>
<details><summary>한 회차에서 에이전트는 무엇을 하나요?</summary><p>저장소를 파악하고 개선 아이디어 5개를 가치·위험·작업량으로 채점해 하나를 고른 뒤 구현하고, 테스트·린트·빌드를 실제로 실행해 통과한 경우에만 커밋합니다. 러너가 PR 을 열어 머지하고, 릴리즈 에이전트가 이전 릴리즈 방식(태그·버전 파일·CHANGELOG·워크플로)을 확인해 같은 방식으로 다음 버전을 냅니다.</p></details>
<details><summary>어떤 프로젝트가 대상인가요?</summary><p>최근 30일 안에 커밋이 있고, 작업트리가 깨끗하며, GitHub 원격이 있는 저장소만 후보가 됩니다. 사람이 작업 중인(미커밋 변경이 있는) 저장소는 자동으로 제외됩니다.</p></details>
<details><summary>'머지(릴리즈 없음)'는 무슨 뜻인가요?</summary><p>개선은 머지됐지만 릴리즈가 만들어지지 않은 회차입니다. 릴리즈 이력이 전혀 없는 신규 저장소(관례를 새로 정하지 않음)이거나, 릴리즈 단계가 실패·미완료된 경우입니다.</p></details>
<details><summary>원본 데이터는 어디서 보나요?</summary><p>회차 기록은 runs.jsonl(한 줄에 한 회차, JSON), 프로젝트별 원장은 GitHub 저장소 hkjang/aidev 의 state/ 폴더, 러너와 프롬프트는 같은 저장소의 bin/ 과 prompt.md 에 있습니다.</p></details>

---
러너·프롬프트·원장은 [https://github.com/hkjang/aidev](https://github.com/hkjang/aidev) 에서 관리한다. 이 페이지는 회차가 끝날 때마다 `bin/report.py` 가 다시 만든다.
