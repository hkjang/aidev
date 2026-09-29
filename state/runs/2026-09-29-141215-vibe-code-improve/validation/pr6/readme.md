# Vibe Code

Vibe Code is an AI-powered coding assistant for Visual Studio Code. It provides an in-editor chat workspace, code explanation and improvement commands, terminal command help, project document search, MCP server integration, and multi-provider model support.

## Features

- Chat-based coding workflow in the VS Code activity bar
- Context-aware code explanation, improvement, and task creation commands
- Terminal context capture, command explanation, and command repair helpers
- Document embedding search for project knowledge
- MCP server support for extensible tools
- Provider support for Anthropic, OpenAI, Google, AWS Bedrock, LM Studio, and compatible local or hosted APIs
- Korean and English localization

## Install From VSIX

1. Open VS Code.
2. Run `Extensions: Install from VSIX...`.
3. Select `release/vibe-code-<version>.vsix` (currently `1.4.5`).
4. Reload VS Code when prompted.

## Quick Start

1. Open the Vibe Code activity bar item.
2. Configure your preferred API provider and key in Vibe Code settings.
3. Select code and use the editor context menu for `Explain Code`, `Improve Code`, or `Add To Context`.
4. Use the chat panel for larger vibe coding tasks across the workspace.

## Settings

```json
{
	"vibe-code.language": "ko",
	"vibe-code.allowedCommands": ["npm test", "npm install", "git status"],
	"vibe-code.commandExecutionTimeout": 0
}
```

Korean is the default language. Source/package analysis, maintenance notes, and the improvement roadmap are documented in `docs/`.

Use `/goal <desired outcome>` in chat to start a persistent autonomous goal loop. Vibe Code stores the active goal in `.vibe-code/goals/current.md`, keeps work queues and verification logs, writes handoff notes under `.vibe-code/sessions/`, and shows the active goal with checklist progress in the VS Code status bar. The command palette also includes `Vibe Code: Open Current Goal`, `Vibe Code: Show Goal Status`, and `Vibe Code: Create Goal Handoff`.

## vibe-coders Proxy

Run `Vibe Code: Apply vibe-coders Proxy` to route Vibe Code model calls through the OpenAI-compatible `vibe-coders` proxy at `http://localhost:8080/v1`. Aggregation is handled by the proxy when model requests pass through it.

The active provider profile before applying the proxy is remembered. Use `Vibe Code: Show vibe-coders Proxy Status` to inspect the current route without making network or model calls, and `Vibe Code: Restore Previous Provider` to switch back.

## Development

Vibe Code's own features live in `src/` (TypeScript) and are bundled into `dist/extension.js`. The upstream core bundle is vendored as `vendor/extension.core.js` and staged next to it as `dist/extension.core.js`; see `vendor/PATCHES.md` for the hook contract.

```powershell
npm install
npm run check   # typecheck + unit tests + build
npm run watch   # rebuild dist/extension.js on change
```

`dist/` is not tracked in git. On a fresh clone, restore the runtime assets (`node_modules`, `i18n`, `*.wasm`, `workers`) from a release VSIX before building:

```powershell
node scripts/restore-dist-assets.mjs --from release/vibe-code-1.4.5.vsix
```

CI (`.github/workflows/ci.yml`) runs `npm run check` on every push and packages a VSIX on Windows from the latest GitHub release's runtime assets.

## Build The VSIX

Run `npm run build` first, then:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/package-vsix.ps1
powershell -ExecutionPolicy Bypass -File scripts/verify-package.ps1
powershell -ExecutionPolicy Bypass -File scripts/smoke-vscode-cli.ps1
powershell -ExecutionPolicy Bypass -File scripts/test-extension-host.ps1
```

The output is written to `release/vibe-code-<version>.vsix`.
