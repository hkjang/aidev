import { describe, expect, it } from 'vitest'
import type { OutboxOperation } from './outbox'
import { MAX_ATTEMPTS, blocked, isBlocked, sendable } from './outboxQueue'

function operation(id:string,sheetId:string,attempts=0):OutboxOperation{
  return {id,sheetId,endpoint:'batch',attempts,createdAt:0,body:{}}
}

describe('isBlocked', () => {
  it('gives up only after the attempt budget runs out', () => {
    expect(isBlocked(operation('a','s',MAX_ATTEMPTS-1))).toBe(false)
    expect(isBlocked(operation('a','s',MAX_ATTEMPTS))).toBe(true)
  })
})

describe('sendable', () => {
  it('sends everything while nothing is blocked', () => {
    const queue=[operation('1','s1'),operation('2','s1')]
    expect(sendable(queue).map(item=>item.id)).toEqual(['1','2'])
  })

  it('holds back the later edits of a sheet whose earlier one is stuck', () => {
    const queue=[operation('1','s1',MAX_ATTEMPTS),operation('2','s1'),operation('3','s1')]
    expect(sendable(queue)).toEqual([])
  })

  it('lets other sheets through so one stuck workbook never stops the rest', () => {
    const queue=[operation('1','stuck',MAX_ATTEMPTS),operation('2','stuck'),operation('3','other')]
    expect(sendable(queue).map(item=>item.id)).toEqual(['3'])
  })

  it('keeps sending a sheet whose earlier attempts have not run out yet', () => {
    const queue=[operation('1','s1',MAX_ATTEMPTS-1),operation('2','s1')]
    expect(sendable(queue).map(item=>item.id)).toEqual(['1','2'])
  })
})

describe('blocked', () => {
  it('lists every operation that gave up, in queue order', () => {
    const queue=[operation('1','s1',MAX_ATTEMPTS),operation('2','s1'),operation('3','s2',MAX_ATTEMPTS+2)]
    expect(blocked(queue).map(item=>item.id)).toEqual(['1','3'])
  })
})
