export default {
 root: '/home/hkjang/.cache/auto-improve-wt/vibe-code',
 cacheDir: new URL('./vite-cache', import.meta.url).pathname,
 resolve: { alias: {vscode: '/home/hkjang/.cache/auto-improve-wt/vibe-code/tests/unit/vscode-stub.ts', vitest: new URL('./node_modules/vitest/dist/index.js', import.meta.url).pathname} },
 test: { include: ['tests/unit/**/*.test.ts'] }
};
