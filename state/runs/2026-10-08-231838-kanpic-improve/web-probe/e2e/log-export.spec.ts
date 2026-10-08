import { expect, test } from '@playwright/test'

// 감사에서는 "그 기간 기록을 주세요" 라고 한다. 화면은 200건에서 끊기므로
// 지금까지는 화면을 긁는 수밖에 없었다. 거른 조건 그대로, 개수를 자르지
// 않고 내려받는 데까지 확인한다.
test('an administrator exports the filtered log as CSV', async ({ page, request }) => {
  await page.goto('/admin?tab=logs')
  await expect(page.getByRole('heading', { name: '서버 로그' })).toBeVisible()
  await expect(page.getByLabel('시작 날짜')).toBeVisible()

  // 화면과 내보내기가 같은 조건을 쓴다.
  await page.getByLabel('시작 날짜').fill('2020-01-01')
  const link = page.getByRole('link', { name: 'CSV 내보내기' })
  await expect(link).toHaveAttribute('href', /from=2020-01-01/)

  const csv = await request.get('/api/v1/admin/logs.csv?from=2020-01-01')
  expect(csv.status()).toBe(200)
  expect(csv.headers()['content-type']).toContain('text/csv')
  expect(csv.headers()['content-disposition']).toContain('.csv')
  const body = await csv.text()
  // 엑셀이 한글을 깨뜨리지 않도록 BOM 을 붙인다. 감사에 넘긴 파일을 받은
  // 사람은 대개 엑셀로 연다.
  expect(body.charCodeAt(0)).toBe(0xFEFF)
  expect(body.split('\n')[0]).toContain('logged_at,level,message,trace_id,attributes')

  // 거꾸로 적은 기간은 조용히 빈 파일을 주지 않고 거절한다.
  const backwards = await request.get('/api/v1/admin/logs.csv?from=2026-01-06&to=2026-01-05')
  expect(backwards.status()).toBeGreaterThanOrEqual(400)
})
