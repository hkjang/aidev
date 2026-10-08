import { createServer } from 'node:http';
import { readFileSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import { createRequire } from 'node:module';
import assert from 'node:assert/strict';
const root = process.cwd();
const require = createRequire(`${root}/web/package.json`);
const ts = require('typescript');
const { chromium } = require('@playwright/test');
const compiled = {};
for (const revision of ['head', 'base']) {
  const source = revision === 'head' ? readFileSync(`${root}/web/src/lib/api.ts`, 'utf8') : execFileSync('git', ['show', 'origin/main:web/src/lib/api.ts'], {encoding:'utf8'});
  compiled[revision] = ts.transpileModule(source, {compilerOptions:{target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.ESNext}}).outputText;
}
let activeMode = '', currentState = { closed: false };
const server = createServer((req, res) => {
  if (req.url === '/') { res.end('<!doctype html><title>SSE review</title>'); return; }
  const moduleMatch = req.url.match(/^\/(head|base)\.js$/);
  if (moduleMatch) { res.setHeader('Content-Type','text/javascript'); res.end(compiled[moduleMatch[1]]); return; }
  res.setHeader('Content-Type','text/event-stream');
  req.resume();
  const requestState = currentState;
  res.on('close', () => { requestState.closed = true; });
  if (activeMode === 'server-error') res.write('event: error\ndata: {"message":"fixture failure"}\n\n');
  else if (['agent', 'legacy'].includes(activeMode)) res.end('event: error\ndata: {"message":"fixture failure"}\n\nevent: done\ndata: {"status":"FAILED"}\n\n');
  else { res.write('data: {"delta":"안녕"}\n\n'); if (activeMode === 'normal') res.end(); }
});
await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
let browser;
try {
  browser = await chromium.launch({headless:true,executablePath:'/home/hkjang/.cache/ms-playwright/chromium-1243/chrome-linux64/chrome'});
  for (const revision of ['head','base']) {
    for (const mode of ['normal','server-error','callback-error','abort','agent','legacy']) {
      activeMode = mode; currentState = { closed: false };
      const page = await browser.newPage();
      await page.goto(`http://127.0.0.1:${server.address().port}/`);
      const result = await page.evaluate(async ({revision,mode}) => {
        const {apiClient,ApiError} = await import(`/${revision}.js`);
        let response;
        const nativeFetch=globalThis.fetch.bind(globalThis);
        globalThis.fetch=async (...args)=>{response=await nativeFetch(...args);return response;};
        const controller=new AbortController();
        const sentinel=new Error('callback fixture');
        const events=[],deltas=[];
        let caught;
        const onDelta=value=>{deltas.push(value);if(mode==='callback-error')throw sentinel;if(mode==='abort')controller.abort();};
        let timer;
        try {
          const work=mode==='agent' ? apiClient.streamAgentRun({prompt:'fixture',onEvent:e=>events.push(e.type)}) : mode==='legacy' ? apiClient.streamLegacyCodeRoom({roomId:'fixture',modelId:1,message:'fixture',onEvent:e=>events.push(e.event)}) : apiClient.streamChat({prompt:'fixture',signal:controller.signal,onDelta});
          await Promise.race([work,new Promise((_,reject)=>{timer=setTimeout(()=>reject(new Error('timeout')),3000);})]);
        } catch(error) {caught=error;} finally {clearTimeout(timer);}
        return {locked:response?.body.locked,message:caught?.message,errorName:caught?.name,identity:caught===sentinel,apiError:caught instanceof ApiError,events,deltas};
      },{revision,mode});
      if(revision==='head') {
        const deadline=Date.now()+3000;
        while(!currentState.closed && Date.now()<deadline) await new Promise(r=>setTimeout(r,20));
        assert.equal(result.locked,false); assert.equal(currentState.closed,true);
      } else assert.equal(result.locked,true);
      if(mode==='callback-error') assert.equal(result.identity,true);
      if(mode==='abort') assert.equal(result.errorName,'AbortError');
      if(['server-error','agent','legacy'].includes(mode)) {assert.equal(result.apiError,true);assert.equal(result.message,'fixture failure');}
      if(['agent','legacy'].includes(mode)) assert.deepEqual(result.events,['error','done']);
      if(mode==='normal') assert.deepEqual(result.deltas,['안녕']);
      console.log(JSON.stringify({revision,mode,...result,serverResponseClosed:currentState.closed}));
      await page.close();
    }
  }
} finally {await browser?.close();server.closeAllConnections();await new Promise(resolve=>server.close(resolve));}
