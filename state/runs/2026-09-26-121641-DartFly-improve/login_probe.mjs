import vm from 'node:vm';
import { repoFile } from '/home/hkjang/.cache/auto-improve-wt/DartFly/test/js/load.mjs';

class El {
  listeners = {}; content = ''; disabled = false; hidden = true;
  elements = { email: { value: 'a@b.c' }, password: { value: 'pw' } };
  set textContent(v) { this.content = v; }
  get textContent() { return this.content; }
  addEventListener(t, l) { this.listeners[t] = l; }
  querySelector() { this.button ??= new El(); return this.button; }
  fire(t, e) { return this.listeners[t]?.(e); }
}
const form = new El(), error = new El(), version = new El();
const els = { '#login-form': form, '#login-error': error, '#version-info': version };
let nextLogin;
const ctx = vm.createContext({
  document: { querySelector: (s) => els[s] ?? new El() },
  window: { location: { search: '', replace(u) { ctx.__went = u; } } },
  URLSearchParams,
  fetch: async (url) => (String(url).includes('system/info') ? { ok: false } : nextLogin),
  shouldAttemptSilentSso: () => false,
  beginSilentSso: () => {},
  clearSilentSsoState: () => {},
  safeReturnTo: () => '/',
  setTimeout, clearTimeout, console,
});
vm.runInContext(repoFile('internal/webui/js/login.js').replace(/^import .*;\n/gm, ''), ctx);

const tick = () => new Promise((r) => setImmediate(r));

nextLogin = {
  ok: false, status: 429, headers: { get: () => '7' },
  json: async () => ({ title: '로그인 시도가 너무 많습니다. 7초 후 다시 시도하세요' }),
};
await form.fire('submit', { preventDefault() {} });
await tick(); await tick();
console.log('429 ->', JSON.stringify(error.textContent), 'buttonDisabled=', form.querySelector().disabled);

nextLogin = {
  ok: false, status: 502, headers: { get: () => null },
  json: async () => { throw new SyntaxError('Unexpected token < in JSON at position 0'); },
};
await form.fire('submit', { preventDefault() {} });
await tick(); await tick();
console.log('502 ->', JSON.stringify(error.textContent), 'buttonDisabled=', form.querySelector().disabled);
