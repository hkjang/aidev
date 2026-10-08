import { AlertTriangle, X } from 'lucide-react'
import { address } from '../lib/api'
import { explainFormulaError } from '../lib/formulaError'
import type { MutationResult } from '../types'

/**
 * 방금 한 편집이 다른 곳의 수식을 깨뜨렸다고 알린다. 행 하나를 지워
 * 열두 곳이 `#REF!` 가 되어도 화면은 아무 말이 없었다. 편집을 막지 않도록
 * 대화상자가 아니라 옆으로 비켜선 안내로 둔다.
 */
export function FormulaIssueNotice({issues,dropped=[],automations=[],unmerged=[],backup,onOpen,onRevert,onClose}:{
  issues:MutationResult['formula_errors']
  /** 다른 사람이 그 행이나 열을 지워 자리를 잃은 편집. */
  dropped?:NonNullable<MutationResult['dropped_cells']>
  /** 이 편집이 실행시킨 자동화 중 실패한 것. 서버 로그에만 남던 실패다. */
  automations?:NonNullable<MutationResult['automation_failures']>
  /** 이 편집이 풀어 버린 병합. 병합된 칸에 붙여넣으면 병합 전체가 풀린다 — 조용히 풀리면 사람은 표가 망가졌다고 읽는다. */
  unmerged?:string[]
  /** 되돌릴 수 없는 편집 직전의 자동 백업. 행·열 삭제에만 붙는다. */
  backup?:{versionId:string;summary:string}
  onOpen:(issue:MutationResult['formula_errors'][number])=>void
  onRevert?:(backup:{versionId:string;summary:string})=>void
  onClose:()=>void
}){
  if(issues.length===0&&dropped.length===0&&automations.length===0&&unmerged.length===0&&!backup)return null
  const first=issues[0]
  const explanation=first?explainFormulaError(first.code):undefined
  return <div className="formula-issue" role="status">
    <AlertTriangle/>
    <div>
      <strong>{automations.length>0
        ?automations.length>1?`자동화 ${automations.length}개가 실패했습니다`:'자동화가 실패했습니다'
        :unmerged.length>0
        ?unmerged.length>1?`병합 ${unmerged.length}개를 풀었습니다`:`${unmerged[0]} 병합을 풀었습니다`
        :dropped.length>0
        ?`편집 ${dropped.length}곳이 반영되지 않았습니다`
        :issues.length>0
          ?issues.length>1?`이 편집으로 수식 ${issues.length}곳이 오류가 되었습니다`:'이 편집으로 수식 한 곳이 오류가 되었습니다'
          :`${backup?.summary}을(를) 삭제했습니다`}</strong>
      <small>{automations.length>0
        ?`${automations[0].message} · 이 편집은 저장되었습니다.`
        :unmerged.length>0
        ?`${unmerged.slice(0,3).join(', ')}: 병합된 칸에 값을 넣으면 병합 전체가 풀립니다. 실행 취소로 되돌릴 수 있습니다.`
        :dropped.length>0
        ?`${dropped.map(cell=>address(cell.row,cell.column)).slice(0,3).join(', ')}: 저장하는 사이에 다른 사람이 그 행이나 열을 지웠습니다.`
        :first
          ?`${address(first.row,first.column)} ${first.code}${explanation?` · ${explanation.summary}`:''}`
          :'실행 취소로는 되돌릴 수 없습니다. 삭제 직전 상태로 복원할 수 있습니다.'}</small>
    </div>
    {first&&<button className="link-button" onClick={()=>onOpen(first)}>보기</button>}
    {backup&&onRevert&&<button className="link-button" onClick={()=>onRevert(backup)}>되돌리기</button>}
    <button className="issue-close" aria-label="알림 닫기" onClick={onClose}><X/></button>
  </div>
}
