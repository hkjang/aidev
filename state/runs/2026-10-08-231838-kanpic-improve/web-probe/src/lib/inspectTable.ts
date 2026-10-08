import type { Cell } from '../types'
import type { GridRegion } from './dataRegion'
import { convertTextNumbers, removeDuplicateRows, trimWhitespace, unmergeAndFill } from './dataCleanup'
import { planRemoveSubtotals } from './subtotal'

/** 검사에서 찾은 것 하나. kind 는 그대로 데이터 정리의 갈래 이름이다. */
export type TableFinding={
  kind:'numbers'|'trim'|'unmerge'|'duplicates'|'subtotals'
  count:number
  title:string
  detail:string
  /** 고쳐 두면 셈이 달라지는 것. 모양만 다듬는 것과 나눠 보여 준다. */
  changesTotals:boolean
}

/**
 * 표를 훑어 고칠 만한 것을 센다.
 *
 * 정리 도구는 여섯 개나 되는데, 사람이 "데이터 정리" 를 열어 볼 생각을
 * 해야 알 수 있었다. 무엇이 잘못됐는지 모르는 채로 열어 보는 사람은 없다.
 * 표를 보고 먼저 말해 주는 쪽이 맞다.
 *
 * 세기만 하고 아무것도 바꾸지 않는다. 무엇을 고칠지는 사람이 각 갈래의
 * 미리보기를 보고 정한다 — 중복 행을 지우는 일 같은 것은 되돌리기 어렵다.
 */
export function inspectTable(cells:Map<string,Cell>,region:GridRegion,headerRows:number):TableFinding[]{
  const findings:TableFinding[]=[]

  const numbers=convertTextNumbers(cells,region).changed
  if(numbers>0)findings.push({kind:'numbers',count:numbers,changesTotals:true,
    title:'글자로 담긴 숫자',
    detail:'=SUM 이 이 칸들을 빼고 셈합니다. 합계가 작게 나오는데 이유는 어디에도 적히지 않습니다.'})

  const merges=unmergeAndFill(cells,region)
  if(merges.ranges>0)findings.push({kind:'unmerge',count:merges.ranges,changesTotals:true,
    title:'병합된 칸',
    detail:'병합된 표는 정렬도 피벗도 되지 않습니다. 이름표가 빈 행은 집계에서 통째로 빠집니다.'})

  const subtotals=planRemoveSubtotals(cells,region).rows.length
  if(subtotals>0)findings.push({kind:'subtotals',count:subtotals,changesTotals:true,
    title:'부분합 행',
    detail:'자료 사이에 낀 소계는 다시 합할 때 두 번 세어집니다.'})

  const duplicates=removeDuplicateRows(cells,region,headerRows).removed
  if(duplicates>0)findings.push({kind:'duplicates',count:duplicates,changesTotals:true,
    title:'중복된 행',
    detail:'같은 내용이 거듭 적힌 행입니다. 세어 보기 전에는 합계가 맞는지 알 수 없습니다.'})

  const trims=trimWhitespace(cells,region).changed
  if(trims>0)findings.push({kind:'trim',count:trims,changesTotals:false,
    title:'앞뒤·가운데 공백',
    detail:'눈에 보이지 않지만 정렬과 견주기에서는 다른 값이 됩니다.'})

  return findings
}
