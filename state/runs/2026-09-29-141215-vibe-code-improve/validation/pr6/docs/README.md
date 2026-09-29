# Vibe Code 유지보수 문서

이 폴더는 `vibe-code` VS Code 확장을 계속 개발하고 릴리즈하기 위한 내부 문서입니다.

Vibe Code가 추가한 기능은 `src/`의 TypeScript 소스이고, 원본 확장 코어는 `vendor/extension.core.js`로 vendoring되어 있습니다. 유지보수의 기준 파일은 `package.json`, `src/`, `vendor/PATCHES.md`, `webview-ui/build`, `dist/i18n`, `assets`, `scripts/package-vsix.ps1` 입니다.

## 문서 목록

- [소스 분석](source-analysis.md): 현재 패키지 구조, 런타임 흐름, 주요 기능 위치
- [유지보수 가이드](maintenance-guide.md): 변경 절차, 검증, 릴리즈, 주의사항
- [개선 로드맵](improvement-roadmap.md): 목표에 맞는 추가 개발 아이디어와 우선순위
- [vibe-coders 프록시 연동](vibe-coders-proxy.md): Vibe Code 모델 호출을 `vibe-coders` OpenAI 호환 프록시로 보내는 방법
- [자율 목표 개발 워크플로](autonomous-goal-workflow.md): `/goal` 슬래시 커맨드 기반 장기 자율 개발 루프

## 빠른 명령

```powershell
npm run check
powershell -ExecutionPolicy Bypass -File scripts/verify-package.ps1
powershell -ExecutionPolicy Bypass -File scripts/package-vsix.ps1
powershell -ExecutionPolicy Bypass -File scripts/smoke-vscode-cli.ps1
powershell -ExecutionPolicy Bypass -File scripts/test-extension-host.ps1
```

패키지를 수정한 뒤에는 패키징 전후로 `verify-package.ps1`를 한 번씩 실행합니다. VSIX가 이미 있다면 스크립트가 내부 필수 파일 포함 여부까지 함께 확인합니다.

## 현재 릴리즈 산출물

- VSIX: `release/vibe-code-<version>.vsix` (버전은 `package.json` 기준, 현재 1.4.5)
- 확장 ID: `vibe-code.vibe-code`
- 기본 언어: 한국어 (`vibe-code.language = ko`)
