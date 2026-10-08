import { describe, expect, it } from 'vitest'
import type { OutboxOperation } from './outbox'
import { forSheet, pendingInWorkbook } from './outboxScope'

function operation(sheetId:string):OutboxOperation{
  return {id:`op-${sheetId}`,sheetId,endpoint:'batch',attempts:0,createdAt:0,body:{}}
}

describe('forSheet', () => {
  it('applies a result that belongs to the open sheet', () => {
    const seen:number[]=[]
    forSheet<{server_version:number}>('sheet-1',result=>seen.push(result.server_version))(operation('sheet-1'),{server_version:7})
    expect(seen).toEqual([7])
  })

  it('drops a result queued by another workbook so its version never lands here', () => {
    const seen:number[]=[]
    const apply=forSheet<{server_version:number}>('sheet-1',result=>seen.push(result.server_version))
    apply(operation('other-workbook-sheet'),{server_version:99})
    expect(seen).toEqual([])
  })

  it('drops every result while no sheet is open', () => {
    const seen:number[]=[]
    forSheet<{server_version:number}>(undefined,result=>seen.push(result.server_version))(operation('sheet-1'),{server_version:7})
    expect(seen).toEqual([])
  })
})

describe('pendingInWorkbook', () => {
  it('counts only the operations belonging to this workbook', () => {
    const queue=[operation('sheet-1'),operation('elsewhere'),operation('sheet-2')]
    expect(pendingInWorkbook(queue,['sheet-1','sheet-2']).map(item=>item.sheetId)).toEqual(['sheet-1','sheet-2'])
  })

  it('ignores a stuck operation from another workbook', () => {
    expect(pendingInWorkbook([operation('elsewhere')],['sheet-1'])).toEqual([])
  })

  it('is empty when the workbook has no sheets yet', () => {
    expect(pendingInWorkbook([operation('sheet-1')],[])).toEqual([])
  })
})
