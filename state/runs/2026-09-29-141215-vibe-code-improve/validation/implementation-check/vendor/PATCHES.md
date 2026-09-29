# vendor/extension.core.js

`extension.core.js` is the upstream extension-host bundle (a rebranded [Roo-Code](https://github.com/RooCodeInc/Roo-Code) build, minified) that ships in the VSIX as `dist/extension.core.js`. Everything Vibe Code adds on top lives in `src/` and is bundled separately into `dist/extension.js`, which loads the core at runtime.

The core is treated as a vendored artifact: do not hand-edit it for features. Feature code belongs in `src/`.

## Hook contract

`src/extension.ts` installs `globalThis.__vibeCode` before requiring the core. The core calls two hooks:

| Hook | Where in the core | When |
|---|---|---|
| `mergeLocaleOverrides(resources)` | i18n module init (`ov.init({lng:"ko", ...})`) | While bundled translations are loaded; `resources` is the i18next language -> namespace -> keys tree and may be mutated in place. |
| `beforeCore(host)` | Start of `activate` (`async function M8d(e)`), right after the `Vibe Code` output channel is created | Awaited before telemetry, the sidebar provider, and the code-index manager are created. |
| `onTask(event)` | `Task.initiateTaskLoop` (`started`), the provider's `taskCompleted` / `taskAborted` listeners (`completed` with `tokenUsage`/`toolUsage`/`isSubtask`, `aborted`), and both `say("condense_context")` sites (`condensed` with `cost`) | Five fire-and-forget call sites. |
| `promptContext()` | End of the OBJECTIVE system-prompt template literal (`return\`====\n\nOBJECTIVE ...\``) | `${globalThis.__vibeCode?.promptContext?.()||""}` — evaluated on every system-prompt build, so the goal state stays fresh. |
| `onCommand(event)` | `execute_command` tool (`class Qwn ... execute()`) and the terminal runner (`async function G_s`) | Three call sites: `{phase:"denied"}` when `askApproval("command")` is refused, `{phase:"approved"}` right after approval (before the terminal runs), and `{phase:"exited", exitCode}` from `onShellExecutionComplete`. Fire-and-forget; never awaited. |

`host` passed to `beforeCore`:

```js
{ context: e, output: ND, pkg: Ls, contextProxy: f6, changeLanguage: VQn, getProvider: () => o }
```

- `context` – the `vscode.ExtensionContext`
- `output` – the shared output channel
- `pkg` – `{ publisher, name, version, outputChannel, sha }`
- `contextProxy` – the core `ContextProxy` class (`getInstance(context)` returns the singleton with `getValue/getValues/setValue/setProviderSettings`)
- `changeLanguage` – switches the core i18next language
- `getProvider` – returns the sidebar `ClineProvider` once the core has created it (undefined while `beforeCore` itself runs)

The TypeScript view of this contract is `src/core/host.ts`.

## Deviations from upstream that remain inside the core

These were part of the original rebrand and are kept in the vendored bundle because they touch upstream internals:

1. **Branding strings** – package name/IDs (`vibe-code.*`), command titles (`Explain with Vibe Code`, ...), and the `.vibe`/`.vibemodes`/`.vibeignore` file names.
2. **Korean-first i18n** – `ov.init({ lng: "ko", fallbackLng: "en", ... })` instead of the upstream English default.
3. **System prompt text** – the response-language rule (`You MUST respond exclusively in "..."`), the list of extension-owned state paths (`.vibe-code/journal/`, `.vibe-code/plans/`), and the "prefer local sources" guidance.
4. **The hook call sites** described above (`beforeCore`, `mergeLocaleOverrides`, three `onCommand` sites, five `onTask` sites, one `promptContext` site), replacing ~85 KB of inline feature code that now lives in `src/`.
5. **Removed upstream i18n bootstrap** – upstream called `initializeI18n(globalState.language ?? vscode.env.language)` in `activate`; the equivalent is `initializeLanguage` in `src/features/language.ts`, invoked through `beforeCore`.

## Refreshing the core

If a newer upstream build is adopted, re-apply items 1–5 and re-run `npm run check`. `scripts/build.mjs` copies this file to `dist/extension.core.js`; the extension host loads it from there, so its `__dirname`-relative assets (`dist/i18n`, `dist/*.wasm`, `dist/workers`, `dist/node_modules`) must stay in `dist/`.
