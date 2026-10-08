import { expect, test, type APIRequestContext } from '@playwright/test'
import { createServer, type Socket } from 'node:net'

// A throwaway SMTP relay that accepts everything and remembers the messages,
// which is how these tests prove kanpic speaks to an unauthenticated internal
// relay rather than asserting against a mock of our own code.
type Received={to:string;body:string}
const received:Received[]=[]
let relayPort=0

const relay=createServer((socket:Socket)=>{
  let buffer='',recipient='',reading=false,body=''
  const write=(line:string)=>socket.write(line+'\r\n')
  write('220 relay.test ESMTP')
  socket.on('data',chunk=>{
    buffer+=chunk.toString('utf8')
    let index=buffer.indexOf('\r\n')
    while(index>=0){
      const line=buffer.slice(0,index)
      buffer=buffer.slice(index+2)
      if(reading){
        if(line==='.'){reading=false;received.push({to:recipient,body});body='';write('250 2.0.0 Ok')}
        else body+=line+'\n'
      }else{
        const upper=line.toUpperCase()
        if(upper.startsWith('EHLO'))write('250-relay.test\r\n250 SIZE 10485760')
        else if(upper.startsWith('RCPT TO')){recipient=line.replace(/.*<|>.*/g,'');write('250 2.1.5 Ok')}
        else if(upper==='DATA'){reading=true;write('354 go ahead')}
        else if(upper==='QUIT'){write('221 bye');socket.end()}
        else write('250 2.0.0 Ok')
      }
      index=buffer.indexOf('\r\n')
    }
  })
  socket.on('error',()=>undefined)
})

test.beforeAll(async()=>{
  await new Promise<void>((resolve,reject)=>{
    relay.once('error',reject)
    relay.listen(0,'0.0.0.0',()=>{
      const address=relay.address()
      if(typeof address==='object'&&address){relayPort=address.port;resolve()}else reject(new Error('relay did not bind'))
    })
  })
})
test.afterAll(async()=>{await new Promise<void>(resolve=>relay.close(()=>resolve()))})

const put=(request:APIRequestContext,key:string,value:unknown,value_type:'string'|'number'|'boolean')=>
  request.put(`/api/v1/admin/settings/${key}`,{data:{key,value,value_type}})

async function enableMail(request:APIRequestContext){
  const host=process.env.KANPIC_E2E_GATEWAY_HOST||'127.0.0.1'
  await put(request,'mail.smtp_host',host,'string')
  await put(request,'mail.smtp_port',relayPort,'number')
  await put(request,'mail.from_address','kanpic@corp.test','string')
  await put(request,'mail.base_url','https://sheet.corp.test','string')
  await put(request,'mail.enabled',true,'boolean')
}

test.afterEach(async ({ request }) => { await put(request,'mail.enabled',false,'boolean') })

// Long Korean subjects are split into several MIME encoded words and folded
// across lines, so the header is unfolded before every word is decoded.
const decodeSubject=(raw:string)=>{
  // Whitespace between adjacent encoded words is not part of the text (RFC 2047).
  const unfolded=raw.replace(/\n[ \t]+/g,'').replace(/\?=[ \t]+=\?/g,'?==?')
  const line=unfolded.split('\n').find(item=>item.startsWith('Subject:'))??''
  return line.slice('Subject:'.length).replace(/=\?utf-8\?q\?([^?]*)\?=/gi,(_match,encoded:string)=>
    Buffer.from(encoded.replace(/_/g,' ').replace(/=([0-9A-F]{2})/gi,(_all,hex:string)=>String.fromCharCode(parseInt(hex,16))),'binary').toString('utf8')).trim()
}

test('the console connects to an unauthenticated relay and sends a test mail', async ({ page, request }) => {
  await enableMail(request)
  const before=received.length

  await page.goto('/admin?tab=mail')
  await expect(page.getByRole('heading',{name:'알림 메일'})).toBeVisible()
  await expect(page.getByLabel('SMTP 서버')).toHaveValue(/\d+\.\d+\.\d+\.\d+|localhost/)

  // The connection test greets the relay without leaving any credentials.
  await page.getByRole('button',{name:'연결 확인'}).click()
  await expect(page.locator('.result-banner')).toContainText('연결 성공')

  await page.getByLabel('테스트 수신 주소').fill('admin@corp.test')
  await page.getByRole('button',{name:'테스트 메일 보내기'}).click()
  await expect(page.locator('.result-banner')).toContainText('테스트 메일을 보냈습니다')
  await expect.poll(()=>received.length,{timeout:10_000}).toBeGreaterThan(before)
  expect(decodeSubject(received[received.length-1].body)).toContain('SMTP 발송 테스트')

  // The delivery is recorded for the administrator.
  await expect(page.locator('.mail-row',{hasText:'admin@corp.test'}).first()).toContainText('발송됨')
})

test('sharing a workbook and mentioning somebody sends mail to the right people', async ({ request }) => {
  await enableMail(request)
  for(const [id,name] of [['mail.owner@corp.test','메일 소유자'],['mail.reader@corp.test','메일 독자'],['mail.mentioned@corp.test','언급 대상']]){
    await request.post('/api/v1/admin/users',{data:{user_id:id,display_name:name,email:id}})
  }
  const workbook=await request.post('/api/v1/workbooks',{
    headers:{'X-Kanpic-Actor':'mail.owner@corp.test'},data:{title:`메일 알림 ${Date.now()}`},
  }).then(response=>response.json())
  const before=received.length

  await request.put(`/api/v1/workbooks/${workbook.id}/shares`,{
    headers:{'X-Kanpic-Actor':'mail.owner@corp.test'},
    data:{principal_type:'user',principal_id:'mail.reader@corp.test',role:'editor'},
  })
  await request.post(`/api/v1/workbooks/${workbook.id}/comments`,{
    headers:{'X-Kanpic-Actor':'mail.reader@corp.test'},
    data:{sheet_id:workbook.sheets[0].id,range:'B2',content:'@mail.mentioned@corp.test 확인 부탁드립니다',idempotency_key:`mail-${Date.now()}`},
  })

  // Each expected mail is awaited on its own, because a late delivery from an
  // earlier test would otherwise satisfy a plain count.
  const arrived=(address:string,needle:string)=>received.slice(before).filter(item=>item.to===address&&decodeSubject(item.body).includes(needle))
  const waitFor=(address:string,needle:string)=>expect.poll(()=>arrived(address,needle).length,{timeout:15_000}).toBeGreaterThan(0)

  // The person a workbook was shared with hears about it, with a display name.
  await waitFor('mail.reader@corp.test','공유되었습니다')
  expect(arrived('mail.reader@corp.test','공유되었습니다')[0].body).toContain('메일 소유자')
  // The owner hears about the comment.
  await waitFor('mail.owner@corp.test','새 댓글')
  // The mentioned person gets a mention mail rather than a copy of the thread.
  await waitFor('mail.mentioned@corp.test','언급했습니다')
  const mentioned=received.slice(before).filter(item=>item.to==='mail.mentioned@corp.test')
  expect(mentioned).toHaveLength(1)
  // Nobody is told about their own action.
  expect(arrived('mail.reader@corp.test','새 댓글')).toHaveLength(0)
  // The body links back to the workbook using the configured address.
  expect(mentioned[0].body).toContain(`https://sheet.corp.test/workbooks/${workbook.id}`)
})

test('turning an event off stops that mail without touching the others', async ({ request }) => {
  await enableMail(request)
  await put(request,'mail.notify_share',false,'boolean')
  const workbook=await request.post('/api/v1/workbooks',{
    headers:{'X-Kanpic-Actor':'mail.owner@corp.test'},data:{title:`알림 끄기 ${Date.now()}`},
  }).then(response=>response.json())
  const before=received.length
  await request.put(`/api/v1/workbooks/${workbook.id}/shares`,{
    headers:{'X-Kanpic-Actor':'mail.owner@corp.test'},
    data:{principal_type:'user',principal_id:'mail.reader@corp.test',role:'viewer'},
  })
  await new Promise(resolve=>setTimeout(resolve,1500))
  expect(received.slice(before).filter(item=>item.to==='mail.reader@corp.test')).toHaveLength(0)
  await put(request,'mail.notify_share',true,'boolean')
})
