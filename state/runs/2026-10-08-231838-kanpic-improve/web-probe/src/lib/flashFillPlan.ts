import type { Cell } from '../types'
import { cellKey } from '../state/editor'
import type { GridRegion } from './dataRegion'
import { formatCellValue } from './cellFormat'
import { applyRule,inferRule,type FillExample,type FillRule } from './flashFill'

/**
 * 채울 계획. 어떤 줄을 예시로 삼았고 어떤 줄에 무엇을 쓸 것인지 사람이 보고
 * 정할 수 있게 담는다.
 */
export type FillPlan={
  column:number
  rule:FillRule
  examples:Array<{row:number;value:string}>
  writes:Array<{row:number;value:string}>
  /** 규칙이 닿지 못한 줄. 조용히 비워 두지 않고 몇 줄인지 말한다. */
  unreached:number
  /** 첫 줄을 머리글로 보고 본보기에서 뺐는지. 사람에게 그렇게 말해야 한다. */
  headerSkipped:boolean
}

export type PlanFailure='no-examples'|'no-rule'|'nothing-to-fill'

const text=(cell?:Cell)=>cell?formatCellValue(cell.value,cell.style):''

/**
 * 사람이 손으로 채워 둔 칸을 예시로 삼아 나머지 칸의 계획을 세운다.
 *
 * 예시는 **채워진 칸** 이고 대상은 **빈 칸** 이다. 이미 쓴 값을 덮어쓰지
 * 않는다 — 빠른 채우기가 남의 값을 지우면 되돌리기 전에는 무엇이 있었는지
 * 알 수 없다.
 */
export function planFlashFill(cells:Map<string,Cell>,region:GridRegion,column:number):FillPlan|PlanFailure{
  // 머리글이 있는지는 미리 알 수 없다. 열 이름은 그 열의 값이 아니므로 규칙을
  // 설명하지 못하고, 그럴 때 첫 줄을 빼고 한 번 더 해 본다. 숫자가 있는지로
  // 머리글을 가리면 글자만 있는 표 - 이름과 주소 - 의 머리글을 본보기로 삼는다.
  const whole=planFrom(cells,region,column,region.startRow,false)
  if(typeof whole!=='string')return whole
  if(whole==='no-rule'&&region.endRow>region.startRow){
    const skipped=planFrom(cells,region,column,region.startRow+1,true)
    if(typeof skipped!=='string')return skipped
  }
  return whole
}

function planFrom(cells:Map<string,Cell>,region:GridRegion,column:number,firstRow:number,headerSkipped:boolean):FillPlan|PlanFailure{
  const sourcesOf=(row:number)=>{
    const values:string[]=[]
    for(let at=region.startColumn;at<=region.endColumn;at+=1)
      values.push(at===column?'':text(cells.get(cellKey(row,at))))
    return values
  }
  const examples:FillExample[]=[]
  const shown:Array<{row:number;value:string}>=[]
  const targets:number[]=[]
  for(let row=firstRow;row<=region.endRow;row+=1){
    const value=text(cells.get(cellKey(row,column))).trim()
    if(value){
      examples.push({sources:sourcesOf(row),output:value})
      shown.push({row,value})
    }else targets.push(row)
  }
  if(examples.length===0)return 'no-examples'
  if(targets.length===0)return 'nothing-to-fill'
  const rule=inferRule(examples)
  if(!rule)return 'no-rule'
  const writes:Array<{row:number;value:string}>=[]
  let unreached=0
  for(const row of targets){
    const value=applyRule(rule,sourcesOf(row))
    if(value==='')unreached+=1
    else writes.push({row,value})
  }
  if(writes.length===0)return 'no-rule'
  return {column,rule,examples:shown,writes,unreached,headerSkipped}
}
