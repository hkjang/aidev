import { expect, test } from '@playwright/test'

// 인쇄물은 셀마다 다른 색과 굵기를 style 속성으로 싣는다. 그런데 앱의 보안
// 정책은 인라인 스타일을 막으므로, 빈 프레임에 그냥 쓰면 그 전부가 조용히
// 버려지고 글자만 남은 종이가 나온다.
//
// 이 시험이 없던 동안 인쇄물에는 색도 굵기도 테두리도 없었다. 인쇄 시험들이
// 글자만 확인했기 때문이다. 그리는 것이 일인 기능은 그려졌는지를 봐야 한다.
test('a printed sheet keeps the colours and weights it was given', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`인쇄 서식 ${stamp}`}}).then(r=>r.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`style-${stamp}`,cells:[
    {row:1,column:1,value:'강조',style:{background:'#ff0000',color:'#ffffff',bold:true}},
    {row:2,column:1,value:'보통'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  // 인쇄 대화 상자를 띄우지 않고 문서만 만든다.
  await page.evaluate(()=>{(window as unknown as {print:()=>void}).print=()=>{}})
  await page.getByRole('menubar',{name:'워크북 메뉴'}).getByRole('menuitem',{name:'파일',exact:true}).click()
  await page.getByRole('menuitem',{name:/^인쇄(?!\s*영역)/}).click()

  const painted=await (async()=>{
    for(let attempt=0;attempt<50;attempt+=1){
      const found=await page.evaluate(()=>{
        for(const frame of [...document.querySelectorAll('iframe')]){
          const doc=frame.contentDocument
          const view=doc?.defaultView
          if(!doc||!view)continue
          const cell=[...doc.querySelectorAll('td')].find(item=>item.textContent?.includes('강조'))
          const table=doc.querySelector('table')
          if(!cell||!table)continue
          return {
            background:view.getComputedStyle(cell).backgroundColor,
            weight:view.getComputedStyle(cell).fontWeight,
            // 문서가 실은 <style> 블록도 적용되어야 표가 표처럼 나온다.
            collapse:view.getComputedStyle(table).borderCollapse,
          }
        }
        return undefined
      })
      if(found)return found
      await page.waitForTimeout(100)
    }
    throw new Error('the print document never appeared')
  })()

  expect(painted.background).toBe('rgb(255, 0, 0)')
  expect(painted.weight).toBe('700')
  expect(painted.collapse).toBe('collapse')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 인쇄 문서에만 스타일을 허용한 것이지 스크립트를 허용한 것이 아니다.
test('the print page allows styling and nothing else', async ({ request }) => {
  const response=await request.get('/print-frame')
  expect(response.status()).toBe(200)
  const policy=response.headers()['content-security-policy']??''
  expect(policy).toContain("default-src 'none'")
  expect(policy).toContain("style-src 'unsafe-inline'")
  // 스크립트도, 바깥으로 나가는 연결도 허용하지 않는다.
  expect(policy).not.toContain('script-src')
  expect(policy).not.toContain('connect-src')
})

// 조건부 서식은 값에 따라 그때그때 정해지므로 셀에 저장돼 있지 않다. 인쇄가
// 따로 묻지 않으면, 읽으라고 칠해 놓은 표가 종이에서는 아무 표시 없는 숫자
// 뭉치가 된다.
test('a printed sheet carries the conditional formatting the reader sees', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`조건부 인쇄 ${stamp}`}}).then(r=>r.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`cf-${stamp}`,cells:[
    {row:1,column:1,value:10},{row:2,column:1,value:500},
  ]}})
  await request.post(`/api/v1/sheets/${sheet}/conditional-formats`,{data:{
    idempotency_key:`rule-${stamp}`,name:'큰 값',range:'A1:A2',rule_type:'value',operator:'greater_than',
    value:100,style:{background:'#00ff00'},priority:1}})
  await request.post(`/api/v1/sheets/${sheet}/conditional-formats`,{data:{
    idempotency_key:`icon-${stamp}`,name:'아이콘',range:'A1:A2',rule_type:'icon_set',icon_style:'3Arrows',priority:2}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.evaluate(()=>{(window as unknown as {print:()=>void}).print=()=>{}})
  await page.getByRole('menubar',{name:'워크북 메뉴'}).getByRole('menuitem',{name:'파일',exact:true}).click()
  await page.getByRole('menuitem',{name:/^인쇄(?!\s*영역)/}).click()

  const printed=await (async()=>{
    for(let attempt=0;attempt<50;attempt+=1){
      const found=await page.evaluate(()=>{
        for(const frame of [...document.querySelectorAll('iframe')]){
          const doc=frame.contentDocument, view=doc?.defaultView
          if(!doc||!view)continue
          const cells=[...doc.querySelectorAll('td')]
          if(cells.length<2)continue
          return cells.map(cell=>({
            text:cell.textContent??'',
            background:view.getComputedStyle(cell).backgroundColor,
            mark:cell.querySelector('.mark')?.textContent??'',
          }))
        }
        return undefined
      })
      if(found)return found
      await page.waitForTimeout(100)
    }
    throw new Error('the print document never appeared')
  })()

  // 규칙에 걸린 칸만 칠해진다.
  expect(printed[1].background).toBe('rgb(0, 255, 0)')
  expect(printed[0].background).toBe('rgba(0, 0, 0, 0)')
  // 아이콘은 값을 밀어낼 뿐 대신하지 않는다.
  expect(printed[0].mark).toBe('▼')
  expect(printed[1].mark).toBe('▲')
  expect(printed[1].text).toContain('500')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
