import { apiClient } from '/home/hkjang/.cache/auto-improve-wt/qurio/web/src/lib/api.ts';
globalThis.localStorage = { getItem: () => null };
globalThis.document = { cookie: '' };
const encoder = new TextEncoder();
let violations = 0;
for (const mode of ['normal', 'server-error', 'callback-error', 'read-error']) {
  let cancelled = 0;
  const sentinel = new Error(mode);
  const body = new ReadableStream({
    start(controller) {
      if (mode === 'read-error') { controller.error(sentinel); return; }
      controller.enqueue(encoder.encode(mode === 'server-error'
        ? 'event: error\ndata: {"message":"provider stopped"}\n\n'
        : 'data: {"delta":"안녕"}\n\n'));
      if (mode === 'normal') controller.close();
    },
    cancel() { cancelled++; },
  });
  globalThis.fetch = async () => new Response(body, { headers: { 'Content-Type': 'text/event-stream' } });
  const deltas = [];
  let error;
  try {
    await apiClient.streamChat({ prompt: 'probe', onDelta(value) {
      if (mode === 'callback-error') throw sentinel;
      deltas.push(value);
    } });
  } catch (caught) { error = caught; }
  console.log(JSON.stringify({ mode, cancelled, locked: body.locked, deltas, error: error?.message, originalError: error === sentinel }));
  if (body.locked || (['server-error', 'callback-error'].includes(mode) && cancelled !== 1)) violations++;
  // Dispose of the still-open underlying source by process exit; no network was opened.
}
process.exitCode = violations ? 1 : 0;
