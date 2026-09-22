import {chromium} from 'playwright';
import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import path from 'node:path';
const base=process.env.MADI_BASE_URL||'http://127.0.0.1:8080';
const output=path.resolve(process.env.MADI_SCREENSHOT_DIR||'test-results/migration');
await mkdir(output,{recursive:true});
const browser=await chromium.launch({headless:true});const context=await browser.newContext({viewport:{width:1440,height:1000},locale:'ko-KR'});const page=await context.newPage();const errors=[];
page.on('pageerror',e=>errors.push(e.message));page.on('response',r=>{if(r.status()>=500)errors.push(`${r.status()} ${r.url()}`)});
async function api(endpoint,method='GET',data){const response=await context.request.fetch(base+'/api/v1'+endpoint,{method,data,headers:{'X-Madi-Request':'1'}});assert.ok(response.ok(),`${method} ${endpoint}: ${response.status()} ${await response.text()}`);return response.json()}
async function shot(name){await page.evaluate(()=>document.fonts.ready);await page.screenshot({path:path.join(output,name+'.png'),fullPage:true})}
try{
 await page.goto(base+'/login');await api('/auth/login','POST',{email:process.env.MADI_TEST_EMAIL||'admin@example.test',password:process.env.MADI_TEST_PASSWORD||'Browser-Test-Password-2026!'});
 const workspace=await api('/workspaces','POST',{name:'가져오기 UI 검증 '+Date.now()});await page.evaluate(id=>localStorage.setItem('madi.workspace',id),workspace.id);
 await page.goto(base+'/app/migrations');await page.locator('details.migration-legacy').first().locator('summary').click();await page.getByRole('heading',{name:'가져오기 센터',exact:true}).waitFor();
 await page.getByLabel('원본 형식',{exact:true}).selectOption('csv');await page.getByLabel('대상 공간',{exact:true}).last().selectOption('');
 await page.getByLabel('가져오기 파일',{exact:true}).setInputFiles({name:'업무 목록.csv',mimeType:'text/csv',buffer:Buffer.from('제목,담당 부서,상태\n운영 정책 정리,플랫폼팀,진행 중\n복구 훈련,보안팀,완료\n')});
 await shot('migration-center');await page.getByRole('button',{name:'미리보기 만들기',exact:true}).click();
 await page.getByRole('dialog').waitFor();await page.getByLabel('가져오기 확인',{exact:true}).waitFor({timeout:45000});
 await shot('migration-preview');const previewURL=page.url();await page.reload();await page.getByLabel('가져오기 확인',{exact:true}).waitFor();assert.equal(page.url(),previewURL);
 await page.getByLabel('가져오기 확인',{exact:true}).fill('IMPORT');await page.getByRole('button',{name:'확인한 데이터 가져오기',exact:true}).click();
 await page.getByRole('link',{name:'데이터베이스 열기',exact:true}).waitFor({timeout:45000});await shot('migration-result');
 await page.getByRole('link',{name:'데이터베이스 열기',exact:true}).click();await page.getByText('운영 정책 정리',{exact:true}).waitFor();
 await page.goto(base+'/admin/migration');await page.locator('details.migration-legacy').first().locator('summary').click();await page.getByRole('heading',{name:'가져오기 센터',exact:true}).waitFor();await shot('admin-migration');
 await page.setViewportSize({width:390,height:844});await page.goto(previewURL);await page.getByRole('dialog').waitFor();await page.getByRole('link',{name:'데이터베이스 열기',exact:true}).waitFor();await shot('migration-result-mobile');
 assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),'mobile overflow');await page.getByRole('button',{name:'닫기',exact:true}).click();
 const historyAction=page.locator('.migration-history-table').getByRole('button',{name:'미리보기 · 결과',exact:true}).first();
 assert.equal(await historyAction.evaluate(element=>getComputedStyle(element).whiteSpace),'nowrap','mobile history action must not wrap one letter per line');
 assert.ok((await historyAction.boundingBox()).height>=44,'mobile history action touch target');
 assert.ok(await page.locator('.migration-history-table table').evaluate(element=>element.getBoundingClientRect().width>=640),'mobile history uses contained horizontal scrolling');
 await shot('migration-center-mobile');
 const historyRegion=page.locator('.migration-history-table');
 assert.ok(await historyRegion.evaluate(element=>element.scrollWidth>element.clientWidth),'mobile history overflows only inside its own scrolling region');
 await historyRegion.evaluate(element=>{element.scrollLeft=element.scrollWidth});
 assert.ok(await historyRegion.evaluate(element=>element.scrollLeft>0),'native horizontal history scrolling');
 await historyAction.scrollIntoViewIfNeeded();
 assert.ok((await historyAction.boundingBox()).width>=120,'readable history action width');
 await page.screenshot({path:path.join(output,'migration-history-mobile-actions.png'),fullPage:false,animations:'disabled'});
 assert.deepEqual(errors,[]);console.log(JSON.stringify({ok:true,workspace_id:workspace.id,screenshots:output}));
}finally{await browser.close()}
