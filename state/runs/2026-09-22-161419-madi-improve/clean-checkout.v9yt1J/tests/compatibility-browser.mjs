import {chromium,firefox,webkit,expect} from 'playwright/test';
import assert from 'node:assert/strict';
import {mkdir,writeFile} from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';

const base=process.env.MADI_BASE_URL,major=process.env.MADI_COMPAT_PG_MAJOR;
assert.ok(base&&['17','18'].includes(major),'isolated Go PostgreSQL 17/18 fixture required');
const output=path.resolve(process.env.MADI_SCREENSHOT_DIR||'test-results/compatibility',`pg${major}`);
await mkdir(output,{recursive:true});
const engines={chromium,firefox,webkit},requested=(process.env.MADI_COMPAT_BROWSERS||'chromium,firefox,webkit').split(',');
assert.ok(requested.length>0&&requested.every(x=>Object.hasOwn(engines,x)),'invalid browser selection');
const report={verified_at:new Date().toISOString(),app_version:process.env.MADI_COMPAT_APP_VERSION,postgresql:process.env.MADI_COMPAT_PG_VERSION,go:process.env.MADI_COMPAT_GO,platform:process.env.MADI_COMPAT_PLATFORM,kernel:os.release(),cpu:os.cpus()[0]?.model,logical_cpus:os.cpus().length,total_memory:os.totalmem(),native_safari:false,actual_os_ime:false,browsers:[]};
for(const name of requested){
 const launch={headless:true};
 // The optional local QA sysroot avoids changing system packages. Use the
 // exact installed Playwright binary; upstream MiniBrowser's shell wrapper
 // replaces LD_LIBRARY_PATH instead of preserving the private dependencies.
 if(name==='webkit'&&process.env.MADI_COMPAT_WEBKIT_BUNDLE){const bundle=path.resolve(process.env.MADI_COMPAT_WEBKIT_BUNDLE);launch.executablePath=path.join(bundle,'bin/MiniBrowser');launch.env={...process.env,WEBKIT_EXEC_PATH:path.join(bundle,'bin'),WEBKIT_INJECTED_BUNDLE_PATH:path.join(bundle,'lib'),WEBKIT_INSPECTOR_RESOURCES_PATH:path.join(bundle,'share'),LD_LIBRARY_PATH:[path.join(bundle,'lib'),path.join(bundle,'sys/lib'),process.env.LD_LIBRARY_PATH||''].join(':')}}
 const start=performance.now(),browser=await engines[name].launch(launch);
 const context=await browser.newContext({viewport:{width:1440,height:1024},locale:'ko-KR',timezoneId:'Asia/Seoul',reducedMotion:'reduce'}),page=await context.newPage();
 const errors=[],external=[],diagnostics=[],trace=process.env.MADI_COMPAT_TRACE==='1';
 const record=(event,details={})=>{if(trace){diagnostics.push({at:new Date().toISOString(),event,page_url:page.url(),...details});if(diagnostics.length>2000)diagnostics.shift()}};
 const sameOrigin=url=>{try{return new URL(url).origin===new URL(base).origin}catch{return false}};
 page.on('pageerror',e=>{errors.push(e.message);record('pageerror',{name:e.name,message:e.message,stack:e.stack})});
 page.on('response',r=>{if(r.status()>=500)errors.push(`${r.status()} ${r.url()}`);if(r.url().includes('/api/v1/profile')||r.status()>=400)record('response',{url:r.url(),status:r.status(),method:r.request().method(),same_origin:sameOrigin(r.url())})});
 page.on('request',r=>{if(!r.url().startsWith(base)&&!r.url().startsWith('blob:')&&!r.url().startsWith('data:'))external.push(r.url());if(r.url().includes('/api/v1/profile')||r.isNavigationRequest())record('request',{url:r.url(),method:r.method(),navigation:r.isNavigationRequest(),same_origin:sameOrigin(r.url())})});
 page.on('requestfailed',r=>record('requestfailed',{url:r.url(),method:r.method(),failure:r.failure(),same_origin:sameOrigin(r.url())}));
 page.on('console',m=>{if(m.type()==='error'||m.type()==='warning')record('console',{type:m.type(),text:m.text(),location:m.location()})});
 page.on('framenavigated',frame=>{if(frame===page.mainFrame())record('navigation',{url:frame.url()})});
 if(trace)await context.addInitScript(()=>{
  const keep=(event,detail)=>{try{const key='madi.compatibility.error-trace',items=JSON.parse(sessionStorage.getItem(key)||'[]');items.push({at:new Date().toISOString(),event,url:location.href,...detail});sessionStorage.setItem(key,JSON.stringify(items.slice(-100)))}catch{}};
  window.addEventListener('error',event=>keep('window.error',{message:event.message,name:event.error?.name,stack:event.error?.stack,filename:event.filename,line:event.lineno,column:event.colno}));
  window.addEventListener('unhandledrejection',event=>keep('unhandledrejection',{name:event.reason?.name,message:event.reason?.message||String(event.reason),stack:event.reason?.stack}));
 });
 const dir=path.join(output,name);await mkdir(dir,{recursive:true});
 async function api(url,method='GET',data){const r=await context.request.fetch(base+'/api/v1'+url,{method,data,headers:{'X-Madi-Request':'1'}});assert.ok(r.ok(),`${url}: ${r.status()} ${await r.text()}`);return r.json()}
 async function shot(file){await page.evaluate(()=>document.fonts.ready);await page.screenshot({path:path.join(dir,file+'.png'),fullPage:false,animations:'disabled'})}
 async function persisted(check){await expect.poll(check,{timeout:15000}).toBe(true)}
 async function overflow(){assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),'viewport horizontal overflow')}
 try{
  await page.goto(base+'/login');await expect(page.locator('.login-version')).toContainText('v'+process.env.MADI_COMPAT_APP_VERSION);await page.getByLabel('이메일',{exact:true}).fill('admin@example.test');await page.getByLabel('비밀번호',{exact:true}).fill(process.env.MADI_TEST_PASSWORD);await page.getByRole('button',{name:'로그인',exact:true}).click();await page.waitForURL(/\/app(?:\/|$)/);await page.locator('.profile-trigger').click();await expect(page.locator('.menu-version')).toContainText('v'+process.env.MADI_COMPAT_APP_VERSION);await page.keyboard.press('Escape');
  const wid=(await api('/workspaces','POST',{name:`PG${major} ${name} 호환성`})).id;
  await context.addInitScript(id=>localStorage.setItem('madi.workspace',id),wid);
  const doc=await api('/documents','POST',{workspace_id:wid,title:`${name} 한글 문서`,markdown:'# 입력 전 원문\n',visibility:'private'});
  await page.goto(base+`/app/documents/${doc.id}?mode=source`);const source=page.getByLabel('Markdown 원문 편집',{exact:true});await source.waitFor();
  const text='# 한글 입력 검증\n\n자료의 출처와 검토 기록을 보존합니다. 🚦\n';await source.fill('');await source.press('Control+Home');await page.keyboard.insertText(text);assert.equal(await source.inputValue(),text);await page.getByRole('button',{name:'저장',exact:true}).click();await persisted(async()=>(await api('/documents/'+doc.id)).markdown===text);
  const tag=page.getByRole('combobox',{name:'문서 태그',exact:true});await tag.dispatchEvent('compositionstart',{data:'한'});await tag.fill('한글태그');await tag.dispatchEvent('keydown',{key:'Enter',code:'Enter',keyCode:229,isComposing:true});assert.equal(await page.getByRole('button',{name:'한글태그 태그 삭제',exact:true}).count(),0,'composition Enter created an incomplete tag');await tag.dispatchEvent('compositionend',{data:'한글태그'});await tag.press('Enter');await page.getByRole('button',{name:'한글태그 태그 삭제',exact:true}).waitFor();await persisted(async()=>(await api('/documents/'+doc.id)).tags.includes('한글태그'));
  await page.reload();await source.waitFor();assert.ok((await source.inputValue()).includes('자료의 출처와 검토 기록'));await page.getByRole('button',{name:'한글태그 태그 삭제',exact:true}).waitFor();assert.equal(new URL(page.url()).searchParams.get('mode'),'source');await overflow();await shot('korean-document');
  const db=await api('/databases','POST',{workspace_id:wid,name:`${name} 선택 입력 검증`,properties:[{id:'name',name:'항목 이름',type:'text'},{id:'status',name:'상태',type:'select',options:['할 일','진행 중','완료']},{id:'tags',name:'분류',type:'multi_select',options:['문서','AI','운영']}]});
  await page.goto(base+'/app/databases/'+db.id);const opener=page.getByRole('button',{name:'새 항목',exact:true});await opener.click();const dialog=page.getByRole('dialog',{name:'새 항목',exact:true});await dialog.waitFor();await page.keyboard.press('Tab');assert.ok(await dialog.evaluate(el=>el.contains(document.activeElement)),'modal focus escaped');await page.keyboard.press('Escape');await dialog.waitFor({state:'hidden'});await expect(opener).toBeFocused();
  await opener.click();await dialog.getByLabel('항목 이름',{exact:true}).fill('한글 단일·다중 선택');await dialog.getByLabel('상태',{exact:true}).selectOption('진행 중');await dialog.getByLabel('문서',{exact:true}).check();await dialog.getByLabel('AI',{exact:true}).check();await dialog.getByLabel('운영',{exact:true}).check();await dialog.getByLabel('운영',{exact:true}).uncheck();await shot('select-modal');await dialog.getByRole('button',{name:'저장',exact:true}).click();await dialog.waitFor({state:'hidden'});await persisted(async()=>{const rows=await api(`/databases/${db.id}/rows`);return rows.some(x=>x.values.name==='한글 단일·다중 선택'&&x.values.status==='진행 중'&&JSON.stringify(x.values.tags)===JSON.stringify(['문서','AI']))});
  await page.reload();await opener.waitFor();await page.getByRole('gridcell').filter({hasText:'한글 단일·다중 선택'}).waitFor();await shot('database-roundtrip');
  await page.goto(base+'/app/profile');const font=page.getByLabel('글꼴',{exact:true});await font.selectOption('serif');await page.getByRole('button',{name:'변경사항 저장',exact:true}).click();await persisted(async()=>(await api('/profile')).preferences.font_family==='serif');await expect(page.getByRole('button',{name:'변경사항 저장',exact:true})).toBeEnabled();await page.reload();await expect(font).toHaveValue('serif');
  await font.selectOption('sans');await page.getByRole('button',{name:'변경사항 저장',exact:true}).click();await persisted(async()=>(await api('/profile')).preferences.font_family==='sans');await expect(page.getByRole('button',{name:'변경사항 저장',exact:true})).toBeEnabled();
  await page.setViewportSize({width:390,height:844});await page.goto(base+`/app/documents/${doc.id}?mode=read`);await page.locator('.markdown-content').waitFor();await overflow();await shot('mobile-document');
  await page.goto(base+'/app/databases/'+db.id);await opener.waitFor();await overflow();await opener.click();await dialog.waitFor();assert.ok(await dialog.evaluate(el=>{const r=el.getBoundingClientRect();return r.left>=-1&&r.right<=innerWidth+1&&r.bottom<=innerHeight+1}),'mobile modal outside viewport');await dialog.getByLabel('항목 이름',{exact:true}).fill('모바일 한글 확인');await dialog.getByLabel('상태',{exact:true}).selectOption('완료');await dialog.getByLabel('운영',{exact:true}).check();await shot('mobile-select');await dialog.getByRole('button',{name:'항목 상세 닫기',exact:true}).click();await dialog.waitFor({state:'hidden'});await expect(opener).toBeFocused();
  assert.deepEqual(errors,[],'JavaScript errors or HTTP 5xx responses');
  assert.deepEqual(external,[],'external page requests');
  report.browsers.push({engine:name,version:browser.version(),private_dependency_sysroot:!!launch.executablePath,status:'passed',duration_ms:Math.round(performance.now()-start),checks:['login','candidate-version','route-refresh','unicode-entry','synthetic-composition-enter','tag-roundtrip','native-single-select','checkbox-multi-select','modal-focus-return','profile-native-select','390px'],javascript_errors:0,external_page_requests:0});
  console.log(`PASS PostgreSQL ${major} ${name} ${browser.version()}`);
 }catch(e){await page.screenshot({path:path.join(dir,'failure.png')}).catch(()=>{});report.browsers.push({engine:name,version:browser.version(),status:'failed',error:e.message,duration_ms:Math.round(performance.now()-start)});throw e}
 finally{if(trace){const window_events=await page.evaluate(()=>JSON.parse(sessionStorage.getItem('madi.compatibility.error-trace')||'[]')).catch(e=>({diagnostic_error:e.message}));await writeFile(path.join(dir,'diagnostics.json'),JSON.stringify({base,errors,external,window_events,events:diagnostics},null,2))}await browser.close();await writeFile(path.join(output,'report.json'),JSON.stringify(report,null,2))}
}
