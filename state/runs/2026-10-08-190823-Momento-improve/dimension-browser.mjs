import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { pathToFileURL } from 'node:url';
const require = createRequire('/tmp/pptr/package.json');
const puppeteer = require('puppeteer-core');
const root = process.cwd();
const { describeDimensionError } = await import(pathToFileURL(path.join(root, 'web/src/pages/adminErrors.ts')));
const dist = path.join(root, 'web/dist');
const site = { id:'s1', site_id:'site-one', name:'테스트 사이트', service_name:'test', allowed_domains:[], session_timeout_minutes:30, timezone:'Asia/Seoul', engagement_threshold_seconds:10, active:true, workspace:'test', organization:'test', created_at:'2026-01-01T00:00:00Z' };
let scenario, requests, reads, rows, releaseResponse;
const initialRow = {id:'dimension-one', name:'membership', query_name:'custom.membership', property_key:'user.membership', scope:'user', data_type:'string', description:'기존 정의', active:true};
const server = http.createServer(async(req,res)=>{
 const url=new URL(req.url,'http://localhost');
 const json=(status,body)=>{res.writeHead(status,{'content-type':'application/json'});res.end(JSON.stringify(body));};
 if(url.pathname==='/api/v1/me')return json(200,{id:'u1',email:'admin@example.com',display_name:'관리자',department:'개발',organization_name:'test',role:'super_admin'});
 if(url.pathname==='/api/v1/sites')return json(200,[site]);
 if(url.pathname.endsWith('/environments'))return json(200,[{name:'prd',label:'운영',active:true}]);
 if(url.pathname==='/api/v1/dimensions'){
  if(req.method==='GET'){reads++;return json(200,rows);}
  if(req.method==='POST'){
   let body='';for await(const chunk of req)body+=chunk;
   requests.push(JSON.parse(body));
   await new Promise(resolve=>{releaseResponse=resolve;});
   if(scenario){return json(scenario[0],{error:{code:scenario[1],message:scenario[2]}});}
   rows=[{...initialRow,...JSON.parse(body)}];
   return json(200,{id:'dimension-one',query_name:'custom.membership'});
  }
 }
 if(url.pathname==='/api/v1/dimensions/dimension-one'&&req.method==='DELETE'){rows=[];return json(200,{ok:true});}
 if(url.pathname.startsWith('/api/'))return json(404,{error:{code:'HARNESS_UNEXPECTED',message:url.pathname}});
 const file=path.join(dist,url.pathname.startsWith('/assets/')?url.pathname:'index.html');
 const mime={'.js':'text/javascript','.css':'text/css','.woff2':'font/woff2','.html':'text/html'}[path.extname(file)]||'application/octet-stream';
 res.writeHead(200,{'content-type':mime});fs.createReadStream(file).pipe(res);
});
await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
const base=`http://127.0.0.1:${server.address().port}`;
const browser=await puppeteer.launch({executablePath:'/usr/bin/google-chrome',headless:true,args:['--no-sandbox','--disable-dev-shm-usage']});
const cases=[
 [400,'INVALID_DIMENSION','name and property_key must use letters, numbers, underscore, dot, or hyphen'],
 [500,'DIMENSION_SAVE_FAILED','ERROR: insert or update on table "dimensions" violates foreign key constraint (SQLSTATE 23503)'],
 [500,'DIMENSION_SAVE_FAILED','no rows in result set'],
 [500,'FUTURE_CODE','future server refusal'],
 [400,'INVALID_SCOPE','scope must be user, session, event, or item'],
 [400,'INVALID_DATA_TYPE','data_type must be string, number, boolean, or date'],
 [400,'INVALID_PAYLOAD','unexpected EOF'],
 [404,'UNKNOWN_SITE','site not found'],
 [502,'REQUEST_FAILED','HTTP 502'],
];
let passed=0;
async function openForm(testCase){
 scenario=testCase;requests=[];reads=0;rows=[initialRow];releaseResponse=null;
 const page=await browser.newPage();
 page.on('dialog',d=>d.accept());
 page.on('pageerror',err=>console.error('PAGE_ERROR',err.message));
 await page.goto(`${base}/admin?section=dimensions`,{waitUntil:'networkidle0'});
 await page.waitForFunction(()=>[...document.querySelectorAll('label')].some(l=>l.textContent==='Dimension 이름'));
 return page;
}
async function input(page,label){return page.evaluateHandle(t=>{const l=[...document.querySelectorAll('label')].find(l=>l.textContent===t);return document.getElementById(l.htmlFor);},label);}
async function button(page,text){return page.evaluateHandle(t=>[...document.querySelectorAll('button')].find(b=>b.textContent.trim()===t),text);}
async function disabled(button){return button.evaluate(b=>b.disabled);}
async function save(page,saveButton){
 await saveButton.click();
 const deadline=Date.now()+5000;while(!releaseResponse&&Date.now()<deadline)await new Promise(r=>setTimeout(r,20));
 assert.ok(releaseResponse,'실제 POST 도달');
 await page.waitForFunction(b=>b.disabled,{},saveButton);
 assert.equal(await disabled(saveButton),true,'pending 저장 버튼 닫힘');
 releaseResponse();
}
try{
 for(const c of cases){
  const page=await openForm(c);
  const saveButton=await button(page,'등록 또는 갱신');
  assert.equal(await disabled(saveButton),true,'빈 폼 저장 버튼 닫힘');
  await (await input(page,'Dimension 이름')).type('bad name');
  assert.equal(await disabled(saveButton),true,'Property key 비면 닫힘');
  await (await input(page,'Property key')).type('membership');
  assert.equal(await disabled(saveButton),false,'기존 입력 조건만 적용');
  await save(page,saveButton);
  await page.waitForSelector('.MuiCard-root .MuiAlert-root');
  const notice=await page.$eval('.MuiCard-root .MuiAlert-root',alert=>({message:alert.querySelector('.MuiTypography-body2')?.textContent??alert.querySelector('.MuiAlert-message').textContent,detail:alert.querySelector('.MuiTypography-caption')?.textContent}));
  const error=Object.assign(new Error(c[2]),{code:c[1],status:c[0]});
  assert.deepEqual(notice,describeDimensionError(error));
  if(c[1]==='DIMENSION_SAVE_FAILED')assert.doesNotMatch(notice.message,/SQLSTATE|violates|no rows|중복|이미 등록/);
  assert.equal(requests.length,1);
  assert.equal(requests[0].site_id,site.site_id);
  assert.equal(await (await input(page,'Dimension 이름')).evaluate(e=>e.value),'bad name','실패 후 입력 보존');
  assert.equal(await disabled(saveButton),false,'실패 후 재시도 가능');
  console.log('PASS',c[1],JSON.stringify(notice));passed++;
  await page.close();
 }
 const page=await openForm(null);
 await (await button(page,'편집')).click();
 assert.equal(await (await input(page,'Dimension 이름')).evaluate(e=>e.value),'membership');
 assert.equal(await (await input(page,'Property key')).evaluate(e=>e.value),'user.membership');
 const readsBefore=reads;
 await save(page,await button(page,'등록 또는 갱신'));
 await page.waitForFunction(()=>{const l=[...document.querySelectorAll('label')].find(l=>l.textContent==='Dimension 이름');return document.getElementById(l.htmlFor).value==='';});
 assert.equal(await (await input(page,'Property key')).evaluate(e=>e.value),'');
 assert.equal(await disabled(await button(page,'등록 또는 갱신')),true);
 assert.ok(reads>readsBefore,'성공 뒤 dimensions 쿼리 무효화');
 assert.equal(requests[0].scope,'user');
 assert.equal(requests[0].data_type,'string');
 await (await button(page,'편집')).evaluate(el=>el.nextElementSibling.click());
 await page.waitForSelector('[role="dialog"]');
 await page.evaluate(()=>[...document.querySelectorAll('[role="dialog"] button')].find(b=>b.textContent==='삭제').click());
 await page.waitForFunction(()=>![...document.querySelectorAll('button')].some(b=>b.textContent.trim()==='편집'));
 assert.deepEqual(rows,[]);
 console.log('PASS 편집·성공 폼 초기화·쿼리 무효화·삭제·저장 버튼 조건');passed++;
 await page.close();
 console.log(`RESULT ${passed}/${cases.length+1} scenarios passed`);
}finally{await browser.close();await new Promise(resolve=>server.close(resolve));}
