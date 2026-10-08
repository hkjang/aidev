import { expect, test } from '@playwright/test'

// 도구 모음을 다 지나야 시트에 닿는다. 마우스 없이 쓰는 사람에게 마흔 번
// 넘는 탭을 요구할 수는 없다.
test('the first tab stop jumps straight to the sheet', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`키보드 ${stamp}`}}).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${workbook.sheets[0].id}/cells:batch`,{data:{idempotency_key:`kb-${stamp}`,
    cells:[{row:1,column:1,value:'항목'},{row:2,column:1,value:10},{row:3,column:1,value:20}]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()

  await page.evaluate(()=>(document.activeElement as HTMLElement|null)?.blur?.())
  await page.keyboard.press('Tab')
  const skip=page.getByRole('link',{name:'시트로 건너뛰기'})
  await expect(skip).toBeFocused()
  // 초점을 받았을 때만 보인다. 평소에는 화면 위로 숨어 있다.
  await expect(skip).toBeInViewport()

  await page.keyboard.press('Enter')
  await page.keyboard.press('ArrowDown')
  await page.keyboard.press('ArrowRight')
  await expect(page.getByLabel('이름 상자')).toHaveValue('B2')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 아이콘만 있는 단추는 이름을 따로 달아 주지 않으면 화면 낭독기에 그냥
// "단추" 라고만 읽힌다. 확대와 축소가 그랬다.
test('every control in the toolbar can be named out loud', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`이름 ${stamp}`}}).then(response=>response.json())
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await expect(page.getByRole('button',{name:'확대',exact:true})).toBeVisible()
  await expect(page.getByRole('button',{name:'축소',exact:true})).toBeVisible()
  const unnamed=await page.evaluate(()=>[...document.querySelectorAll('.toolbar button,.toolbar select,.toolbar input')]
    .filter(element=>{
      const control=element as HTMLElement
      if(control.offsetParent===null)return false
      const name=(control.getAttribute('aria-label')||control.getAttribute('title')||control.textContent||'').trim()
      return name===''&&!((control as HTMLInputElement).labels?.length)
    })
    .map(element=>element.className))
  expect(unnamed).toEqual([])
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 대화상자는 초점을 안에 붙들고, Esc로 닫히고, 열었던 자리로 초점을
// 돌려줘야 한다. 돌려주지 않으면 키보드 사용자는 처음부터 다시 탭한다.
test('a dialog holds focus and hands it back where it came from', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`대화상자 ${stamp}`}}).then(response=>response.json())
  await request.patch(`/api/v1/sheets/${workbook.sheets[0].id}/cells:batch`,{data:{idempotency_key:`dl-${stamp}`,
    cells:[{row:1,column:1,value:'항목'},{row:2,column:1,value:10}]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()

  const trigger=page.getByRole('button',{name:'범위 정렬'})
  await trigger.focus()
  await page.keyboard.press('Enter')
  await expect(page.getByRole('dialog')).toBeVisible()
  for(let press=0;press<8;press+=1){
    await page.keyboard.press('Tab')
    expect(await page.evaluate(()=>Boolean(document.activeElement?.closest('[role="dialog"]')))).toBe(true)
  }
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(trigger).toBeFocused()
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
