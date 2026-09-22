import assert from 'node:assert/strict';
import {test} from 'node:test';
import {mkdtemp,mkdir,readFile,writeFile,rm,readdir,symlink,utimes} from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import {randomUUID} from 'node:crypto';
import {createServer} from 'node:http';
import {spawn} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import {publishVerifiedScreenshots} from './publish-verified-screenshots.mjs';
import {approvedScreenshots,readPNG,snapshotScreenshots,freshScreenshots} from './screenshot-evidence.mjs';

const png=Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jS9sAAAAASUVORK5CYII=','base64');
const bundle='main-FinalFixture.js';
async function fixture(t) {
 const root=await mkdtemp(path.join(os.tmpdir(),'madi-screenshot-evidence-'));
 t.after(()=>rm(root,{recursive:true,force:true})); // Exact test-owned mkdtemp.
 const batch=randomUUID(),base=Date.now()-60000,results=[];
 const report=path.join(root,'test-results/regression-shared/report.json');
 await mkdir(path.dirname(report),{recursive:true});await mkdir(path.join(root,'docs/screenshots'),{recursive:true});
 const save=()=>writeFile(report,JSON.stringify(results));
 for(const [index,[suite,names]] of Object.entries(approvedScreenshots).entries()) {
  const relative=`test-results/regression-shared/screenshots/${suite}`,dir=path.join(root,relative);await mkdir(dir,{recursive:true});
  const start=base+index*500,end=start+400;
  const screenshots=[];
  const name=suite==='browser'?'graph.png':names[0];
  if(name) {
   const file=path.join(dir,name);await writeFile(file,png);await utimes(file,(start+100)/1000,(start+100)/1000);
   const {bytes,...evidence}=await readPNG(file);screenshots.push({name,...evidence});
  }
  results.push({evidence_version:2,batch_id:batch,suite,bundle,bundle_before:bundle,bundle_after:bundle,started_at:new Date(start).toISOString(),checked_at:new Date(end).toISOString(),seconds:0.4,ok:true,exit_code:0,timed_out:false,evidence_error:'',screenshot_dir:relative,screenshots});
 }
 await save();return {root,results,save,report,publish:options=>publishVerifiedScreenshots({root,bundle,...options})};
}

test('25 suites publish only approved PNGs and choose newest successful duplicate',async t=>{
 const f=await fixture(t);
 f.results.find(r=>r.suite==='browser').screenshots.push(...['failure.png','debug.png','raw.png','diagnostic.png','unapproved.png','unknown-normal.png'].map(name=>({name})));
 await f.save();
 const result=await f.publish();assert.equal(result.suite_count,25);assert.equal(result.skipped.length,6);
 assert.equal(result.published.find(item=>item.target.endsWith('/graph.png')).suite,'graph');
 assert.deepEqual(result.superseded,[{name:'graph.png',kept_suite:'graph',replaced_suite:'browser',reason:'later verified capture time'}]);
 assert.equal((await readdir(path.join(f.root,'docs/screenshots'))).length,result.published.length);
 for(const item of result.published)assert.equal((await readPNG(path.join(f.root,item.target))).sha256,item.sha256);
 assert.equal(JSON.parse(await readFile(path.join(f.root,'test-results/regression-shared/published-screenshots.json'),'utf8')).bundle,bundle);
});

for(const [name,mutate,pattern] of [
 ['missing suite',f=>f.results.pop(),/All 25/],
 ['failed suite',f=>f.results[1].ok=false,/unverified/],
 ['successful process after timeout',f=>f.results[1].timed_out=true,/unverified/],
 ['old report',f=>delete f.results[1].evidence_version,/unverified/],
 ['old bundle',f=>f.results[1].bundle='main-Old.js',/bundle/],
 ['bundle changed mid-suite',f=>f.results[1].bundle_after='main-Other.js',/bundle/],
 ['mixed batches',f=>f.results[1].batch_id=randomUUID(),/different regression batches/],
 ['stale screenshot',f=>f.results[1].screenshots[0].mtime_ms-=60000,/Stale/],
 ['post-run screenshot',f=>f.results[1].screenshots[0].mtime_ms+=60000,/post-run/],
 ['forged digest',f=>f.results[1].screenshots[0].sha256='0'.repeat(64),/changed after/],
 ['legacy output directory',f=>f.results[1].screenshot_dir='test-results/foundations',/directory/],
 ['future completion',f=>f.results[1].checked_at=new Date(Date.now()+60000).toISOString(),/time/],
 ['duration mismatch',f=>f.results[1].seconds=10,/duration/],
 ['duplicate suite',f=>f.results.push(f.results[0]),/duplicate suite/],
 ['no approved screenshots',f=>f.results[1].screenshots=[],/No fresh/],
])test(`refuses ${name} before changing public files`,async t=>{
 const f=await fixture(t);mutate(f);await f.save();await assert.rejects(f.publish(),pattern);
 assert.deepEqual(await readdir(path.join(f.root,'docs/screenshots')),[]);
});

test('rejects modified files, symlink sources and symlink targets',async t=>{
 const f=await fixture(t),r=f.results[1],source=path.join(f.root,r.screenshot_dir,r.screenshots[0].name);
 await writeFile(source,Buffer.concat([png,Buffer.from('changed')]));await assert.rejects(f.publish(),/changed after/);
 await rm(source);await symlink(path.join(f.root,'outside.png'),source);await writeFile(path.join(f.root,'outside.png'),png);await assert.rejects(f.publish(),/regular PNG/);
 await rm(source);await writeFile(source,png);const {bytes,...evidence}=await readPNG(source);r.screenshots[0]={name:r.screenshots[0].name,...evidence};
 const start=Date.now()-100,end=Date.now()+100;r.started_at=new Date(start).toISOString();r.checked_at=new Date(end).toISOString();r.seconds=0.2;await f.save();
 await symlink(path.join(f.root,'outside.png'),path.join(f.root,'docs/screenshots',r.screenshots[0].name));await assert.rejects(f.publish(),/target is not a regular file/);
});

test('dry-run writes no PNG and older evidence cannot replace newer published capture',async t=>{
 const f=await fixture(t);await f.publish({dryRun:true});assert.deepEqual(await readdir(path.join(f.root,'docs/screenshots')),[]);
 const result=await f.publish();result.published[0].captured_at=new Date(Date.now()).toISOString();
 await writeFile(path.join(f.root,'test-results/regression-shared/published-screenshots.json'),JSON.stringify(result));await assert.rejects(f.publish(),/newer verified publication/);
});

test('fresh inventory excludes unchanged previous-run captures',async t=>{
 const root=await mkdtemp(path.join(os.tmpdir(),'madi-screenshot-fresh-'));t.after(()=>rm(root,{recursive:true,force:true}));
 await writeFile(path.join(root,'old.png'),png);const before=await snapshotScreenshots(root),started=Date.now();
 await writeFile(path.join(root,'fresh.png'),png);await writeFile(path.join(root,'failure.png'),png);
 const after=await freshScreenshots(root,before,started,Date.now());assert.deepEqual(after.map(x=>x.name),['fresh.png']);
});

for(const changed of [false,true])test(`runner records actual bundle, interval and fresh digest (${changed?'changed bundle fails':'stable bundle passes'})`,async t=>{
 const root=await mkdtemp(path.join(os.tmpdir(),'madi-screenshot-runner-'));t.after(()=>rm(root,{recursive:true,force:true}));
 await mkdir(path.join(root,'tests'));
 await writeFile(path.join(root,'tests/document-async.mjs'),`import {writeFile} from 'node:fs/promises';import path from 'node:path';await writeFile(path.join(process.env.MADI_SCREENSHOT_DIR,'fresh.png'),Buffer.from(${JSON.stringify(png.toString('base64'))},'base64'));`);
 const shots=path.join(root,'test-results/regression-shared/screenshots/document-async');await mkdir(shots,{recursive:true});await writeFile(path.join(shots,'old.png'),png);
 let requests=0;const server=createServer((req,res)=>{requests++;res.end(`<script src="/assets/${changed&&requests>=3?'main-Changed.js':bundle}"></script>`)});
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));t.after(()=>new Promise(resolve=>server.close(resolve)));
 const result=await new Promise((resolve,reject)=>{
  const child=spawn(process.execPath,[fileURLToPath(new URL('./regression-shared.mjs',import.meta.url)),'document-async'],{cwd:root,env:{...process.env,MADI_BASE_URL:`http://127.0.0.1:${server.address().port}`,MADI_REGRESSION_SCREENSHOT_ROOT:shots.replace(/\/document-async$/,'')},stdio:'pipe'});
  let output='';child.stdout.on('data',data=>output+=data);child.stderr.on('data',data=>output+=data);child.on('error',reject);child.on('exit',code=>resolve({code,output}));
 });
 assert.equal(result.code,changed?1:0,result.output);
 const [report]=JSON.parse(await readFile(path.join(root,'test-results/regression-shared/report.json'),'utf8'));
 assert.equal(report.evidence_version,2);assert.equal(report.bundle_before,bundle);assert.equal(report.bundle_after,changed?'main-Changed.js':bundle);assert.equal(report.ok,!changed);
 assert.ok(Date.parse(report.started_at)<=Date.parse(report.checked_at));assert.deepEqual(report.screenshots.map(item=>item.name),['fresh.png']);assert.equal(report.screenshots[0].sha256,(await readPNG(path.join(shots,'fresh.png'))).sha256);
});
