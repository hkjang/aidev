import assert from 'node:assert/strict';
import {mkdir,readFile,writeFile,lstat,rename} from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {approvedScreenshots,digest,normalPNG,readPNG,safeDirectory,validBundle} from './screenshot-evidence.mjs';

// Never infer success from old directories or the currently checked-out bundle.
export async function publishVerifiedScreenshots({root=process.cwd(),bundle,dryRun=false}={}) {
 assert.ok(validBundle(bundle),'Pass --bundle with the exact verified served bundle');
 root=await safeDirectory(root);
 const directory=path.join(root,'test-results/regression-shared');
 const reportBytes=await readFile(path.join(directory,'report.json'));
 const results=JSON.parse(reportBytes),suites=Object.keys(approvedScreenshots),latest=new Map();
 assert.ok(Array.isArray(results),'Invalid suite report');
 for(const result of results) {
  if(!suites.includes(result.suite))continue;
  assert.ok(!latest.has(result.suite),'Ambiguous duplicate suite report: '+result.suite);
  latest.set(result.suite,result);
 }
 assert.equal(latest.size,suites.length,'All 25 successful suite reports are required');
 const batchIDs=new Set(),candidates=new Map(),skipped=[],superseded=[];
 let totalBytes=0;
 for(const suite of suites) {
  const result=latest.get(suite),start=Date.parse(result.started_at),end=Date.parse(result.checked_at);
  assert.ok(result.evidence_version===2&&result.ok===true&&result.exit_code===0&&!result.timed_out&&!result.evidence_error,'Refusing unverified screenshots: '+suite);
  assert.ok(typeof result.batch_id==='string'&&/^[a-f0-9-]{36}$/.test(result.batch_id),'Missing batch identity');batchIDs.add(result.batch_id);
  assert.ok(result.bundle===bundle&&result.bundle_before===bundle&&result.bundle_after===bundle,'Wrong or changed bundle: '+suite);
  assert.ok(Number.isFinite(start)&&Number.isFinite(end)&&start<=end&&end<=Date.now()+2000&&end-start<=245000,'Invalid suite time: '+suite);
  assert.ok(Number.isFinite(result.seconds)&&Math.abs(result.seconds-(end-start)/1000)<=0.11,'Inconsistent suite duration: '+suite);
  const relative=`test-results/regression-shared/screenshots/${suite}`;
  assert.equal(result.screenshot_dir,relative,'Unexpected screenshot directory: '+suite);
  const sourceDir=await safeDirectory(path.join(root,relative));
  assert.ok(Array.isArray(result.screenshots),'Missing screenshot inventory: '+suite);
  const names=new Set();let eligible=0;
  for(const evidence of result.screenshots) {
   assert.ok(typeof evidence.name==='string'&&!names.has(evidence.name),'Invalid duplicate screenshot');names.add(evidence.name);
   if(!normalPNG(evidence.name)||!approvedScreenshots[suite].includes(evidence.name)) {skipped.push({suite,name:evidence.name,reason:'not an approved normal capture'});continue}
   assert.ok(Number.isFinite(evidence.mtime_ms)&&evidence.mtime_ms>=start-2000&&evidence.mtime_ms<=end+2000,'Stale or post-run screenshot: '+evidence.name);
   const file=await readPNG(path.join(sourceDir,evidence.name));
   assert.ok(file.sha256===evidence.sha256&&file.size===evidence.size&&Math.abs(file.mtime_ms-evidence.mtime_ms)<0.001&&Math.abs(file.ctime_ms-evidence.ctime_ms)<0.001,'Screenshot changed after successful suite: '+evidence.name);
   totalBytes+=file.size;assert.ok(totalBytes<=256*1024*1024,'Screenshot publication byte limit');eligible++;
   const candidate={suite,bundle,batch_id:result.batch_id,started_at:result.started_at,checked_at:result.checked_at,captured_at:new Date(file.mtime_ms).toISOString(),mtime_ms:file.mtime_ms,sha256:file.sha256,bytes:file.size,source:`${relative}/${evidence.name}`,target:`docs/screenshots/${evidence.name}`,data:file.bytes};
   const old=candidates.get(evidence.name);
   if(old) {
    assert.ok(old.mtime_ms!==candidate.mtime_ms||old.sha256===candidate.sha256,'Ambiguous equal-time screenshot collision: '+evidence.name);
    const newer=candidate.mtime_ms>old.mtime_ms||(candidate.mtime_ms===old.mtime_ms&&Date.parse(candidate.checked_at)>Date.parse(old.checked_at));
    superseded.push({name:evidence.name,kept_suite:newer?suite:old.suite,replaced_suite:newer?old.suite:suite,reason:'later verified capture time'});
    if(!newer)continue;
   }
   candidates.set(evidence.name,candidate);
  }
  if(approvedScreenshots[suite].length)assert.ok(eligible>0,'No fresh approved screenshots: '+suite);
 }
 assert.equal(batchIDs.size,1,'Refusing to combine different regression batches');
 const previousPath=path.join(directory,'published-screenshots.json');
 let previous;
 try {previous=JSON.parse(await readFile(previousPath,'utf8'))}catch(error){if(error.code!=='ENOENT')throw error}
 const targetDir=path.join(root,'docs/screenshots');await mkdir(targetDir,{recursive:true});await safeDirectory(targetDir);
 for(const [name,candidate] of candidates) {
  const target=path.join(targetDir,name);let info;
  try {info=await lstat(target)}catch(error){if(error.code!=='ENOENT')throw error}
  assert.ok(!info||(info.isFile()&&!info.isSymbolicLink()),'Publication target is not a regular file');
  const old=previous?.format==='madi-screenshot-publication-v2'?previous.published.find(item=>item.target===candidate.target):null;
  if(old&&Date.parse(old.captured_at)>candidate.mtime_ms) {
   assert.ok(info&&digest(await readFile(target))===old.sha256,'Newer published evidence no longer matches its target');
   throw new Error('Refusing to replace a newer verified publication: '+name);
  }
 }
 const published=[...candidates.values()].map(({data,mtime_ms,...entry})=>entry);
 const manifest={format:'madi-screenshot-publication-v2',published_at:new Date().toISOString(),bundle,batch_id:[...batchIDs][0],report_sha256:digest(reportBytes),suite_count:suites.length,published,superseded,skipped};
 if(!dryRun) {
  // Finish all validation before changing public files. Write verified bytes,
  // not a source path that could have changed after the hash check.
  for(const candidate of candidates.values()) {
   const target=path.join(root,candidate.target),temporary=target+'.publishing-'+manifest.batch_id;
   await writeFile(temporary,candidate.data,{flag:'wx'});await rename(temporary,target);
  }
  await writeFile(previousPath,JSON.stringify(manifest,null,2)+'\n');
 }
 return manifest;
}

if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
 const args=process.argv.slice(2),index=args.indexOf('--bundle');
 assert.ok(index>=0&&args[index+1]&&args.every((arg,i)=>i===index||i===index+1||arg==='--dry-run'),'Usage: node tests/publish-verified-screenshots.mjs --bundle main-HASH.js [--dry-run]');
 const result=await publishVerifiedScreenshots({bundle:args[index+1],dryRun:args.includes('--dry-run')});
 console.log(`${args.includes('--dry-run')?'Verified':'Published'} ${result.published.length} normal PNGs from ${result.suite_count} successful suites (${result.bundle}); ${result.superseded.length} duplicate names resolved, ${result.skipped.length} unapproved captures excluded`);
}
