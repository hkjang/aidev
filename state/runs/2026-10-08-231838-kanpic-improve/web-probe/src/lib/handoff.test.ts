import { afterEach, describe, expect, it, vi } from 'vitest'
import { fetchHandoffOrigin, handoffAddress, handoffOriginDetail, handoffOriginLabel, handoffResource, sendHandoff, type HandoffOrigin, type HandoffTarget } from './handoff'

const ptium: HandoffTarget = { name: 'ptium', origin: 'https://ptium.intra', formats: ['csv', 'xlsx'] }
const issued = { claim: 'abc+def/ghi', source: 'https://kanpic.intra', filename: '실적.xlsx', content_type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet', bytes: 10, expires_at: '2026-09-14T09:05:00+09:00' }

describe('handoffAddress', () => {
  it('opens the receiver with source and claim in the query', () => {
    const address = new URL(handoffAddress(ptium, issued))
    expect(address.origin + address.pathname).toBe('https://ptium.intra/handoff')
    expect(address.searchParams.get('source')).toBe('https://kanpic.intra')
    expect(address.searchParams.get('claim')).toBe('abc+def/ghi')
  })
  it('names one sheet only for csv', () => {
    expect(handoffResource('wb', 'csv', 'sh')).toBe('wb/sheets/sh')
    expect(handoffResource('wb', 'xlsx', 'sh')).toBe('wb')
    expect(handoffResource('wb', 'csv')).toBe('wb')
  })
})

describe('sendHandoff', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('opens the window before asking for the claim, then points it at the receiver', async () => {
    const fetchMock = vi.fn(async (_input: string, _init?: RequestInit) => new Response(JSON.stringify(issued), { status: 201, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)
    const popup = { location: { href: 'about:blank' }, close: vi.fn() }
    const open = vi.fn(() => popup)
    await sendHandoff(ptium, 'wb', 'xlsx', open)
    // The window is opened synchronously in the click, before any await.
    expect(open).toHaveBeenCalledTimes(1)
    expect(open.mock.invocationCallOrder[0]).toBeLessThan(fetchMock.mock.invocationCallOrder[0])
    const [path, init] = fetchMock.mock.calls[0]!
    expect(path).toBe('/api/v1/handoff/claims')
    expect(JSON.parse(String(init?.body))).toEqual({ resource: 'wb', format: 'xlsx' })
    expect(popup.location.href).toBe(handoffAddress(ptium, issued))
    expect(popup.close).not.toHaveBeenCalled()
  })

  it('closes the window and reports when no claim is issued', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({ error: { message: '소유자가 뷰어의 내보내기와 복사를 제한했습니다.' } }), { status: 403 })))
    const popup = { location: { href: 'about:blank' }, close: vi.fn() }
    await expect(sendHandoff(ptium, 'wb', 'csv', () => popup)).rejects.toThrow('소유자가 뷰어의 내보내기와 복사를 제한했습니다.')
    expect(popup.close).toHaveBeenCalledTimes(1)
    expect(popup.location.href).toBe('about:blank')
  })
})

describe('handoff origin', () => {
  afterEach(() => vi.unstubAllGlobals())
  const received: HandoffOrigin = { workbook_id: 'wb', source: 'https://ptium.intra', service: 'ptium', filename: '실적.csv', received_by: 'bob', received_at: '2026-09-14T09:05:00+09:00' }

  it('names the sender by its allow-list name, or by host once it has left the list', () => {
    expect(handoffOriginLabel(received)).toBe('ptium 에서 받음')
    expect(handoffOriginLabel({ ...received, service: undefined })).toBe('ptium.intra 에서 받음')
    expect(handoffOriginLabel({ source: 'not a url', service: '' })).toBe('not a url 에서 받음')
    expect(handoffOriginDetail(received)).toContain('https://ptium.intra 에서 실적.csv 을(를) ')
  })

  it('reads a missing origin as "not handed off" and reports anything else', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({ error: { code: 'not_found', message: '요청한 리소스를 찾을 수 없습니다.' } }), { status: 404 })))
    await expect(fetchHandoffOrigin('wb')).resolves.toBeNull()
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify(received), { status: 200, headers: { 'Content-Type': 'application/json' } })))
    await expect(fetchHandoffOrigin('wb')).resolves.toEqual(received)
    vi.stubGlobal('fetch', vi.fn(async () => new Response('', { status: 500 })))
    await expect(fetchHandoffOrigin('wb')).rejects.toThrow('요청 실패 (500)')
  })
})
