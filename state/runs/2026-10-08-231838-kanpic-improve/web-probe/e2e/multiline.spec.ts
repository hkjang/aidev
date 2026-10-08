import { expect, test, type Page } from '@playwright/test'

/**
 * Emulates what a Korean IME does in Chrome: a composition that updates as
 * jamo arrive and is confirmed with Enter. The confirming Enter arrives while
 * isComposing is true, which is exactly the case that used to swallow input.
 */
async function composeHangul(page:Page,label:string,syllables:string[]){
  await page.getByLabel(label).evaluate((element,values)=>{
    const field=element as HTMLTextAreaElement
    // React tracks the value it set, so the native setter is what makes a
    // programmatic change look like typing to it.
    const setValue=Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype,'value')!.set!
    field.focus()
    for(const syllable of values){
      field.dispatchEvent(new CompositionEvent('compositionstart',{bubbles:true}))
      setValue.call(field,field.value+syllable)
      field.dispatchEvent(new InputEvent('input',{bubbles:true,data:syllable,isComposing:true}))
      field.dispatchEvent(new CompositionEvent('compositionend',{bubbles:true,data:syllable}))
      field.dispatchEvent(new InputEvent('input',{bubbles:true,data:syllable}))
    }
  },syllables)
}

// A second line inside one cell is written with Alt+Enter, the way it is in
// every other spreadsheet.
test('Alt+Enter writes a line break and Enter still commits', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`줄바꿈 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')

  const editor=page.getByLabel('A1 셀 입력')
  await editor.fill('첫 줄')
  await editor.press('Alt+Enter')
  await editor.type('둘째 줄')
  await expect(editor).toHaveValue('첫 줄\n둘째 줄')
  // Enter without Alt still finishes the cell and moves down.
  await editor.press('Enter')
  await expect.poll(async()=>{
    const items=(await (await request.get(`/api/v1/sheets/${sheet}/ranges/A1:A1`)).json()).items
    return items[0]?.value
  }).toBe('첫 줄\n둘째 줄')
  await expect(page.getByLabel('A2 셀 입력')).toBeVisible()
})

// The editor is a text area now, so the composition path is checked again:
// the jamo must survive and the confirming Enter must not leave a stray line.
test('Hangul input still commits cleanly in the multi-line editor', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`한글 입력 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')

  // F2 opens the cell for editing, which is where an IME composition lands.
  await page.getByLabel('A1 셀 입력').press('F2')
  await composeHangul(page,'A1 셀 입력',['안','녕'])
  await expect(page.getByLabel('A1 셀 입력')).toHaveValue(/안녕$/)
  await page.getByLabel('A1 셀 입력').press('Enter')
  await expect.poll(async()=>{
    const items=(await (await request.get(`/api/v1/sheets/${sheet}/ranges/A1:A1`)).json()).items
    return items[0]?.value
  }).toMatch(/안녕$/)
  // No stray newline came from confirming the composition.
  const stored=(await (await request.get(`/api/v1/sheets/${sheet}/ranges/A1:A1`)).json()).items[0].value
  expect(String(stored)).not.toContain('\n')
})

// Editing a multi-line cell from the formula bar must not quietly flatten it.
test('the formula bar keeps the line breaks of a multi-line cell', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`수식 입력창 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`seed-${Date.now()}`,cells:[
    {row:1,column:1,value:'첫 줄\n둘째 줄'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await page.waitForSelector('.grid-canvas')

  const bar=page.getByLabel('수식 입력창')
  await expect(bar).toHaveValue('첫 줄\n둘째 줄')
  await bar.click()
  // End stops at the end of the line it is on, so the caret goes to the very end.
  await bar.press('Control+End')
  await bar.type('!')
  await bar.press('Enter')
  await expect.poll(async()=>{
    const items=(await (await request.get(`/api/v1/sheets/${sheet}/ranges/A1:A1`)).json()).items
    return items[0]?.value
  }).toBe('첫 줄\n둘째 줄!')
})

// 줄바꿈 뒤에 캐럿을 다음 프레임에 다시 맞추면, 그 프레임이 늦게 도착했을 때
// 그 사이에 친 글자가 앞으로 끌려간다. "둘째 줄" 이 "줄둘째 " 가 되는 식이다.
// CI 처럼 부하가 걸린 기계에서만 보이던 것이라, 프레임을 일부러 늦춰 재현한다.
test('a late frame does not drag typing back to where the line break was', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`캐럿 ${stamp}`}}).then(r=>r.json())
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  // 부하가 걸린 기계에서 프레임이 늦는 것과 같은 상황을 만든다.
  await page.evaluate(()=>{
    window.requestAnimationFrame=(callback:FrameRequestCallback)=>
      window.setTimeout(()=>callback(performance.now()),150) as unknown as number
  })
  await page.locator('.grid-viewport').press('Enter')
  const editor=page.getByLabel('A1 셀 입력')
  await editor.type('첫 줄')
  await editor.press('Alt+Enter')
  await editor.type('둘째')
  // 늦은 프레임이 여기서 도착한다. 캐럿을 줄바꿈 직후로 되돌리면 안 된다.
  await page.waitForTimeout(300)
  await editor.type(' 줄')
  await expect(editor).toHaveValue('첫 줄\n둘째 줄')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 수식 입력줄에도 같은 코드가 있었다. 격자만 고치고 여기를 두면 같은 버그가
// 같은 워크북 안에 그대로 남는다.
test('a late frame does not scramble typing in the formula bar either', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`수식줄 캐럿 ${stamp}`}}).then(r=>r.json())
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.evaluate(()=>{
    window.requestAnimationFrame=(callback:FrameRequestCallback)=>
      window.setTimeout(()=>callback(performance.now()),150) as unknown as number
  })
  const bar=page.locator('.formula-input')
  await bar.click()
  await bar.type('첫 줄')
  await bar.press('Alt+Enter')
  await bar.type('둘째')
  await page.waitForTimeout(300)
  await bar.type(' 줄')
  await expect(bar).toHaveValue('첫 줄\n둘째 줄')
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
