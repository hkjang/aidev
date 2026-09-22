import { defineConfig, type Plugin } from "vite";
import react from "@vitejs/plugin-react";
import { pdfjsAssets } from "./vite-pdfjs-assets";
function offlineAssets(): Plugin {
  return {
    name: "madi-offline-static-only",
    enforce: "post",
    generateBundle: {
      order: "post",
      handler(_, bundle) {
        const assets = Object.keys(bundle)
          .filter(
            (name) => name.startsWith("assets/") || name === "offline.html",
          )
          .map((name) => "/" + name);
        assets.push("/favicon.svg");
        const version = assets
          .join("|")
          .split("")
          .reduce(
            (hash, char) =>
              Math.imul(hash ^ char.charCodeAt(0), 16777619) >>> 0,
            2166136261,
          )
          .toString(16);
        this.emitFile({
          type: "asset",
          fileName: "sw.js",
          source: `const CACHE='madi-static-${version}';const ASSETS=${JSON.stringify(assets)};const ALLOWED=new Set(ASSETS);self.addEventListener('install',event=>event.waitUntil(caches.open(CACHE).then(cache=>cache.addAll(ASSETS))));self.addEventListener('activate',event=>event.waitUntil(Promise.all([caches.keys().then(keys=>Promise.all(keys.filter(key=>key.startsWith('madi-static-')&&key!==CACHE).map(key=>caches.delete(key)))),self.clients.claim()])));self.addEventListener('fetch',event=>{const url=new URL(event.request.url);if(url.origin!==self.location.origin||event.request.method!=='GET')return;if(ALLOWED.has(url.pathname)){event.respondWith(caches.open(CACHE).then(cache=>cache.match(url.pathname).then(value=>value||fetch(event.request))));return;}if(event.request.mode==='navigate'&&(url.pathname==='/app'||url.pathname.startsWith('/app/')||url.pathname==='/offline.html'))event.respondWith(fetch(event.request).catch(()=>caches.open(CACHE).then(cache=>cache.match('/offline.html')).then(value=>value||new Response('서버 연결을 확인하세요.',{status:503}))));});`,
        });
      },
    },
  };
}
// web/dist/.gitkeep is tracked so `go build ./...` works on a clean checkout:
// `//go:embed all:dist` (web/embed.go) rejects a missing directory. Vite empties
// dist/ before every build, so re-emit the placeholder to keep the tree clean.
function keepDistPlaceholder(): Plugin {
  return {
    name: "madi-keep-dist-placeholder",
    generateBundle() {
      this.emitFile({ type: "asset", fileName: ".gitkeep", source: "" });
    },
  };
}
export default defineConfig({
  plugins: [react(), pdfjsAssets(), offlineAssets(), keepDistPlaceholder()],
  server: {
    proxy: {
      "/api": "http://localhost:8080",
      "/auth": "http://localhost:8080",
      "/healthz": "http://localhost:8080",
    },
  },
  build: {
    chunkSizeWarningLimit: 1300,
    rollupOptions: { input: { main: "index.html", offline: "offline.html" } },
  },
});
