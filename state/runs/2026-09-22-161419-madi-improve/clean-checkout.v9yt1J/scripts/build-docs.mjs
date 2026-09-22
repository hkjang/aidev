// Rebuild offline-readable manuals and the complete, real-screenshot gallery.
// npm --prefix web ci first; no network access is needed by this generator.
import { readFile, writeFile, readdir, mkdir } from "node:fs/promises";
import { createHash } from "node:crypto";
import { createRequire } from "node:module";
import { fileURLToPath, pathToFileURL } from "node:url";
import path from "node:path";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const docs = path.join(root, "docs");
const require = createRequire(path.join(root, "web/package.json"));
const { Marked } = await import(pathToFileURL(require.resolve("marked")).href);
const origin = "https://hkjang.github.io/madi/";
const version = (await readFile(path.join(root, "VERSION"), "utf8"))
  .trim()
  .replace(/^v/, "");
const escape = (s) =>
  String(s).replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ],
  );
const slug = (s) =>
  s
    .toLowerCase()
    .replace(/<[^>]*>/g, "")
    .replace(/[^\p{L}\p{N}\s_-]/gu, "")
    .trim()
    .replace(/\s+/g, "-");
function page(title, description, content, target, depth = "", schema = {}) {
  return `<!doctype html>
<html lang="ko"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="theme-color" content="#164e43">
<title>${escape(title)} — madi</title><meta name="description" content="${escape(description)}"><link rel="canonical" href="${origin + target}"><link rel="icon" href="${depth}assets/favicon.svg" type="image/svg+xml"><link rel="stylesheet" href="${depth}assets/style.css">
<meta property="og:type" content="website"><meta property="og:locale" content="ko_KR"><meta property="og:site_name" content="madi"><meta property="og:title" content="${escape(title)} — madi"><meta property="og:description" content="${escape(description)}"><meta property="og:url" content="${origin + target}"><meta property="og:image" content="${origin}screenshots/workspace.png"><meta name="twitter:card" content="summary_large_image">
<script type="application/ld+json">${JSON.stringify({ "@context": "https://schema.org", "@type": "TechArticle", headline: title, description, inLanguage: "ko", url: origin + target, isPartOf: { "@type": "WebSite", name: "madi", url: origin }, ...schema }).replace(/</g, "\\u003c")}</script></head>
<body><a class="skip" href="#main">본문으로 바로가기</a><header class="site-header"><div class="container header-inner"><a class="brand" href="${depth}index.html" aria-label="madi 홈"><img src="${depth}assets/logo.svg" alt="madi" width="113" height="40"></a><button class="mobile-toggle" type="button" aria-expanded="false" aria-controls="navigation" aria-label="탐색 메뉴 열기">☰</button><nav class="nav" id="navigation" aria-label="주 탐색"><a href="${depth}index.html#features">기능</a><a href="${depth}screenshots.html">화면 둘러보기</a><a href="${depth}guide.html">시작 가이드</a><a href="${depth}manuals.html">전체 매뉴얼</a><a href="${depth}api.html">API · MCP</a><a href="https://github.com/hkjang/madi">GitHub ↗</a></nav></div></header>
<main id="main" class="container">${content}</main>
<footer class="site-footer"><div class="container"><div class="footer-top"><a class="brand" href="${depth}index.html"><img src="${depth}assets/logo.svg" alt="madi" width="90" height="32"></a><div class="footer-links"><a href="${depth}manuals.html">사용자·관리자 매뉴얼</a><a href="${depth}screenshots.html">실제 화면</a><a href="https://github.com/hkjang/madi">소스 저장소 ↗</a></div></div><p class="footer-bottom">madi v${escape(version)} · 우리 조직의 지식이 연결되는 공간.</p></div></footer>
<dialog class="lightbox" aria-label="서비스 화면 확대"><img alt=""><div class="lightbox-caption"><span data-caption></span><button class="lightbox-close" type="button">닫기 ×</button></div></dialog><script src="${depth}assets/site.js" defer></script></body></html>\n`;
}

const files = (await readdir(docs))
  .filter((name) => name.endsWith(".md") && !name.endsWith("-design.md"))
  .sort();
const manuals = [];
await mkdir(path.join(docs, "manuals"), { recursive: true });
for (const file of files) {
  const markdown = await readFile(path.join(docs, file), "utf8");
  const title = markdown.match(/^#\s+(.+)$/m)?.[1] || file.replace(/\.md$/, "");
  const headings = [],
    seen = new Map();
  const marked = new Marked({ gfm: true });
  marked.use({
    renderer: {
      heading(token) {
        const text = this.parser.parseInline(token.tokens),
          base = slug(token.text),
          count = seen.get(base) || 0;
        seen.set(base, count + 1);
        const id = base + (count ? `-${count}` : "");
        if (token.depth > 1 && token.depth <= 3)
          headings.push({ text: token.text, id });
        return `<h${token.depth} id="${escape(id)}">${text}</h${token.depth}>\n`;
      },
    },
  });
  let body = marked.parse(markdown);
  body = body.replace(
    /href="\.\.\/(deploy|scripts|desktop|extensions)\/([^"]+)"/g,
    'href="https://github.com/hkjang/madi/blob/main/$1/$2"',
  );
  body = body
    .replace(/href="([^"#]+)\.md(#[^"]*)?"/g, (whole, base, hash = "") => {
      if (base.startsWith("http") || base.startsWith("/")) return whole;
      if (!base.includes("/") && files.includes(base + ".md"))
        return `href="${base}.html${hash}"`;
      return `href="https://github.com/hkjang/madi/blob/main/${escape(path.posix.normalize("docs/" + base + ".md"))}${hash}"`;
    })
    .replace(
      /(src|href)="((?:screenshots|benchmarks)\/[^\"]+)"/g,
      '$1="../$2"',
    );
  body = body
    .replace(/<table>/g, '<div class="table-wrap"><table>')
    .replace(/<\/table>/g, "</table></div>");
  const target = "manuals/" + file.replace(/\.md$/, ".html");
  const description = `${title}. madi의 실제 설정 방법, 사용자 권한, 기능 흐름과 운영상 주의점을 안내합니다.`;
  const content = `<div class="page-hero"><a class="crumb" href="../manuals.html">madi / 전체 매뉴얼</a><p>${escape(description)}</p><a href="../${escape(file)}" download>Markdown 원문 내려받기</a></div><div class="guide-layout"><aside class="guide-sidebar" aria-label="문서 목차"><strong>이 문서의 내용</strong>${headings.map((h) => `<a href="#${escape(h.id)}">${escape(h.text)}</a>`).join("")}</aside><article class="prose manual-prose">${body}</article></div>`;
  await writeFile(
    path.join(docs, target),
    page(title, description, content, target, "../"),
  );
  manuals.push({ title, target });
}
const manualList = `<div class="page-hero"><a class="crumb" href="index.html">madi / 매뉴얼</a><h1>사용자에서 운영자까지,<br>필요한 안내를 한곳에.</h1><p>문서 작성, 팀 운영, AI·에이전트, 기업 시스템 연결과 폐쇄망 배포를 위한 상세 가이드입니다. 모든 본문은 이 사이트에서 바로 읽을 수 있습니다.</p></div><div class="manual-grid">${manuals.map((m) => `<a class="manual-link" href="${m.target}"><h2>${escape(m.title)}</h2><span>가이드 읽기 →</span></a>`).join("")}</div>`;
await writeFile(
  path.join(docs, "manuals.html"),
  page(
    "사용자·관리자 전체 매뉴얼",
    "madi의 문서·협업·검색·AI·Agent·기업 연동·정보보호·폐쇄망 운영 상세 가이드 모음.",
    manualList,
    "manuals.html",
    "",
    { "@type": "CollectionPage" },
  ),
);

const words = {
  compatibility: "호환성",
  chromium: "Chromium",
  firefox: "Firefox",
  webkit: "WebKit",
  korean: "한국어 입력",
  access: "접근",
  approved: "승인 완료",
  attachment: "첨부",
  body: "본문",
  builder: "작성 도구",
  bundle: "묶음",
  cards: "카드",
  checkpoints: "재개 지점",
  cleanup: "정리 제안",
  collaboration: "공동 편집",
  committed: "반영 완료",
  comparison: "비교",
  completed: "완료",
  compose: "작성",
  conditions: "적용 조건",
  conflict: "충돌",
  conflicts: "불일치 후보",
  context: "연결 맥락",
  css: "CSS",
  csv: "CSV",
  date: "날짜",
  definition: "조회 정의",
  diagnostics: "진단",
  dictionary: "조직 용어 사전",
  distribution: "지식 배포",
  drift: "기대값과 차이",
  empty: "결과 없음",
  errors: "입력 오류 확인",
  evaluation: "검색 평가",
  evidence: "근거 보관",
  exception: "예외 검토",
  extraction: "본문 추출",
  focused: "이어 쓰기",
  generation: "색인 세대",
  generations: "색인 세대",
  grant: "권한 부여",
  grouped: "문서별 묶음",
  historical: "과거 근거",
  home: "홈",
  independent: "독립 검토",
  inline: "인라인 편집",
  inspector: "문서 정보 패널",
  invalid: "입력 확인",
  learning: "학습",
  my: "나의",
  observation: "실제 관찰",
  official: "공식 답변",
  office: "오피스 문서",
  origin: "출처 연결",
  original: "원본",
  overview: "개요",
  package: "패키지",
  passport: "문서 정보 카드",
  paste: "붙여넣기",
  path: "경로",
  paused: "일시 중단",
  pdf: "PDF",
  position: "위치",
  positions: "근거 위치",
  practice: "실습",
  proposal: "변경안",
  purpose: "작업 방식",
  question: "관리 질문",
  range: "범위",
  readable: "읽기 편한 표시",
  ready: "준비 완료",
  receipt: "반영 기록",
  recheck: "현재 기준 재확인",
  record: "보관 기록",
  recovery: "복구 안내",
  reference: "참조",
  removed: "접근 회수",
  request: "요청",
  requests: "요청 목록",
  resume: "재개형 가져오기",
  retracted: "근거 회수",
  revoked: "권한 회수",
  rollback: "이전 세대로 전환",
  roundtrip: "내보내기·재가져오기",
  row: "행",
  schema: "속성 구성",
  selection: "선택 구간",
  sent: "요청 접수",
  session: "세션",
  sharing: "공유 확인",
  split: "문서 분리",
  stale: "기준 변경",
  status: "상태",
  steps: "단계",
  structured: "문서에서 데이터 정리",
  system: "시스템",
  team: "팀",
  text: "텍스트",
  time: "시점 기준",
  tools: "도구",
  trust: "신뢰 키",
  unavailable: "현재 접근 불가",
  undo: "실행 취소",
  unobserved: "관찰 전",
  verified: "검증 결과",
  views: "보기 설정",
  work: "작업",
  worksets: "개인 작업 묶음",
  workset: "작업 묶음",
  zoom: "확대",
  actions: "작업",
  api: "API",
  artifacts: "보관 파일",
  captured: "수집 결과",
  detail: "상세",
  imap: "IMAP",
  private: "비공개",
  registered: "등록 결과",
  s3: "S3",
  schedule: "예약",
  webhooks: "웹훅",
  channel: "채널",
  delivery: "전송 이력",
  deliveries: "전송 이력",
  automations: "자동화",
  job: "작업",
  admin: "관리자",
  workspace: "워크스페이스",
  settings: "설정",
  document: "문서",
  documents: "모든 문서",
  editor: "편집기",
  source: "원문",
  location: "위치",
  foundations: "기본 편집",
  history: "이력",
  diff: "변경 비교",
  graph: "지식 그래프",
  local: "현재 문서 중심",
  ai: "AI",
  citations: "출처",
  citation: "인용",
  verification: "검증",
  draft: "초안",
  review: "검토",
  natural: "자연어",
  search: "검색",
  selected: "선택 문서",
  summary: "요약",
  knowledge: "지식",
  gap: "보완 제안",
  personal: "개인",
  rag: "RAG",
  index: "색인",
  consent: "동의",
  agent: "Agent",
  action: "작업",
  confirm: "확인",
  run: "실행",
  approval: "승인",
  inbox: "수집함",
  block: "블록",
  organizer: "구조 편집",
  canvas: "캔버스",
  drawing: "드로잉",
  command: "명령",
  palette: "팔레트",
  database: "데이터베이스",
  databases: "데이터베이스 목록",
  advanced: "고급 속성",
  board: "보드",
  calendar: "달력",
  gallery: "갤러리",
  list: "목록",
  table: "표",
  timeline: "타임라인",
  desktop: "데스크톱",
  offline: "오프라인",
  devices: "기기",
  discussion: "토론",
  lifecycle: "생명주기",
  policy: "정책",
  watermark: "워터마크",
  export: "내보내기",
  import: "가져오기",
  center: "센터",
  results: "결과",
  result: "결과",
  favorites: "즐겨찾기",
  git: "Git",
  sync: "동기화",
  preview: "미리보기",
  classify: "분류",
  information: "정보",
  protection: "보호",
  keyboard: "단축키",
  health: "건강 상태",
  quality: "품질",
  score: "점수",
  login: "로그인",
  members: "멤버",
  migration: "가져오기",
  folder: "폴더",
  mobile: "모바일",
  branding: "브랜딩",
  menu: "메뉴",
  navigation: "탐색",
  operations: "운영 상태",
  plugins: "플러그인",
  plugin: "플러그인",
  permissions: "권한",
  permission: "권한",
  spaces: "공간",
  space: "공간",
  support: "지원 진단",
  tasks: "할 일",
  task: "할 일",
  teams: "팀",
  telemetry: "텔레메트리",
  templates: "템플릿",
  template: "템플릿",
  universal: "통합",
  vault: "보관함",
  organizations: "조직",
  presentation: "프레젠테이션",
  profile: "개인 설정",
  dark: "다크 테마",
  expanded: "고급 설정",
  public: "공개",
  share: "공유",
  create: "생성",
  password: "비밀번호",
  runbook: "Runbook",
  execution: "실행",
  code: "코드",
  filters: "필터",
  start: "시작",
  properties: "속성",
  html: "HTML",
  kanban: "칸반",
  library: "라이브러리",
  trash: "휴지통",
  web: "웹",
  clipper: "클리퍼",
  features: "기능 플래그",
  audit: "감사",
  backup: "백업",
  dashboard: "대시보드",
  oidc: "OIDC",
  security: "보안",
  storage: "저장소",
  workflow: "승인 절차",
  users: "사용자",
  user: "사용자",
  identity: "기업 인증",
  ldap: "LDAP",
  saml: "SAML",
  scim: "SCIM",
  connector: "커넥터",
  connectors: "커넥터",
  sql: "SQL",
  sources: "소스",
  text2sql: "Text2SQL",
  enterprise: "기업 데이터",
  entity: "엔터티",
  entities: "엔터티",
  notification: "알림",
  notifications: "알림",
  capture: "수집",
  channels: "채널",
  inbound: "수신",
  automation: "자동화",
  jobs: "작업 큐",
  webhook: "웹훅",
  data: "외부 데이터",
  directory: "디렉터리",
  group: "그룹",
  mapping: "역할 매핑",
  plan: "계획",
  query: "조회",
  proposals: "추천 후보",
  topic: "주제",
  keys: "키",
  preferences: "개인 설정",
};
// Descriptive titles for multi-step product screens. These are verified UI
// states, not performance promises or fabricated usability observations.
const screenshotTitles = {
  "focused-home": "홈 · 이어 쓰기와 빠른 개인 메모",
  "navigation-purpose": "개인노트·팀 위키·지식 DB·운영 탐색 방식",
  "my-work": "나의 작업 · 문서와 할 일",
  "document-inspector": "문서 오른쪽 패널 · 속성·참조·댓글·AI·이력",
  "document-inspector-mobile": "모바일 문서 정보 패널",
  "document-context-tools": "문서에서 연결된 도구로 이동",
  "document-query-builder": "선언형 조회 표 만들기",
  "document-query-definition": "문서에 보관한 조회 정의",
  "document-query-results": "현재 권한으로 실행한 조회 표",
  "document-query-revoked": "권한 변경 시 조회 결과 제거",
  "document-query-mobile": "모바일 선언형 조회 표",
  "document-passport": "문서 정보 카드 · 소유자와 최신 상태",
  "document-cleanup-preview": "원문 기반 제목·태그 정리 제안 비교",
  "document-split-review": "원문과 새 하위 문서의 분리 미리보기",
  "document-split-conflict": "문서 분리 중 변경 충돌 안내",
  "document-session-recovery": "세션 만료 후 문서 복구 안내",
  "document-conflict-review": "서버 정본과 내 변경 비교",
  "document-paste-mobile": "모바일 붙여넣기 검토",
  "collaboration-diagnostics": "공동 편집 진단 · 서버 확정과 이력 압축",
  "collaboration-diagnostics-mobile": "모바일 공동 편집 진단",
  "selection-ai-review": "선택 구간 AI 제안과 원문 비교",
  "selection-ai-conflict": "선택 구간 적용 전 변경 충돌 확인",
  "selection-ai-mobile": "모바일 선택 구간 AI 요청",
  "selection-ai-mobile-comparison": "모바일 AI 제안 비교",
  "sharing-move-impact-review": "공유·이동 전 접근 범위 변경 확인",
  "access-grant-review": "접근 요청의 대상·권한 확인",
  "access-request-sent": "문서 접근 요청 접수",
  "access-request-sent-mobile": "모바일 문서 접근 요청 접수",
  "access-requests-mobile": "모바일 접근 요청함",
  "database-range-errors": "범위 붙여넣기 · 잘못된 선택 값 수정",
  "database-range-review": "범위 붙여넣기 · 모든 셀 검토",
  "database-inline-conflict": "인라인 셀 변경 충돌 · 입력 유지",
  "database-team-view-review": "팀 공유 보기 저장 전 확인",
  "database-private-views-mobile": "모바일 개인 보기 설정",
  "database-row-panel-mobile": "모바일 데이터 행 상세 패널",
  "migration-resume-source": "재개형 가져오기 · 원본 파일 선택",
  "migration-resume-paused": "전송을 중단하고 재개 지점 보존",
  "migration-resume-checkpoints": "파일별 업로드·검사 재개 지점",
  "migration-resume-diff": "가져올 원문과 정본의 변경 비교",
  "migration-resume-csv": "CSV 열 타입과 셀 변환 검토",
  "migration-resume-confirm": "검사한 자료의 명시적 가져오기 확인",
  "migration-resume-completed": "재개형 가져오기 완료와 실제 문서",
  "migration-resume-roundtrip": "문서 수정·내보내기 후 재가져오기 검토",
  "migration-resume-mobile": "모바일 재개형 가져오기",
  "admin-attachment-extraction": "첨부 본문 추출·OCR 운영 정책",
  "attachment-text-position": "첨부 텍스트의 원문 구간 확인",
  "attachment-pdf-original": "PDF 원본 페이지와 근거 위치",
  "attachment-office-position": "오피스 첨부의 표·시트·문단 위치",
  "attachment-position-mobile": "모바일 첨부 원본 위치 확인",
  "attachment-extraction-review": "첨부 본문 추출 전 파일·정책 확인",
  "attachment-body-search": "첨부 본문을 포함한 현재 권한 검색",
  "attachment-ai-consent": "첨부에서 선택한 구간의 AI 전송 동의",
  "attachment-ai-citation": "첨부 위치를 가리키는 AI 답변 인용",
  "attachment-ai-retracted": "첨부 권한 변경 후 AI 근거 회수",
  "attachment-evidence-consent": "첨부 근거 사본의 개인 보관 동의",
  "attachment-evidence-positions": "보관한 첨부 근거의 원본 위치",
  "attachment-evidence-historical": "첨부 변경 후 보관 시점 근거 확인",
  "attachment-evidence-mobile": "모바일 첨부 근거 보관함",
  "attachment-package-selection": "지식 패키지에 넣을 첨부 근거 선택",
  "attachment-package-ready": "첨부 근거가 포함된 지식 패키지",
  "attachment-package-stale": "첨부 변경 시 이전 패키지 내보내기 차단",
  "mobile-database-cards": "모바일 데이터베이스 카드 보기",
  "mobile-date-confirmation": "모바일 날짜 입력 확인",
  "mobile-date-invalid": "모바일 잘못된 날짜 입력 안내",
  "mobile-home-320": "320px 홈 화면",
  "mobile-home-actions": "모바일 홈 · 빠른 작업",
  "mobile-preferences-readable": "모바일 글자·표시 밀도 설정",
  "profile-css-zoom-200": "200% 확대 개인 설정 화면",
  "profile-navigation-mobile": "모바일 개인 탐색 방식 설정",
  "search-grouped-preview": "문서별 검색 결과와 오른쪽 미리보기",
  "search-grouped-mobile": "모바일 문서별 검색 결과",
  "search-preview-mobile": "모바일 검색 결과 미리보기",
  "search-empty-recovery-mobile": "모바일 검색 결과 없음 · 조건 조정",
  "search-dictionary": "워크스페이스 조직 용어 사전",
  "search-diagnostics": "현재 권한의 검색 실행 진단",
  "search-evaluation": "제공한 정답으로 한국어 검색 평가",
  "rag-generation-create": "새 벡터 색인 세대 준비",
  "rag-generation-document-consent": "색인 세대별 문서 전송 동의",
  "rag-generation-verified": "벡터 색인 세대 검사 결과",
  "rag-generation-rollback": "이전 벡터 세대 전환 확인",
  "rag-generations-mobile": "모바일 벡터 색인 세대 운영",
  "system-status-policy": "기대값·관찰값 수집 정책",
  "system-status-unobserved": "시스템 현황 · 관찰 전 상태",
  "system-status-drift": "문서 기대값과 실제 관찰값의 차이",
  "system-status-mobile": "모바일 시스템 현황 비교",
  "system-status-observation-mobile": "모바일 실제 관찰 기록",
  "evidence-policy": "개인 AI 근거 보관 정책",
  "evidence-record": "명시 동의로 보관한 AI 답변과 근거",
  "evidence-historical": "보관 시점 원문과 현재 기준 비교",
  "knowledge-package-policy": "지식 패키지 필수 근거·예산 정책",
  "knowledge-package-compose": "목적과 근거를 선택한 지식 패키지",
  "knowledge-package-record": "현재 권한으로 확인한 지식 패키지",
  "knowledge-impact": "문서 변경의 연결 영향 검토",
  "knowledge-impact-mobile": "모바일 문서 변경 영향 검토",
  "knowledge-proposal": "원문과 변경안 비교 · 명시 반영",
  "knowledge-proposal-mobile": "모바일 문서 변경안 검토",
  "knowledge-time": "시점과 기준을 지정한 지식 조회",
  "knowledge-time-mobile": "모바일 시점 기준 조회",
  "impact-exception-request": "변경 영향 예외 검토 요청",
  "impact-exception-review": "현재 원문 근거로 예외 검토",
  "impact-exception-approved": "변경 영향 예외 승인 기록",
  "impact-exception-mobile": "모바일 변경 영향 예외 검토",
  "distribution-policy": "지식 배포 서명·반입 정책",
  "distribution-trust": "배포 발신자의 신뢰 키 설정",
  "distribution-export-confirm": "지식 묶음 반출 전 검토",
  "distribution-export-ready": "서명된 지식 묶음 다운로드",
  "distribution-import-preview": "서명 확인 후 비공개 반입 미리보기",
  "distribution-import-receipt": "명시 반입의 문서·첨부 연결 기록",
  "distribution-mobile": "모바일 지식 묶음 반입",
  "distribution-bundle-review": "지식 묶음 전체 배포 검토 요청",
  "distribution-bundle-approval": "별도 검토자의 배포 승인",
  "distribution-bundle-approved": "승인 후 신청자가 확인하는 서명 배포",
  "question-personal-consent": "개인 AI 대화에서 답변 정리 동의",
  "question-private-draft": "관리 질문의 비공개 답변 초안",
  "question-official-conditions": "공식 답변의 근거·담당자·적용 조건",
  "question-stale-evidence": "근거 변경으로 공식 답변 재확인",
  "question-mobile": "모바일 관리 질문과 공식 답변",
  "conflicts-selection": "비교할 문서와 규칙기반 검토 범위 선택",
  "conflicts-comparison": "같은 주제의 수치·정책 문구 비교",
  "conflicts-review": "불일치 후보에 대한 사람의 판단 기록",
  "conflicts-history": "문서 간 불일치 검토 이력",
  "conflicts-proposal-consent": "양쪽 근거를 확인한 변경안 생성 동의",
  "conflicts-proposal-origin": "문서 변경안의 불일치 검토 출처",
  "conflicts-stale-origin": "출처 변경 시 이전 변경안 반영 차단",
  "conflicts-mobile": "모바일 문서 간 불일치 후보",
  "conflicts-mobile-comparison": "모바일 양쪽 원문 비교",
  "conflicts-access-removed": "현재 원문 접근 회수 후 후보 내용 제거",
  "structured-workspace": "문서에서 데이터 정리 · 원문과 대상 선택",
  "structured-selection": "구조화할 원문 구간·일반 속성 선택",
  "structured-ai-proposal": "완료된 AI 값 제안 · 자동 저장 없음",
  "structured-private-draft": "개인 구조화 초안 · 값과 인용 수정",
  "structured-sharing-preview": "새 데이터 행 공유 전 값·근거 확인",
  "structured-committed": "사람이 확인해 반영한 값과 근거",
  "structured-source-changed": "원문 변경 후 과거 구조화 근거 표시",
  "structured-schema-changed": "속성 변경 시 이전 미리보기 폐기",
  "structured-access-removed": "원문 권한 회수 후 구조화 내용 제거",
  "structured-mobile": "모바일 문서에서 데이터 정리",
  "structured-mobile-sharing": "모바일 새 행 공유 동의",
  "learning-path-overview": "역할별 학습 경로와 개인 진행",
  "learning-path-practice": "단계별 실습 결과와 근거 제출",
  "learning-path-independent-review": "독립 검토자가 확인하는 실습 원문",
  "learning-path-config-review": "학습 경로 공유·설정 변경 확인",
  "learning-path-completed": "현재 기준으로 확인한 학습 완료",
  "learning-path-source-recheck": "원문 변경 후 학습 단계 재확인",
  "learning-path-history": "개인 학습 진행 이력",
  "learning-path-mobile": "모바일 역할별 학습 경로",
  "learning-path-steps-mobile": "모바일 학습 단계와 순서",
  "worksets-overview": "개인 작업 묶음과 복원 위치",
  "workset-reference-comparison": "작업 묶음의 보관 위치와 현재 참조 비교",
  "worksets-mobile": "모바일 개인 작업 묶음",
  "home-worksets-mobile": "모바일 홈에서 작업 묶음 이어가기",
  "reference-unavailable-mobile": "모바일 저장한 참조의 현재 접근 불가 안내",
  "trash-undo": "문서 휴지통 이동 후 실행 취소",
  "trash-undo-conflict": "후속 변경이 있는 문서의 실행 취소 차단",
};
const successfulDiagnosticScreenshots = new Set([
  "collaboration-diagnostics",
  "collaboration-diagnostics-mobile",
  "search-diagnostics",
  "database-range-errors",
]);
const isDiagnostic = (name) =>
  /(?:^|-)(?:fail(?:ed|ure)?|debug|diagnostics?|errors?)(?:-|$)/.test(name);
const groups = {
  workspace: "문서·협업·탐색",
  data: "데이터·할 일·기업 지식",
  ai: "검색·AI·Agent",
  knowledge: "근거·검토·지식 운영",
  admin: "서비스·조직 운영",
  personal: "개인화·공유·기기",
  mobile: "모바일",
};
function category(name) {
  if (/(^|-)mobile($|-)/.test(name)) return "mobile";
  if (
    /^(evidence-policy|knowledge-package-policy|distribution-(policy|trust)|system-status-policy)$/.test(
      name,
    )
  )
    return "admin";
  if (
    /^(admin|support|approval-admin|organizations|space|members|teams|workspace-(settings|branding|features))/.test(
      name,
    )
  )
    return "admin";
  if (
    /^(evidence|knowledge-(package|impact|proposal|time|health)|distribution|question|conflicts|impact-exception|learning-path|system-status|attachment-(evidence|package))/.test(
      name,
    )
  )
    return "knowledge";
  if (/(search|^graph-ai|^ai-|^agent-|^rag-|rag-index|personal-ai)/.test(name))
    return "ai";
  if (
    /^(database|data-source|task|runbook|enterprise|entity|entities|sql-|text2sql|connector|structured|attachment)/.test(
      name,
    )
  )
    return "data";
  if (
    /^(profile|login|api-keys|public-share|devices|desktop|offline|clipper|web-clipper|keyboard|notification|worksets|workset-|my-work|reference-unavailable)/.test(
      name,
    )
  )
    return "personal";
  return "workspace";
}
const screenshots = [];
for (const file of (await readdir(path.join(docs, "screenshots"))).sort()) {
  if (!file.endsWith(".png") || /^product-/.test(file)) continue;
  const name = file.slice(0, -4),
    bytes = await readFile(path.join(docs, "screenshots", file));
  if (isDiagnostic(name) && !successfulDiagnosticScreenshots.has(name))
    continue;
  if (
    !bytes.subarray(0, 8).equals(Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]))
  )
    throw new Error("Invalid PNG: " + file);
  screenshots.push({
    file,
    title:
      screenshotTitles[name] ||
      name
        .split("-")
        .map((part) => words[part] || part)
        .join(" · "),
    category: category(name),
    width: bytes.readUInt32BE(16),
    height: bytes.readUInt32BE(20),
    sha256: createHash("sha256").update(bytes).digest("hex"),
  });
}
const featured = [
  "workspace.png",
  "documents.png",
  "document.png",
  "editor-source.png",
  "document-summary-list.png",
];
screenshots.sort((a, b) => {
  const rank = (name) =>
    featured.includes(name) ? featured.indexOf(name) : featured.length;
  return rank(a.file) - rank(b.file) || a.file.localeCompare(b.file);
});
let content = `<div class="page-hero"><a class="crumb" href="index.html">madi / 실제 화면</a><h1>사람과 AI가 함께 쓰는 공간,<br>모든 화면을 둘러보세요.</h1><p>실제 서비스에서 데모 계정·문서로 촬영한 ${screenshots.length}개 화면입니다. 문서와 관리자 설정, 연결과 동의 절차, 모바일 화면을 포함합니다. 가상 목업이 아니며 비밀값은 표시하지 않습니다.</p></div><form class="gallery-filters" role="search" aria-label="화면 검색"><label>화면 이름<input id="gallery-query" aria-label="화면 이름" type="search" placeholder="검색, 승인, 키, 문서…"></label><label>분류<select id="gallery-category" aria-label="분류"><option value="">모든 분류</option>${Object.entries(
  groups,
)
  .map(([id, label]) => `<option value="${id}">${label}</option>`)
  .join(
    "",
  )}</select></label><p id="gallery-count" role="status">전체 ${screenshots.length}개 화면</p></form>`;
for (const [id, title] of Object.entries(groups)) {
  content += `<section class="gallery-group" id="${id}" data-gallery-group><h2>${title}</h2><div class="screenshot-grid">`;
  content += screenshots
    .filter((s) => s.category === id)
    .map(
      (s) =>
        `<figure class="screen-card${s.width < 700 ? " mobile-screen" : ""}" data-gallery-card data-category="${id}" data-title="${escape(s.title)}"><button class="screen-button" type="button" data-screenshot aria-label="${escape(s.title)} 화면 확대"><img src="screenshots/${s.file}" alt="madi ${escape(s.title)} 실제 화면" loading="lazy" decoding="async" width="${s.width}" height="${s.height}"><span class="zoom-label">확대해서 보기 ↗</span></button><figcaption><h3>${escape(s.title)}</h3><p>실제 서비스 화면 · ${s.width} × ${s.height}</p></figcaption></figure>`,
    )
    .join("");
  content += "</div></section>";
}
await writeFile(
  path.join(docs, "screenshots.html"),
  page(
    "사용자·관리자·모바일 전체 화면",
    "madi의 문서·협업·검색·그래프·AI·기업 데이터·관리자 설정·모바일 실제 화면을 검색하고 확대해 보세요.",
    content,
    "screenshots.html",
    "",
    { "@type": "CollectionPage" },
  ),
);
await writeFile(
  path.join(docs, "screenshots/manifest.json"),
  JSON.stringify(
    {
      version,
      description:
        "실제 브라우저 성공 화면 목록. 실패 진단 화면은 제외합니다. 이미지 해시는 마지막 재생성 시점 기준입니다.",
      screenshots,
    },
    null,
    2,
  ) + "\n",
);
const urls = [
  "",
  "guide.html",
  "api.html",
  "screenshots.html",
  "manuals.html",
  ...manuals.map((m) => m.target),
];
await writeFile(
  path.join(docs, "sitemap.xml"),
  `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${urls.map((u) => `  <url><loc>${origin}${u}</loc></url>`).join("\n")}\n</urlset>\n`,
);
console.log(
  JSON.stringify({
    manuals: manuals.length,
    screenshots: screenshots.length,
    sitemap: urls.length,
  }),
);
