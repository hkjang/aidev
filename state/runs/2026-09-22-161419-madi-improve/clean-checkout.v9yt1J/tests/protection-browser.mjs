import {chromium} from 'playwright';
import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import path from 'node:path';
const base=process.env.MADI_BASE_URL||'http://127.0.0.1:8080',output=path.resolve(process.env.MADI_SCREENSHOT_DIR||'test-results/protection');
await mkdir(output,{recursive:true});
const browser=await chromium.launch({headless:true}),context=await browser.newContext({viewport:{width:1440,height:1000},locale:'ko-KR'}),page=await context.newPage();
const errors=[];page.on('pageerror',e=>errors.push(e.message));page.on('response',r=>{if(r.status()>=500)errors.push(`${r.status()} ${r.url()}`)});
async function api(endpoint,method='GET',data){const response=await context.request.fetch(base+'/api/v1'+endpoint,{method,data,headers:{'X-Madi-Request':'1'}});assert.ok(response.ok(),`${method} ${endpoint}: ${response.status()} ${await response.text()}`);return response.json()}
async function shot(name,view=page){await view.evaluate(async()=>{await document.fonts.ready;window.scrollTo(0,0);await new Promise(resolve=>requestAnimationFrame(resolve))});await view.screenshot({path:path.join(output,name+'.png'),fullPage:true,animations:'disabled'})}
let oldPolicy,policyChanged=false;
try{
 await page.goto(base+'/login');await api('/auth/login','POST',{email:process.env.MADI_TEST_EMAIL||'admin@example.test',password:process.env.MADI_TEST_PASSWORD||'Browser-Test-Password-2026!'});
 const stamp=Date.now(),workspace=await api('/workspaces','POST',{name:'정보보호 UI 검증 '+stamp});await page.evaluate(id=>localStorage.setItem('madi.workspace',id),workspace.id);
 oldPolicy=await api('/admin/information-protection');
 // Use warn + a fixture-only literal: never shut down other test workspaces'
 // live CRDT connections or scan unrelated content during shared browser QA.
 await api('/admin/information-protection','PUT',{revision:oldPolicy.revision,settings:{...oldPolicy.settings,enabled:true,mode:'warn',detectors:[],custom_terms:['보호검증_'+stamp],watermark_enabled:true,watermark_min_classification:'restricted',public_shares_enabled:true,public_share_max_classification:'restricted',public_share_require_password:true,public_share_allow_download:true}});policyChanged=true;
 await page.goto(base+'/admin/information-protection');await page.getByRole('heading',{name:'정보보호 정책',exact:true}).waitFor();
 await page.getByLabel('민감정보 처리 방식',{exact:true}).selectOption('mask');await page.getByLabel('민감정보 처리 방식',{exact:true}).selectOption('warn');
 await page.getByLabel('검사할 수 없는 첨부',{exact:true}).selectOption('block');await page.getByLabel('검사할 수 없는 첨부',{exact:true}).selectOption('warn');
 await page.getByRole('button',{name:'정보보호 정책 저장',exact:true}).click();await page.getByText('정보보호 정책을 저장했습니다',{exact:true}).waitFor();await shot('admin-information-protection');
 const document=await api('/documents','POST',{workspace_id:workspace.id,title:'외부 검토를 위한 정책 안내 '+stamp,visibility:'private',markdown:'# 외부 검토 안내\n\n이 문서는 소유자가 명시적으로 공유한 자료입니다.\n\n보호검증_'+stamp+'\n\n- [x] 만료·암호·IP 정책 확인\n- [ ] 검토 후 공유 폐기'});
 await api(`/documents/${document.id}/knowledge`,'PUT',{version:document.version,classification:'restricted'});
 await page.goto(base+'/app/documents/'+document.id);await page.getByText('상속 적용 등급: 제한',{exact:true}).waitFor();await shot('document-watermark');
 await page.getByRole('tab',{name:'속성',exact:true}).click();await page.getByRole('button',{name:'공개 링크',exact:true}).click();await page.getByRole('button',{name:'공개 링크 만들기',exact:true}).click();
 await page.getByLabel('공유 암호',{exact:true}).fill('Browser-Public-Share-Password!');await page.getByLabel('화면 텍스트 복사 허용',{exact:true}).uncheck();await page.getByLabel('링크를 가진 방문자에게 이 문서를 공개함을 확인했습니다',{exact:true}).check();
 await shot('public-share-create');await page.getByRole('button',{name:'공유 설정 저장',exact:true}).click();await page.getByRole('textbox',{name:'새 공개 링크',exact:true}).waitFor();const url=await page.getByRole('textbox',{name:'새 공개 링크',exact:true}).inputValue();assert.ok(url.includes('#'));
 await page.keyboard.press('Escape');
 const visitor=await browser.newContext({viewport:{width:1280,height:900},locale:'ko-KR'}),external=await visitor.newPage();external.on('pageerror',e=>errors.push(e.message));external.on('response',r=>{if(r.status()>=500)errors.push(`${r.status()} ${r.url()}`)});
 const response=await external.goto(url);assert.ok((await response.allHeaders())['x-robots-tag'].includes('noindex'));await external.getByRole('heading',{name:'공유 암호 확인',exact:false}).waitFor();assert.equal(await external.getByText('이 문서는 소유자가 명시적으로 공유한 자료입니다.',{exact:true}).count(),0);await shot('public-share-password',external);
 await external.getByLabel('공유 암호',{exact:true}).fill('Browser-Public-Share-Password!');await external.getByRole('button',{name:'공유 문서 열기',exact:true}).click();await external.getByRole('heading',{name:'외부 검토 안내',exact:true}).waitFor();await shot('public-share-document',external);
 await external.setViewportSize({width:390,height:844});await shot('public-share-mobile',external);assert.ok(await external.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1));
 await page.getByRole('button',{name:'폐기',exact:true}).first().click();await page.getByRole('button',{name:'공개 링크 폐기 확인',exact:true}).click();await page.getByText('공개 링크를 폐기했습니다',{exact:true}).waitFor();
 await external.reload();await external.getByText('공유 링크가 없거나 사용할 수 없습니다',{exact:false}).waitFor();assert.equal(await external.getByRole('heading',{name:'외부 검토 안내',exact:true}).count(),0);await visitor.close();
 await page.goto(base+'/admin/information-protection');await page.getByRole('heading',{name:'정보보호 정책',exact:true}).waitFor();await page.setViewportSize({width:390,height:844});await shot('information-protection-mobile');assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1));
 assert.deepEqual(errors,[]);console.log(JSON.stringify({ok:true,workspace_id:workspace.id,document_id:document.id,screenshots:output}));
}finally{
 if(policyChanged&&oldPolicy){const current=await api('/admin/information-protection');await api('/admin/information-protection','PUT',{revision:current.revision,settings:oldPolicy.settings})}
 await browser.close();
}
