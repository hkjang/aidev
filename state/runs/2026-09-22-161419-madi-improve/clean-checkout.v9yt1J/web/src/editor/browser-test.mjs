import {chromium} from '../../../tests/node_modules/playwright/index.mjs';
import assert from 'node:assert/strict';
import {expect} from '../../../tests/node_modules/playwright/test.mjs';
const base=process.env.MADI_BASE_URL;
assert.ok(base,'isolated Go test server required');
const browser=await chromium.launch({headless:true});
const context=await browser.newContext();
const page=await context.newPage();
const issues=[],external=[];
page.on('pageerror',e=>issues.push(e.message));
page.on('response',r=>{if(r.status()>=500)issues.push(`${r.status()} ${r.url()}`)});
await context.route('**/*',route=>{if(!route.request().url().startsWith(base)&&!route.request().url().startsWith('data:')){external.push(route.request().url());return route.abort()};return route.continue()});
async function api(path,method='GET',data){const response=await context.request.fetch(base+'/api/v1'+path,{method,data,headers:{'X-Madi-Request':'1'}});assert.ok(response.ok(),`${method} ${path} ${response.status()} ${await response.text()}`);return response.json()}
async function waitForSaved(){await page.locator('.collaboration-bar[data-state="connected"] [data-save-state="confirmed"]').waitFor()}
async function open(id){await page.goto(base+'/app/documents/'+id);await waitForSaved()}
async function end(){await page.locator('.tiptap-content').focus();await page.keyboard.press('Control+End')}
async function added(id,type,expected){await end();await page.getByLabel('고급 블록 추가',{exact:true}).selectOption(type);await expect.poll(async()=> (await api('/documents/'+id)).markdown.includes(expected)).toBe(true);await waitForSaved()}
try {
 await api('/auth/login','POST',{email:'admin@example.test',password:'Integration-Test-Password-2026!'});
 const wid=(await api('/workspaces'))[0].id;
 const create=markdown=>api('/documents','POST',{workspace_id:wid,title:'고급 블록 검증',markdown});
 const doc=await create('---\naliases: [테스트]\n---\n# 고급 블록\n\n내용');
 await open(doc.id);
 for(const [type,expected] of [['callout','> [!NOTE]'],['details',':::details'],['columns2',':::columns'],['blockMath','$$'],['inlineMath','$E=mc^2$'],['mermaid','```mermaid'],['footnote','[^주-'],['bookmark',':::bookmark']]) await added(doc.id,type,expected);
 await page.locator('.katex').first().waitFor();
 await page.getByRole('img',{name:'Mermaid 다이어그램'}).first().waitFor();
 let saved=await api('/documents/'+doc.id);
 assert.ok(saved.markdown.startsWith('---\naliases: [테스트]\n---\n'));
 const types=saved.block_metadata.blocks.map(b=>b.type);
 for(const type of ['callout','details','columns','blockMath','codeBlock','footnoteDefinition','bookmark'])assert.ok(types.includes(type),`missing ${type}: ${JSON.stringify(types)}`);
 const canonical=saved.markdown;
 await page.reload();await waitForSaved();assert.equal((await api('/documents/'+doc.id)).markdown,canonical);
 const roundtrip=await create(canonical);await open(roundtrip.id);
 const roundtripDoc=await api('/documents/'+roundtrip.id);
 for(const type of ['callout','details','columns','blockMath','codeBlock','footnoteDefinition','bookmark'])assert.ok(roundtripDoc.block_metadata.blocks.some(block=>block.type===type),`source roundtrip lost ${type}: ${roundtripDoc.markdown}`);
 await page.getByRole('button',{name:'읽기',exact:true}).click().catch(()=>{});
 console.log('PASS advanced native insertion, durable Markdown/frontmatter/IDs, local math and Mermaid');

 // A REST/source document containing merged cells must survive first CRDT seed.
 const table=await create('<table><tr><th colspan="2" colwidth="120,180"><p>병합 제목</p></th></tr><tr><td><p style="text-align:center">가운데</p></td><td><p>오른쪽</p></td></tr></table>');
 await open(table.id);
 saved=await api('/documents/'+table.id);
 assert.ok(saved.markdown.includes('colspan="2"'),saved.markdown);
 assert.ok(saved.markdown.includes('colwidth="120,180"'),saved.markdown);
 assert.ok(saved.markdown.includes('text-align:center'),saved.markdown);
 assert.equal(await page.locator('.tiptap-content th[colspan="2"]').count(),1);
 await page.locator('.tiptap-content td').first().click();
 await page.getByLabel('문단 정렬',{exact:true}).selectOption('right');
 await expect.poll(async()=> (await api('/documents/'+table.id)).markdown.includes('text-align:right')).toBe(true);
 console.log('PASS merged/resized/aligned table source survives CRDT projection');

 const unknown=await create('<custom-company-diagram secret="keep">보존할 원문</custom-company-diagram>');
 await page.goto(base+'/app/documents/'+unknown.id);
 await page.getByText('원문 보존 모드',{exact:true}).waitFor();
 assert.equal((await api('/documents/'+unknown.id)).markdown,unknown.markdown);
 assert.equal(await page.locator('.collaboration-bar').count(),0);
 console.log('PASS unknown import is not silently rewritten');

 // Whole-document synced references update and clear on revoked source ACL.
 const source=await create('임베드 원본 최초');
 const target=await create(`![[${source.id}]]`);
 await open(target.id);await page.locator('.synced-embed').getByText('임베드 원본 최초',{exact:true}).waitFor();
 let sourceCurrent=await api('/documents/'+source.id);
 await api('/documents/'+source.id,'PUT',{version:sourceCurrent.version,markdown:'임베드 원본 갱신'});
 await page.locator('.synced-embed').getByText('임베드 원본 갱신',{exact:true}).waitFor();
 sourceCurrent=await api('/documents/'+source.id);
 await api('/documents/'+source.id,'DELETE');
 await page.waitForFunction(()=>!document.querySelector('.synced-embed')?.textContent.includes('임베드 원본 갱신'));
 console.log('PASS synced reference tracks original and clears deleted content');
 const protectedSource=await create('권한 회수 전에 보이는 내용');
 const protectedTarget=await create(`![[${protectedSource.id}]]`);
 const reader=await browser.newContext();
 const login=await reader.request.post(base+'/api/v1/auth/login',{data:{email:'collaborator@example.test',password:'Collaboration-Password-2026!'},headers:{'X-Madi-Request':'1'}});assert.ok(login.ok());
 const readerPage=await reader.newPage();readerPage.on('pageerror',e=>issues.push(e.message));
 await readerPage.goto(base+'/app/documents/'+protectedTarget.id);
 await readerPage.locator('.synced-embed').getByText('권한 회수 전에 보이는 내용',{exact:true}).waitFor();
 await api('/documents/'+protectedSource.id,'PUT',{version:protectedSource.version,visibility:'private'});
 await readerPage.waitForFunction(()=>!document.querySelector('.synced-embed')?.textContent.includes('권한 회수 전에 보이는 내용'));
 assert.ok((await readerPage.locator('.synced-embed').innerText()).includes('권한'));
 await reader.close();
 const cycleA=await create('순환 A'),cycleB=await create(`![[${cycleA.id}]]`);
 await api('/documents/'+cycleA.id,'PUT',{version:cycleA.version,markdown:`![[${cycleB.id}]]`});
 await open(cycleA.id);await page.locator('.synced-embed.error').getByText('순환 참조 또는 최대 5단계 중첩입니다.',{exact:false}).waitFor();
 assert.ok(await page.locator('.synced-embed').count()<6);
 console.log('PASS source ACL revocation clears cached embed and cycles are bounded');
 assert.deepEqual(issues,[]);assert.deepEqual(external,[],'offline UI must not fetch remote assets');
} catch(error) {await page.screenshot({path:'/tmp/madi-editor-failure.png',fullPage:true}).catch(()=>{});console.error('ISSUES',issues);console.error(await page.locator('body').innerText());throw error;} finally {await browser.close()}
