import { chromium } from '../../../tests/node_modules/playwright/index.mjs';
import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
const base=process.env.MADI_BASE_URL,doc=process.env.MADI_RUNBOOK_DOCUMENT;assert.ok(base&&doc);
const browser=await chromium.launch({headless:true}),context=await browser.newContext({viewport:{width:1440,height:1000}}),page=await context.newPage();
const issues=[],external=[],shots=fileURLToPath(new URL('../../../docs/screenshots/',import.meta.url));await mkdir(shots,{recursive:true});
page.on('pageerror',e=>issues.push(e.message));page.on('response',r=>{if(r.status()>=500)issues.push(`${r.status()} ${r.url()}`)});
await context.route('**/*',route=>{const url=route.request().url();if(!url.startsWith(base)&&!url.startsWith('data:')){external.push(url);return route.abort()};return route.continue()});
async function api(path,method='GET',data){const r=await context.request.fetch(base+'/api/v1'+path,{method,data,headers:{'X-Madi-Request':'1'}});assert.ok(r.ok(),`${method} ${path} ${r.status()} ${await r.text()}`);return r.json()}
async function capture(name){await page.evaluate(()=>window.scrollTo(0,0));await page.waitForFunction(()=>scrollY===0);await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));await page.screenshot({path:shots+name,fullPage:true})}
try{
 await api('/auth/login','POST',{email:'admin@example.test',password:'Integration-Test-Password-2026!'});
 await page.goto(base+'/admin/runbook');await page.getByRole('heading',{name:'격리 실행 · Runbook',exact:true}).waitFor();
 await page.getByRole('button',{name:/사내 AWX · 사용 · v1/}).click();
 await page.getByLabel('작업 이름',{exact:true}).fill('사내 표준 안전 점검');
 await page.getByLabel('실행 허용 역할',{exact:true}).selectOption('editor');
 await page.getByRole('button',{name:'실행기 저장',exact:true}).click();await page.getByRole('button',{name:/사내 AWX · 사용 · v2/}).waitFor();
 await page.getByRole('button',{name:'저장된 설정으로 연결·격리 검사',exact:true}).click();await page.getByText('사전 검사 결과 · 원격 작업은 실행하지 않았습니다',{exact:true}).waitFor();
 await capture('runbook-admin.png');
 await page.reload();await page.getByRole('button',{name:/사내 AWX · 사용 · v2/}).waitFor();
 await page.goto(base+`/app/documents/${doc}/runbook`);await page.getByRole('heading',{name:'격리 운영 절차',exact:true}).waitFor();
 await page.getByLabel('대상 *',{exact:true}).selectOption('production');await page.getByRole('button',{name:'운영 절차 저장',exact:true}).click();await page.getByRole('button',{name:'운영 절차 저장',exact:true}).waitFor({state:'visible'});await page.waitForFunction(()=>[...document.querySelectorAll('button')].some(b=>b.textContent.includes('운영 절차 저장')&&b.disabled));
 await page.getByRole('tab',{name:'검증 단계',exact:true}).click();await page.getByLabel('대상 *',{exact:true}).selectOption('production');await page.getByRole('button',{name:'운영 절차 저장',exact:true}).click();await page.waitForFunction(()=>[...document.querySelectorAll('button')].some(b=>b.textContent.includes('운영 절차 저장')&&b.disabled));
 await capture('runbook-document.png');
 await page.getByRole('button',{name:'검증 계획 준비',exact:true}).click();await page.waitForURL('**/app/runbook/executions/*');
 const id=new URL(page.url()).pathname.split('/').at(-1);await page.getByRole('heading',{name:'검증을 직접 확인하세요',exact:true}).waitFor();
 assert.equal(await page.getByRole('button',{name:'검토 요청',exact:true}).count(),0,'approval disabled must hide review');
 assert.equal(await page.getByRole('button',{name:'확인한 계획 실행',exact:true}).isEnabled(),false);
 await page.reload();await page.getByLabel('실행 확인 문구',{exact:true}).fill(`EXECUTE ${id}`);await page.getByRole('button',{name:'확인한 계획 실행',exact:true}).click();
 await page.waitForFunction(async id=>(await(await fetch('/api/v1/runbook/executions/'+id)).json()).status==='succeeded',id,{timeout:25000});
 await page.getByText('점검 완료',{exact:false}).first().waitFor({timeout:12000});
 const output=await page.locator('.runbook-output').innerText();assert.ok(output.includes('점검 완료'));assert.ok(!output.includes('test-runner-secret'));
 await capture('runbook-execution.png');
 await page.setViewportSize({width:390,height:844});await page.reload();await page.getByRole('heading',{name:'변경할 수 없는 실행 원본',exact:true}).waitFor();await page.getByText('점검 완료',{exact:false}).first().waitFor();
 assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),'mobile execution overflow');await capture('runbook-mobile.png');
 assert.equal((await api(`/documents/${doc}/runbook`)).last_tested_at!==null,true);
 assert.deepEqual(issues,[]);assert.deepEqual(external,[]);console.log('PASS Runbook admin save/preflight, typed select roundtrip, no approval workflow when disabled, explicit execution confirmation, durable output+refresh, exact last-tested, mobile, zero JS/HTTP500/external assets');
}catch(e){await page.screenshot({path:'/tmp/madi-runbook-browser-failure.png',fullPage:true}).catch(()=>{});console.error(await page.locator('body').innerText());console.error(issues,external);throw e}finally{await browser.close()}
