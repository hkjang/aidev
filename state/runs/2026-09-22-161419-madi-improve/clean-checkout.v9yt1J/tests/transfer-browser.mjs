import {chromium} from 'playwright';
import assert from 'node:assert/strict';
import {mkdir,readFile} from 'node:fs/promises';
import {createRequire} from 'node:module';
import path from 'node:path';
const require=createRequire(import.meta.url),{unzipSync}=require('../web/node_modules/fflate');
const base=process.env.MADI_BASE_URL||'http://127.0.0.1:8080';
assert.ok(['localhost','127.0.0.1'].includes(new URL(base).hostname),'disposable local service only');
const output=path.resolve(process.env.MADI_SCREENSHOT_DIR||'test-results/transfer');await mkdir(output,{recursive:true});
const browser=await chromium.launch({headless:true}),context=await browser.newContext({viewport:{width:1480,height:1050},locale:'ko-KR',timezoneId:'Asia/Seoul',reducedMotion:'reduce'}),page=await context.newPage(),errors=[];
page.on('pageerror',e=>errors.push(e.message));page.on('response',r=>{if(r.status()>=500)errors.push(`${r.status()} ${r.url()}`)});
await context.route('**/*',route=>{const u=new URL(route.request().url());if(u.origin!==new URL(base).origin&&!['data:','blob:'].includes(u.protocol)){errors.push('unexpected external request '+u.origin);return route.abort()};return route.continue()});
async function api(endpoint,method='GET',data){const response=await context.request.fetch(base+'/api/v1'+endpoint,{method,data,headers:{'X-Madi-Request':'1'}});assert.ok(response.ok(),`${method} ${endpoint}: ${response.status()} ${await response.text()}`);return response.json()}
async function poll(fn){const deadline=Date.now()+60000;while(Date.now()<deadline){if(await fn())return;await new Promise(r=>setTimeout(r,250))};throw new Error('transfer job timeout')}
async function shot(name){await page.evaluate(()=>document.fonts.ready);const close=page.locator('.toast .icon-button');if(await close.count())await close.first().click().catch(()=>{});await page.screenshot({path:path.join(output,name+'.png'),fullPage:true,animations:'disabled'})}
let oldPolicy,changed=false;
try{
 await page.goto(base+'/login');await api('/auth/login','POST',{email:process.env.MADI_TEST_EMAIL||'admin@example.test',password:process.env.MADI_TEST_PASSWORD||'Browser-Test-Password-2026!'});
 const workspace=await api('/workspaces','POST',{name:'자료 이동 검증 '+Date.now()});await page.evaluate(id=>localStorage.setItem('madi.workspace',id),workspace.id);
 const raw='# 원문 그대로\n\nMarkdown 줄 끝의 공백을 보존합니다.  \n\n```go\nfmt.Println("madi")\n```\n';const doc=await api('/documents','POST',{workspace_id:workspace.id,title:'내보내기 원문 검증',visibility:'private',markdown:raw});
 const canonical=await api('/documents/'+doc.id);
 const database=await api('/databases','POST',{workspace_id:workspace.id,name:'CSV 이동 검증',properties:[{id:'title',name:'제목',type:'text',options:[]},{id:'status',name:'상태',type:'select',options:['진행 중','완료']}]});await api(`/databases/${database.id}/rows`,'POST',{values:{title:'=검증',status:'완료'}});
 oldPolicy=await api('/admin/exports/settings');await page.goto(base+'/admin/exports');await page.getByRole('heading',{name:'내보내기 정책',exact:true}).waitFor();await page.getByLabel('내보내기 사용',{exact:true}).selectOption('true');await page.getByLabel('결과 파일 최대 크기 (MB)',{exact:true}).fill('100');await page.getByRole('button',{name:'정책 저장',exact:true}).click();await page.getByText('내보내기 정책을 저장했습니다',{exact:true}).waitFor();changed=true;await shot('admin-export-policy');
 await page.goto(base+'/app/import');await page.locator('details.migration-legacy').first().locator('summary').click();await page.getByRole('heading',{name:'가져오기 센터',exact:true}).waitFor();await page.getByLabel('파일 선택 방식',{exact:true}).selectOption('folder');await page.locator('input[webkitdirectory]').setInputFiles(path.resolve('tests/fixtures/transfer'));await page.getByRole('button',{name:'미리보기 만들기',exact:true}).click();
 await poll(async()=>new URL(page.url()).searchParams.has('import'));const importID=new URL(page.url()).searchParams.get('import');await poll(async()=>{const v=await api('/migrations/'+importID);if(v.job_status==='failed')throw new Error(JSON.stringify(v));return v.status==='ready'});
 await page.getByLabel('가져오기 확인',{exact:true}).waitFor({timeout:15000});await shot('migration-folder-preview');await page.getByLabel('가져오기 확인',{exact:true}).fill('IMPORT');await page.getByRole('button',{name:'확인한 데이터 가져오기',exact:true}).click();await poll(async()=>{const v=await api('/migrations/'+importID);if(v.job_status==='failed')throw new Error(JSON.stringify(v));return v.status==='completed'});await page.getByText(/가져오기 완료: 문서/).waitFor({timeout:15000});await shot('migration-folder-result');await page.keyboard.press('Escape');
 await page.goto(base+'/app/export');await page.getByRole('heading',{name:'내보내기 센터',exact:true}).waitFor();await page.getByRole('checkbox',{name:/내보내기 원문 검증/}).check();await shot('export-center');
 const resultIDs={};
 for(const format of ['markdown','portable','html','json','csv']){
  await page.getByLabel('내보내기 형식',{exact:true}).selectOption(format);if(format==='csv')await page.getByLabel('내보낼 데이터베이스',{exact:true}).selectOption(database.id);
  await page.getByRole('button',{name:'선택한 자료 내보내기',exact:true}).click();const previous=resultIDs[format];await poll(async()=>{const id=new URL(page.url()).searchParams.get('export');return !!id&&id!==previous&&!Object.values(resultIDs).includes(id)});const id=new URL(page.url()).searchParams.get('export');resultIDs[format]=id;
  await poll(async()=>{const v=await api('/exports/'+id);if(['failed','cancelled'].includes(v.status))throw new Error(JSON.stringify(v));return v.status==='ready'});
  const card=page.locator('.transfer-card.selected');await card.getByRole('button',{name:'파일 다운로드',exact:true}).waitFor({timeout:15000});const event=page.waitForEvent('download');await card.getByRole('button',{name:'파일 다운로드',exact:true}).click();const download=await event,filePath=await download.path();assert.equal(await download.failure(),null);const data=await readFile(filePath);
  if(format==='markdown'){const files=unzipSync(data);assert.equal(new TextDecoder().decode(files['내보내기 원문 검증.md']),canonical.markdown)}
  if(format==='json'){const v=JSON.parse(data);assert.equal(v.format,'madi-json-vault');assert.equal(Buffer.from(v.files[0].data_base64,'base64').toString(),canonical.markdown)}
  if(format==='html'){const files=unzipSync(data);assert.ok(files['index.html']);assert.match(new TextDecoder().decode(files['내보내기 원문 검증.html']),/Content-Security-Policy/)}
  if(format==='csv')assert.match(data.toString(),/'=검증/);
 }
 await shot('export-results');await page.reload();await page.getByRole('heading',{name:'내보내기 센터',exact:true}).waitFor();assert.equal(new URL(page.url()).searchParams.get('export'),resultIDs.csv);
 await page.setViewportSize({width:390,height:844});await shot('mobile-export');assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1));
 await page.goto(base+'/app/import');await page.getByRole('heading',{name:'가져오기',exact:true}).waitFor();await shot('mobile-import');assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1));
 await page.goto(base+'/admin/exports');await page.getByRole('heading',{name:'내보내기 정책',exact:true}).waitFor();await shot('mobile-export-policy');assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1));
 assert.deepEqual(errors,[]);console.log(JSON.stringify({ok:true,workspace_id:workspace.id,import_id:importID,exports:resultIDs,screenshots:output}));
}finally{try{if(changed&&oldPolicy){const current=await api('/admin/exports/settings');await api('/admin/exports/settings','PUT',{...oldPolicy,revision:current.revision})}}finally{await browser.close()}}
