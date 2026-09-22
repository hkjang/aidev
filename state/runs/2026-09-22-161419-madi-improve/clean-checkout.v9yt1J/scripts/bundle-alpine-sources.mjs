#!/usr/bin/env node
// Connected BUILD stage only. Runtime does not download anything.
// Official signed APK metadata identifies the exact aports commit. abuild then
// verifies each upstream archive against that commit's SHA-512 checksums.
import {createHash} from 'node:crypto';
import {createReadStream} from 'node:fs';
import {readFile,writeFile,appendFile,mkdir,readdir,lstat,copyFile,readlink,symlink,realpath} from 'node:fs/promises';
import {spawn} from 'node:child_process';
import {lookup} from 'node:dns/promises';
import path from 'node:path';
import {bundlePDFFontSources} from './bundle-pdf-font-sources.mjs';
const [database,output]=process.argv.slice(2);
if(!database||!output||!path.isAbsolute(output)||output==='/'||!output.endsWith('/madi-sources'))throw new Error('Usage: bundle-alpine-sources.mjs /runtime-apk-installed /build/madi-sources');
await mkdir(output,{recursive:true,mode:0o755});
const records=(await readFile(database,'utf8')).split(/\n\n+/).filter(Boolean).map(record=>Object.fromEntries(record.split('\n').filter(line=>line[1]===':').map(line=>[line[0],line.slice(2)])));
const packages=records.map(r=>({name:r.P,version:r.V,license:r.L,origin:r.o,commit:r.c,architecture:r.A})).filter(r=>r.name).sort((a,b)=>a.name.localeCompare(b.name));
const selected=new Map();
for(const pkg of packages){if(!/^[a-zA-Z0-9+_.-]+$/.test(pkg.origin)||!/^[a-f0-9]{40}$/.test(pkg.commit))throw new Error(`Missing exact source provenance for ${pkg.name}`);if(pkg.origin==='tesseract-ocr')continue;selected.set(`${pkg.origin}/${pkg.commit}`,pkg);}
const headers={'User-Agent':'madi-corresponding-source-builder','Accept':'application/vnd.github+json'};
async function get(url,limit=32<<20,allow404=false){const response=await fetch(url,{headers,signal:AbortSignal.timeout(120000)});if(response.status===404&&allow404)return null;if(!response.ok)throw new Error(`Official source fetch ${response.status}: ${url}`);const chunks=[];let bytes=0;for await(const chunk of response.body){bytes+=chunk.length;if(bytes>limit)throw new Error('Official source metadata size limit');chunks.push(chunk);}return Buffer.concat(chunks);}
async function hashFile(file,algorithm='sha256'){const hash=createHash(algorithm);for await(const chunk of createReadStream(file))hash.update(chunk);return hash.digest('hex');}
// Smart Git avoids unauthenticated REST API quotas. This is a fixed official
// read-only remote, not a user repository; no credentials, hooks or submodules.
const gitDir='/madi-source-cache/aports', fetched=new Set();let gitReady=false;
// Resolve through the same libc resolver as Node's verified HTTPS downloads.
// Some rootless BuildKit networks time out only in curl's c-ares resolver.
// No address is hard-coded; Git still verifies github.com's TLS certificate.
const gitAddress=(await lookup('github.com',{family:4})).address;
async function git(args){return await new Promise((resolve,reject)=>{const child=spawn('git',['-c','core.hooksPath=/dev/null','-c','credential.helper=','-c',`http.curloptResolve=github.com:443:${gitAddress}`,...args],{cwd:gitDir,env:{...process.env,GIT_CONFIG_NOSYSTEM:'1',GIT_CONFIG_GLOBAL:'/dev/null',GIT_TERMINAL_PROMPT:'0'},stdio:['ignore','pipe','inherit']});const chunks=[];let bytes=0;const timer=setTimeout(()=>child.kill('SIGTERM'),180000);child.stdout.on('data',chunk=>{bytes+=chunk.length;if(bytes>4<<20)child.kill('SIGTERM');else chunks.push(chunk);});child.on('error',e=>{clearTimeout(timer);reject(e)});child.on('exit',code=>{clearTimeout(timer);code===0&&bytes<=4<<20?resolve(Buffer.concat(chunks).toString('utf8')):reject(new Error('Official aports Git checkout failed'));});});}
async function recipeFiles(repoPath,commit,target){
 if(!gitReady){await mkdir(gitDir,{recursive:true});await git(['init','--quiet']);await git(['config','remote.origin.url','https://github.com/alpinelinux/aports.git']);await git(['config','remote.origin.promisor','true']);await git(['config','remote.origin.partialclonefilter','blob:none']);gitReady=true;}
 if(!fetched.has(commit)){await git(['fetch','--quiet','--no-tags','--depth=1','--filter=blob:none','origin',commit]);fetched.add(commit);}
 const files=(await git(['ls-tree','-r','-z','--name-only',commit,'--',repoPath])).split('\0').filter(Boolean);if(!files.length)return false;if(files.length>1000)throw new Error('Recipe file count limit');
 for(const name of files){const relative=name.slice(repoPath.length+1);if(!name.startsWith(repoPath+'/')||relative.split('/').some(part=>!/^[-a-zA-Z0-9+_.@]+$/.test(part)||part==='..')||relative.split('/').length>5)throw new Error('Unsafe official recipe path');}
 await git(['restore','--source='+commit,'--worktree','--',repoPath]);
 for(const name of files){const source=path.join(gitDir,name),stat=await lstat(source),destination=path.join(target,name.slice(repoPath.length+1));await mkdir(path.dirname(destination),{recursive:true});if(stat.isSymbolicLink()){const link=await readlink(source),resolved=path.posix.normalize(path.posix.join(path.posix.dirname(name),link));if(path.isAbsolute(link)||!resolved.startsWith(repoPath+'/')||!files.includes(resolved))throw new Error('Official recipe link escapes its tracked source directory');await symlink(link,destination);}else{if(!stat.isFile()||stat.size>32<<20)throw new Error(`Recipe is not a bounded regular file: ${name}`);await copyFile(source,destination);}}
 return true;
}
async function run(program,args,cwd){await new Promise((resolve,reject)=>{const child=spawn(program,args,{cwd,stdio:'inherit',env:{...process.env,LC_ALL:'C.UTF-8',srcdir:path.join('/tmp/madi-source-links',path.basename(path.dirname(cwd)))}});const timer=setTimeout(()=>{child.kill('SIGTERM');reject(new Error('Corresponding source verification exceeded 20 minutes'));},20*60*1000);child.on('error',e=>{clearTimeout(timer);reject(e);});child.on('exit',code=>{clearTimeout(timer);code===0?resolve():reject(new Error(`Source verification failed (${program}, ${code})`));});});}
async function inventory(dir,prefix=''){const files=[];for(const name of (await readdir(dir)).sort()){const file=path.join(dir,name),st=await lstat(file);if(st.isSymbolicLink()){const resolved=await realpath(file);if(!resolved.startsWith(output+'/')||!(await lstat(resolved)).isFile())throw new Error('Source bundle link escapes its output directory');files.push({path:prefix+name,symlink:await readlink(file),sha256:await hashFile(file)});}else if(st.isDirectory())files.push(...await inventory(file,`${prefix}${name}/`));else if(st.isFile())files.push({path:prefix+name,bytes:st.size,sha256:await hashFile(file)});}return files;}
const origins=[];
for(const pkg of selected.values()){
 const target=path.join(output,'aports',pkg.origin,pkg.commit),recipe=path.join(target,'recipe'),distfiles=path.join(target,'distfiles');
 let repository='';for(const candidate of ['main','community','testing']){if(await recipeFiles(`${candidate}/${pkg.origin}`,pkg.commit,recipe)){repository=candidate;break;}}
 if(!repository)throw new Error(`Corresponding APKBUILD unavailable: ${pkg.origin}@${pkg.commit}`);
 await mkdir(distfiles,{recursive:true});const cached=path.join('/madi-source-cache/distfiles',pkg.origin,pkg.commit);await mkdir(cached,{recursive:true});
 // fetch/verify do not build or execute the packaged program. The authenticated
 // official APKBUILD is the build recipe, not a user-supplied document.
 await run('abuild',['-F','-s',cached,'fetch','verify'],recipe);
 for(const file of await inventory(cached)){const destination=path.join(distfiles,file.path);await mkdir(path.dirname(destination),{recursive:true});await copyFile(path.join(cached,file.path),destination);}
 origins.push({origin:pkg.origin,commit:pkg.commit,repository,license:pkg.license,recipe:`aports/${pkg.origin}/${pkg.commit}/recipe/APKBUILD`,recipeFiles:await inventory(recipe),files:await inventory(distfiles)});
}
// Apache-2.0 Tesseract/model notices accompany the two installed language files;
// the source collector above intentionally does not fetch all languages.
const tesseractDir=path.join(output,'tesseract');await mkdir(tesseractDir,{recursive:true});
const tessPackage=packages.find(p=>p.name==='tesseract-ocr');
if(!tessPackage||tessPackage.version!=='5.5.2-r0'||!/^[a-f0-9]{40}$/.test(tessPackage.commit))throw new Error('Unexpected OCR APK version');
if(!await recipeFiles('community/tesseract-ocr',tessPackage.commit,path.join(tesseractDir,'recipe')))throw new Error('OCR APKBUILD unavailable');
const engine=await get('https://github.com/tesseract-ocr/tesseract/archive/5.5.2.tar.gz',100<<20);
const expected='e9103c68ba186821aedd38de4d9949cd6732da93a2d0764de18aaaac70eb9c305384a6eb1fe656a8a269bee833178a583a91dd72027ae26d27c8329ed722f4a9';
if(createHash('sha512').update(engine).digest('hex')!==expected)throw new Error('Tesseract upstream SHA-512 mismatch');
await writeFile(path.join(tesseractDir,'tesseract-5.5.2.tar.gz'),engine);
await writeFile(path.join(tesseractDir,'LICENSE'),await get('https://raw.githubusercontent.com/tesseract-ocr/tesseract/5.5.2/LICENSE'));
await writeFile(path.join(tesseractDir,'TESSDATA-LICENSE'),await get('https://raw.githubusercontent.com/tesseract-ocr/tessdata/4.1.0/LICENSE'));
await copyFile(database,path.join(output,'runtime-apk-installed'));
const models=[];for(const lang of ['eng','kor'])models.push({language:lang,path:`/usr/share/tessdata/${lang}.traineddata`,sha256:await hashFile(`/runtime-tessdata/${lang}.traineddata`)});
const pdfFonts=await bundlePDFFontSources('/runtime-pdfjs',output,get);
await writeFile(path.join(output,'manifest.json'),JSON.stringify({format:'madi-runtime-sources-v1',packages,origins,tesseract:{version:'5.5.2-r0',commit:tessPackage.commit,recipe:'tesseract/recipe/APKBUILD',modelRelease:'4.1.0',models,upstreamSHA512:expected},notice:'Exact installed package metadata and ALL official APKBUILD/patches and SHA-512-verified upstream sources, including permissive-license copyright/notice texts inside original archives. Tesseract includes its engine and the two installed language files only. No source downloads occur at runtime.'},null,2)+'\n');
await writeFile(path.join(output,'README.txt'),'madi 오프라인 런타임 대응 소스\n\nmanifest.json: 실제 APK 버전, SPDX 라이선스, aports origin/commit.\naports/: GPL/LGPL/MPL 및 MIT/BSD 등 모든 런타임 origin의 원본 APKBUILD·패치와 검증된 upstream 소스. 저작권·라이선스·NOTICE 원문도 각 원본 압축 파일 안에 보존됩니다. 해당 Alpine 릴리스/아키텍처에서 APKBUILD의 의존성을 설치하고 abuild로 재빌드할 수 있습니다. 빌드 레시피를 함께 제공하며 파일을 변경하지 않았습니다. 별도 upstream이 없는 메타패키지·공개키는 recipe 자체가 원본입니다.\ntesseract/: Apache-2.0 OCR 엔진 원본과 영어·한국어 학습데이터 라이선스. 실제 모델 파일은 /usr/share/tessdata에 포함됩니다.\n런타임에는 인터넷·패키지 설치·다운로드가 필요하지 않습니다. 각 구성 요소의 원래 라이선스가 적용됩니다.\n');
await writeFile(path.join(output,'PDF-FONT-SOURCES.json'),JSON.stringify(pdfFonts,null,2)+'\n');
await appendFile(path.join(output,'README.txt'),'\npdfjs-liberation/: APK에 속하지 않는 PDF.js Liberation Sans 1.07.4의 SFD 원본·FontForge 빌드 스크립트·GPLv2/예외, 실제 포함 TTF·해시·재빌드 안내. PDF-FONT-SOURCES.json에 정확한 버전과 고정 소스 URL/SHA-256이 기록됩니다. 자세한 빌드/폰트 매핑 제약은 REBUILD.txt를 참조하세요.\n');
await writeFile(path.join(output,'SHA256SUMS'),(await inventory(output)).map(f=>`${f.sha256}  ${f.path}`).join('\n')+'\n');
console.log(`Verified source bundle: ${origins.length} complete origins + Tesseract, ${packages.length} runtime APKs`);
