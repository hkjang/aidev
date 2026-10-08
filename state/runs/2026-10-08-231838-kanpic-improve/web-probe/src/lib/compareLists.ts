import type { Cell } from '../types'
import { cellKey } from '../state/editor'
import type { GridRegion } from './dataRegion'
import { significantDigits, textToNumber } from './spreadsheetNumber'
import { parseFilterRange } from './filter'

export type CompareSide='left'|'right'
export type CompareRow={key:string;label:string;row:number;count:number}
export type CompareResult={
  onlyLeft:CompareRow[]
  onlyRight:CompareRow[]
  both:number
  /** 한쪽에 같은 키가 두 번 이상 나온 것. 대사에서는 이것 자체가 발견이다. */
  duplicated:Array<{side:CompareSide;key:string;label:string;count:number}>
  /** 키 칸이 비어 있어 견줄 수 없던 행. 조용히 빼면 수가 맞지 않는다. */
  blank:{left:number;right:number}
}

/**
 * 두 목록을 견주는 열쇠를 만든다.
 *
 * 은행에서 받은 내역은 금액이 "1,234" 처럼 글자로 오고 장부에는 숫자로
 * 적혀 있다. 사람 눈에는 같은 값인데 글자 그대로 견주면 하나도 맞지 않는다.
 * 숫자로 읽히는 것은 숫자로 견준다 — 세는 규칙은 다른 곳과 같은 것을 쓴다.
 *
 * 앞뒤 빈칸은 버리고 영문 대소문자는 가리지 않는다. 다른 곳에서 옮겨 적힌
 * 이름은 그 정도로 흔들리고, 그 흔들림까지 "없는 항목" 으로 세면 대사표가
 * 쓸모없어진다.
 */
/**
 * 수로 바꾸면 안 되는 것을 가려낸다.
 *
 *   "007"    앞의 0은 번호를 뜻한다. 수로 보면 7 과 같아져,
 *            007번 전표와 7번 전표가 한 항목이 된다.
 *   20자리   배정밀도가 정확히 담는 것은 열다섯 자리까지다. 서로 다른
 *            계좌번호 둘이 같은 키가 되어 "맞았다" 고 나온다.
 *
 * 대사에서 틀리게 맞았다고 하는 것은 안 맞았다고 하는 것보다 나쁘다.
 * 안 맞으면 사람이 들여다보지만, 맞았다고 하면 거기서 끝난다.
 *
 * 쉼표나 통화 기호가 붙은 금액은 여기서 걸리지 않는다. 그것은 번호가 아니라
 * 금액이고, "1,000" 과 1000 은 같아야 한다.
 */
function looksLikeIdentifier(text:string){
  const digits=text.replace(/^[+-]/,'')
  if(!/^\d+(\.\d+)?$/.test(digits))return false
  if(/^0\d/.test(digits))return true
  return significantDigits(digits)>15
}

export function compareKey(value:unknown):string|undefined{
  if(value==null)return undefined
  if(typeof value==='boolean')return value?'TRUE':'FALSE'
  if(typeof value==='number')return Number.isFinite(value)?String(value):undefined
  if(typeof value!=='string')return undefined
  const text=value.trim()
  if(text==='')return undefined
  if(looksLikeIdentifier(text))return text.toLowerCase()
  const numeric=Number(text)
  if(Number.isFinite(numeric))return String(numeric)
  const loose=textToNumber(text)
  if(loose!==undefined)return String(loose)
  return text.toLowerCase()
}

function readSide(cells:Map<string,Cell>,region:GridRegion,keyColumn:number,headerRows:number){
  const byKey=new Map<string,CompareRow>()
  let blank=0
  for(let row=region.startRow+headerRows;row<=region.endRow;row+=1){
    const cell=cells.get(cellKey(row,keyColumn))
    const key=compareKey(cell?.value)
    if(key===undefined){
      // 행 전체가 비어 있으면 목록의 끝일 뿐이다. 키만 비어 있는 행은 셈한다.
      if(rowHasAnything(cells,region,row))blank+=1
      continue
    }
    const existing=byKey.get(key)
    if(existing){existing.count+=1;continue}
    byKey.set(key,{key,label:displayText(cell),row,count:1})
  }
  return {byKey,blank}
}

function rowHasAnything(cells:Map<string,Cell>,region:GridRegion,row:number){
  for(let column=region.startColumn;column<=region.endColumn;column+=1){
    const cell=cells.get(cellKey(row,column))
    if(cell?.formula||(cell?.value!=null&&cell.value!==''))return true
  }
  return false
}

const displayText=(cell?:Cell)=>cell?.value==null?'':typeof cell.value==='string'?cell.value:String(cell.value)

/**
 * 두 목록을 키 열로 견준다. 어느 쪽에만 있는지, 양쪽에 있는지, 한쪽에서
 * 거듭 나오는지를 낸다.
 */
export function compareLists(
  left:{cells:Map<string,Cell>;region:GridRegion;keyColumn:number;headerRows:number},
  right:{cells:Map<string,Cell>;region:GridRegion;keyColumn:number;headerRows:number},
):CompareResult{
  const a=readSide(left.cells,left.region,left.keyColumn,left.headerRows)
  const b=readSide(right.cells,right.region,right.keyColumn,right.headerRows)
  const onlyLeft:CompareRow[]=[],onlyRight:CompareRow[]=[]
  let both=0
  for(const row of a.byKey.values()){if(b.byKey.has(row.key))both+=1;else onlyLeft.push(row)}
  for(const row of b.byKey.values())if(!a.byKey.has(row.key))onlyRight.push(row)
  const duplicated=[
    ...[...a.byKey.values()].filter(row=>row.count>1).map(row=>({side:'left' as const,key:row.key,label:row.label,count:row.count})),
    ...[...b.byKey.values()].filter(row=>row.count>1).map(row=>({side:'right' as const,key:row.key,label:row.label,count:row.count})),
  ]
  return {onlyLeft,onlyRight,both,duplicated,blank:{left:a.blank,right:b.blank}}
}

/**
 * "장부!A1:C100" 을 시트 이름과 범위로 나눈다. 느낌표가 없으면 시트 이름이
 * 없는 것으로 보고 undefined 를 낸다 — 부르는 쪽이 지금 시트를 넣는다.
 *
 * 시트 이름에도 느낌표가 들어갈 수 있으므로 **마지막** 느낌표에서 자른다.
 * 앞에서 자르면 "1분기!실적" 이라는 이름의 시트를 영영 못 고른다.
 */
export function splitSheetRange(value:string):{sheetName?:string;region:GridRegion}|undefined{
  const text=value.trim()
  if(text==='')return undefined
  const split=text.lastIndexOf('!')
  const region=parseFilterRange(text.slice(split+1))
  if(!region)return undefined
  if(split<0)return {region}
  // 이름에 빈칸이 있으면 따옴표로 감싸는 것이 스프레드시트의 버릇이다.
  const sheetName=text.slice(0,split).trim().replace(/^'(.*)'$/,'$1')
  return sheetName===''?undefined:{sheetName,region}
}
