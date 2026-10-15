## 정찰 노트
- 저장소는 성숙하다(시험 76개, TODO/FIXME 0건, 금칙어·메타 자리까지 빌드가 검사). 그래서 버그 사냥보다 **어제 공개한 대표 영상이 공유 미리보기에서 빠져 있는** 공백을 골랐다 — `shell.html` 의 `og:type` 은 전 쪽 고정 `website`, `og:video`·`twitter:*` 자리는 없음(직접 확인).
- JSON-LD(VideoObject)가 SEO 가치는 더 크지만 `scripts/verify-build.mjs:36-37` 이 인라인 `<script>` 0개를 단정하고 CSP 가 `script-src 'self'` 다 — **보안 가드를 느슨하게 하는 변경**이 본체가 되므로 제쳐 ideas.json 에 분리해 남겼다. 같은 회차에 섞지 말 것.
- 추측으로 적지 않은 것: `serveObject` 의 ETag 처리 여부, 대표 영상의 `probe.hasAudio` 값, `application/ld+json` 이 CSP `script-src` 대상인지 — 모두 ideas.json 에 "미확인" 으로 표기.
- 구현자가 조심할 것: `shell.html` 과 `seo.ts` 는 **짝으로** 고쳐야 한다(한쪽만 고치면 아무 일도 안 일어나거나 전 쪽에 빈 `og:video` 가 남는다). `setAttribute` 경로에 `esc()` 를 쓰면 이중 이스케이프가 된다.
- 증명은 `npm run build` → `npm run test:api` 의 **응답 본문**으로. `shell.html` 소스 문자열 검사는 근거가 아니다(Worker 가 `/shell` 을 다시 쓴다).
