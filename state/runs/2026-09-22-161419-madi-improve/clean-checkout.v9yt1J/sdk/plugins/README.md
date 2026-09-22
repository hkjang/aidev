# madi Plugin SDK v1

인터넷 없이 설치하는 프런트엔드 확장 SDK입니다. 서비스 관리자 설치 → 워크스페이스 관리자 권한 승인 → 사용자 실행 순서입니다. ZIP의 코드를 서버에서 실행하지 않습니다.

## 예제 패키지 만들기

저장소 루트에서 실행합니다. 생성되는 ZIP은 서비스 릴리즈 이미지와 별개의 개발용 설치 패키지이며, 릴리즈 자산에 추가하지 않습니다.

```sh
node sdk/plugins/package.mjs sdk/plugins/example knowledge-helper-1.0.0.zip
```

`/admin/plugins`에서 ZIP을 설치하고, `/app/plugins?manage=1`에서 필요한 권한만 승인합니다. `/app/plugins/knowledge-helper`에서 실제 문서 조회·개인 문서 생성·파일 가져오기·내보내기를 확인할 수 있습니다. AI는 관리자가 설정한 OpenAI-compatible 공급자를 사용합니다.

## 패키지 계약

ZIP 루트에는 `manifest.json`, `main.js`가 필수입니다. 선택적으로 `style.css`, JS/CSS/JSON/TXT/MD, PNG/JPEG/GIF/WebP/WOFF2 에셋을 포함할 수 있습니다. HTML/SVG, 원격 스크립트, 서버 바이너리는 허용하지 않습니다.

한도: ZIP 10MB, 압축 해제 20MB, 파일당 4MB, 총 200개 항목, manifest 64KB. 절대경로·상위경로·중복 이름·심볼릭 링크·특수 파일·과도한 압축률·CRC 오류를 거부합니다. 파일은 경로에 추출하지 않고 PostgreSQL에 저장되어 백업에 포함됩니다.

```json
{
  "id": "my-extension",
  "name": "나의 확장",
  "version": "1.0.0",
  "api_version": 1,
  "description": "확장 설명",
  "author": "개발 팀",
  "entry": "main.js",
  "capabilities": ["document:read"],
  "contributions": { "commands": [{ "id": "search", "title": "문서 찾기" }] }
}
```

같은 ID를 교체할 때 관리자는 `REPLACE`를 입력해야 합니다. 업데이트는 모든 워크스페이스 활성 상태와 권한을 초기화합니다. 코드 변경 후 반드시 재승인이 필요합니다. 사용 중지와 업그레이드는 저장한 개인 데이터를 삭제하지 않습니다.

## 보안 모델

플러그인 코드는 `allow-scripts`만 부여된 opaque-origin iframe 안의 Blob Worker에서 실행됩니다. iframe에는 신뢰된 선언형 UI 렌더러만 있습니다. Worker에는 `document`, cookie, DOM, 페이지 이동 API가 없어 자기 프레임 이동을 통한 정보 유출도 차단됩니다. CSP는 외부 연결, importScripts, 프레임, 폼, 원격 이미지·폰트와 임의 스크립트를 허용하지 않습니다. 임의 HTML과 JavaScript 평가 API를 제공하지 않습니다.

부모 브리지는 `event.source`와 매 실행마다 새로 생성한 nonce를 모두 검사합니다. `origin === "null"`만으로 신뢰하지 않습니다. 서버는 요청마다 설치 상태, 워크스페이스 승인, 현재 사용자 ACL, 문서/공간 범위, 작업 scope를 다시 확인합니다. 서비스 관리자도 다른 사용자의 개인 문서 권한을 우회하지 않습니다. AI API 키와 사용자 세션은 Worker에 전달하지 않습니다. 파일은 사용자 선택·실행·다운로드 확인을 거칩니다.

권한 취소 후 새 API는 즉시 거부됩니다. 진행 중 AI는 1초 간격으로 승인 상태를 검사하여 취소하고, 화면 실행은 최대 5초 안에 종료됩니다. 이미 정당하게 전달된 데이터는 회수할 수 없으므로 설치 전 코드 검토가 필요합니다. 플러그인은 신뢰되지 않은 입력을 처리하는 제3자 코드입니다. 무한 루프 등 과도한 자원 사용 시 사용자가 재실행하거나 관리자가 사용을 중지할 수 있습니다.

## 선언형 UI

`await madi.ready()` 후 등록 및 렌더링합니다. 타입 정의는 [madi.d.ts](madi.d.ts)에 있습니다.

```js
await madi.ready();
madi.registerCommand("search", async () => {
  const documents = await madi.documents.list({ q: "운영" });
  madi.ui.render({
    type: "stack",
    children: [
      { type: "heading", text: "운영 지식" },
      ...documents.map((doc) => ({
        type: "button",
        text: doc.title,
        onClick: () => madi.navigate("/app/documents/" + doc.id),
      })),
    ],
  });
});
```

지원 노드: stack, row, text, heading, button, input, textarea, select, checkbox, code, image, divider, badge. `image.asset`은 ZIP 내부 에셋 경로입니다. HTML 문자열, 임의 속성, 원격 링크를 렌더링하지 않습니다. 최대 500개 요소, 깊이 20, select 옵션 100개를 지원합니다. `onClick`, `onChange`는 Worker 콜백으로 전달됩니다. 입력 변경은 기본 `change` 이벤트(포커스 이동 또는 선택 완료) 시 전달됩니다.

## 권한

| 권한                           | 범위                                                                       |
| ------------------------------ | -------------------------------------------------------------------------- |
| document:read / document:write | 현재 워크스페이스의 접근 가능한 문서 조회 / 작성·수정·휴지통 이동          |
| database:read / database:write | 현재 워크스페이스와 공간 권한이 허용된 DB 조회 / 행 변경                   |
| ai:execute                     | 관리자 설정 AI의 실제 SSE; 워크스페이스 근거 조회에는 document:read도 필요 |
| storage:personal               | 플러그인·워크스페이스·사용자별 JSON 저장, 100개 키 × 64KB                  |
| ui:notify / ui:navigate        | 텍스트 알림 / ACL 확인된 서비스 내부 화면 이동                             |
| file:import / file:export      | 사용자 선택 파일 전달 / 확인 후 문서 다운로드                              |

수정 API는 문서 `version`을 전달하여 충돌을 방지합니다. API JSON 요청은 최대 2MB, 응답 4MB, 파일 UI는 1MB, 동시 요청 16개, 사용자·플러그인별 서버 요청은 분당 120회입니다. 초당 100회 넘는 Worker 메시지나 4초 이상의 무응답은 실행을 종료합니다. 이는 브라우저 메모리의 하드웨어 수준 격리나 절대 메모리 한도를 보장하지 않으므로 관리자의 코드 검토가 여전히 필요합니다. 오류는 Promise rejection으로 전달되므로 `try/catch`로 화면에 안내하세요. 개인 저장소는 비밀키 보관용이 아닙니다.

## 확장 등록

`documents.list`는 최근 100개 문서의 메타데이터와 최대 2,000자 `snippet`만 반환합니다. 전체 Markdown은 `documents.get({id})`로 명시적으로 요청하세요.

manifest `contributions`에 선언한 ID만 등록할 수 있습니다. `registerBlock`, `registerCommand`, `registerSidebar`, `registerMenu`, `registerImporter`, `registerExporter`, `registerAIProvider`를 제공합니다. 등록한 명령·사이드바·메뉴는 플러그인 실행 화면의 확장 영역에서 사용합니다. 애플리케이션 DOM과 전역 메뉴를 임의로 수정하지 않습니다.

Importer는 `{name,content}`를 받아 처리합니다. Exporter는 `{name,content}` 텍스트를 반환하며 사용자가 검토 후 저장합니다. manifest extensions에는 `.md`, `.txt`, `.json`, `.csv`, `.yaml`, `.yml` 중 허용할 형식을 선언합니다. `registerAIProvider`는 `provider:"workspace"`만 지원하며 별도 공급자 URL·비밀키를 등록하지 않습니다.

AI 직접 사용은 `madi.ai.chat({prompt}, chunk => ...)`로 실시간 `text`, `sources`를 받습니다. max_tokens는 관리자 설정 한도 및 최대 262144 이내에서 검증됩니다. 지원 여부와 실제 출력 한도는 연결한 모델에 달려 있습니다.

## Markdown 플러그인 블록

````markdown
```madi-plugin
{"plugin_id":"knowledge-helper","block_id":"summary-card","data":{"title":"팀의 결정","summary":"결정의 이유를 함께 남깁니다."}}
```
````

읽기 화면은 원본을 보존하며 사용자가 실행 버튼을 누를 때만 Worker를 시작합니다. 다른 사람이 작성한 문서가 열리자마자 플러그인 작업을 실행시키지 않습니다. 확장이 없거나 권한이 없으면 원본 JSON을 볼 수 있습니다.

## 검증

샌드박스의 Worker·CSP 동작은 [MDN Web Workers 안내](https://developer.mozilla.org/en-US/docs/Web/API/Web_Workers_API/Using_web_workers)의 Blob Worker 정책 상속 설명을 기준으로 설계하고 실제 Chromium에서 검증합니다.

```sh
go test ./internal/server -run 'TestPlugin|TestPostgresPlugin'
node tests/plugins-sandbox.mjs
node tests/plugins.mjs
```

PostgreSQL 통합 검증에는 격리 테스트 DB의 `MADI_TEST_POSTGRES_DSN`을 설정합니다. 브라우저 검증은 별도 개발 인스턴스에서만 수행하세요. 테스트가 실제 QA용 문서와 플러그인을 만듭니다.
