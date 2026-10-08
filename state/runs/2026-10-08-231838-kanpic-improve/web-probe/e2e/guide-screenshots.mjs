// 가이드 문서(docs/USER_GUIDE.md·docs/ADMIN_GUIDE.md)가 싣는 화면 캡처를 찍는다.
//
// 실행 방법 (버려도 되는 로컬 배포에 대고 돌린다):
//   cd web && KANPIC_GUIDE_BASE_URL=http://localhost:8123 node e2e/guide-screenshots.mjs
//
// 이 스크립트는 e2e 시나리오와 대상 주소를 나눠 쓰지 않는다. 캡처 전용
// 환경 변수 하나만 보고, 값이 없으면 그 자리에서 멈춘다. localhost 가 아닌
// 곳을 찍으려면 그 배포가 버려도 되는 것임을 KANPIC_GUIDE_DISPOSABLE=yes 로
// 밝혀야 한다 — 캡처는 워크북과 사용자를 만들어 넣기 때문이다.
//
// 전역 설정(PUT /api/v1/admin/settings/*)은 건드리지 않는다. 읽기만 하므로
// 되돌릴 것도 없다.
import { chromium } from '@playwright/test'
import { mkdir } from 'node:fs/promises'
import path from 'node:path'

const baseURL = process.env.KANPIC_GUIDE_BASE_URL
if (!baseURL) {
  console.error('KANPIC_GUIDE_BASE_URL 이 없습니다. 캡처할 배포 주소를 정해 주세요.')
  process.exit(1)
}
const host = new URL(baseURL).hostname
if (!['localhost', '127.0.0.1', '::1'].includes(host) && process.env.KANPIC_GUIDE_DISPOSABLE !== 'yes') {
  console.error(`${host} 는 로컬이 아닙니다. 버려도 되는 배포라면 KANPIC_GUIDE_DISPOSABLE=yes 로 밝혀 주세요.`)
  process.exit(1)
}

const outputDir = path.resolve(process.cwd(), '..', 'docs/images/guide')
const viewport = { width: 1440, height: 900 }

/** 화면에 남는 값은 전부 가짜다 — 실명·실제 주소·실제 비밀값을 찍지 않는다. */
const ROSTER = [
  ['dawon.ko', '고다원', 'dawon.ko@example.com', '재무팀'],
  ['haneul.im', '임하늘', 'haneul.im@example.com', '영업기획'],
  ['jiho.baek', '백지호', 'jiho.baek@example.com', '물류운영'],
  ['mirae.song', '송미래', 'mirae.song@example.com', '경영지원'],
  ['siwoo.jang', '장시우', 'siwoo.jang@example.com', '데이터팀'],
]

const SALES = [
  ['지점', '1분기', '2분기', '3분기', '합계'],
  ['서울 본점', 128400000, 141200000, 155900000, null],
  ['부산 지점', 76300000, 71800000, 88400000, null],
  ['대전 지점', 54100000, 60500000, 63200000, null],
  ['광주 지점', 41900000, 45300000, 47700000, null],
  ['대구 지점', 62800000, 66100000, 70400000, null],
]

const WORKBOOKS = [
  {
    title: '2026 분기별 지점 매출',
    rows: SALES,
    formulas: [
      { row: 2, column: 5, formula: '=SUM(B2:D2)' },
      { row: 3, column: 5, formula: '=SUM(B3:D3)' },
      { row: 4, column: 5, formula: '=SUM(B4:D4)' },
      { row: 5, column: 5, formula: '=SUM(B5:D5)' },
      { row: 6, column: 5, formula: '=SUM(B6:D6)' },
      { row: 7, column: 1, value: '전체' },
      { row: 7, column: 2, formula: '=SUM(B2:B6)' },
      { row: 7, column: 3, formula: '=SUM(C2:C6)' },
      { row: 7, column: 4, formula: '=SUM(D2:D6)' },
      { row: 7, column: 5, formula: '=SUM(E2:E6)' },
    ],
  },
  {
    title: '부서 예산 집행 현황',
    rows: [
      ['부서', '배정 예산', '집행액', '집행률'],
      ['재무팀', 42000000, 31800000, null],
      ['영업기획', 68000000, 59200000, null],
      ['물류운영', 51000000, 47600000, null],
      ['데이터팀', 39000000, 22100000, null],
    ],
    formulas: [
      { row: 2, column: 4, formula: '=C2/B2' },
      { row: 3, column: 4, formula: '=C3/B3' },
      { row: 4, column: 4, formula: '=C4/B4' },
      { row: 5, column: 4, formula: '=C5/B5' },
    ],
  },
  {
    title: '물류 창고 재고 대장',
    rows: [
      ['품목코드', '품목명', '창고', '수량', '안전재고'],
      ['A-1001', '포장 박스 중형', '이천 1창고', 1240, 800],
      ['A-1002', '포장 박스 대형', '이천 1창고', 430, 600],
      ['B-2011', '완충재 롤', '용인 2창고', 980, 500],
      ['B-2012', '테이프 72mm', '용인 2창고', 2100, 1200],
    ],
    formulas: [],
  },
  {
    title: '고객 문의 접수 대장',
    rows: [
      ['접수일', '채널', '문의 유형', '담당', '상태'],
      ['2026-09-01', '이메일', '배송 지연', '고다원', '처리 완료'],
      ['2026-09-02', '전화', '반품 요청', '임하늘', '처리 중'],
      ['2026-09-03', '이메일', '세금계산서', '송미래', '처리 완료'],
      ['2026-09-04', '채팅', '제품 문의', '장시우', '접수'],
    ],
    formulas: [],
  },
]

const shot = async (page, name) => {
  const file = path.join(outputDir, `${name}.png`)
  await page.screenshot({ path: file })
  console.log(`찍음: ${path.relative(path.resolve(process.cwd(), '..'), file)}`)
}

/** 격자는 Canvas 라 그려질 때까지 기다린다. 스피너가 박제된 캡처는 쓸 수 없다. */
const settle = async (page, ms = 900) => {
  await page.waitForLoadState('networkidle').catch(() => {})
  await page.waitForTimeout(ms)
}

const run = async () => {
  await mkdir(outputDir, { recursive: true })
  const browser = await chromium.launch({ executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH || undefined })
  const context = await browser.newContext({ baseURL, viewport, locale: 'ko-KR', timezoneId: 'Asia/Seoul' })
  const page = await context.newPage()
  const request = context.request

  // 로그인 화면은 사람이 실제로 보는 모습으로 찍는다. 자격 증명은 전용 환경
  // 변수로만 받는다 — 아이디도 비밀번호도 이 파일에 적지 않는다.
  const adminID = process.env.KANPIC_GUIDE_ADMIN_ID
  const adminPassword = process.env.KANPIC_GUIDE_ADMIN_PASSWORD
  await page.goto('/login')
  await settle(page, 600)
  await shot(page, 'login')
  if (adminID && adminPassword) {
    await page.getByLabel('아이디').fill(adminID)
    await page.getByLabel('비밀번호').fill(adminPassword)
    await page.getByRole('button', { name: /관리자로 로그인/ }).click()
    await page.waitForURL(url => !url.pathname.startsWith('/login'))
    await settle(page, 600)
  }

  // ── 가짜 데이터 채우기 ────────────────────────────────────────────────
  const csv = ['user_id,display_name,email,note', ...ROSTER.map(row => row.join(','))].join('\n')
  await request.post('/api/v1/admin/users:import', { data: { csv } })
  await request.post(`/api/v1/admin/users/${ROSTER[4][0]}/roles`, { data: { role: 'kanpic-analyst' } })
  await request.post(`/api/v1/admin/users/${ROSTER[0][0]}/roles`, { data: { role: 'kanpic-admin' } })

  const headquarters = await request.post('/api/v1/departments', { data: { name: '영업본부' } }).then(r => r.json())
  const team = await request.post('/api/v1/departments', { data: { name: '영업1팀', parent_id: headquarters.id } }).then(r => r.json())
  const support = await request.post('/api/v1/departments', { data: { name: '경영지원본부' } }).then(r => r.json())
  await request.post(`/api/v1/departments/${team.id}/members`, { data: { user_ids: [ROSTER[1][0], ROSTER[4][0]] } })
  await request.post(`/api/v1/departments/${support.id}/members`, { data: { user_ids: [ROSTER[0][0], ROSTER[3][0]] } })
  await request.post(`/api/v1/departments/${headquarters.id}/managers`, { data: { user_ids: [ROSTER[1][0]] } })

  // 키 목록 화면이 비어 있지 않게 두 개를 만들어 둔다. 원문은 만든 직후에만
  // 보이는 값이라 캡처하는 화면(목록)에는 남지 않는다.
  await request.post('/api/v1/me/api-keys', { data: { name: '분석 대시보드 연동', scopes: ['range.read', 'formula.read'], expires_at: null } })
  await request.post('/api/v1/me/api-keys', { data: { name: 'MCP 에이전트', scopes: ['mcp.use', 'range.read', 'range.write'], expires_at: null } })

  const created = []
  for (const item of WORKBOOKS) {
    const workbook = await request.post('/api/v1/workbooks', { data: { title: item.title, workspace_id: 'default' } }).then(r => r.json())
    const sheet = workbook.sheets[0].id
    const cells = item.rows.flatMap((row, rowIndex) => row
      .map((value, column) => (value === null ? null : { row: rowIndex + 1, column: column + 1, value }))
      .filter(Boolean))
    await request.patch(`/api/v1/sheets/${sheet}/cells:batch`, {
      data: { idempotency_key: `guide-${workbook.id}`, cells: [...cells, ...item.formulas] },
    })
    created.push(workbook)
  }
  const sales = created[0]

  // ── 사용자 가이드 화면 ────────────────────────────────────────────────
  await page.goto('/')
  await settle(page)
  await shot(page, 'home')

  // 가져오기 미리보기 — 실제 파일을 골라 서버가 읽어 본 결과를 그대로 찍는다.
  const importCSV = ['지점,1분기,2분기,3분기',
    '서울 본점,128400000,141200000,155900000',
    '부산 지점,76300000,71800000,88400000',
    '대전 지점,54100000,60500000,63200000'].join('\n')
  await page.locator('input[type="file"]').setInputFiles({ name: '지점별 매출.csv', mimeType: 'text/csv', buffer: Buffer.from(`﻿${importCSV}`, 'utf8') })
  await page.waitForSelector('.import-modal')
  await settle(page, 700)
  await shot(page, 'home-import')
  await page.keyboard.press('Escape')

  await page.goto(`/workbooks/${sales.id}`)
  await page.waitForSelector('canvas.grid-canvas')
  await settle(page)
  // 오른쪽 패널은 접고 격자를 넓게 찍는다.
  await page.locator('.toolbar').getByRole('button', { name: 'AI 도우미', exact: true }).click()
  await settle(page, 400)
  await shot(page, 'editor')

  const toolbar = page.locator('.toolbar')
  // 버전 이력은 이름을 붙여 둔 버전이 하나라도 있는 상태로 찍는다.
  await toolbar.getByRole('button', { name: '버전 이력', exact: true }).click()
  await page.getByPlaceholder('예: 2026년 3분기 확정').fill('3분기 실적 확정')
  await page.getByRole('button', { name: '저장', exact: true }).click()
  await settle(page, 1000)
  await shot(page, 'editor-history')
  await toolbar.getByRole('button', { name: '버전 이력', exact: true }).click()

  await page.getByRole('button', { name: /공유/ }).first().click()
  await page.waitForSelector('[role="dialog"]')
  await settle(page, 600)
  await shot(page, 'editor-share')
  await page.keyboard.press('Escape')

  await toolbar.getByRole('button', { name: 'AI 도우미', exact: true }).click()
  await settle(page, 600)
  await shot(page, 'editor-ai')
  await toolbar.getByRole('button', { name: 'AI 도우미', exact: true }).click()

  // 이름 상자로 범위를 잡고 차트를 만든다 — 차트 패널이 실제로 그린 그림을 싣는다.
  await page.getByRole('combobox', { name: '이름 상자' }).fill('A1:D6')
  await page.getByRole('combobox', { name: '이름 상자' }).press('Enter')
  await toolbar.getByRole('button', { name: '차트 패널', exact: true }).click()
  await page.getByRole('button', { name: '새 차트' }).click()
  const dialog = page.getByRole('dialog', { name: '차트 만들기' })
  await dialog.getByLabel('차트 제목').fill('지점별 분기 매출')
  await dialog.getByRole('button', { name: '차트 저장' }).click()
  await page.waitForSelector('[data-chart-id]')
  await settle(page)
  await shot(page, 'editor-chart')

  await page.goto('/preferences')
  await settle(page, 700)
  await shot(page, 'preferences')
  await page.getByRole('button', { name: 'API 키' }).click()
  await settle(page, 600)
  await shot(page, 'preferences-keys')

  // ── 관리자 가이드 화면 ────────────────────────────────────────────────
  const adminTabs = [
    ['overview', 'admin-overview'],
    ['users', 'admin-users'],
    ['workbooks', 'admin-workbooks'],
    ['departments', 'admin-departments'],
    ['settings', 'admin-settings'],
    ['logs', 'admin-logs'],
    ['keys', 'admin-keys'],
    ['system', 'admin-system'],
  ]
  for (const [tab, name] of adminTabs) {
    await page.goto(`/admin?tab=${tab}`)
    await settle(page, 800)
    await shot(page, name)
  }

  // 일괄 등록은 미리 보기까지 눌러 무엇이 바뀌는지 보여 주는 화면을 찍는다.
  await page.goto('/admin?tab=users')
  await settle(page, 600)
  await page.getByRole('button', { name: '일괄 등록…' }).click()
  const roster = page.getByRole('dialog', { name: '사용자 일괄 등록' })
  await roster.getByLabel('CSV 내용').fill([
    '사용자 ID,이름,이메일',
    'nuri.han,한누리,nuri.han@example.com',
    'yeojin.oh,오여진,yeojin.oh@example.com',
    `${ROSTER[2][0]},${ROSTER[2][1]},${ROSTER[2][2]}`,
  ].join('\n'))
  await roster.getByRole('button', { name: '미리 보기' }).click()
  await page.waitForSelector('.user-import-counts')
  await settle(page, 600)
  await shot(page, 'admin-users-import')
  await page.keyboard.press('Escape')

  await context.close()
  await browser.close()
}

run().catch(error => {
  console.error(error)
  process.exit(1)
})
