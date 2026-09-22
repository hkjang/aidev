import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import path from 'node:path';
import { chromium, expect } from 'playwright/test';

const base=process.env.MADI_BASE_URL||'http://127.0.0.1:8080';
const output=path.resolve(process.env.MADI_SCREENSHOT_DIR||'test-results/ux-structure');
const browser=await chromium.launch(), context=await browser.newContext({viewport:{width:1512,height:1080},locale:'ko-KR',reducedMotion:'reduce'}),page=await context.newPage();
const errors=[];page.on('pageerror',e=>errors.push(e.message));
async function api(url,method='GET',data,status=200){const r=await context.request.fetch(base+'/api/v1'+url,{method,data,headers:{'X-Madi-Request':'1'}});assert.equal(r.status(),status,`${method} ${url}: ${await r.text()}`);return r.json()}
async function settled(check,message){await expect.poll(check,{timeout:15000,message}).toBe(true)}
async function shot(name){await mkdir(output,{recursive:true});await page.evaluate(()=>document.fonts.ready);await page.screenshot({path:path.join(output,name+'.png'),fullPage:page.viewportSize().width>500&&!(await page.getByRole('dialog').count()),animations:'disabled'})}
async function overflow(){assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1),'horizontal overflow')}
const tab=name=>page.getByRole('tab',{name,exact:true});
let ws,user,doc;
try{
 await api('/auth/login','POST',{email:'admin@example.test',password:process.env.MADI_ADMIN_PASSWORD||'Browser-Test-Password-2026!'});
 const password='UX-Structure-Password-2026!',email=`ux-${Date.now()}@example.test`;
 user=await api('/admin/users','POST',{email,password,name:'문서 탐색 사용자',role:'editor',kind:'user'});
 await api('/auth/logout','POST',{});await api('/auth/login','POST',{email,password});
 ws=await api('/workspaces','POST',{name:'목적별 지식 탐색 검증'});
 const markdown='# 흐름을 잃지 않는 지식\n\n'+Array.from({length:100},(_,i)=>`검증 문장 ${i+1}: 연결과 기록의 흐름을 이어갑니다.\n`).join('\n')+'\n- [ ] 현재 문서 작업\n';
 doc=await api('/documents','POST',{workspace_id:ws.id,title:'문서 맥락 검증',markdown,visibility:'workspace'});
 const other=await api('/documents','POST',{workspace_id:ws.id,title:'다른 문서',markdown:'- [ ] 다른 문서 작업\n',visibility:'workspace'});
 await page.goto(base+'/app');await page.getByLabel('워크스페이스 선택',{exact:true}).selectOption(ws.id);
 await expect(page.getByLabel('내 작업 방식',{exact:true})).toHaveValue('wiki');
 assert.ok(await page.getByRole('navigation',{name:'주요 메뉴',exact:true}).getByRole('link').count()<=9);
 await page.getByLabel('내 작업 방식',{exact:true}).selectOption('database');
 await settled(async()=>(await api('/profile')).preferences.nav_preset==='database','preset stored');
 await page.reload();await expect(page.getByLabel('내 작업 방식',{exact:true})).toHaveValue('database');
 await expect(page.getByRole('button',{name:'바로 메모',exact:true})).toBeVisible();
 const beforeCapture=await api(`/captures?workspace_id=${ws.id}`);await page.getByRole('button',{name:'바로 메모',exact:true}).click();
 await page.waitForURL(/\/app\/documents\/[^/]+\?mode=edit$/);const memoID=new URL(page.url()).pathname.split('/').at(-1);
 const memo=await api('/documents/'+memoID);assert.equal(memo.visibility,'private');assert.equal(memo.title,'새 개인 메모');assert.equal((await api(`/captures?workspace_id=${ws.id}`)).length,beforeCapture.length+1);
 await page.goto(base+'/app');await shot('focused-home');await page.getByRole('link',{name:'내 처리함 열기',exact:true}).click();await page.getByRole('heading',{name:'내 처리함',exact:true}).waitFor();await expect(page.getByRole('link',{name:'새 개인 메모',exact:true}).last()).toBeVisible();await shot('my-work');await page.goto(base+'/app');
 await expect(page.getByRole('navigation',{name:'주요 메뉴',exact:true}).getByRole('link',{name:'데이터베이스',exact:true})).toBeVisible();
 await page.getByRole('button',{name:'전체 도구 보기',exact:true}).click();await page.getByLabel('전체 도구 검색',{exact:true}).fill('작업');await expect(page.locator('#advanced-navigation').getByRole('link',{name:'작업 이력',exact:true})).toBeVisible();await page.getByRole('button',{name:'작업 이력 도구 고정',exact:true}).click();await settled(async()=>(await api('/profile')).preferences.navigation_pins?.includes('/app/jobs')===true,'tool pinned');await page.locator('#advanced-navigation').getByRole('link',{name:'작업 이력',exact:true}).click();
 await page.getByRole('button',{name:'전체 도구 접기',exact:true}).click();
 await expect(page.getByRole('navigation',{name:'주요 메뉴',exact:true}).getByRole('link',{name:'작업 이력',exact:true})).toHaveClass(/active/);
 await page.reload();await expect(page.getByRole('navigation',{name:'주요 메뉴',exact:true}).getByRole('link',{name:'작업 이력',exact:true})).toBeVisible();
 await shot('navigation-purpose');
 const settings=await api(`/workspaces/${ws.id}/settings`);await api(`/workspaces/${ws.id}/settings`,'PUT',{version:settings.version,data:{feature_flags:{canvas:false}}});
 await page.goto(base+'/app');await page.getByRole('button',{name:'전체 도구 보기',exact:true}).click();
 await expect(page.locator('.sidebar').getByRole('link',{name:'캔버스',exact:true})).toHaveCount(0);
 await page.goto(base+`/app/documents/${doc.id}?mode=source`);await page.getByLabel('Markdown 원문 편집',{exact:true}).waitFor();
 const tagInput=page.getByRole('combobox',{name:'문서 태그',exact:true});await tagInput.dispatchEvent('compositionstart',{data:'태'});await tagInput.fill('태그검증');await tagInput.dispatchEvent('keydown',{key:'Enter',code:'Enter',keyCode:229,isComposing:true});assert.equal(await page.getByRole('button',{name:'태그검증 태그 삭제',exact:true}).count(),0,'IME composing Enter must not create a tag');await tagInput.dispatchEvent('compositionend',{data:'태그검증'});await tagInput.press('Enter');await page.getByRole('button',{name:'태그검증 태그 삭제',exact:true}).waitFor();await settled(async()=>(await api('/documents/'+doc.id)).tags.includes('태그검증'),'tag chip persisted');
 await page.getByRole('button',{name:'태그검증 태그 삭제',exact:true}).click();await settled(async()=>!(await api('/documents/'+doc.id)).tags.includes('태그검증'),'tag chip removed');
 await expect(page.locator('.editor-mode-bar').getByRole('button',{name:'Markdown',exact:true})).toHaveCount(0);await page.getByRole('button',{name:'문서 보기 도구',exact:true}).click();await page.getByRole('menuitem',{name:'Markdown 원문',exact:true}).click();await page.getByLabel('Markdown 원문 편집',{exact:true}).waitFor();
 await expect(page.locator('.document-states')).toContainText('저장');await expect(page.locator('.document-states')).toContainText('내부 공유');await expect(page.locator('.document-states')).toContainText('게시 단계');await expect(page.locator('.document-states')).toContainText('외부 링크');
 for(const name of ['속성','댓글','이력','AI','연결']){await tab(name).click();await expect(tab(name)).toHaveAttribute('aria-selected','true');await overflow()}
 await tab('속성').click();await page.getByRole('button',{name:'공동 편집 진단',exact:true}).click();await page.getByRole('dialog').waitFor();await page.getByRole('dialog').getByRole('button',{name:'닫기',exact:true}).click();
 await tab('댓글').click();await settled(async()=>(await api('/profile')).preferences.document_panel==='comments','panel persisted');
 await page.reload();await expect(tab('댓글')).toHaveAttribute('aria-selected','true');
 await tab('연결').click();await shot('document-inspector');
 const input=page.getByLabel('Markdown 원문 편집',{exact:true});
 const sourcePositionVersion=await page.locator('[data-document-id]').getAttribute('data-document-version');
 await input.evaluate(el=>{el.focus();el.setSelectionRange(50,75);el.scrollTop=420;window.scrollTo(0,240)});
 await page.locator('.document-inspector').getByRole('link',{name:'문서의 할 일 확인',exact:true}).click();
 await expect(page.getByText('현재 문서 작업',{exact:true})).toBeVisible();await expect(page.getByText('다른 문서 작업',{exact:true})).toHaveCount(0);
 const peek=page.getByRole('button',{name:'문서 맥락 검증 문서 미리보기',exact:true});await peek.click();const preview=page.getByRole('dialog',{name:'문서 미리보기',exact:true});await expect(preview).toContainText('현재 문서 작업');await expect(preview.getByRole('link',{name:'문서 전체 열기'})).toHaveAttribute('href',/line=202/);await preview.getByRole('button',{name:'닫기',exact:true}).click();await expect(peek).toBeFocused();
 const departedPosition=await page.evaluate(({actor,workspace,document})=>JSON.parse(sessionStorage.getItem(`madi.position.${actor}.${workspace}`)||'{}')[`/app/documents/${document}?mode=source`],{actor:user.id,workspace:ws.id,document:doc.id});
 assert.deepEqual(departedPosition?.selection,[50,75],'queued navigation saves must preserve the departed source selection');
 assert.equal(departedPosition?.editorY,420,'a task/loading DOM must not erase the source editor position');
 assert.equal(departedPosition?.version,sourcePositionVersion,'departing source version is retained until an actual new document is loaded');
 await page.getByRole('link',{name:/문서로 돌아가기/}).click();await input.waitFor();
 await settled(async()=>input.evaluate(el=>el.selectionStart===50&&el.selectionEnd===75),'source selection restored');
 assert.equal(await input.evaluate(el=>el.scrollTop),420,'source editor scroll restored at normal viewport height');
 // The old window offset can become unreachable after resizing on another
 // page. Its height must not gate restoration of a still-current selection.
 await input.evaluate(el=>{el.focus();el.setSelectionRange(90,115);el.scrollTop=280;window.scrollTo(0,240)});
 await page.locator('.document-inspector').getByRole('link',{name:'문서의 할 일 확인',exact:true}).click();
 await expect(page.getByText('현재 문서 작업',{exact:true})).toBeVisible();
 await page.setViewportSize({width:1512,height:1800});
 await page.getByRole('link',{name:/문서로 돌아가기/}).click();await input.waitFor();
 assert.ok(await page.evaluate(()=>document.documentElement.scrollHeight-innerHeight<236),'previous window y=240 is unreachable in the taller viewport');
 await settled(async()=>input.evaluate(el=>el.selectionStart===90&&el.selectionEnd===115&&el.scrollTop===280),'selection and editor scroll restored independently of window height');
 // A pending window-position restoration must not replay over a later user
 // selection when the viewport becomes scrollable again.
 await input.evaluate(el=>{el.focus();el.setSelectionRange(120,140)});
 await page.setViewportSize({width:1512,height:1080});
 await settled(async()=>page.evaluate(()=>Math.abs(scrollY-240)<=4),'pending window position restored after resize');
 assert.deepEqual(await input.evaluate(el=>[el.selectionStart,el.selectionEnd]),[120,140],'later selection is not overwritten by window restoration');
 // Position memory is deliberately invalid across a canonical version change.
 await input.evaluate(el=>{el.focus();el.setSelectionRange(160,180);el.scrollTop=300;window.scrollTo(0,240)});
 await page.locator('.document-inspector').getByRole('link',{name:'문서의 할 일 확인',exact:true}).click();
 await expect(page.getByText('현재 문서 작업',{exact:true})).toBeVisible();
 const priorPositionVersion=await api('/documents/'+doc.id);
 const newerPositionVersion=await api('/documents/'+doc.id,'PUT',{version:priorPositionVersion.version,title:priorPositionVersion.title+' 최신'});
 await page.getByRole('link',{name:/문서로 돌아가기/}).click();await input.waitFor();
 await expect(page.locator('[data-document-id="'+doc.id+'"]')).toHaveAttribute('data-document-version',String(newerPositionVersion.version));
 assert.deepEqual(await input.evaluate(el=>[el.selectionStart,el.selectionEnd,el.scrollTop]),[0,0,0],'a different source version does not receive stale selection/editor scroll');
 const position=await page.evaluate(()=>Object.entries(sessionStorage).filter(([k])=>k.startsWith('madi.position.')).map(([,v])=>v).join(''));
 assert.ok(!position.includes('연결과 기록의 흐름'),'position storage contains no source text');
 // A real concurrent REST write must produce a 409, followed by explicit CAS review.
 const local=markdown+'\n내가 작성한 미확정 추가 내용\n';await input.fill(local);
 const current=await api('/documents/'+doc.id);await api('/documents/'+doc.id,'PUT',{version:current.version,title:'서버에서 바뀐 제목',markdown:markdown+'\n다른 편집자의 변경\n'});
 await page.getByRole('heading',{name:'다른 변경과 충돌했습니다',exact:true}).waitFor();
 await page.getByRole('button',{name:'서버 내용과 비교',exact:true}).click();
 const review=page.getByRole('dialog',{name:'서버 내용과 내 변경 비교',exact:true});await review.waitFor();
 await expect(review).toContainText('다른 편집자의 변경');await expect(review).toContainText('내가 작성한 미확정 추가 내용');await shot('document-conflict-review');
 await review.getByRole('button',{name:'취소',exact:true}).click();assert.equal(await input.inputValue(),local);
 await page.getByRole('button',{name:'서버 내용과 비교',exact:true}).click();await review.getByRole('button',{name:'확인한 버전을 기준으로 저장',exact:true}).click();
 await settled(async()=>(await api('/documents/'+doc.id)).markdown===local,'confirmed CAS saved local draft');
 // The UI must retain pending text after actual session revocation, without retrying blindly.
 await api('/auth/logout','POST',{});await input.fill(local+'세션 종료 후 로컬 초안');await page.getByRole('heading',{name:'로그인이 필요합니다',exact:true}).waitFor();await expect(page.getByRole('button',{name:'내 변경 보관',exact:true})).toBeVisible();await expect(page.locator('.recovery-notice').getByRole('button',{name:'다시 시도',exact:true})).toHaveCount(0);
 await shot('document-session-recovery');await api('/auth/login','POST',{email,password});
 // Clear only this test's local draft after proving it was preserved, then inspect mobile layouts.
 await page.evaluate(id=>sessionStorage.removeItem(`madi.draft.${id.user}.${id.doc}`),{user:user.id,doc:doc.id});
 await page.goto(base+`/app/documents/${doc.id}?mode=preview`);await page.locator('.document-body').waitFor();await page.setViewportSize({width:390,height:844});
 for(const name of ['연결','속성','댓글','AI','이력']){await tab(name).click();await overflow()}
 await shot('document-inspector-mobile');
 await page.goto(base+'/app/profile');await expect(page.getByLabel('기본 작업 방식',{exact:true})).toHaveValue('database');await expect(page.getByLabel('고급 도구 메뉴',{exact:true})).toHaveValue('on');await overflow();await shot('profile-navigation-mobile');
 assert.deepEqual(errors,[]);console.log(JSON.stringify({ok:true,workspace:ws.id,document:doc.id,other:other.id,checks:['presets','current-route','feature-gate','home-private-capture','my-work','tool-search-pins','IME-tag-chips','two-modes-four-states','inspector-tabs','server-preferences','task-filter','preview-focus-return','selection-context','selection-height-independent','selection-user-preserved','selection-version-guard','409-CAS-review','session-recovery','mobile-390'],screenshots:output}));
}catch(e){console.log('navigation geometry',await page.evaluate(()=>{const el=document.querySelector('.markdown-source');return{path:location.pathname+location.search,version:document.querySelector('[data-document-version]')?.dataset.documentVersion,length:el?.value.length,selection:el?[el.selectionStart,el.selectionEnd]:null,editorY:el?.scrollTop,y:scrollY,height:innerHeight,maxScroll:document.documentElement.scrollHeight-innerHeight,positions:Object.entries(sessionStorage).filter(([key])=>key.startsWith('madi.position.'))}}).catch(()=>null));await mkdir(output,{recursive:true});await page.screenshot({path:path.join(output,'failure.png'),fullPage:true}).catch(()=>{});throw e}finally{await browser.close()}
