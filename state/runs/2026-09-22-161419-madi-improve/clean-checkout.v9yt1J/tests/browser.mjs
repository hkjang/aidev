import {documentTool,documentPanel} from "./document-ui.mjs";
import {chromium} from 'playwright';
import {expect} from 'playwright/test';
import assert from 'node:assert/strict';
import {mkdir, readFile, writeFile} from 'node:fs/promises';
import path from 'node:path';

// Run against a dedicated disposable madi database. These tests create documents,
// change settings and issue/revoke keys; they are never intended for production.
const base = process.env.MADI_BASE_URL || 'http://127.0.0.1:8080';
const email = process.env.MADI_TEST_EMAIL || 'admin@example.test';
const password = process.env.MADI_TEST_PASSWORD || 'Browser-Test-Password-2026!';
const expectedVersion = (await readFile(new URL('../VERSION', import.meta.url), 'utf8')).trim();
const out = path.resolve(process.env.MADI_SCREENSHOT_DIR || 'docs/screenshots');
await mkdir(out,{recursive:true});
const browser = await chromium.launch({headless:true});
const context = await browser.newContext({viewport:{width:1512,height:1080},locale:'ko-KR',timezoneId:'Asia/Seoul',reducedMotion:'reduce'});
const page = await context.newPage();
const issues = [];
const failures = [];
const screenshots = [];
page.on('pageerror',error=>issues.push(error.message));
page.on('response',response=>{if(response.status()>=500)issues.push(`${response.status()} ${response.url()}`)});
page.on('console',message=>{if(message.type()==='error'&&!message.text().includes('401 (Unauthorized)'))issues.push(message.text())});
await context.route('**/*',route=>{
 const url=new URL(route.request().url());
 if(url.origin!==new URL(base).origin && !['data:','blob:'].includes(url.protocol)){issues.push(`External runtime asset: ${url}`);return route.abort()}
 return route.continue();
});
async function api(endpoint,method='GET',data){
 const response=await context.request.fetch(`${base}/api/v1${endpoint}`,{method,data,headers:{'X-Madi-Request':'1'}});
 assert.ok(response.ok(),`${method} ${endpoint}: ${response.status()} ${await response.text()}`);
 return response.json();
}
async function shot(name){
 await page.evaluate(()=>document.fonts.ready);
 if(await page.locator('.toast .icon-button').count())await page.locator('.toast .icon-button').click().catch(()=>{});
 await page.evaluate(()=>window.scrollTo(0,0));
 const floating=(await page.getByRole('dialog').count())+(await page.getByRole('menu').count());
 await page.screenshot({path:path.join(out,`${name}.png`),fullPage:!floating,animations:'disabled'});
 screenshots.push(name);
}
async function check(name,fn){try{await fn();console.log(`PASS ${name}`)}catch(error){failures.push({name,error:error.message});console.error(`FAIL ${name}: ${error.message}`);await shot(`failure-${name.replace(/[^a-z0-9]+/gi,'-')}`).catch(()=>{})}}
try {
 await page.goto(`${base}/login`);
 await page.getByRole('heading',{name:'다시 만나 반가워요'}).waitFor();
 await page.locator('.login-version').getByText('v'+expectedVersion,{exact:true}).waitFor();
 assert.equal((await api('/public')).version,expectedVersion,'public metadata must match the candidate VERSION');
 await shot('login');
 await page.getByLabel('이메일',{exact:true}).fill(email);
 await page.getByLabel('비밀번호',{exact:true}).fill(password);
 await page.getByRole('button',{name:'로그인',exact:true}).click();
 await page.locator('.sidebar').waitFor();
 const workspace=await api('/workspaces','POST',{name:'플랫폼 지식 워크스페이스'});
 const wid=workspace.id;
 await context.addInitScript(id=>localStorage.setItem('madi.workspace',id),wid);
 const titles=['팀 지식 운영 가이드','AI 플랫폼 아키텍처','Kubernetes 운영 노트','서비스 배포 체크리스트','주간 회의록'];
 let docs=await api(`/documents?workspace_id=${wid}`);
 const content=[
  '# 팀 지식 운영 가이드\n\n흩어진 경험을 연결하면, 팀의 다음 결정이 더 쉬워집니다.\n\n## 기록하는 문화\n\n결론뿐 아니라 **왜 그렇게 결정했는지** 함께 남깁니다. 문서의 맥락을 이어 읽을 수 있도록 [[AI 플랫폼 아키텍처]]와 관련 운영 문서를 연결합니다.\n\n## 문서 작성 원칙\n\n| 항목 | 작성 기준 |\n| --- | --- |\n| 제목 | 찾기 쉬운 구체적인 이름 |\n| 소유자 | 현재 내용을 관리하는 담당자 |\n| 링크 | 관련 문서와 결정 근거 |\n\n> 좋은 문서는 누군가의 다음 질문에 답합니다.\n\n## 이번 주 할 일\n\n- [ ] [[서비스 배포 체크리스트]] 검토\n- [ ] 신규 팀원 온보딩 문서 보완\n- [x] 워크스페이스 운영 원칙 정리\n',
  '# AI 플랫폼 아키텍처\n\n조직의 지식과 AI를 안전하게 연결하는 구성입니다.\n\n## 구성 요소\n\n- **madi** — 문서 원문, 권한, 검색과 MCP\n- **PostgreSQL** — 문서와 운영 설정의 영속 저장소\n- **사내 LLM** — OpenAI 호환 API를 제공하는 추론 서버\n\n## 운영 문서\n\n[[Kubernetes 운영 노트]] · [[팀 지식 운영 가이드]]\n\n```yaml\nservice: madi\nauthentication: oidc\nnetwork: internal\n```\n',
  '# Kubernetes 운영 노트\n\n## 점검 순서\n\n1. 워크로드의 상태를 확인합니다.\n2. readiness와 최근 이벤트를 살펴봅니다.\n3. [[서비스 배포 체크리스트]]로 변경사항을 검증합니다.\n\n```shell\nkubectl get pods -n knowledge\nkubectl get events -n knowledge\n```\n\n- [ ] 리소스 사용량 점검\n',
  '# 서비스 배포 체크리스트\n\n## 배포 전\n\n- [x] 변경 이력 검토\n- [x] 데이터베이스 백업 확인\n- [ ] 폐쇄망 이미지 반입 및 체크섬 확인\n- [ ] 서비스 시작 후 로그인·문서 편집 확인\n\n관련 문서: [[Kubernetes 운영 노트]], [[AI 플랫폼 아키텍처]]\n',
  '# 주간 회의록\n\n## 안건\n\n1. 지식관리 운영 현황\n2. 다음 주 개선 과제\n\n## 결정 사항\n\n문서 작성 기준은 [[팀 지식 운영 가이드]]에 통합합니다.\n\n## 실행 항목\n\n- [ ] 문서 소유자와 태그 정리\n- [ ] 팀별 온보딩 템플릿 검토\n'
 ];
 for(let i=0;i<titles.length;i++)if(!docs.some(d=>d.title===titles[i]))await api('/documents','POST',{workspace_id:wid,title:titles[i],markdown:content[i].replaceAll('\\n','\n'),tags:[['팀 문화','가이드'],['AI','아키텍처'],['운영','인프라'],['배포','체크리스트'],['회의','팀 문화']][i]});
 docs=await api(`/documents?workspace_id=${wid}`);
 const demo=docs.find(d=>d.title===titles[0]);
 if(!demo.is_favorite)await api(`/documents/${demo.id}/favorite`,'POST');
 let databases=await api(`/databases?workspace_id=${wid}`);
 let database=databases.find(d=>d.name==='팀 프로젝트');
 if(!database){database=await api('/databases','POST',{workspace_id:wid,name:'팀 프로젝트',properties:[{id:'name',name:'프로젝트',type:'text',options:[]},{id:'status',name:'상태',type:'select',options:['할 일','진행 중','완료']},{id:'owner',name:'담당 팀',type:'text',options:[]},{id:'date',name:'목표일',type:'date',options:[]},{id:'tags',name:'분류',type:'multi_select',options:['문서','운영','AI']}]});
 const date=new Date().toISOString().slice(0,8);
 for(const [name,status,owner,day,tags] of [['지식관리 운영 가이드 정리','완료','플랫폼팀','08',['문서']],['사내 AI 연동 검증','진행 중','AI팀','15',['AI','운영']],['서비스 운영 문서 최신화','진행 중','운영팀','20',['문서','운영']],['신규 입사자 온보딩','할 일','플랫폼팀','25',['문서']]])await api(`/databases/${database.id}/rows`,'POST',{values:{name,status,owner,date:date+day,tags}});}

 await check('document edit and refresh',async()=>{
  await page.goto(`${base}/app/documents/${demo.id}`);
  await documentTool(page,'Markdown 원문');
  const source=page.getByRole('textbox',{name:'Markdown 원문 편집'});
  await source.waitFor();
  const original=await source.inputValue();
  await source.fill(original+'\n브라우저 왕복 저장 확인.\n');
  await page.getByRole('button',{name:'저장',exact:true}).click();
  await page.locator('.save-state').filter({hasText:'저장됨'}).waitFor();
  await page.reload();
  await documentTool(page,'Markdown 원문');
  assert.ok((await source.inputValue()).includes('브라우저 왕복 저장 확인.'));
  await source.fill(original);
  await page.getByRole('button',{name:'저장',exact:true}).click();
  await page.locator('.save-state').filter({hasText:'저장됨'}).waitFor();
  await shot('editor-source');
  await page.getByRole('button',{name:'읽기',exact:true}).click();
  await page.locator('.markdown-content h1').waitFor();
  await shot('document');
 });
 const routes=[['workspace','/app'],['documents','/app/documents'],['search','/app/search?q=운영'],['favorites','/app/favorites'],['graph','/app/graph'],['tasks','/app/tasks'],['databases','/app/databases'],['templates','/app/templates'],['trash','/app/trash'],['import','/app/import'],['members','/app/members'],['profile','/app/profile'],['api-keys','/app/keys'],['admin-dashboard','/admin'],['admin-users','/admin/users'],['admin-settings','/admin/settings'],['admin-settings-oidc','/admin/settings?tab=auth'],['admin-settings-ai','/admin/settings?tab=ai'],['admin-settings-security','/admin/settings?tab=security'],['admin-settings-workflow','/admin/settings?tab=workflow'],['admin-settings-storage','/admin/settings?tab=storage'],['admin-settings-history','/admin/settings?tab=history'],['admin-audit','/admin/audit'],['admin-backup','/admin/backup']];
 for(const [name,route] of routes)await check(`route ${name}`,async()=>{
  await page.goto(base+route);await page.locator('.page-heading h1:visible').waitFor();
  await expect(page.locator('.loading:visible')).toHaveCount(0,{timeout:30000});
  assert.equal(await page.locator('.notice.error').count(),0,await page.locator('.notice.error').allTextContents());
  const before=new URL(page.url()).pathname;await page.reload();await page.locator('.page-heading h1:visible').waitFor();await expect(page.locator('.loading:visible')).toHaveCount(0,{timeout:30000});assert.equal(new URL(page.url()).pathname,before);
  await shot(name);
 });
 await check('SSO email verification policy',async()=>{
  await page.goto(base+'/admin/settings?tab=auth');
  const toggle=page.getByRole('checkbox',{name:/^SSO 이메일 검증 필수/});
  await toggle.waitFor();
  assert.equal((await api('/admin/settings')).oidc_require_verified_email,false,'default admission does not require email_verified');
  await expect(toggle).not.toBeChecked();
  try {
   await toggle.click();
   await page.getByRole('button',{name:'설정 저장',exact:true}).click();
   await expect.poll(async()=>(await api('/admin/settings')).oidc_require_verified_email).toBe(true);
   await page.reload();
   await expect(toggle).toBeChecked();
   await toggle.click();
   await page.getByRole('button',{name:'설정 저장',exact:true}).click();
   await expect.poll(async()=>(await api('/admin/settings')).oidc_require_verified_email).toBe(false);
   await page.reload();
   await expect(toggle).not.toBeChecked();
   await shot('admin-settings-oidc');
  } finally {await api('/admin/settings','PUT',{oidc_require_verified_email:false})}
 });
 await check('database select and multiselect',async()=>{
  await page.goto(`${base}/app/databases/${database.id}`);await page.getByRole('button',{name:'새 항목',exact:true}).waitFor();
  await page.getByRole('button',{name:'새 항목',exact:true}).click();
  const modal=page.getByRole('dialog');
  await modal.getByLabel('프로젝트',{exact:true}).fill('UI 옵션 검증');
  await modal.getByLabel('상태',{exact:true}).selectOption('진행 중');
  await modal.getByLabel('담당 팀',{exact:true}).fill('검증팀');
  await modal.getByLabel('문서',{exact:true}).check();await modal.getByLabel('AI',{exact:true}).check();
  await modal.getByRole('button',{name:'저장',exact:true}).click();await modal.waitFor({state:'hidden'});
  await page.getByRole('button',{name:'UI 옵션 검증',exact:true}).waitFor();
  const rows=await api(`/databases/${database.id}/rows`);const tested=rows.find(r=>r.values.name==='UI 옵션 검증');assert.deepEqual(tested.values.tags,['문서','AI']);assert.equal(tested.values.status,'진행 중');
  await api(`/databases/${database.id}/rows/${tested.id}`,'DELETE');await page.reload();await page.locator('.data-table').waitFor();await shot('database-table');
  await page.getByRole('button',{name:'보드',exact:true}).click();await shot('database-board');
  await page.getByRole('button',{name:'캘린더',exact:true}).click();await shot('database-calendar');
 });
 await check('profile preferences',async()=>{
  await page.goto(base+'/app/profile');await page.getByLabel('글자 크기',{exact:true}).selectOption('18');await page.getByLabel('테마',{exact:true}).selectOption('dark');await page.getByRole('button',{name:'변경사항 저장'}).click();
  await page.waitForFunction(()=>document.documentElement.dataset.theme==='dark');await page.reload();await page.getByLabel('테마',{exact:true}).waitFor();assert.equal(await page.getByLabel('글자 크기',{exact:true}).inputValue(),'18');await shot('profile-dark');
  await page.getByLabel('글자 크기',{exact:true}).selectOption('16');await page.getByLabel('테마',{exact:true}).selectOption('light');await page.getByRole('button',{name:'변경사항 저장'}).click();await page.waitForFunction(()=>document.documentElement.dataset.theme==='light');
 });
 await check('version context menu',async()=>{await page.locator('.profile-trigger').click();await page.locator('.menu-version').waitFor();await page.locator('.menu-version').getByText('v'+expectedVersion,{exact:true}).waitFor();const notice=page.getByRole('menuitem',{name:'오픈소스 고지',exact:true});assert.equal(await notice.getAttribute('href'),'/licenses.txt');const license=await context.request.get(base+'/licenses.txt');assert.equal(license.status(),200);assert.ok((await license.text()).startsWith('madi — Third-party attribution'));await shot('profile-menu');await page.keyboard.press('Escape')});
 await check('mobile navigation',async()=>{
  await page.setViewportSize({width:390,height:844});await page.goto(base+'/app');await page.locator('.page-heading h1').waitFor();
  assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth+1),'mobile horizontal overflow');await shot('mobile-workspace');
  await page.getByRole('button',{name:'메뉴 열기',exact:true}).click();await page.locator('.sidebar').waitFor();await shot('mobile-menu');
  await page.getByRole('link',{name:'서비스 관리자',exact:false}).click();await page.locator('.page-heading h1').waitFor();assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth+1),'mobile admin horizontal overflow');await shot('mobile-admin');
 });
 assert.deepEqual(issues,[],'Browser runtime errors');
} catch(error){failures.push({name:'fatal',error:error.stack});console.error(error)} finally {
 await browser.close();
 await writeFile(path.join(out,'verification.json'),JSON.stringify({checked_at:new Date().toISOString(),base,screenshots,issues,failures},null,2));
}
if(failures.length){console.error(JSON.stringify(failures,null,2));process.exitCode=1}else console.log(`Verified ${screenshots.length} screenshots with zero runtime errors.`);
