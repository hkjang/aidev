import { afterEach, describe, expect, it, vi } from 'vitest'
import { address, api } from './api'

afterEach(() => vi.restoreAllMocks())

describe('api client', () => {
  it('lets the browser set the multipart boundary for FormData', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    const body = new FormData()
    body.append('file', new Blob(['name,amount\nalpha,10\n']), 'data.csv')

    await api('/api/v1/imports:preview', { method: 'POST', body })

    const request = fetchMock.mock.calls[0][1]
    expect(request?.headers).not.toHaveProperty('Content-Type')
  })
})

describe('cell address', () => {
  it.each([
    [1, 1, 'A1'],
    [7, 26, 'Z7'],
    [9, 27, 'AA9'],
    [3, 703, 'AAA3'],
  ])('maps row %i and column %i to %s', (row, column, expected) => {
    expect(address(row, column)).toBe(expected)
  })
})
