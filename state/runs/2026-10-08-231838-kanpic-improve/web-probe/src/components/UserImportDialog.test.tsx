import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { UserImportDialog } from './UserImportDialog'

afterEach(()=>{cleanup();vi.restoreAllMocks()})

// jsdom 의 File 은 arrayBuffer 가 없다. 브라우저가 주는 그 하나만 채운다.
function pickedFile(name:string,type:string,text:string){
  const file=new File([text],name,{type})
  Object.defineProperty(file,'arrayBuffer',{value:async()=>new TextEncoder().encode(text).buffer})
  return file
}

describe('UserImportDialog',()=>{
  // 서버는 탭으로 가른 명단도 읽는다(엑셀에서 칸을 골라 붙여 넣으면 그 꼴이고,
  // 파일로 저장하면 .tsv 나 .txt 다). 파일 고르기가 .csv 만 보이면 그 파일은
  // 붙여 넣기로만 들어온다.
  it('offers tab-separated roster files and reads a picked one as it is',async()=>{
    render(<UserImportDialog onClose={vi.fn()} onDone={vi.fn()}/>)
    const input=screen.getByLabelText('CSV 파일') as HTMLInputElement
    const accept=input.accept.split(',')
    expect(accept).toEqual(expect.arrayContaining(['.csv','.tsv','.txt']))
    const roster='user_id\tdisplay_name\nkim.nara\t김나라'
    fireEvent.change(input,{target:{files:[pickedFile('roster.tsv','text/tab-separated-values',roster)]}})
    await waitFor(()=>expect((screen.getByLabelText('CSV 내용') as HTMLTextAreaElement).value).toBe(roster))
  })
})
