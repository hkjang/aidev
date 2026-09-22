import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import path from 'node:path';
import {chromium,expect} from 'playwright/test';
const base=process.env.MADI_BASE_URL,wid=process.env.MADI_WORKSPACE_ID,doc=process.env.MADI_DOCUMENT_ID,ann=process.env.MADI_HAS_PGVECTOR==='true';
const output=path.resolve(process.env.MADI_SCREENSHOT_DIR||'test-results/rag-generations');
const browser=await chromium.launch(),context=await browser.newContext({viewport:{width:1512,height:1000},locale:'ko-KR',reducedMotion:'reduce'}),page=await context.newPage();
const issues=[];page.on('pageerror',e=>issues.push(e.message));page.on('response',r=>{if(r.url().startsWith(base+'/api/')&&r.status()>=500)issues.push(`HTTP${r.status()} ${r.url()}`)});
async function api(url,method='GET',data,status=200){const r=await context.request.fetch(base+'/api/v1'+url,{method,data,headers:{'X-Madi-Request':'1'}});assert.equal(r.status(),status,`${url}: ${await r.text()}`);return r.json()}
async function shot(name){await mkdir(output,{recursive:true});await page.evaluate(()=>document.fonts.ready);await page.screenshot({path:path.join(output,name+'.png'),fullPage:page.viewportSize().width>500,animations:'disabled'})}
async function fits(){await expect.poll(()=>page.evaluate(()=>document.documentElement.scrollWidth-innerWidth),{message:'mobile no horizontal overflow'}).toBeLessThanOrEqual(1)}
async function verify(mode){await page.getByLabel('검증하고 전환할 모드',{exact:true}).selectOption(mode);await page.getByRole('checkbox',{name:/GPU 세대 검증 문서/}).check();await page.getByLabel('검증 질문',{exact:true}).fill('GPU 장애 운영 근거');await page.getByRole('checkbox',{name:/위 공급자·모델로 이 질문/}).check();await page.getByRole('button',{name:'실제 검증 실행',exact:true}).click();await expect(page.getByLabel('벡터 검증 결과',{exact:true})).toBeVisible({timeout:15000})}
async function transition(button){await expect(page.getByRole('button',{name:button,exact:true})).toBeDisabled();await page.getByRole('checkbox',{name:/검증은 선택한 문서와 질문 한 건/}).check();await page.getByRole('button',{name:button,exact:true}).click();await page.getByRole('dialog').getByRole('button',{name:'확인하고 계속',exact:true}).click();await expect(page.getByText('검증한 세대와 공급자 설정을 함께 전환했습니다. 현재 권한과 동의가 일치하는 자료만 검색합니다.',{exact:true})).toBeVisible()}
try{
 await api('/auth/login','POST',{email:'admin@example.test',password:'Integration-Test-Password-2026!'});
 await page.goto(base+'/app/search/generations');await page.getByLabel('워크스페이스 선택',{exact:true}).selectOption(wid);await page.goto(base+'/app/search/generations');
 await page.getByRole('heading',{name:'벡터 색인 세대',exact:true}).waitFor();await page.getByRole('button',{name:'새 세대',exact:true}).click();
 const create=page.getByRole('dialog',{name:'새 벡터 색인 세대',exact:true});await create.getByLabel('세대 이름',{exact:true}).fill('브라우저 새 모델 세대');await create.getByLabel('임베딩 모델',{exact:true}).fill('browser-generation');await create.getByLabel('고정 벡터 차원',{exact:true}).fill('3');
 await expect(create.getByRole('button',{name:'세대 만들기',exact:true})).toBeDisabled();await create.getByRole('checkbox',{name:/모델·차원·전송 주소를 확인/}).check();await shot('rag-generation-create');await create.getByRole('button',{name:'세대 만들기',exact:true}).click();
 await expect(page.getByRole('heading',{name:'브라우저 새 모델 세대',exact:true})).toBeVisible();
 let state=await api(`/workspaces/${wid}/rag-generations`),shadow=state.generations.find(g=>g.name==='브라우저 새 모델 세대'),baseline=state.active_id;assert.notEqual(shadow.id,baseline);assert.equal(shadow.index.consented_documents,0);
 await page.getByRole('button',{name:'세대별 동의·색인',exact:true}).click();const modal=page.getByRole('dialog',{name:'문서 검색 색인',exact:true});
 await expect(modal).toContainText('browser-generation');await expect(modal.getByRole('button',{name:'동의하고 색인 시작',exact:true})).toBeDisabled();
 await expect(modal.getByRole('checkbox',{name:/이 문서가 변경되면/})).not.toBeChecked();await expect(modal.getByRole('checkbox',{name:/위 재정렬 공급자에게도/})).not.toBeChecked();
 await modal.getByRole('checkbox',{name:/위 공급자에게 이 문서의 저장된 내용/}).check();await modal.getByRole('button',{name:'동의하고 색인 시작',exact:true}).click();
 await expect(modal.getByText('현재 문서와 동의 상태에 맞는 색인입니다.',{exact:true})).toBeVisible({timeout:20000});await shot('rag-generation-document-consent');await modal.getByRole('button',{name:'닫기',exact:true}).click();
 if(ann){await page.getByRole('button',{name:'HNSW 인덱스 준비',exact:true}).click();await page.getByRole('dialog').getByRole('button',{name:'확인하고 계속',exact:true}).click();await expect(page.getByText('HNSW 준비됨',{exact:true})).toBeVisible({timeout:20000})}
 await verify(ann?'verify':'exact');if(ann){await expect(page.getByLabel('벡터 검증 결과')).toContainText('100.0%');await expect(page.getByLabel('벡터 검증 결과')).toContainText('실제 HNSW 실행 계획 확인')}
 await shot('rag-generation-verified');
 // Effective policy changes invalidate a receipt and reset explicit consent.
 await api('/admin/settings','PUT',{rag_scan_limit:6000});await expect(page.getByLabel('벡터 검증 결과')).toHaveCount(0,{timeout:9000});await expect(page.getByRole('checkbox',{name:/위 공급자·모델로 이 질문/})).not.toBeChecked();
 await verify(ann?'verify':'exact');await transition('검증한 세대로 전환');
 await expect.poll(async()=>(await api(`/workspaces/${wid}/rag-generations`)).active_id).toBe(shadow.id);
 await page.setViewportSize({width:390,height:844});await page.evaluate(()=>scrollTo(0,0));await fits();await shot('rag-generations-mobile');
 await page.setViewportSize({width:1512,height:1000});await page.getByRole('navigation',{name:'색인 세대 목록'}).getByRole('button',{name:/기존 설정 기준 세대/}).click();
 await expect(page.getByLabel('벡터 검증 결과')).toHaveCount(0);await verify('exact');await shot('rag-generation-rollback');await transition('검증한 이전 세대로 복귀');
 await expect.poll(async()=>(await api(`/workspaces/${wid}/rag-generations`)).active_id).toBe(baseline);
 await page.getByRole('navigation',{name:'색인 세대 목록'}).getByRole('button',{name:/브라우저 새 모델 세대/}).click();await page.getByRole('button',{name:'이 세대 삭제',exact:true}).click();await page.getByRole('dialog').getByRole('button',{name:'확인하고 계속',exact:true}).click();
 await expect(page.getByText('선택한 세대의 설정·동의·검증 기록·파생 색인을 삭제했습니다. 원본 문서는 유지했습니다.',{exact:true})).toBeVisible();assert.equal((await api('/documents/'+doc)).version,1);
 assert.deepEqual(issues,[]);console.log(JSON.stringify({ok:true,ann,checks:['immutable-generation','document-specific-consent','real-index-job','actual-verification','changed-policy-reconsent','CAS-switch','explicit-reverified-rollback','safe-derived-delete','mobile390'],screenshots:output}));
}catch(e){await shot('failure').catch(()=>{});console.error(await page.locator('body').innerText());throw e}finally{await browser.close()}
