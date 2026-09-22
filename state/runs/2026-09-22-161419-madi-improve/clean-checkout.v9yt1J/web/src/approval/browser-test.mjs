import { chromium } from '../../../tests/node_modules/playwright/index.mjs';
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
const base=process.env.MADI_BASE_URL;assert.ok(base);
const browser=await chromium.launch({headless:true});
const screenshots=fileURLToPath(new URL('../../../docs/screenshots/',import.meta.url));await mkdir(screenshots,{recursive:true});
const contexts=[];const issues=[],external=[];
async function client(email,password){const context=await browser.newContext();contexts.push(context);const page=await context.newPage();page.on('pageerror',e=>issues.push(e.message));page.on('response',r=>{if(r.status()>=500)issues.push(`${r.status()} ${r.url()}`)});await context.route('**/*',route=>{if(!route.request().url().startsWith(base)&&!route.request().url().startsWith('data:')){external.push(route.request().url());return route.abort()};return route.continue()});async function api(path,method='GET',data){const response=await context.request.fetch(base+'/api/v1'+path,{method,data,headers:{'X-Madi-Request':'1'}});assert.ok(response.ok(),`${method} ${path}: ${response.status()} ${await response.text()}`);return response.json()};await api('/auth/login','POST',{email,password});return{context,page,api}}
let admin,lead,security;
try{
 admin=await client('admin@example.test','Integration-Test-Password-2026!');const wid=(await admin.api('/workspaces'))[0].id;
 const people=[];
 for(const name of ['lead','security']){const u=await admin.api('/admin/users','POST',{email:`${name}@example.test`,name:name==='lead'?'운영 팀장':'보안 담당',password:'Approval-browser-2026!',role:'editor'});await admin.api(`/workspaces/${wid}/members`,'PUT',{email:u.email,role:'viewer'});people.push(u)}
 lead=await client('lead@example.test','Approval-browser-2026!');security=await client('security@example.test','Approval-browser-2026!');
 await admin.page.goto(base+'/admin/approvals');await admin.page.getByRole('heading',{name:'검토·승인 정책',exact:true}).waitFor();
 await admin.page.getByLabel('서비스 검토·승인 사용',{exact:true}).check();await admin.page.waitForFunction(async()=>(await(await fetch('/api/v1/admin/settings')).json()).approval_enabled);
 await admin.page.getByRole('button',{name:'정책 만들기',exact:true}).click();const dialog=admin.page.getByRole('dialog');
 await dialog.getByLabel('정책 이름',{exact:true}).fill('출시 운영 검토');await dialog.getByLabel('워크스페이스',{exact:true}).selectOption(wid);
 await dialog.getByLabel('1단계 이름',{exact:true}).fill('운영 및 보안');await dialog.getByLabel('1단계 1번 대상 유형',{exact:true}).selectOption('user');await dialog.getByLabel('검토 사용자',{exact:true}).selectOption(people[0].id);
 await dialog.getByRole('button',{name:'병렬 검토 대상 추가',exact:true}).click();await dialog.getByLabel('1단계 2번 대상 유형',{exact:true}).selectOption('user');await dialog.getByLabel('검토 사용자',{exact:true}).nth(1).selectOption(people[1].id);await dialog.getByRole('button',{name:'정책 저장',exact:true}).click();await dialog.waitFor({state:'hidden'});
 await admin.page.reload();await admin.page.getByText('출시 운영 검토',{exact:true}).waitFor();
 const doc=await admin.api('/documents','POST',{workspace_id:wid,title:'운영 전환 검토',markdown:'# 정확한 검토 원본\n\n원본은 변경하지 않습니다.',visibility:'workspace'});
 await admin.page.goto(base+'/app/documents/'+doc.id);await admin.page.getByRole('button',{name:'현재 버전 검토 요청',exact:true}).click();
 await admin.page.getByRole('button',{name:'검토 원본·결정 보기',exact:true}).waitFor();let state=await admin.api(`/documents/${doc.id}/approval`);const requestID=state.request.id;
 await lead.page.goto(base+'/app/approvals?request='+requestID);await lead.page.getByRole('dialog').waitFor();await lead.page.getByText('원본은 변경하지 않습니다.',{exact:false}).first().waitFor();
 await lead.page.screenshot({path:screenshots+'approval-review.png',fullPage:true});
 assert.equal(await lead.page.getByRole('button',{name:'검토 승인',exact:true}).isEnabled(),false);
 await lead.page.getByLabel('이 요청의 고정된 원본과 승인 단계를 확인했습니다.',{exact:true}).check();await lead.page.getByLabel('검토 의견',{exact:true}).fill('운영 검증 완료');await lead.page.getByRole('button',{name:'검토 승인',exact:true}).click();await lead.page.getByRole('dialog').waitFor({state:'hidden'});
 state=await admin.api(`/documents/${doc.id}/approval`);assert.equal(state.request.status,'pending');assert.equal(state.decisions.length,1);
 await lead.page.screenshot({path:screenshots+'approval-inbox.png',fullPage:true});
 await security.page.goto(base+'/app/approvals?request='+requestID);await security.page.getByRole('dialog').waitFor();await security.page.reload();await security.page.getByRole('dialog').waitFor();
 await security.page.getByLabel('이 요청의 고정된 원본과 승인 단계를 확인했습니다.',{exact:true}).check();await security.page.getByRole('button',{name:'검토 승인',exact:true}).click();await security.page.getByRole('dialog').waitFor({state:'hidden'});assert.equal((await admin.api('/documents/'+doc.id)).status,'published');
 console.log('PASS Korean administrator policy selects, parallel read-only reviewers, frozen snapshot confirmation and refresh deep-link');
 await admin.api(`/documents/${doc.id}/approval`,'POST',{action:'submit'});state=await admin.api(`/documents/${doc.id}/approval`);
 await lead.page.goto(base+'/app/approvals?request='+state.request.id);await lead.page.getByLabel('이 요청의 고정된 원본과 승인 단계를 확인했습니다.',{exact:true}).check();
 let latest=await admin.api('/documents/'+doc.id);await admin.api('/documents/'+doc.id,'PUT',{version:latest.version,markdown:'검토 이후 수정된 원본'});
 await lead.page.getByRole('dialog').getByText('검토 요청 후 대상 내용이 변경되었습니다. 새 버전으로 다시 검토를 요청하세요',{exact:true}).waitFor({timeout:12000});assert.equal(await lead.page.getByRole('button',{name:'검토 승인',exact:true}).count(),0);
 console.log('PASS open review detects source mutation and removes approval controls');
 latest=await admin.api('/documents/'+doc.id);await admin.api('/documents/'+doc.id,'PUT',{version:latest.version,visibility:'private'});
 await lead.page.getByRole('dialog').getByText('대상을 찾을 수 없습니다.',{exact:false}).waitFor({timeout:12000}).catch(async()=>{await lead.page.waitForFunction(()=>!document.querySelector('.approval-snapshot'),null,{timeout:12000})});assert.equal(await lead.page.locator('.approval-snapshot').count(),0);
 await admin.page.setViewportSize({width:390,height:844});await admin.page.goto(base+'/admin/approvals');await admin.page.getByRole('heading',{name:'검토·승인 정책',exact:true}).waitFor();assert.ok(await admin.page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth+1),'mobile admin overflow');
 await admin.page.screenshot({path:screenshots+'approval-admin-mobile.png',fullPage:true});await admin.page.setViewportSize({width:1440,height:1000});await admin.page.waitForFunction(()=>{const sidebar=document.querySelector('.sidebar');return sidebar&&sidebar.getBoundingClientRect().left>=-1});await admin.page.screenshot({path:screenshots+'approval-admin.png',fullPage:true});
 assert.deepEqual(issues,[]);assert.deepEqual(external,[]);console.log('PASS ACL revocation hides frozen content, mobile layout, no JS/HTTP500/external assets');
}catch(e){for(const [name,c] of [['admin',admin],['lead',lead],['security',security]])if(c){await c.page.screenshot({path:`/tmp/madi-approval-${name}-failure.png`,fullPage:true}).catch(()=>{});console.error(name,await c.page.locator('body').innerText())}console.error('ISSUES',issues);throw e}finally{await browser.close()}
