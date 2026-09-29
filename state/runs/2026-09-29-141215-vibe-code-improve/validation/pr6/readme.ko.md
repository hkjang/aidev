# Vibe Code (한국어 가이드)

Visual Studio Code용 자율 AI 코딩 어시스턴트. 채팅 기반 워크플로우, 코드 설명/개선/수정, 터미널 보조, 프로젝트 문서 검색, MCP 서버, 멀티 LLM 프로바이더를 한국어 환경에 최적화하여 제공합니다.

## 주요 기능

- 액티비티 바 사이드바에서 채팅 기반 자율 코딩
- 컨텍스트 인식 `코드 설명`, `코드 개선`, `코드 수정`, `컨텍스트에 추가` 명령
- 터미널 컨텍스트 캡처, 명령어 설명/수정 도우미
- 프로젝트 문서 임베딩 검색 (LanceDB)
- MCP 서버 통합 (Model Context Protocol)
- 멀티 LLM 프로바이더: Anthropic, OpenAI, Google, AWS Bedrock, OpenRouter, Groq, Mistral, LM Studio, Ollama, VS Code LM API 등
- 한국어 우선 UI/응답, 톤 선택 (격식체/친근체/간결)

## VSIX로 설치

1. VS Code 실행
2. 명령 팔레트(`Ctrl+Shift+P`) → `Extensions: Install from VSIX...`
3. `release/vibe-code-<버전>.vsix` 선택 (현재 `1.4.5`)
4. 메시지에 따라 VS Code 다시 로드

## 개발

Vibe Code 자체 기능은 `src/`의 TypeScript 소스이며 `npm run build`로 `dist/extension.js`에 번들됩니다. 원본 확장 코어는 `vendor/extension.core.js`로 분리되어 있고 훅 계약은 `vendor/PATCHES.md`에 있습니다.

```powershell
npm install
npm run check   # typecheck + 단위 테스트 + 빌드
npm run watch   # src/ 변경 시 자동 빌드
node scripts/restore-dist-assets.mjs --from release/vibe-code-1.4.5.vsix   # 새 clone: dist/ 런타임 자산 복원
```

## 개발/유지보수 문서

현재 패키지 구조 분석, 릴리즈 절차, 향후 개선 로드맵은 `docs/` 폴더에 정리되어 있습니다.

- `docs/source-analysis.md`
- `docs/maintenance-guide.md`
- `docs/improvement-roadmap.md`
- `docs/vibe-coders-proxy.md`

## vibe-coders 프록시 연동

명령 팔레트에서 `Vibe Code: vibe-coders 프록시 적용`을 실행하면 Vibe Code provider profile이 `http://localhost:8080/v1` OpenAI 호환 프록시를 향하도록 저장됩니다. 이후 모델 호출은 `vibe-coders`를 통과하며 사용량/토큰/비용 집계는 proxy 쪽에서 자동 처리됩니다.

프록시 적용 전 provider profile은 복원용으로 기억됩니다. 현재 라우팅 상태만 확인하려면 `Vibe Code: vibe-coders 프록시 상태 보기`를 실행하세요. 이 명령은 네트워크 요청이나 모델 호출을 만들지 않습니다. 원래 설정으로 돌아가려면 `Vibe Code: 이전 프로바이더로 복원`을 실행하세요.

릴리즈 검증 명령:

```powershell
npm run check
powershell -ExecutionPolicy Bypass -File scripts/package-vsix.ps1
powershell -ExecutionPolicy Bypass -File scripts/verify-package.ps1
powershell -ExecutionPolicy Bypass -File scripts/smoke-vscode-cli.ps1
powershell -ExecutionPolicy Bypass -File scripts/test-extension-host.ps1
```

## 빠른 시작

1. 좌측 액티비티 바에서 **Vibe Code** 아이콘 클릭
2. 사이드바 우상단 톱니바퀴(설정) → 사용할 API 프로바이더와 키 입력
3. 코드 선택 후 우클릭 → `코드 설명` / `코드 개선` 등 사용
4. 큰 작업은 사이드바 채팅으로 자연어 지시 ("이 함수에 단위 테스트 작성해줘")

## 한국어 환경 설정

VS Code `settings.json`에서:

```json
{
  "vibe-code.language": "ko",
  "vibe-code.tone": "casual",
  "vibe-code.commandExecutionTimeout": 300
}
```

- `vibe-code.language`: `ko` | `en` (기본 `ko`)
- `vibe-code.tone`: `formal` (격식체) | `casual` (친근체, 기본) | `terse` (간결)

## 번역 오버라이드

UI 번역을 직접 수정하고 싶다면:

```
~/.vibe-code/locales/ko/common.json
```

파일을 만들고 덮어쓸 키만 작성하세요. 확장 활성화 시 자동 딥머지됩니다.

예시:
```json
{
  "buttons": {
    "approve": "승인",
    "reject": "거부"
  }
}
```

## 자율 에이전트 모드

사이드바 상단의 모드 선택기에서 자율성 등급 선택:

- **Ask** — 질문만 답변, 코드 수정 안 함
- **Code** — 일반 코딩 작업 (기본)
- **Architect** — 설계/계획 우선
- **Debug** — 버그 추적 모드

## 사용 팁

- **컨텍스트 자동 압축**: 토큰이 컨텍스트의 75%에 도달하면 자동 요약 (한국어 토큰 효율 맞춰 조정됨)
- **체크포인트**: 큰 작업 전 자동 스냅샷 — 잘못되면 롤백 가능
- **`@` 멘션**: `@파일경로`, `@폴더`, `@URL`, `@git` 등으로 컨텍스트 추가
- **사용자 정의 모드**: `.vibe-code/modes/` 폴더에 직접 정의
- **자동 승인 + 타이머**: 사이드바 설정 → 자동 승인 → 후속 질문에 5~30초 타이머를 걸어두면 화면을 안 보고 있을 때 기본 선택지로 자동 진행
- **기본 모드 architect**: 새 워크스페이스는 architect 모드로 시작 — 큰 변경 전 계획을 먼저 세움
- **병렬 도구 호출**: 설정 → Experimental → Multiple Native Tool Calls 활성 시, 독립적인 파일 읽기 등을 한 번에 병렬 처리
- **장기 목표 루프**: `/goal <원하는 결과>`로 시작하면 `.vibe-code/goals/current.md`에 목표/완료 기준/작업 큐/검증 로그를 저장하며 오래 이어지는 자율 개발을 진행. active goal은 진행률과 함께 상태바에 표시되고, 명령 팔레트의 `Vibe Code: 현재 목표 열기`, `Vibe Code: 목표 상태 보기`, `Vibe Code: 목표 핸드오프 생성`, `Vibe Code: 최신 계획 열기`, `Vibe Code: 현재 계획 진행`, `Vibe Code: 현재 계획 상태 변경`, `Vibe Code: 현재 계획 우선순위 변경`, `Vibe Code: 현재 계획 목표 연결`, `Vibe Code: 완료 계획 보관`, `Vibe Code: 보관 계획 복원`, `Vibe Code: 현재 계획 선택`, `Vibe Code: 최고 우선순위 계획 선택`, `Vibe Code: 계획 목록 보기`, `Vibe Code: 계획 이력 보기`, `Vibe Code: 목표-계획 연결 보기`와 액티비티 바의 `현재 목표` / `현재 계획` / `목표-계획 맵` 뷰로 이어서 작업할 수 있습니다. 계획 목록과 linked plan은 우선순위 `P0 -> P3`로 정렬되며, `목표-계획 맵`에서 linked plan을 클릭하면 active plan 선택 또는 archive 복원이 바로 수행됩니다.

## 문제 해결

| 증상 | 해결 |
|---|---|
| 사이드바가 안 보임 | `Ctrl+Shift+P` → "View: Reset View Locations" |
| 응답이 영어로 옴 | `vibe-code.language` 확인, 한 번 재로드 |
| 활성화 시 오류 | 출력 패널 → "Vibe Code" 채널 확인 |
| 토큰 비용 추적 | 상태바 우측 `$(rocket) Vibe` 위젯 |

## 오프라인 / 폐쇄망 운영 가이드

이 확장은 폐쇄망에서도 동작하도록 설계되었습니다. 인터넷 의존이 있는 기능은 자동 감지하고, AI 에이전트도 OFFLINE-FIRST 정책을 따릅니다.

### 권장 LLM 구성 (인터넷 미사용)

| 옵션 | 설명 | 모델 추천 |
|---|---|---|
| **Ollama** | 가장 간편. `ollama serve` 후 endpoint `http://localhost:11434` | `qwen2.5-coder:14b`, `deepseek-coder-v2`, `gemma2:9b`, `eeve-korean-10.8b` |
| **LM Studio** | GUI로 모델 다운/관리. OpenAI 호환 API 자동 노출 | 위와 동일 |
| **vLLM** | 서버급 추론, multi-GPU | `Qwen/Qwen2.5-Coder-32B-Instruct` |
| **llama.cpp 서버** | 경량/단일 머신 | GGUF 양자화 모델 |
| **사내 LLM 게이트웨이** | 회사가 직접 호스팅한 OpenAI 호환 endpoint | (회사 모델) |

사이드바 → 설정 → API Provider → **OpenAI Compatible** 또는 **Ollama** 선택 후 endpoint 입력.

### 프록시 / 인증서

사내 프록시 환경에서는:

```jsonc
// VS Code settings.json
{
  "http.proxy": "http://proxy.company.local:8080",
  "http.proxyStrictSSL": false,
  "http.proxyAuthorization": null
}
```

또는 환경 변수로:
```powershell
$env:HTTPS_PROXY = "http://proxy.company.local:8080"
$env:NODE_EXTRA_CA_CERTS = "C:\certs\company-root-ca.pem"
```

self-signed 인증서가 있는 경우 `NODE_EXTRA_CA_CERTS` 에 CA 번들 경로를 지정하면 fetch / axios / undici 모두 인식합니다.

### 외부 통신이 필요한 기능 vs 오프라인 가능

| 기능 | 인터넷 필요 | 비고 |
|---|---|---|
| LLM 호출 | ✅ 외부 API / ❌ 로컬 모델 | Ollama/LM Studio 사용 시 불필요 |
| 코드베이스 임베딩 인덱싱 | ❌ | 로컬 LanceDB |
| Tree-sitter 코드 분석 | ❌ | WASM 번들 포함 |
| MCP 서버 (npx) | ✅ | 미리 설치된 경로 사용 가능 (`.vibe-code/mcp-recommendations.json` 참고) |
| WebFetch / 브라우저 자동화 | ✅ | 사용자가 명시적으로 요청할 때만 |
| 자동 업데이트 | ❌ (로컬 release/) | `vibe-code.updateSourcePath` 로 사내 공유 폴더 지정 가능 |
| 텔레메트리 | 기본 OFF | 한국 환경 default disabled |

### 사내 자동 업데이트

`settings.json` 에서:
```json
{
  "vibe-code.updateSourcePath": "\\\\fileserver\\dev-tools\\vibe-code"
}
```

매일 한 번 해당 경로에서 더 높은 버전의 VSIX 를 발견하면 알림이 뜹니다.

## 한국어 코드 환경 권장 설정

### 폰트
한글과 영문이 같은 셀폭을 가지는 폰트가 코드 가독성을 크게 높입니다.

| 폰트 | 특징 |
|---|---|
| **D2Coding** | 네이버 무료 코딩 폰트, 한글 가독성 최상 |
| **Pretendard Coding** | 모던한 디자인, 합자(ligature) 지원 |
| **Sarasa Mono K** | 한국어 + 일본어 + 영문 균형 |
| **JetBrains Mono** + Noto Sans KR | 영문 합자 + 한글 fallback |

`settings.json`:
```json
{
  "editor.fontFamily": "'D2Coding', 'Consolas', monospace",
  "editor.fontLigatures": true,
  "terminal.integrated.fontFamily": "'D2Coding', monospace"
}
```

### 한글 입력기 (IME)
- **Windows**: Win+Space 로 영/한 토글. VS Code 채팅창에서 한글 입력 시 변환 중인 글자 보존이 잘 됨.
- **macOS**: Caps Lock 한/영 전환 권장. 시스템 환경설정 → 키보드 → 입력 소스.
- **Linux**: fcitx5 + Hangul 모듈 권장 (ibus 도 가능).

### 단어 선택 (한글)
한글은 영문보다 단어 경계가 다르므로 VS Code 의 단어 선택이 어색할 수 있습니다.

```json
{
  "editor.wordSeparators": "`~!@#$%^&*()-=+[{]}\\|;:'\",.<>/?，。、；："
}
```

### 줄 너비
한글은 시각 폭이 영문 대비 1.5~2배입니다. 한국어 주석이 많은 파일은 `editor.rulers` 를 100~110 으로 늘리는 것이 편합니다.

```json
{
  "editor.rulers": [100, 120]
}
```

## 라이선스

LICENSE.txt 참고.
