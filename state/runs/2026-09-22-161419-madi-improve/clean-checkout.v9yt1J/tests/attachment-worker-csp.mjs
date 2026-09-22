import {chromium} from 'playwright';
import assert from 'node:assert/strict';
const base=process.env.MADI_BASE_URL;if(!base)throw new Error('Isolated fixture URL required');
const browser=await chromium.launch({headless:true});const context=await browser.newContext();const page=await context.newPage();const external=[];
await context.route('https://**',route=>{external.push(route.request().url());return route.abort();});
try{
 const main=await page.goto(base+'/app');assert.ok(!main.headers()['content-security-policy'].includes('wasm-unsafe-eval'));
 await page.waitForFunction(()=>document.getElementById('result')?.textContent.startsWith('{'));
 const result=JSON.parse(await page.locator('#result').textContent());
 assert.deepEqual(result,{pdf:{wasm:true,dynamic:false,external:false},ordinary:{wasm:false,dynamic:false,external:false},pageWasm:false});
 assert.deepEqual(external,[]);
 const pdf=await context.request.get(base+'/assets/pdf.worker.min-csp.mjs');assert.ok(pdf.headers()['content-security-policy'].includes("'wasm-unsafe-eval'"));assert.ok(!pdf.headers()['content-security-policy'].includes("'unsafe-eval'"));
 const plugin=await context.request.get(base+'/plugin-sandbox.html');assert.ok(!plugin.headers()['content-security-policy'].includes('wasm-unsafe-eval'));assert.match(plugin.headers()['content-security-policy'],/connect-src 'none'/);
 const missing=await context.request.get(base+'/assets/nested/pdf.worker.min-missing.mjs');assert.ok(!missing.headers()['content-security-policy'].includes('wasm-unsafe-eval'));
 console.log('PASS PDF worker WASM only; JavaScript eval/external network blocked; ordinary worker/page/plugin CSP unchanged');
}finally{await browser.close();}
