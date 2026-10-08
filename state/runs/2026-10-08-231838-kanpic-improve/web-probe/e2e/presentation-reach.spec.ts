import { expect, test } from '@playwright/test'

// 만든 프레젠테이션 목록은 오른쪽 패널에 있는데, 도구 모음에 아이콘이
// 없어 데이터 메뉴를 두 번 들어가야만 열 수 있었다. 다른 패널(차트·피벗·
// 자동화)은 모두 아이콘이 있으므로, 이것만 없으면 없는 기능처럼 보인다.
//
// 프레젠테이션 설정이 꺼져 있으면 아이콘도 나오면 안 된다. 눌러도 아무
// 일도 일어나지 않는 단추는 없는 것만 못하다.
test('the presentation panel is reachable from the toolbar when it is turned on', async ({ page, request }) => {
  const config=await request.get('/api/v1/presentation/config').then(response=>response.json())
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`프레젠테이션 ${stamp}`}}).then(response=>response.json())
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()

  // 패널을 열면 "프레젠테이션 패널 닫기" 단추가 생긴다. 부분 일치로 고르면
  // 둘이 걸리므로 이름을 정확히 맞춘다.
  const icon=page.getByRole('button',{name:'프레젠테이션 패널',exact:true})
  if(!config.enabled){
    // 꺼져 있으면 아이콘도 메뉴도 없다.
    await expect(icon).toHaveCount(0)
    await request.delete(`/api/v1/workbooks/${workbook.id}`)
    return
  }

  await expect(icon).toBeVisible()
  await icon.click()
  await expect(icon).toHaveClass(/active/)
  // 한 번 더 누르면 닫힌다.
  await icon.click()
  await expect(icon).not.toHaveClass(/active/)

  // 추가 도구 메뉴에서도 열 수 있어야 한다.
  await page.getByRole('button',{name:'추가 도구'}).click()
  // 켜짐 여부를 함께 보여주는 항목은 menuitemcheckbox 로 그려진다.
  await expect(page.getByRole('menuitemcheckbox',{name:'프레젠테이션 목록'})).toBeVisible()
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
