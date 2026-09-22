import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
import path from 'node:path';
import {chromium,expect} from 'playwright/test';

const base=process.env.MADI_BASE_URL, wid=process.env.MADI_WORKSPACE_ID,doc=process.env.MADI_DOCUMENT_ID;
const output=path.resolve(process.env.MADI_SCREENSHOT_DIR||'test-results/search-operations');
const browser=await chromium.launch(),context=await browser.newContext({viewport:{width:1512,height:1000},locale:'ko-KR',reducedMotion:'reduce'}),page=await context.newPage();
const issues=[];page.on('pageerror',error=>issues.push(error.message));
page.on('response',r=>{if(r.url().startsWith(base+'/api/')&&r.status()>=500)issues.push(`HTTP${r.status()} ${r.url()}`)});
async function api(url,method='GET',data,status=200){const r=await context.request.fetch(base+'/api/v1'+url,{method,data,headers:{'X-Madi-Request':'1'}});assert.equal(r.status(),status,`${url}: ${await r.text()}`);return r.json()}
async function shot(name){await mkdir(output,{recursive:true});await page.evaluate(()=>document.fonts.ready);await page.screenshot({path:path.join(output,name+'.png'),fullPage:page.viewportSize().width>500,animations:'disabled'})}
async function overflow(){const v=await page.evaluate(()=>({width:innerWidth,actual:document.documentElement.scrollWidth,layout:[...document.querySelectorAll('body,.topbar,.content,.universal-search,.search-reading-layout,.search-reading-results,.search-result')].slice(0,9).map(el=>({class:el.className,rect:el.getBoundingClientRect().toJSON(),css:{display:getComputedStyle(el).display,grid:getComputedStyle(el).gridTemplateColumns,position:getComputedStyle(el).position,marginRight:getComputedStyle(el).marginRight}})),elements:[...document.querySelectorAll('body *')].map(el=>({tag:el.tagName,class:el.className,left:el.getBoundingClientRect().left,right:el.getBoundingClientRect().right,width:el.getBoundingClientRect().width})).filter(r=>r.width>0&&(r.right>innerWidth+1||r.left< -1)).slice(0,15)}));assert.ok(v.actual<=v.width+1,'horizontal overflow '+JSON.stringify(v))}
async function settled(){await expect(page.locator('.search-results-header')).not.toContainText('찾고 있습니다');await expect(page.locator('.search-result')).not.toHaveCount(0)}
async function searchAction(action){const response=page.waitForResponse(r=>r.url().startsWith(base+'/api/v1/search?'));await action();await response;await settled()}
try {
 await api('/auth/login','POST',{email:'admin@example.test',password:'Integration-Test-Password-2026!'});
 await page.goto(base+'/app/search/operations');await page.getByLabel('워크스페이스 선택',{exact:true}).selectOption(wid);
 await page.goto(base+'/app/search/operations');
 await page.getByRole('heading',{name:'검색 운영',exact:true}).waitFor();
 await page.getByRole('button',{name:'용어 추가',exact:true}).click();
 await page.getByLabel('표준 용어 1',{exact:true}).fill('쿠버네티스');
 await page.getByLabel('별칭 1 · 쉼표로 구분',{exact:true}).fill('k8s, Kubernetes, ');
 await expect(page.getByRole('button',{name:'사전 저장',exact:true})).toBeDisabled();
 await page.getByRole('checkbox',{name:/이 용어를 워크스페이스/}).check();
 await page.getByRole('button',{name:'사전 저장',exact:true}).click();
 await expect(page.getByText('사전을 저장했습니다. 다음 검색부터 새 용어 해석을 적용합니다.',{exact:true})).toBeVisible();
 assert.deepEqual((await api(`/workspaces/${wid}/search-dictionary`)).entries[0].aliases,['k8s','Kubernetes']);
 await shot('search-dictionary');
 await page.getByRole('tab',{name:'검색 진단',exact:true}).click();await page.getByLabel('진단할 검색어',{exact:true}).fill('k8s 운영');
 await page.getByRole('button',{name:'현재 권한으로 진단 실행',exact:true}).click();
 await expect(page.getByRole('heading',{name:'진단 결과',exact:true})).toBeVisible();
 await expect(page.locator('.search-plan')).toBeVisible();
 await shot('search-diagnostics');
 await page.getByRole('tab',{name:'한국어 평가',exact:true}).click();await page.getByLabel('평가 검색어 1',{exact:true}).fill('k8s 운영');
 await page.getByLabel('정답 문서 1 · 여러 개는 Ctrl/⌘로 선택',{exact:true}).selectOption(doc);
 await page.getByLabel('상위 결과 수 k',{exact:true}).selectOption('5');
 // Title sorting is not the evaluation ranking. Choose a genuinely retrieved top result.
 const real=await api(`/search?workspace_id=${wid}&q=${encodeURIComponent('k8s 운영')}&type=document&limit=5`);
 await page.getByLabel('정답 문서 1 · 여러 개는 Ctrl/⌘로 선택',{exact:true}).selectOption(real.results[0].id);
 await page.getByRole('button',{name:'제공한 정답으로 평가 실행',exact:true}).click();await expect(page.locator('.search-operation-result')).toContainText('100.0%');
 await shot('search-evaluation');
 await page.goto(base+'/app/search?q='+encodeURIComponent('k8s 운영')+'&sort=title');await settled();
 await expect(page.locator('.search-result')).toHaveCount(40);await expect(page.locator('.search-results-header')).toContainText('1페이지 · 40개 묶음');
 await expect(page.locator('.search-result').first()).toContainText('쿠버네티스 운영 01');
 await expect(page.locator('.search-result').first()).toContainText('개 일치');
 await page.locator('.search-result').first().getByRole('button',{name:'문서 미리보기',exact:true}).click();
 await expect(page.locator('.document-preview-panel')).toContainText('장애 대응 근거');await overflow();await shot('search-grouped-preview');
 await page.getByRole('button',{name:'문서 미리보기 닫기',exact:true}).click();
 await searchAction(()=>page.getByRole('button',{name:'다음 결과',exact:true}).click());
 await expect(page.locator('.search-result')).toHaveCount(5);await expect(page.locator('.search-results-header')).toContainText('2페이지 · 5개 묶음');
 await searchAction(()=>page.getByRole('button',{name:'이전 결과',exact:true}).click());
 await expect(page.locator('.search-result')).toHaveCount(40);
 await searchAction(()=>page.locator('.search-result').first().getByRole('button',{name:'#운영검증',exact:true}).click());
 await expect(page.getByRole('button',{name:'태그: #운영검증 조건 해제',exact:true})).toBeVisible();
 await searchAction(()=>page.getByRole('button',{name:'태그: #운영검증 조건 해제',exact:true}).click());assert.ok(!new URL(page.url()).searchParams.has('tag'));
 const chosen=page.locator('.search-result').nth(8);await chosen.getByRole('button',{name:'문서 미리보기',exact:true}).click();
 const selectedURL=page.url();await page.locator('.document-preview-panel').getByRole('link',{name:'문서 전체 열기',exact:true}).click();await page.locator('.document-body').waitFor();
 await page.goBack();await settled();assert.equal(page.url(),selectedURL);await expect(page.locator('.document-preview-panel')).toContainText('쿠버네티스 운영 09');
 await expect.poll(()=>page.evaluate(()=>window.scrollY),{message:'search scroll position restored on back'}).toBeGreaterThan(100);
 const resizeStarted=Date.now();await page.setViewportSize({width:390,height:844});
 await page.waitForFunction(()=>matchMedia('(max-width:640px)').matches&&getComputedStyle(document.querySelector('.main')).marginLeft==='0px'&&getComputedStyle(document.querySelector('.topbar')).height==='62px',null,{timeout:2000});
 console.log(JSON.stringify({mobile_media_layout_ms:Date.now()-resizeStarted}));
 await expect(page.getByRole('dialog',{name:'문서 미리보기',exact:true})).toBeVisible();await overflow();await shot('search-preview-mobile');
 await page.getByRole('dialog').getByRole('button',{name:'닫기',exact:true}).click();await page.evaluate(()=>scrollTo(0,0));await overflow();await shot('search-grouped-mobile');
 // Expired/tampered pages have explicit recovery, no fallback that claims success.
 await page.goto(base+'/app/search?q=k8s&cursor=enc%3Ainvalid&page=1');await expect(page.getByRole('group',{name:'검색 복구',exact:true})).toBeVisible();
 await page.getByRole('button',{name:'첫 페이지에서 다시 검색',exact:true}).click();await settled();
 await page.getByLabel('통합 검색어',{exact:true}).fill('일치하지않는합성검색');await page.getByRole('button',{name:'검색',exact:true}).click();await expect(page.getByText('현재 조건과 접근 범위에서 일치하는 결과가 없습니다. 다른 범위에 문서가 있는지는 표시하지 않습니다.',{exact:true})).toBeVisible();await shot('search-empty-recovery-mobile');
 // A silent viewer loses preview text after the document becomes private.
 const viewerContext=await browser.newContext({viewport:{width:1512,height:1000},locale:'ko-KR'});
 try {
  let r=await viewerContext.request.post(base+'/api/v1/auth/login',{data:{email:'collaborator@example.test',password:'Collaboration-Password-2026!'},headers:{'X-Madi-Request':'1'}});assert.equal(r.status(),200);
  const viewerPage=await viewerContext.newPage();await viewerPage.goto(base+`/app/search?q=k8s&sort=title&preview=${doc}`);await expect(viewerPage.locator('.document-preview-panel')).toContainText('장애 대응 근거');
  const current=await api('/documents/'+doc);await api('/documents/'+doc,'PUT',{version:current.version,visibility:'private'});
  await expect(viewerPage.locator('.document-preview-panel')).not.toContainText('장애 대응 근거',{timeout:7000});
  await viewerPage.reload();await expect(viewerPage.locator('.search-result').filter({hasText:'쿠버네티스 운영 01'})).toHaveCount(0);
 } finally { await viewerContext.close(); }
 assert.deepEqual(issues,[]);console.log(JSON.stringify({ok:true,checks:['dictionary-shared-consent-CAS','comma-options','real-plan','real-recall-evaluation','Korean-alias-spacing','group-before-pagination','keyset-next-previous','filter-chip','ACL-preview','back-context','mobile390','expired-cursor-recovery','private-safe-empty'],screenshots:output}));
} catch(error) {await shot('failure').catch(()=>{});console.error(await page.locator('body').innerText());throw error} finally {await browser.close()}
