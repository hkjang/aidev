import { chromium } from '../../../tests/node_modules/playwright/index.mjs';
import assert from 'node:assert/strict';
import { mkdir, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

const base=process.env.MADI_BASE_URL;
const control=process.env.MADI_COLLABORATION_CONTROL;
assert.ok(base&&control,'isolated Go process harness required');
const browser=await chromium.launch({headless:true});
const contexts=await Promise.all([browser.newContext({viewport:{width:1440,height:1000}}),browser.newContext()]);
const [a,b]=await Promise.all(contexts.map(c=>c.newPage()));
const folder=new URL('../../../test-results/collaboration-soak/',import.meta.url);
await mkdir(folder,{recursive:true});
const report={started_at:new Date().toISOString(),duration_target_seconds:120,rounds:0,markers:0,composition:{},checkpoints:[],issues:[]};
for(const page of [a,b]){
  page.on('pageerror',e=>report.issues.push(e.message));
  page.on('response',r=>{if(r.status()>=500)report.issues.push(`${r.status()} ${new URL(r.url()).pathname}`)});
  await page.addInitScript(()=>{
    const Native=window.WebSocket;window.__sockets=[];window.__dropAck=false;window.__lostAck=null;window.__composition={compositionstart:0,compositionupdate:0,compositionend:0};
    for(const key of Object.keys(window.__composition))document.addEventListener(key,()=>window.__composition[key]++);
    window.WebSocket=class extends Native{
      constructor(...args){super(...args);window.__sockets.push(this)}
      set onmessage(fn){super.onmessage=event=>{
        const message=JSON.parse(event.data);
        if(window.__dropAck&&message.type==='ack'&&message.committed&&message.id){
          window.__lostAck={version:message.version,sequence:message.sequence,id:message.id};
          window.__dropAck=false;return;
        }
        fn?.call(this,event);
      }}};
  });
}
const editable=page=>page.locator('.tiptap-content[contenteditable="true"]');
const connected=page=>page.locator('.collaboration-bar[data-state="connected"]').waitFor({timeout:30000});
const confirmed=page=>page.waitForFunction(()=>!!document.querySelector('[data-save-state="confirmed"]'),{},{timeout:20000});
function content(){const root=document.querySelector('.tiptap-content')?.cloneNode(true);root?.querySelectorAll('.collaboration-carets__caret').forEach(n=>n.remove());return root?.textContent||''}
async function api(context,path,method='GET',data){const r=await context.request.fetch(base+'/api/v1'+path,{method,data,headers:{'X-Madi-Request':'1'}});assert.ok(r.ok(),`${method} ${path} ${r.status()} ${await r.text()}`);return r.json()}
async function append(page,text){await editable(page).click();await page.keyboard.press('Control+End');await page.keyboard.insertText(text)}
async function waitMarkers(page,markers){await page.waitForFunction(markers=>{const el=document.querySelector('.tiptap-content')?.cloneNode(true);el?.querySelectorAll('.collaboration-carets__caret').forEach(n=>n.remove());const text=el?.textContent||'';return markers.every(m=>text.includes(m))},markers,{timeout:20000})}
const exactlyOnce=(text,markers)=>{for(const marker of markers)assert.equal(text.split(marker).length-1,1,`marker must occur exactly once: ${marker}`)};
try{
  await api(contexts[0],'/auth/login','POST',{email:'admin@example.test',password:'Integration-Test-Password-2026!'});
  await api(contexts[1],'/auth/login','POST',{email:'collaborator@example.test',password:'Collaboration-Password-2026!'});
  const [workspace]=await api(contexts[0],'/workspaces');
  const prefix='---\ntags: [장시간, 한글]\n---\n';
  const doc=await api(contexts[0],'/documents','POST',{workspace_id:workspace.id,title:'공동 편집 장시간 재시작 검증',markdown:prefix+'# 함께 쓰는 기록\n\n시작'});
  await a.goto(base+'/app/documents/'+doc.id);await connected(a);
  await b.goto(base+'/app/documents/'+doc.id);await connected(b);
  const cdp=await contexts[0].newCDPSession(a);
  const markers=[];let previousVersion=doc.version;const started=Date.now();
  while(Date.now()-started<120000){
    const index=String(report.rounds).padStart(3,'0');const am=`「한글${index}😀」`,bm=`「공동${index}✅」`;
    await editable(a).click();await a.keyboard.press('Control+End');
    await cdp.send('Input.imeSetComposition',{text:'ㅎ',selectionStart:1,selectionEnd:1});
    // A genuine remote Yjs transaction arrives during the local composition.
    await Promise.all([append(b,bm),cdp.send('Input.imeSetComposition',{text:'한그',selectionStart:2,selectionEnd:2})]);
    await cdp.send('Input.imeSetComposition',{text:am,selectionStart:am.length,selectionEnd:am.length});
    await cdp.send('Input.insertText',{text:am});
    markers.push(am,bm);report.rounds++;report.markers=markers.length;
    await Promise.all([waitMarkers(a,markers),waitMarkers(b,markers),confirmed(a),confirmed(b)]);
    const current=await api(contexts[0],'/documents/'+doc.id);
    assert.ok(current.markdown.startsWith(prefix),'front matter must remain byte-exact');
    exactlyOnce(current.markdown,markers);
    assert.ok(current.version>previousVersion,'canonical CAS version must increase after writes');previousVersion=current.version;
    if(report.rounds%10===0){report.checkpoints.push({seconds:(Date.now()-started)/1000,version:current.version,markers:markers.length});console.log(`soak ${report.rounds} rounds / ${markers.length} committed markers / v${current.version}`)}
    await new Promise(resolve=>setTimeout(resolve,800));
  }
  report.actual_edit_seconds=(Date.now()-started)/1000;
  report.composition=await a.evaluate(()=>window.__composition);
  assert.ok(report.composition.compositionstart>=report.rounds&&report.composition.compositionend>=report.rounds,'Chromium must dispatch real composition start/end events');
  await a.evaluate(()=>window.__dropAck=true);
  const lost='「確定ACK消失한글😀」';markers.push(lost);await append(a,lost);
  await a.waitForFunction(()=>!!window.__lostAck);
  const lostAck=await a.evaluate(()=>window.__lostAck);
  const beforeRestart=await api(contexts[0],'/documents/'+doc.id);
  exactlyOnce(beforeRestart.markdown,markers);
  assert.equal(beforeRestart.version,lostAck.version,'dropped ACK describes a committed canonical version');
  assert.equal(await a.locator('[data-save-state="confirmed"]').count(),0,'dropped ACK must not produce false saved indicator');
  const restartAt=Date.now();
  const restart=await fetch(control+'/restart',{method:'POST',headers:{'X-Test-Control':process.env.MADI_COLLABORATION_CONTROL_SECRET}});
  assert.equal(restart.status,200);report.restart=await restart.json();
  await Promise.all([connected(a),connected(b),confirmed(a),confirmed(b)]);
  await Promise.all([waitMarkers(a,markers),waitMarkers(b,markers)]);
  const afterRestart=await api(contexts[0],'/documents/'+doc.id);
  exactlyOnce(afterRestart.markdown,markers);
  assert.equal(afterRestart.markdown,beforeRestart.markdown,'restart/retransmit cannot duplicate or lose committed text');
  assert.equal(afterRestart.version,beforeRestart.version,'idempotent unacknowledged replay cannot manufacture content version');
  report.restart.recovered_ms=Date.now()-restartAt;
  console.log('PASS actual distinct server process restart after dropped commit ACK; same canonical content/version');
  await b.evaluate(()=>window.__sockets.at(-1).send=()=>{});
  const offline='「오래된초안자동반영금지」';await append(b,offline);await contexts[1].setOffline(true);await b.evaluate(()=>window.__sockets.at(-1).close());
  const diagnostics=await api(contexts[0],'/documents/'+doc.id+'/collaboration/diagnostics');
  await api(contexts[0],'/documents/'+doc.id+'/collaboration/compact','POST',{expected_version:afterRestart.version,expected_epoch:diagnostics.epoch,confirm:true});
  await contexts[1].setOffline(false);
  for(const page of [a,b])await page.locator('.collaboration-bar[data-state="conflict"]').waitFor();
  assert.equal((await api(contexts[0],'/documents/'+doc.id)).markdown,afterRestart.markdown);
  const downloadEvent=b.waitForEvent('download');await b.getByRole('button',{name:'현재 초안 다운로드'}).click();
  const download=await downloadEvent;const stream=await download.createReadStream();let recovered='';for await(const part of stream)recovered+=part.toString();assert.ok(recovered.includes(offline));
  report.offline_recovery={download_contains_local_draft:true,canonical_unchanged:true};
  await b.screenshot({path:fileURLToPath(new URL('recovery.png',folder))});
  for(const page of [a,b]){await page.reload();await connected(page)}
  const users=await api(contexts[0],'/admin/users');const user=users.find(u=>u.email==='collaborator@example.test');
  const beforeRevoked=await b.evaluate(content);const revokedAt=Date.now();
  await api(contexts[0],'/admin/users/'+user.id,'PUT',{disabled:true});
  await b.locator('.collaboration-bar[data-state="error"]').waitFor();
  report.revocation_observed_ms=Date.now()-revokedAt;
  await append(a,'「철회뒤비공개변경」');await confirmed(a);
  assert.equal(await b.evaluate(content),beforeRevoked,'silent revoked peer receives no future content');
  await a.screenshot({path:fileURLToPath(new URL('confirmed.png',folder))});
  report.final_version=(await api(contexts[0],'/documents/'+doc.id)).version;
  report.completed_at=new Date().toISOString();assert.deepEqual(report.issues,[]);
  await writeFile(new URL('browser.json',folder),JSON.stringify(report,null,2));
  console.log(JSON.stringify(report));
}catch(error){
  report.failure=String(error);await writeFile(new URL('browser-failure.json',folder),JSON.stringify(report,null,2));
  await a.screenshot({path:fileURLToPath(new URL('failure-a.png',folder))}).catch(()=>{});
  await b.screenshot({path:fileURLToPath(new URL('failure-b.png',folder))}).catch(()=>{});
  console.error('A',await a.locator('body').innerText().catch(()=>''));console.error('B',await b.locator('body').innerText().catch(()=>''));throw error;
}finally{await browser.close()}
