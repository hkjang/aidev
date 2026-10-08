import { describe, expect, it } from 'vitest'
import { decodeAnnounced, readFileText } from './fileText'

const bytes=(...values:number[])=>new Uint8Array(values)
function utf16(text:string,littleEndian:boolean,mark:number[]){
  const out=[...mark]
  for(let index=0;index<text.length;index++){
    const unit=text.charCodeAt(index)
    out.push(...(littleEndian?[unit&0xFF,unit>>8]:[unit>>8,unit&0xFF]))
  }
  return bytes(...out)
}
const roster='user_id,display_name\nkim.nara,김나라'

describe('decodeAnnounced',()=>{
  it('reads what the file says it is',()=>{
    // PowerShell's Export-Csv and Excel's "유니코드 텍스트" save write these.
    const littleEndian=utf16(roster,true,[0xFF,0xFE]),bigEndian=utf16(roster,false,[0xFE,0xFF])
    // What the browser hands back on its own — always UTF-8 — is not the roster.
    expect(new TextDecoder().decode(littleEndian)).not.toBe(roster)
    expect(decodeAnnounced(littleEndian)).toBe(roster)
    expect(decodeAnnounced(bigEndian)).toBe(roster)
    // Excel's "CSV UTF-8" save marks the file too; the mark is not part of the header.
    expect(decodeAnnounced(bytes(0xEF,0xBB,0xBF,...new TextEncoder().encode(roster)))).toBe(roster)
  })

  it('guesses nothing about a file that announces nothing',()=>{
    const plain=new TextEncoder().encode(roster)
    expect(decodeAnnounced(plain)).toBe(roster)
    expect(decodeAnnounced(bytes())).toBe('')
  })

  it('does not mistake UTF-32 for UTF-16',()=>{
    // UTF-32LE opens with the UTF-16LE mark. Reading it as UTF-16 would make
    // characters out of the padding zeros, so it stays unread for the server
    // to refuse rather than becoming a roster of nonsense.
    expect(decodeAnnounced(bytes(0xFF,0xFE,0x00,0x00,0x61,0x00,0x00,0x00))).not.toBe('a')
  })
})

// jsdom 의 File 은 arrayBuffer 만 있고 브라우저의 나머지는 없다. 고른 파일에서
// 바이트를 받아 위 규칙에 넘기는지만 본다.
describe('readFileText',()=>{
  it('reads a picked file by what it announces',async()=>{
    const picked=utf16(roster,true,[0xFF,0xFE])
    const file={arrayBuffer:async()=>picked.buffer} as unknown as File
    expect(await readFileText(file)).toBe(roster)
  })
})
