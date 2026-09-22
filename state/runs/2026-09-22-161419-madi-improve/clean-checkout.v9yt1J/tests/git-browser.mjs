import { chromium } from 'playwright';
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdir } from 'node:fs/promises';
import path from 'node:path';

const base=process.env.MADI_BASE_URL||'http://127.0.0.1:8080';
assert.ok(/^http:\/\/(127\.0\.0\.1|localhost):\d+$/.test(base),'Git browser fixture is restricted to a disposable local service');
const output=path.resolve(process.env.MADI_SCREENSHOT_DIR||'test-results/git');await mkdir(output,{recursive:true});
const fixture=spawn(path.resolve('.local/git-http-fixture'),[],{stdio:['ignore','pipe','pipe']});
const remote=await new Promise((resolve,reject)=>{let text='';const timer=setTimeout(()=>reject(new Error('Git TLS fixture startup timed out')),10000);fixture.stdout.on('data',chunk=>{text+=chunk;if(text.includes('\n')){clearTimeout(timer);try{resolve(JSON.parse(text.split('\n')[0]))}catch(e){reject(e)}}});fixture.once('error',reject)});
const browser=await chromium.launch({headless:true}),context=await browser.newContext({viewport:{width:1440,height:1000},locale:'ko-KR'}),page=await context.newPage();
const errors=[];page.on('pageerror',e=>errors.push(e.message));page.on('response',r=>{if(r.status()>=500&&r.url().startsWith(base))errors.push(`${r.status()} ${r.url()}`)});
async function api(endpoint,method='GET',data){const response=await context.request.fetch(base+'/api/v1'+endpoint,{method,data,headers:{'X-Madi-Request':'1'}});assert.ok(response.ok(),`${method} ${endpoint}: ${response.status()} ${await response.text()}`);return response.json()}
async function shot(name){await page.evaluate(async()=>{await document.fonts.ready;window.scrollTo(0,0);await new Promise(resolve=>requestAnimationFrame(resolve))});await page.screenshot({path:path.join(output,name+'.png'),fullPage:true,animations:'disabled'})}
let oldPolicy,changed=false,connection;
try{
 await page.goto(base+'/login');await api('/auth/login','POST',{email:process.env.MADI_TEST_EMAIL||'admin@example.test',password:process.env.MADI_TEST_PASSWORD||'Browser-Test-Password-2026!'});const user=await api('/auth/me');
 const workspace=await api('/workspaces','POST',{name:'Git 지식 동기화 검증 '+Date.now()});await page.evaluate(id=>localStorage.setItem('madi.workspace',id),workspace.id);
 const doc=await api('/documents','POST',{workspace_id:workspace.id,title:'Git 기반 지식 운영 안내',visibility:'private',markdown:'# Git 기반 지식 운영\n\n선택한 문서만 확인하고 전송합니다.\n\n- [x] 고정 원격과 브랜치 검증\n- [x] 현재 문서 권한 검사\n- [ ] 외부 열람 권한 확인'});
 oldPolicy=await api('/admin/git-sync/settings');await api('/admin/git-sync/settings','PUT',{revision:oldPolicy.revision,settings:{...oldPolicy.settings,enabled:true,allowed_hosts:[...new Set([...(oldPolicy.settings.allowed_hosts||[]),'127.0.0.1'])],allow_private_networks:true}});changed=true;
 await page.goto(base+'/admin/git-sync');await page.getByRole('heading',{name:'Git 연동 정책',exact:true}).waitFor();await page.getByLabel('Git 기능',{exact:true}).selectOption('false');await page.getByLabel('Git 기능',{exact:true}).selectOption('true');await page.getByRole('button',{name:'정책 저장',exact:true}).click();await page.getByText('Git 정책을 저장했습니다. 이전 미리보기 동의는 무효화됩니다',{exact:true}).waitFor();await shot('admin-git-sync');
 await page.goto(base+'/app/git-sync');await page.getByRole('button',{name:'연결 추가',exact:true}).click();await page.getByLabel('연결 이름',{exact:true}).fill('내부 지식 저장소 · 검증');await page.getByLabel('원격 URL (HTTPS 또는 SSH)',{exact:true}).fill(remote.url);await page.getByLabel('HTTPS 사용자 이름',{exact:true}).fill('fixture');await page.getByLabel('HTTPS 암호·개인 토큰',{exact:true}).fill('Git-Browser-Fixture-Only!');await page.getByLabel('HTTPS 내부 CA 인증서 (선택)',{exact:true}).fill(remote.ca_pem);await page.getByLabel('연결 상태',{exact:true}).selectOption('true');await page.getByRole('button',{name:'연결 저장',exact:true}).click();await page.getByText('암호화된 Git 연결을 저장했습니다',{exact:true}).waitFor();
 connection=(await api(`/git-sync/connections?workspace_id=${workspace.id}`))[0];assert.equal(connection.owner_id,user.id);assert.equal(connection.config.password,undefined);
 await page.getByRole('button',{name:'읽기 연결 진단',exact:true}).click();await page.getByText(/빈 저장소의 읽기 연결을 확인|저장된 대상의 브랜치 읽기 연결/).waitFor();
 await page.getByRole('button',{name:'연결 설정',exact:true}).click();assert.equal(await page.getByLabel('HTTPS 암호·개인 토큰',{exact:true}).inputValue(),'');await page.getByRole('button',{name:'연결 저장',exact:true}).click();await page.getByText('암호화된 Git 연결을 저장했습니다',{exact:true}).waitFor();
 await page.getByLabel('이 고정 원격 저장소의 브랜치·파일을 읽겠습니다',{exact:true}).check();await page.getByRole('checkbox',{name:/Git 기반 지식 운영 안내/}).check();await page.getByRole('button',{name:'원격 읽기·미리보기',exact:true}).click();await page.getByRole('button',{name:'목록 확인 후 실행 동의',exact:true}).waitFor({timeout:45000});await shot('git-sync-preview');
 await page.reload();await page.getByRole('button',{name:'목록 확인 후 실행 동의',exact:true}).waitFor();await page.getByRole('button',{name:'목록 확인 후 실행 동의',exact:true}).click();await page.getByLabel('전송 목록·독립된 Git 권한·변경 결과를 확인하고 동의합니다',{exact:true}).check();await shot('git-sync-confirm');await page.getByRole('button',{name:'동의한 변경 실행',exact:true}).click();
 const runID=new URL(page.url()).searchParams.get('run');await assertPoll(async()=>{const run=await api(`/git-sync/runs/${runID}`);if(['failed','conflict','unknown'].includes(run.status))throw new Error(JSON.stringify(run));return run.status==='succeeded'},45000);await page.getByText('완료',{exact:true}).first().waitFor();await shot('git-sync');
 await page.setViewportSize({width:390,height:844});await shot('mobile-git-sync');assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1));await page.goto(base+'/admin/git-sync');await page.getByRole('heading',{name:'Git 연동 정책',exact:true}).waitFor();await shot('mobile-git-policy');assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1));
 assert.deepEqual(errors,[]);console.log(JSON.stringify({ok:true,workspace_id:workspace.id,document_id:doc.id,connection_id:connection.id,run_id:runID,screenshots:output}));
}finally{
 try{
  if(connection){const current=(await api(`/git-sync/connections?workspace_id=${connection.workspace_id}`)).find(c=>c.id===connection.id);if(current){await api(`/git-sync/connections/${current.id}`,'PUT',{...current,enabled:false,config:Object.fromEntries(Object.entries(current.config).filter(([key,value])=>!key.endsWith('_configured')&&typeof value==='string'))})}}
 }finally{
  try{if(changed&&oldPolicy){const current=await api('/admin/git-sync/settings');await api('/admin/git-sync/settings','PUT',{revision:current.revision,settings:oldPolicy.settings})}}
  finally{await browser.close();fixture.kill('SIGTERM')}
 }
}
async function assertPoll(fn,timeout){const deadline=Date.now()+timeout;while(Date.now()<deadline){if(await fn())return;await new Promise(resolve=>setTimeout(resolve,250))}throw new Error('Git job timed out')}
